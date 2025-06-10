package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger 是用于测试的mock logger
type MockLogger struct {
	t *testing.T
}

func (m *MockLogger) Debug(format string, v ...interface{}) {}
func (m *MockLogger) Info(format string, v ...interface{})  {}
func (m *MockLogger) Warn(format string, v ...interface{})  {}
func (m *MockLogger) Error(format string, v ...interface{}) {}
func (m *MockLogger) Fatal(format string, v ...interface{}) {}

func TestCalculateFileHash(t *testing.T) {
	logger := &MockLogger{t}
	fs := NewFileScanner(logger)

	t.Run("计算文件哈希", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		content := "test content"
		require.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

		hash, err := fs.CalculateFileHash(testFile)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("处理不存在文件", func(t *testing.T) {
		_, err := fs.CalculateFileHash("nonexistent.txt")
		assert.Error(t, err)
	})
}

func TestLoadIgnoreRules(t *testing.T) {
	logger := &MockLogger{t}
	fs := &FileScanner{logger: logger}

	t.Run("仅使用默认规则", func(t *testing.T) {
		tempDir := t.TempDir()
		ignore := fs.loadIgnoreRules(tempDir)
		require.NotNil(t, ignore)

		// 测试默认规则
		assert.True(t, ignore.MatchesPath("node_modules/file"))
		assert.True(t, ignore.MatchesPath("dist/index.js"))
		assert.True(t, ignore.MatchesPath(".git/config"))
		assert.False(t, ignore.MatchesPath("src/main.go"))
	})

	t.Run("合并.gitignore规则", func(t *testing.T) {
		tempDir := t.TempDir()
		gitignoreContent := "/build\n*.log\n"
		require.NoError(t, os.WriteFile(
			filepath.Join(tempDir, ".gitignore"),
			[]byte(gitignoreContent), 0644))

		ignore := fs.loadIgnoreRules(tempDir)
		require.NotNil(t, ignore)

		assert.True(t, ignore.MatchesPath("build/main.go"))     // .gitignore规则
		assert.True(t, ignore.MatchesPath("src/main.log"))      // .gitignore规则
		assert.True(t, ignore.MatchesPath("node_modules/file")) // 默认规则
		assert.False(t, ignore.MatchesPath("src/main.go"))      // 不应匹配
	})
}

func TestScanDirectory(t *testing.T) {
	logger := &MockLogger{t}
	fs := NewFileScanner(logger)

	setupTestDir := func(t *testing.T) string {
		tempDir := t.TempDir()

		// 创建测试文件结构
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

	t.Run("扫描目录并过滤文件", func(t *testing.T) {
		codebasePath := setupTestDir(t)
		hashTree, err := fs.ScanDirectory(codebasePath)
		require.NoError(t, err)

		// 验证包含的文件
		_, ok := hashTree[filepath.Join("src", "main.go")]
		assert.True(t, ok, "应该包含src/main.go")
		_, ok = hashTree[filepath.Join("src", "pkg", "utils.go")]
		assert.True(t, ok, "应该包含src/pkg/utils.go")

		// 验证排除的文件
		_, ok = hashTree[filepath.Join("build", "main.exe")]
		assert.False(t, ok, "应该排除build/main.exe")

		_, ok = hashTree[filepath.Join("node_modules", "module")]
		assert.False(t, ok, "应该排除node_modules/module")
	})

	t.Run("Windows路径格式兼容", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("仅Windows系统运行此测试")
		}
		codebasePath := setupTestDir(t)
		hashTree, err := fs.ScanDirectory(codebasePath)
		require.NoError(t, err)

		// 使用Windows风格路径验证
		_, ok := hashTree["src\\main.go"]
		assert.True(t, ok, "应该识别Windows路径格式")
	})
}

func benchmarkScanDirectory(t *testing.T, fileCount int) (*MockLogger, ScannerInterface, string) {
	logger := &MockLogger{t}
	fs := NewFileScanner(logger)

	tempDir := t.TempDir()

	// 创建子目录和文件
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

func BenchmarkScanDirectory_10000Files(b *testing.B) {
	t := &testing.T{} // 创建临时testing.T实例
	logger, fs, tempDir := benchmarkScanDirectory(t, 10000)
	_ = logger // 避免未使用变量警告

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fs.ScanDirectory(tempDir)
		require.NoError(b, err)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.ReportMetric(float64(m.TotalAlloc)/1024/1024, "malloc_mb")
	b.ReportMetric(float64(m.HeapInuse)/1024/1024, "heap_inuse_mb")
}

func TestCalculateFileChanges(t *testing.T) {
	logger := &MockLogger{t}
	fs := NewFileScanner(logger)

	t.Run("识别文件变化", func(t *testing.T) {
		local := map[string]string{
			"added.txt":    "hash1",
			"modified.txt": "hash2", // 与remote中不同
		}

		remote := map[string]string{
			"modified.txt": "hash1",
			"deleted.txt":  "hash3",
		}

		changes := fs.CalculateFileChanges(local, remote)

		// 验证变化
		assert.Equal(t, 3, len(changes))

		// 验证新增文件
		var added *FileStatus
		for _, c := range changes {
			if c.Path == "added.txt" {
				added = c
				break
			}
		}
		require.NotNil(t, added)
		assert.Equal(t, FILE_STATUS_ADDED, added.Status)

		// 验证修改文件
		var modified *FileStatus
		for _, c := range changes {
			if c.Path == "modified.txt" {
				modified = c
				break
			}
		}
		require.NotNil(t, modified)
		assert.Equal(t, FILE_STATUS_MODIFIED, modified.Status)

		// 验证删除文件
		var deleted *FileStatus
		for _, c := range changes {
			if c.Path == "deleted.txt" {
				deleted = c
				break
			}
		}
		require.NotNil(t, deleted)
		assert.Equal(t, FILE_STATUS_DELETED, deleted.Status)
	})
}
