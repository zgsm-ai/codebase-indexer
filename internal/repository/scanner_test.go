package repository

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/utils"
	"codebase-indexer/test/mocks"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	scannerConfig = &config.ScannerConfig{
		// IgnorePatterns: []string{".git", ".idea", "node_modules/", "vendor/", "dist/", "build/"},
		FileIgnorePatterns:   []string{".*", "*.bak"},
		FolderIgnorePatterns: []string{".*", "build/", "dist/", "node_modules/", "vendor/"},
		// MaxFileSizeMB:  10,
		MaxFileSizeKB: 100,
	}
)

func TestCalculateFileHash(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	fs := NewFileScanner(logger)

	t.Run("Calculate file hash", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		content := "test content"
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

		hash, err := fs.CalculateFileHash(testFile)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		logger.AssertCalled(t, "Debug", "file hash calculated for %s, time taken: %v, hash: %s", mock.Anything, mock.Anything)
	})

	t.Run("Handle nonexistent file", func(t *testing.T) {
		_, err := fs.CalculateFileHash("nonexistent.txt")
		assert.Error(t, err)
	})
}

func TestLoadIgnoreRules(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Warn", mock.Anything, mock.Anything).Return()
	fs := &FileScanner{scannerConfig: scannerConfig, logger: logger}

	t.Run("Use default rules only", func(t *testing.T) {
		tempDir := t.TempDir()
		ignore := fs.LoadIgnoreRules(tempDir)
		logger.AssertCalled(t, "Warn", "Failed to read .gitignore file: %v", mock.Anything)
		require.NotNil(t, ignore)

		// Test default rules
		assert.True(t, ignore.MatchesPath("node_modules/file"))
		assert.True(t, ignore.MatchesPath("dist/index.js"))
		assert.True(t, ignore.MatchesPath(".git/config"))
		assert.False(t, ignore.MatchesPath("src/main.go"))
	})

	t.Run("Merge gitignore rules", func(t *testing.T) {
		tempDir := t.TempDir()
		gitignoreContent := "/build\n*.log\n"
		require.NoError(t, os.WriteFile(
			filepath.Join(tempDir, ".gitignore"),
			[]byte(gitignoreContent), 0644))

		ignore := fs.LoadIgnoreRules(tempDir)
		require.NotNil(t, ignore)

		assert.True(t, ignore.MatchesPath("build/main.go"))     // .gitignore rule
		assert.True(t, ignore.MatchesPath("src/main.log"))      // .gitignore rule
		assert.True(t, ignore.MatchesPath("node_modules/file")) // Default rule
		assert.False(t, ignore.MatchesPath("src/main.go"))      // Should not match
	})
}

func TestScanDirectory(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	fs := NewFileScanner(logger)

	setupTestDir := func(t *testing.T) string {
		tempDir := t.TempDir()

		// Create test file structure
		dirs := []string{"src", filepath.Join("src", "pkg"), "build", "dist", "node_modules"}
		for _, dir := range dirs {
			require.NoError(t, os.MkdirAll(filepath.Join(tempDir, dir), 0755))
		}

		files := map[string]string{
			filepath.Join("src", "main.go"):         "package main",
			filepath.Join("src", "pkg", "utils.go"): "package utils",
			filepath.Join("build", "main.exe"):      "binary content",
			".gitignore":                            "/build\n*.exe\n",
			filepath.Join("node_modules", "module"): "module content",
		}
		for path, content := range files {
			require.NoError(t, os.WriteFile(
				filepath.Join(tempDir, path),
				[]byte(content), 0644))
		}

		return tempDir
	}

	t.Run("Scan codebase and filter files", func(t *testing.T) {
		codebasePath := setupTestDir(t)
		hashTree, err := fs.ScanCodebase(codebasePath)
		logger.AssertCalled(t, "Info", "starting codebase scan: %s", mock.Anything)
		require.NoError(t, err)

		// Verify included files
		_, ok := hashTree[filepath.Join("src", "main.go")]
		assert.True(t, ok, "should include src/main.go")
		_, ok = hashTree[filepath.Join("src", "pkg", "utils.go")]
		assert.True(t, ok, "should include src/pkg/utils.go")

		// Verify excluded files
		_, ok = hashTree[filepath.Join("build", "main.exe")]
		assert.False(t, ok, "should exclude build/main.exe")

		_, ok = hashTree[filepath.Join("node_modules", "module")]
		assert.False(t, ok, "should exclude node_modules/module")
	})

	t.Run("Windows path compatibility", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("skip: only run on Windows system")
		}
		codebasePath := setupTestDir(t)
		hashTree, err := fs.ScanCodebase(codebasePath)
		require.NoError(t, err)

		// Verify with Windows-style paths
		_, ok := hashTree["src\\main.go"]
		assert.True(t, ok, "should recognize Windows path format")
	})
}

func benchmarkScanCodebase(t *testing.T, fileCount int) (*mocks.MockLogger, ScannerInterface, string) {
	logger := &mocks.MockLogger{}
	fs := NewFileScanner(logger)

	tempDir := t.TempDir()

	// Create subdirectories and files
	for i := 0; i < fileCount/10; i++ {
		subDir := filepath.Join(tempDir, "dir"+strconv.Itoa(i))
		require.NoError(t, os.MkdirAll(subDir, 0755))

		for j := 0; j < 10; j++ {
			filePath := filepath.Join(subDir, "file"+strconv.Itoa(j)+".txt")
			require.NoError(t, os.WriteFile(filePath, []byte("test"), 0644))
		}
	}

	return logger, fs, tempDir
}

func BenchmarkScanCodebase_10000Files(b *testing.B) {
	t := &testing.T{} // Create temp testing.T instance
	logger, fs, tempDir := benchmarkScanCodebase(t, 10000)
	_ = logger // Avoid unused variable warning

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fs.ScanCodebase(tempDir)
		require.NoError(b, err)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.ReportMetric(float64(m.TotalAlloc)/1024/1024, "malloc_mb")
	b.ReportMetric(float64(m.HeapInuse)/1024/1024, "heap_inuse_mb")
}

func TestCalculateFileChanges(t *testing.T) {
	logger := &mocks.MockLogger{}
	fs := NewFileScanner(logger)

	t.Run("Detect file changes", func(t *testing.T) {
		local := map[string]string{
			"added.txt":    "hash1",
			"modified.txt": "hash2", // Different from remote
		}

		remote := map[string]string{
			"modified.txt": "hash1",
			"deleted.txt":  "hash3",
		}

		changes := fs.CalculateFileChanges(local, remote)

		// Verify changes
		assert.Equal(t, 3, len(changes))

		// Verify added file
		var added *utils.FileStatus
		for _, c := range changes {
			if c.Path == "added.txt" {
				added = c
				break
			}
		}
		require.NotNil(t, added)
		assert.Equal(t, utils.FILE_STATUS_ADDED, added.Status)

		// Verify modified file
		var modified *utils.FileStatus
		for _, c := range changes {
			if c.Path == "modified.txt" {
				modified = c
				break
			}
		}
		require.NotNil(t, modified)
		assert.Equal(t, utils.FILE_STATUS_MODIFIED, modified.Status)

		// Verify deleted file
		var deleted *utils.FileStatus
		for _, c := range changes {
			if c.Path == "deleted.txt" {
				deleted = c
				break
			}
		}
		require.NotNil(t, deleted)
		assert.Equal(t, utils.FILE_STATUS_DELETED, deleted.Status)
	})
}
