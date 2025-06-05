package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// verifyZipContent 验证ZIP文件内容
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
		// 获取testFile相对于tempDir的相对路径
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
		// 获取testFile相对于tempDir的相对路径
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

		// 验证路径转换是否成功
		expectedPathInZip := "win/path/test.txt"
		verifyZipContent(t, zipFile, map[string]string{
			expectedPathInZip: "windows content",
		})
	})
}
