package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Processor 文件处理器
type Processor struct {
	excludedDirs  []string
	excludedFiles []string
	config        *SimpleConfig
	logger        logger.Logger
}

// NewProcessor 创建新的文件处理器
func NewProcessor(config *SimpleConfig, logger logger.Logger) *Processor {
	return &Processor{
		excludedDirs:  []string{".git", "node_modules", "vendor", "__pycache__"},
		excludedFiles: []string{"*.min.js", "*.min.css", "*.log", "*.tmp"},
		config:        config,
		logger:        logger,
	}
}

// ProcessRepository 处理仓库目录
func (p *Processor) ProcessRepository(ctx context.Context, repoPath string) ([]*FileMeta, error) {
	var files []*FileMeta
	var totalFiles int
	var processedFiles int
	var successFiles int
	var failedFiles int

	p.logger.Info("Starting to process repository: %s", repoPath)

	// 遍历目录
	err := filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			// 跳过隐藏目录和常见排除目录
			dirName := d.Name()
			if strings.HasPrefix(dirName, ".") ||
				dirName == "node_modules" ||
				dirName == "vendor" ||
				dirName == "target" ||
				dirName == "build" ||
				dirName == "dist" ||
				dirName == "__pycache__" ||
				dirName == ".git" {
				return filepath.SkipDir
			}

			// 检查是否在排除目录列表中
			for _, excludedDir := range p.excludedDirs {
				if dirName == excludedDir {
					return filepath.SkipDir
				}
			}

			return nil
		}

		// 统计总文件数
		totalFiles++

		// 检查文件是否应该包含
		if !p.shouldIncludeFile(path) {
			return nil
		}

		// 收集文件元数据（不读取内容）
		fileMeta, err := p.collectFileMeta(path)
		if err != nil {
			// 记录错误但继续处理其他文件
			failedFiles++
			p.logger.Warn("Failed to collect file metadata: %s - %v", path, err)
			return nil
		}

		files = append(files, fileMeta)
		successFiles++
		processedFiles++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// 限制文件数量 - 使用配置中的限制
	originalFileCount := len(files)
	if len(files) > p.config.MaxFiles {
		files = files[:p.config.MaxFiles]
		p.logger.Info("File count exceeds limit, truncated: %d -> %d", originalFileCount, len(files))
	}

	// 打印最终统计信息
	p.logger.Info("File processing completed: total_files=%d, processed=%d, success=%d, failed=%d", totalFiles, processedFiles, successFiles, failedFiles)
	p.logger.Info("Final included file count: %d", len(files))

	return files, nil
}

// shouldIncludeFile 判断文件是否应该包含
func (p *Processor) shouldIncludeFile(filePath string) bool {
	name := filepath.Base(filePath)
	ext := strings.ToLower(filepath.Ext(name))

	// 检查是否在排除文件列表中
	for _, excludedFile := range p.excludedFiles {
		// 支持文件名匹配和扩展名匹配
		if name == excludedFile || ext == excludedFile || strings.Contains(name, excludedFile) {
			return false
		}
	}

	//// 排除测试文件
	//if strings.Contains(name, "test") || strings.Contains(name, "_test") {
	//	return false
	//}
	//
	//// 排除文档文件
	//if ext == ".txt" || ext == ".rst" {
	//	return false
	//}

	// 排除配置文件
	configExts := []string{".env", ".lock"}
	configFiles := []string{"license"}

	for _, configExt := range configExts {
		if ext == configExt {
			return false
		}
	}

	for _, configFile := range configFiles {
		if strings.ToLower(name) == configFile {
			return false
		}
	}

	// 排除二进制文件
	binaryExts := []string{".exe", ".dll", ".so", ".dylib", ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".mp3", ".mp4", ".avi", ".mov", ".wmv", ".flv"}
	for _, binaryExt := range binaryExts {
		if ext == binaryExt {
			return false
		}
	}

	// 排除临时文件
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "#") || strings.HasPrefix(name, "~") || strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, ".swp") {
		return false
	}

	return true
}

// readFileContent 读取文件内容
func (p *Processor) readFileContent(filePath string) (*FileContent, error) {
	// 获取文件信息
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 检查文件大小 - 使用配置中的限制
	if info.Size() > p.config.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum limit %d", info.Size(), p.config.MaxFileSize)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 检测是否为二进制文件
	isBinary := p.isBinaryFile(data)

	var content string
	var encoding string

	if isBinary {
		content = "[Binary file content not displayed]"
		encoding = "binary"
	} else {
		content = string(data)
		encoding = "utf-8"
	}

	// 创建文件信息
	fileInfo := &FileInfo{
		Path:      filePath,
		Name:      info.Name(),
		Size:      info.Size(),
		Extension: strings.TrimPrefix(filepath.Ext(info.Name()), "."),
		Language:  p.detectLanguage(filePath),
		IsBinary:  isBinary,
		Metadata:  make(map[string]string),
	}

	return &FileContent{
		Info:     fileInfo,
		Content:  content,
		Encoding: encoding,
		Size:     int64(len(content)),
	}, nil
}

// isBinaryFile 检测是否为二进制文件
func (p *Processor) isBinaryFile(data []byte) bool {
	// 简单的二进制文件检测：检查是否包含null字节
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// detectLanguage 检测文件语言
func (p *Processor) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	if len(ext) > 0 {
		ext = ext[1:] // 移除点号
	}

	languageMap := map[string]string{
		"go":    "Go",
		"java":  "Java",
		"js":    "JavaScript",
		"ts":    "TypeScript",
		"jsx":   "JavaScript",
		"tsx":   "TypeScript",
		"py":    "Python",
		"rs":    "Rust",
		"c":     "C",
		"cpp":   "C++",
		"h":     "C",
		"hpp":   "C++",
		"cs":    "C#",
		"php":   "PHP",
		"rb":    "Ruby",
		"swift": "Swift",
		"kt":    "Kotlin",
		"scala": "Scala",
		"clj":   "Clojure",
		"hs":    "Haskell",
		"elm":   "Elm",
		"dart":  "Dart",
		"lua":   "Lua",
		"r":     "R",
		"m":     "Objective-C",
		"mm":    "Objective-C++",
		"sh":    "Shell",
		"bash":  "Shell",
		"zsh":   "Shell",
		"fish":  "Shell",
		"ps1":   "PowerShell",
		"bat":   "Batch",
		"cmd":   "Batch",
		"sql":   "SQL",
		"html":  "HTML",
		"css":   "CSS",
		"scss":  "SCSS",
		"sass":  "Sass",
		"less":  "Less",
		"xml":   "XML",
		"json":  "JSON",
		"yaml":  "YAML",
		"yml":   "YAML",
		"toml":  "TOML",
		"ini":   "INI",
		"cfg":   "Configuration",
		"conf":  "Configuration",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "Unknown"
}

// collectFileMeta 收集文件元数据（仅包含路径和更新时间，不读取内容）
func (p *Processor) collectFileMeta(filePath string) (*FileMeta, error) {
	// 获取文件信息
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 检查文件大小 - 使用配置中的限制
	if info.Size() > p.config.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum limit %d", info.Size(), p.config.MaxFileSize)
	}

	// 检测是否为二进制文件（只读取前1024字节进行检测）
	isBinary := false
	if info.Size() > 0 {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		buffer := make([]byte, 1024)
		n, err := file.Read(buffer)
		if err != nil && err.Error() != "EOF" {
			return nil, fmt.Errorf("failed to read file header: %w", err)
		}

		// 检测是否为二进制文件
		isBinary = p.isBinaryFile(buffer[:n])
	}

	// 创建文件元数据
	fileMeta := &FileMeta{
		Path:         filePath,
		ModTime:      info.ModTime().Unix(),
		Size:         info.Size(),
		Extension:    strings.TrimPrefix(filepath.Ext(info.Name()), "."),
		Language:     p.detectLanguage(filePath),
		IsBinary:     isBinary,
		ContentReady: false, // 初始状态为未加载内容
	}

	return fileMeta, nil
}

// LoadFileContent 延迟加载文件内容
func (p *Processor) LoadFileContent(fileMeta *FileMeta) (*FileContent, error) {
	// 如果内容已经加载，直接返回
	if fileMeta.ContentReady {
		// 这里需要从缓存中获取，暂时重新读取
		return p.readFileContentFromPath(fileMeta.Path)
	}

	// 读取文件内容
	fileContent, err := p.readFileContentFromPath(fileMeta.Path)
	if err != nil {
		return nil, err
	}

	// 标记内容已加载
	fileMeta.ContentReady = true

	return fileContent, nil
}

// readFileContentFromPath 从文件路径读取内容
func (p *Processor) readFileContentFromPath(filePath string) (*FileContent, error) {
	// 获取文件信息
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 检查文件大小 - 使用配置中的限制
	if info.Size() > p.config.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum limit %d", info.Size(), p.config.MaxFileSize)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 检测是否为二进制文件
	isBinary := p.isBinaryFile(data)

	var content string
	var encoding string

	if isBinary {
		content = "[Binary file content not displayed]"
		encoding = "binary"
	} else {
		content = string(data)
		encoding = "utf-8"
	}

	// 创建文件信息
	fileInfo := &FileInfo{
		Path:      filePath,
		Name:      info.Name(),
		Size:      info.Size(),
		Extension: strings.TrimPrefix(filepath.Ext(info.Name()), "."),
		Language:  p.detectLanguage(filePath),
		IsBinary:  isBinary,
		Metadata:  make(map[string]string),
	}

	return &FileContent{
		Info:     fileInfo,
		Content:  content,
		Encoding: encoding,
		Size:     int64(len(content)),
	}, nil
}
