package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// verifyZipContent validates zip file contents
func verifyZipContent(t *testing.T, zipPath string, expected map[string]string) {
	t.Helper()

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal("OpenReader failed:", err)
	}
	defer r.Close()

	for name, expectedContent := range expected {
		found := false
		for _, f := range r.File {
			t.Log("zip file: ", f.Name)
			fmt.Println("zip file: ", f.Name)
			if f.Name == name {
				found = true

				rc, err := f.Open()
				if err != nil {
					t.Fatalf("failed to open zip entry %q: %v", name, err)
				}
				defer rc.Close()

				content, err := io.ReadAll(rc)
				if err != nil {
					t.Fatal(err)
				}

				if string(content) != expectedContent {
					t.Errorf("zip content mismatch for %q: expected %q, got %q",
						name, expectedContent, string(content))
				}
				break
			}
		}
		if !found {
			t.Errorf("file %q not found in zip", name)
		}
	}
}

func TestAddFileToZip(t *testing.T) {
	t.Run("successfully add file to zip", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}
		// Value testFile's relative path to tempDir
		testFileRelPath, err := filepath.Rel(tempDir, testFile)
		if err != nil {
			t.Fatal(err)
		}

		zipFile := filepath.Join(tempDir, "test.zip")
		zipFileHandle, err := os.Create(zipFile)
		if err != nil {
			t.Fatal(err)
		}
		defer zipFileHandle.Close()

		zipWriter := zip.NewWriter(zipFileHandle)
		err = AddFileToZip(zipWriter, testFileRelPath, tempDir)
		if err == nil {
			err = zipWriter.Close()
		}
		if err != nil {
			t.Fatalf("AddFileToZip failed: %v", err)
		}

		verifyZipContent(t, zipFile, map[string]string{
			testFileRelPath: "test content",
		})
	})

	t.Run("return error for non-existent file", func(t *testing.T) {
		tempDir := t.TempDir()
		zipFile := filepath.Join(tempDir, "test.zip")
		zipFileHandle, err := os.Create(zipFile)
		if err != nil {
			t.Fatal(err)
		}
		defer zipFileHandle.Close()

		zipWriter := zip.NewWriter(zipFileHandle)
		defer zipWriter.Close()

		err = AddFileToZip(zipWriter, "nonexistent.txt", tempDir)
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("handle windows path correctly", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("only runs on windows")
		}

		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "win\\path\\test.txt")
		if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(testFile, []byte("windows content"), 0644); err != nil {
			t.Fatal(err)
		}
		// Value testFile's relative path to tempDir
		testFileRelPath, err := filepath.Rel(tempDir, testFile)
		if err != nil {
			t.Fatal(err)
		}

		zipFile := filepath.Join(tempDir, "windows.zip")
		zipFileHandle, err := os.Create(zipFile)
		if err != nil {
			t.Fatal(err)
		}
		defer zipFileHandle.Close()

		zipWriter := zip.NewWriter(zipFileHandle)
		err = AddFileToZip(zipWriter, testFileRelPath, tempDir)
		if err == nil {
			err = zipWriter.Close()
		}
		if err != nil {
			t.Fatalf("AddFileToZip failed: %v", err)
		}

		// Verify path conversion
		expectedPathInZip := "win/path/test.txt"
		verifyZipContent(t, zipFile, map[string]string{
			expectedPathInZip: "windows content",
		})
	})
}

func TestCalculateFileHash(t *testing.T) {
	t.Run("calculate file hash", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}
		hash, err := CalculateFileHash(testFile)
		if err != nil {
			t.Fatal(err)
		}
		require.NoError(t, err)
		assert.NotEmpty(t, hash)

	})

	t.Run("file not found", func(t *testing.T) {
		_, err := CalculateFileHash("nonexistentfile.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})
}

func TestCalculateFileChanges(t *testing.T) {
	t.Run("new files only", func(t *testing.T) {
		local := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}
		remote := map[string]string{}

		changes := CalculateFileChanges(local, remote)

		require.Len(t, changes, 2)

		// Create a map for easier lookup
		changeMap := make(map[string]*FileStatus)
		for _, change := range changes {
			changeMap[change.Path] = change
		}

		// Verify file1.txt
		assert.Contains(t, changeMap, "file1.txt")
		assert.Equal(t, "hash1", changeMap["file1.txt"].Hash)
		assert.Equal(t, FILE_STATUS_ADDED, changeMap["file1.txt"].Status)

		// Verify file2.txt
		assert.Contains(t, changeMap, "file2.txt")
		assert.Equal(t, "hash2", changeMap["file2.txt"].Hash)
		assert.Equal(t, FILE_STATUS_ADDED, changeMap["file2.txt"].Status)
	})

	t.Run("modified files only", func(t *testing.T) {
		local := map[string]string{
			"file1.txt": "hash1_new",
			"file2.txt": "hash2_new",
		}
		remote := map[string]string{
			"file1.txt": "hash1_old",
			"file2.txt": "hash2_old",
		}

		changes := CalculateFileChanges(local, remote)

		require.Len(t, changes, 2)

		// Create a map for easier lookup
		changeMap := make(map[string]*FileStatus)
		for _, change := range changes {
			changeMap[change.Path] = change
		}

		// Verify file1.txt
		assert.Contains(t, changeMap, "file1.txt")
		assert.Equal(t, "hash1_new", changeMap["file1.txt"].Hash)
		assert.Equal(t, FILE_STATUS_MODIFIED, changeMap["file1.txt"].Status)

		// Verify file2.txt
		assert.Contains(t, changeMap, "file2.txt")
		assert.Equal(t, "hash2_new", changeMap["file2.txt"].Hash)
		assert.Equal(t, FILE_STATUS_MODIFIED, changeMap["file2.txt"].Status)
	})

	t.Run("deleted files only", func(t *testing.T) {
		local := map[string]string{}
		remote := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}

		changes := CalculateFileChanges(local, remote)

		require.Len(t, changes, 2)

		// Create a map for easier lookup
		changeMap := make(map[string]*FileStatus)
		for _, change := range changes {
			changeMap[change.Path] = change
		}

		// Verify file1.txt
		assert.Contains(t, changeMap, "file1.txt")
		assert.Equal(t, "", changeMap["file1.txt"].Hash)
		assert.Equal(t, FILE_STATUS_DELETED, changeMap["file1.txt"].Status)

		// Verify file2.txt
		assert.Contains(t, changeMap, "file2.txt")
		assert.Equal(t, "", changeMap["file2.txt"].Hash)
		assert.Equal(t, FILE_STATUS_DELETED, changeMap["file2.txt"].Status)
	})

	t.Run("mixed changes", func(t *testing.T) {
		local := map[string]string{
			"new_file.txt":       "new_hash",
			"modified_file.txt":  "modified_hash",
			"unchanged_file.txt": "unchanged_hash",
		}
		remote := map[string]string{
			"modified_file.txt":  "old_hash",
			"unchanged_file.txt": "unchanged_hash",
			"deleted_file.txt":   "deleted_hash",
		}

		changes := CalculateFileChanges(local, remote)

		// Should have 3 changes: new, modified, deleted
		require.Len(t, changes, 3)

		// Find and verify each change
		var newFound, modifiedFound, deletedFound bool

		for _, change := range changes {
			switch change.Path {
			case "new_file.txt":
				assert.Equal(t, "new_hash", change.Hash)
				assert.Equal(t, FILE_STATUS_ADDED, change.Status)
				newFound = true
			case "modified_file.txt":
				assert.Equal(t, "modified_hash", change.Hash)
				assert.Equal(t, FILE_STATUS_MODIFIED, change.Status)
				modifiedFound = true
			case "deleted_file.txt":
				assert.Equal(t, "", change.Hash)
				assert.Equal(t, FILE_STATUS_DELETED, change.Status)
				deletedFound = true
			}
		}

		assert.True(t, newFound, "New file change not found")
		assert.True(t, modifiedFound, "Modified file change not found")
		assert.True(t, deletedFound, "Deleted file change not found")
	})

	t.Run("no changes", func(t *testing.T) {
		local := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}
		remote := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}

		changes := CalculateFileChanges(local, remote)

		assert.Empty(t, changes)
	})

	t.Run("empty maps", func(t *testing.T) {
		local := map[string]string{}
		remote := map[string]string{}

		changes := CalculateFileChanges(local, remote)

		assert.Empty(t, changes)
	})
}

func TestCalculateFileChangesWithoutDelete(t *testing.T) {
	t.Run("new files only", func(t *testing.T) {
		local := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}
		remote := map[string]string{}

		changes := CalculateFileChangesWithoutDelete(local, remote)

		require.Len(t, changes, 2)

		// Create a map for easier lookup
		changeMap := make(map[string]*FileStatus)
		for _, change := range changes {
			changeMap[change.Path] = change
		}

		// Verify file1.txt
		assert.Contains(t, changeMap, "file1.txt")
		assert.Equal(t, "hash1", changeMap["file1.txt"].Hash)
		assert.Equal(t, FILE_STATUS_ADDED, changeMap["file1.txt"].Status)

		// Verify file2.txt
		assert.Contains(t, changeMap, "file2.txt")
		assert.Equal(t, "hash2", changeMap["file2.txt"].Hash)
		assert.Equal(t, FILE_STATUS_ADDED, changeMap["file2.txt"].Status)
	})

	t.Run("modified files only", func(t *testing.T) {
		local := map[string]string{
			"file1.txt": "hash1_new",
			"file2.txt": "hash2_new",
		}
		remote := map[string]string{
			"file1.txt": "hash1_old",
			"file2.txt": "hash2_old",
		}

		changes := CalculateFileChangesWithoutDelete(local, remote)

		require.Len(t, changes, 2)

		// Create a map for easier lookup
		changeMap := make(map[string]*FileStatus)
		for _, change := range changes {
			changeMap[change.Path] = change
		}

		// Verify file1.txt
		assert.Contains(t, changeMap, "file1.txt")
		assert.Equal(t, "hash1_new", changeMap["file1.txt"].Hash)
		assert.Equal(t, FILE_STATUS_MODIFIED, changeMap["file1.txt"].Status)

		// Verify file2.txt
		assert.Contains(t, changeMap, "file2.txt")
		assert.Equal(t, "hash2_new", changeMap["file2.txt"].Hash)
		assert.Equal(t, FILE_STATUS_MODIFIED, changeMap["file2.txt"].Status)
	})

	t.Run("deleted files should be ignored", func(t *testing.T) {
		local := map[string]string{}
		remote := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}

		changes := CalculateFileChangesWithoutDelete(local, remote)

		assert.Empty(t, changes, "Deleted files should not be included in changes")
	})

	t.Run("mixed changes without delete", func(t *testing.T) {
		local := map[string]string{
			"new_file.txt":       "new_hash",
			"modified_file.txt":  "modified_hash",
			"unchanged_file.txt": "unchanged_hash",
		}
		remote := map[string]string{
			"modified_file.txt":  "old_hash",
			"unchanged_file.txt": "unchanged_hash",
			"deleted_file.txt":   "deleted_hash",
		}

		changes := CalculateFileChangesWithoutDelete(local, remote)

		// Should have 2 changes: new and modified (deleted should be ignored)
		require.Len(t, changes, 2)

		// Find and verify each change
		var newFound, modifiedFound bool

		for _, change := range changes {
			switch change.Path {
			case "new_file.txt":
				assert.Equal(t, "new_hash", change.Hash)
				assert.Equal(t, FILE_STATUS_ADDED, change.Status)
				newFound = true
			case "modified_file.txt":
				assert.Equal(t, "modified_hash", change.Hash)
				assert.Equal(t, FILE_STATUS_MODIFIED, change.Status)
				modifiedFound = true
			case "deleted_file.txt":
				t.Errorf("Deleted file should not be included in changes")
			}
		}

		assert.True(t, newFound, "New file change not found")
		assert.True(t, modifiedFound, "Modified file change not found")
	})

	t.Run("no changes", func(t *testing.T) {
		local := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}
		remote := map[string]string{
			"file1.txt": "hash1",
			"file2.txt": "hash2",
		}

		changes := CalculateFileChangesWithoutDelete(local, remote)

		assert.Empty(t, changes)
	})

	t.Run("empty maps", func(t *testing.T) {
		local := map[string]string{}
		remote := map[string]string{}

		changes := CalculateFileChangesWithoutDelete(local, remote)

		assert.Empty(t, changes)
	})
}
