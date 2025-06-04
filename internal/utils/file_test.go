package utils

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreateZipFile(t *testing.T) {
	t.Run("successful zip creation with single file", func(t *testing.T) {
		// 准备测试文件
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}

		zipFile := filepath.Join(tempDir, "test.zip")

		// 调用函数
		err := CreateZipFile([]string{testFile}, zipFile)
		if err != nil {
			t.Fatalf("CreateZipFile failed: %v", err)
		}

		// 验证ZIP文件
		if !fileExists(zipFile) {
			t.Error("zip file was not created")
		}

		verifyZipContent(t, zipFile, map[string]string{
			testFile: "test content",
		})
	})

	t.Run("create zip with multiple files", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建多个测试文件
		file1 := filepath.Join(tempDir, "file1.txt")
		file2 := filepath.Join(tempDir, "file2.txt")
		os.WriteFile(file1, []byte("file1 content"), 0644)
		os.WriteFile(file2, []byte("file2 content"), 0644)

		zipFile := filepath.Join(tempDir, "multi.zip")

		err := CreateZipFile([]string{file1, file2}, zipFile)
		if err != nil {
			t.Fatal(err)
		}

		verifyZipContent(t, zipFile, map[string]string{
			file1: "file1 content",
			file2: "file2 content",
		})
	})

	t.Run("empty file list should create empty zip", func(t *testing.T) {
		tempDir := t.TempDir()
		zipFile := filepath.Join(tempDir, "empty.zip")

		err := CreateZipFile([]string{}, zipFile)
		if err != nil {
			t.Fatal(err)
		}

		if !fileExists(zipFile) {
			t.Error("empty zip file was not created")
		}

		// 验证空ZIP文件
		r, err := zip.OpenReader(zipFile)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		if len(r.File) != 0 {
			t.Errorf("expected empty zip, got %d files", len(r.File))
		}
	})

	t.Run("non-existent files should return error", func(t *testing.T) {
		tempDir := t.TempDir()
		zipFile := filepath.Join(tempDir, "error.zip")

		err := CreateZipFile([]string{"nonexistent.txt"}, zipFile)
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("invalid zip file path should return error", func(t *testing.T) {
		err := CreateZipFile([]string{"somefile.txt"}, "/invalid/path/zipfile.zip")
		if err == nil {
			t.Error("expected error for invalid zip file path")
		}
	})

	t.Run("should handle directory input correctly", func(t *testing.T) {
		tempDir := t.TempDir()

		// 创建测试目录和文件
		subDir := filepath.Join(tempDir, "subdir")
		os.Mkdir(subDir, 0755)
		fileInDir := filepath.Join(subDir, "file.txt")
		os.WriteFile(fileInDir, []byte("dir content"), 0644)

		zipFile := filepath.Join(tempDir, "dir.zip")

		err := CreateZipFile([]string{subDir, fileInDir}, zipFile)
		if err != nil {
			t.Fatal(err)
		}

		verifyZipContent(t, zipFile, map[string]string{
			fileInDir: "dir content",
		})
	})
}

// verifyZipContent 验证ZIP文件内容
func verifyZipContent(t *testing.T, zipPath string, expected map[string]string) {
	t.Helper()

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	for name, expectedContent := range expected {
		found := false
		for _, f := range r.File {
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestAddFileToZip(t *testing.T) {
	t.Run("successfully add file to zip", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}

		zipFile := filepath.Join(tempDir, "test.zip")
		zipFileHandle, err := os.Create(zipFile)
		if err != nil {
			t.Fatal(err)
		}
		defer zipFileHandle.Close()

		zipWriter := zip.NewWriter(zipFileHandle)
		defer zipWriter.Close()

		err = AddFileToZip(zipWriter, testFile, tempDir)
		if err != nil {
			t.Fatalf("AddFileToZip failed: %v", err)
		}

		verifyZipContent(t, zipFile, map[string]string{
			testFile: "test content",
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

		zipFile := filepath.Join(tempDir, "windows.zip")
		zipFileHandle, err := os.Create(zipFile)
		if err != nil {
			t.Fatal(err)
		}
		defer zipFileHandle.Close()

		zipWriter := zip.NewWriter(zipFileHandle)
		defer zipWriter.Close()

		err = AddFileToZip(zipWriter, testFile, tempDir)
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
