package repository

import (
	"codebase-indexer/internal/config"
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
		// FileIgnorePatterns:   []string{".*", "*.bak"},
		FolderIgnorePatterns: []string{".*", "build/", "dist/", "node_modules/", "vendor/", "!.costrict/wiki/"},
		FileIncludePatterns:  []string{".go"},
		// MaxFileSizeMB:  10,
		MaxFileSizeKB: 100,
		MaxFileCount:  10000,
	}
)

func TestLoadIgnoreRules(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Info", "no .gitignore file: %v", mock.Anything).Return()
	logger.On("Info", "no .coignore file: %v", mock.Anything).Return()
	fs := &FileScanner{scannerConfig: scannerConfig, logger: logger}

	t.Run("Use default rules only", func(t *testing.T) {
		tempDir := t.TempDir()
		ignore := fs.LoadIgnoreRules(tempDir)
		logger.AssertCalled(t, "Info", "no .gitignore file: %v", mock.Anything)
		logger.AssertCalled(t, "Info", "no .coignore file: %v", mock.Anything)
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

	t.Run("Test .costrict/wiki directory inclusion with ! pattern", func(t *testing.T) {
		tempDir := t.TempDir()
		fs := &FileScanner{scannerConfig: scannerConfig, logger: logger}

		ignore := fs.LoadIgnoreRules(tempDir)
		require.NotNil(t, ignore)

		// Test that .costrict/wiki directory is NOT ignored due to ! pattern
		assert.False(t, ignore.MatchesPath(".costrict/wiki/"), "should not ignore .costrict/wiki/ directory")
		assert.False(t, ignore.MatchesPath(".costrict/wiki/file.md"), "should not ignore files inside .costrict/wiki/ directory")

		// Test that other dot directories are still ignored
		assert.True(t, ignore.MatchesPath(".git/"), "should ignore .git/ directory")
		assert.True(t, ignore.MatchesPath(".idea/"), "should ignore .idea/ directory")

		// Test that .costrict directory without wiki is still ignored
		assert.True(t, ignore.MatchesPath(".costrict/other/"), "should ignore .costrict/other/ directory")
		assert.True(t, ignore.MatchesPath(".costrict/config.json"), "should ignore .costrict/config.json file")
	})
}

func TestScanDirectory(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()
	logger.On("Debug", mock.Anything, mock.Anything).Return()
	_ = NewFileScanner(logger)
	// TODO 测试待校验
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
		logger := &mocks.MockLogger{}
		logger.On("Warn", "reached maximum file count limit: %d, stopping scan, time taken: %v", mock.Anything, mock.Anything).Return()
		logger.On("Info", "starting codebase scan: %s", mock.Anything).Return()
		logger.On("Debug", "skipping file excluded by ignore: %s", mock.Anything).Return()
		logger.On("Debug", "skipping ignored directory: %s", mock.Anything).Return()
		logger.On("Info", mock.Anything, mock.Anything).Return()
		logger.On("Debug", mock.Anything, mock.Anything).Return()

		codebasePath := setupTestDir(t)
		fs := &FileScanner{scannerConfig: scannerConfig, logger: logger}

		// Debug: check file include patterns
		includeFiles := fs.LoadIncludeFiles()
		t.Logf("File include patterns: %v", includeFiles)

		// Debug: check ignore rules
		ignore := fs.LoadIgnoreRules(codebasePath)
		t.Logf("Ignore rules loaded: %v", ignore != nil)
		if ignore != nil {
			t.Logf("Ignore patterns:")
			// Note: We can't easily inspect the internal patterns of the ignore object
			// but we can test specific paths
			t.Logf("  src/main.go matches: %v", ignore.MatchesPath("src/main.go"))
			t.Logf("  src/pkg/utils.go matches: %v", ignore.MatchesPath("src/pkg/utils.go"))
		}

		// Debug: check scanner config
		t.Logf("MaxFileSizeKB: %d", scannerConfig.MaxFileSizeKB)
		t.Logf("MaxFileCount: %d", scannerConfig.MaxFileCount)

		// Debug: check actual file sizes
		mainGoPath := filepath.Join(codebasePath, "src", "main.go")
		utilsGoPath := filepath.Join(codebasePath, "src", "pkg", "utils.go")
		if info, err := os.Stat(mainGoPath); err == nil {
			t.Logf("src/main.go size: %d bytes", info.Size())
		}
		if info, err := os.Stat(utilsGoPath); err == nil {
			t.Logf("src/pkg/utils.go size: %d bytes", info.Size())
		}

		ignoreConfig := fs.LoadIgnoreConfig(codebasePath)
		hashTree, err := fs.ScanCodebase(ignoreConfig, codebasePath)
		logger.AssertCalled(t, "Info", "starting codebase scan: %s", mock.Anything)
		require.NoError(t, err)

		// Debug: print all files in hashTree
		t.Logf("Files in hashTree:")
		for path := range hashTree {
			t.Logf("  %s", path)
		}

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
		logger := &mocks.MockLogger{}
		logger.On("Warn", "reached maximum file count limit: %d, stopping scan, time taken: %v", mock.Anything, mock.Anything).Return()
		logger.On("Info", "starting codebase scan: %s", mock.Anything).Return()
		logger.On("Debug", "skipping file excluded by ignore: %s", mock.Anything).Return()
		logger.On("Debug", "skipping ignored directory: %s", mock.Anything).Return()
		logger.On("Info", mock.Anything, mock.Anything).Return()
		logger.On("Debug", mock.Anything, mock.Anything).Return()

		codebasePath := setupTestDir(t)
		fs := &FileScanner{scannerConfig: scannerConfig, logger: logger}
		ignoreConfig := fs.LoadIgnoreConfig(codebasePath)
		hashTree, err := fs.ScanCodebase(ignoreConfig, codebasePath)
		require.NoError(t, err)

		// Verify with Windows-style paths
		windowsPath := filepath.Join("src", "main.go")
		_, ok := hashTree[windowsPath]
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
	logger.On("Info", mock.Anything, mock.Anything).Maybe().Return()
	logger.On("Warn", mock.Anything, mock.Anything).Maybe().Return()
	logger.On("Error", mock.Anything, mock.Anything).Maybe().Return()
	logger.On("Debug", mock.Anything, mock.Anything).Maybe().Return()
	_ = logger // Avoid unused variable warning

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ignoreConfig := fs.LoadIgnoreConfig(tempDir)
		_, err := fs.ScanCodebase(ignoreConfig, tempDir)
		require.NoError(b, err)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.ReportMetric(float64(m.TotalAlloc)/1024/1024, "malloc_mb")
	b.ReportMetric(float64(m.HeapInuse)/1024/1024, "heap_inuse_mb")
}
