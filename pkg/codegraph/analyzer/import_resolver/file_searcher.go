package import_resolver

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileSearcher 文件搜索器（带索引）
type FileSearcher struct {
	projectPath string
	fileIndex   *FileIndex
	mu          sync.RWMutex
}

// FileIndex 文件索引
type FileIndex struct {
	// 快速查找：文件名 -> 完整路径列表
	nameToPath map[string][]string

	// 目录索引：目录路径 -> 文件列表
	dirToFiles map[string][]string

	// 扩展名索引：扩展名 -> 文件列表
	extToFiles map[string][]string
}

// NewFileSearcher 创建文件搜索器
func NewFileSearcher(projectPath string) *FileSearcher {
	return &FileSearcher{
		projectPath: projectPath,
		fileIndex: &FileIndex{
			nameToPath: make(map[string][]string),
			dirToFiles: make(map[string][]string),
			extToFiles: make(map[string][]string),
		},
	}
}

// BuildIndex 构建文件索引
func (fs *FileSearcher) BuildIndex() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// 清空现有索引
	fs.fileIndex = &FileIndex{
		nameToPath: make(map[string][]string),
		dirToFiles: make(map[string][]string),
		extToFiles: make(map[string][]string),
	}

	// 遍历项目目录
	return filepath.Walk(fs.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续遍历
		}

		// 跳过目录和隐藏文件
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过常见的排除目录
		if shouldSkipPath(path) {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(fs.projectPath, path)
		if err != nil {
			return nil
		}

		// 索引文件名
		fileName := info.Name()
		fs.fileIndex.nameToPath[fileName] = append(fs.fileIndex.nameToPath[fileName], relPath)

		// 索引目录
		dir := filepath.Dir(relPath)
		fs.fileIndex.dirToFiles[dir] = append(fs.fileIndex.dirToFiles[dir], relPath)

		// 索引扩展名
		ext := filepath.Ext(fileName)
		if ext != "" {
			fs.fileIndex.extToFiles[ext] = append(fs.fileIndex.extToFiles[ext], relPath)
		}

		return nil
	})
}

// shouldSkipPath 判断是否应该跳过路径
func shouldSkipPath(path string) bool {
	skipDirs := []string{
		"node_modules",
		".git",
		".idea",
		".vscode",
		"vendor",
		"target",
		"build",
		"dist",
		"__pycache__",
		".pytest_cache",
	}

	for _, skip := range skipDirs {
		if strings.Contains(path, string(filepath.Separator)+skip+string(filepath.Separator)) ||
			strings.HasSuffix(path, string(filepath.Separator)+skip) {
			return true
		}
	}
	return false
}

// ListFilesInDir 列出目录中的文件
func (fs *FileSearcher) ListFilesInDir(dir string, ext string) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// 规范化目录路径
	dir = filepath.Clean(dir)
	if filepath.IsAbs(dir) {
		relDir, err := filepath.Rel(fs.projectPath, dir)
		if err != nil {
			return nil, err
		}
		dir = relDir
	}

	files, ok := fs.fileIndex.dirToFiles[dir]
	if !ok {
		// 如果索引中没有，直接读取目录
		return fs.listFilesInDirDirect(dir, ext)
	}

	// 过滤扩展名
	if ext == "" {
		return files, nil
	}

	var result []string
	for _, file := range files {
		if strings.HasSuffix(file, ext) {
			result = append(result, file)
		}
	}

	return result, nil
}

// listFilesInDirDirect 直接读取目录（不使用索引）
func (fs *FileSearcher) listFilesInDirDirect(dir string, ext string) ([]string, error) {
	fullPath := filepath.Join(fs.projectPath, dir)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if ext == "" || strings.HasSuffix(entry.Name(), ext) {
			relPath := filepath.Join(dir, entry.Name())
			result = append(result, relPath)
		}
	}

	return result, nil
}

// FindFilesByName 根据文件名查找
func (fs *FileSearcher) FindFilesByName(name string) []string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	paths, ok := fs.fileIndex.nameToPath[name]
	if ok {
		return paths
	}

	return nil
}

// FindFilesByExtension 根据扩展名查找所有文件
func (fs *FileSearcher) FindFilesByExtension(ext string) []string {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	paths, ok := fs.fileIndex.extToFiles[ext]
	if ok {
		// 返回副本以避免并发修改
		result := make([]string, len(paths))
		copy(result, paths)
		return result
	}

	return nil
}

// FileExists 检查文件是否存在
func (fs *FileSearcher) FileExists(relPath string) bool {
	fullPath := filepath.Join(fs.projectPath, relPath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// FindInDirs 在指定目录列表中查找文件
func (fs *FileSearcher) FindInDirs(filename string, dirs []string) (string, error) {
	for _, dir := range dirs {
		filePath := filepath.Join(dir, filename)
		if fs.FileExists(filePath) {
			return filePath, nil
		}
	}
	return "", nil
}

// TryExtensions 尝试多个扩展名
func (fs *FileSearcher) TryExtensions(basePath string, extensions []string) []string {
	var result []string

	for _, ext := range extensions {
		path := basePath + ext
		if fs.FileExists(path) {
			result = append(result, path)
		}
	}

	return result
}
