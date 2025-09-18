package wiki

import (
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
	"fmt"
	"time"
)

// WikiSection Wiki章节 - 与参考实现保持一致
type WikiSection struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Pages       []string `json:"pages"`
	Subsections []string `json:"subsections,omitempty"`
}

// WikiPage Wiki页面 - 与参考实现保持一致
type WikiPage struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	FilePaths    []string `json:"filePaths"`
	Importance   string   `json:"importance"` // 'high', 'medium', 'low'
	RelatedPages []string `json:"relatedPages"`
	ParentID     string   `json:"parentId,omitempty"`
	IsSection    bool     `json:"isSection,omitempty"`
	Children     []string `json:"children,omitempty"`
}

// WikiStructure Wiki结构 - 与参考实现保持一致
type WikiStructure struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	Pages        []WikiPage    `json:"pages"`
	Sections     []WikiSection `json:"sections"`
	RootSections []string      `json:"rootSections"`
}

// FileInfo 文件信息
type FileInfo struct {
	Path      string            `json:"path"`
	Name      string            `json:"name"`
	Size      int64             `json:"size"`
	Extension string            `json:"extension"`
	Language  string            `json:"language"`
	IsBinary  bool              `json:"is_binary"`
	Metadata  map[string]string `json:"metadata"`
}

// FileMeta 文件元数据（仅包含路径和更新时间，用于延迟加载）
type FileMeta struct {
	Path         string `json:"path"`
	ModTime      int64  `json:"mod_time"` // 时间戳
	Size         int64  `json:"size"`
	Extension    string `json:"extension"`
	Language     string `json:"language"`
	IsBinary     bool   `json:"is_binary"`
	ContentReady bool   `json:"content_ready"` // 内容是否已加载
}

// FileContent 文件内容
type FileContent struct {
	Info     *FileInfo `json:"info"`
	Content  string    `json:"content"`
	Encoding string    `json:"encoding"`
	Size     int64     `json:"size"`
}

// RepoInfo 仓库信息 - 简化版本
type RepoInfo struct {
	LocalPath string `json:"local_path"`
}

// GenerateWikiPromptData 包含生成Wiki提示词所需的数据
type GenerateWikiPromptData struct {
	PageTitle      string
	FileLinks      string
	OutputLanguage string
}

// GenerateWikiStructPromptData 包含生成Wiki结构所需的数据
type GenerateWikiStructPromptData struct {
	FileTree       string
	ReadmeContent  string
	OutputLanguage string
	PageCount      string
	WikiType       string
}

// GenerateCodeRulesStructPromptData 包含生成代码规则结构所需的数据 - 简化版
type GenerateCodeRulesStructPromptData struct {
	FileTree       string
	ReadmeContent  string
	OutputLanguage string
	PageCount      string
	// 简化的项目信息
	ProjectName    string              // 项目名称
	KeyDirectories map[string][]string // 关键目录结构
}

// GenerateCodeRulesPromptData 包含生成代码规则页面所需的数据 - 超增强版
type GenerateCodeRulesPromptData struct {
	PageTitle      string
	FileLinks      string
	OutputLanguage string
	GuidelineCount string
	FileTree       string
	ReadmeContent  string
	FileContents   string // 相关文件的实际内容
	// 超丰富的分析维度
	BusinessContext    string   // 业务上下文
	RelatedFiles       []string // 相关文件列表
	CodePatterns       []string // 代码模式
	ConfigurationFiles []string // 配置文件
	TestFiles          []string // 测试文件
	RuntimePatterns    []string // 运行时行为模式
	BusinessSemantics  []string // 业务语义模式
	CodeTemplates      []string // 代码模板模式
	DependencyNetwork  []string // 依赖网络关系
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(logger logger.Logger, progress float64, stage, message string)

// SimpleConfig 简化的配置结构
type SimpleConfig struct {
	APIKey         string        `json:"api_key"`         // LLM API密钥
	BaseURL        string        `json:"base_url"`        // API基础URL
	Model          string        `json:"model"`           // 模型名称
	RequestTimeout time.Duration `json:"request_timeout"` // 模型请求超时时间
	StoreBasePath  string        `json:"store_base_path"`
	Language       string        `json:"language"`        // 输出语言
	MaxTokens      int           `json:"max_tokens"`      // 最大令牌数
	Temperature    float64       `json:"temperature"`     // 温度参数
	MaxFiles       int           `json:"max_files"`       // 最大文件数
	MaxFileSize    int64         `json:"max_file_size"`   // 最大文件大小
	OutputDir      string        `json:"output_dir"`      // 输出目录
	Concurrency    int           `json:"concurrency"`     // 请求并行度
	PromptTemplate string        `json:"prompt_template"` // Prompt模板
}

// DefaultSimpleConfig 返回默认配置
func DefaultSimpleConfig() *SimpleConfig {
	return &SimpleConfig{
		APIKey:         "",
		BaseURL:        "",
		Model:          "",
		Language:       "zh",
		RequestTimeout: time.Minute * 5,
		MaxTokens:      8192,
		Temperature:    0.7,
		MaxFiles:       1000,
		MaxFileSize:    10 * 1024 * 1024, // 10MB
		OutputDir:      "./output",
		StoreBasePath:  utils.DeepwikiDir,
	}
}

// ProgressStats 进度统计信息
type ProgressStats struct {
	TotalFiles     int `json:"total_files"`     // 总文件数
	ProcessedFiles int `json:"processed_files"` // 已处理文件数
	SuccessFiles   int `json:"success_files"`   // 成功文件数
	FailedFiles    int `json:"failed_files"`    // 失败文件数
	TotalPages     int `json:"total_pages"`     // 总页面数
	GeneratedPages int `json:"generated_pages"` // 已生成页面数
	SuccessPages   int `json:"success_pages"`   // 成功页面数
	FailedPages    int `json:"failed_pages"`    // 失败页面数
	TotalWikis     int `json:"total_wikis"`     // 总wiki数
	GeneratedWikis int `json:"generated_wikis"` // 已生成wiki数
}

// PerformanceStats 性能统计信息
type PerformanceStats struct {
	TotalDuration       time.Duration `json:"total_duration"`       // 总耗时
	FileProcessing      time.Duration `json:"file_processing"`      // 文件处理耗时
	StructureGeneration time.Duration `json:"structure_generation"` // 结构生成耗时
	ContentGeneration   time.Duration `json:"content_generation"`   // 内容生成耗时
	StartTime           time.Time     `json:"start_time"`           // 开始时间
	EndTime             time.Time     `json:"end_time"`             // 结束时间
	StageCount          int           `json:"stage_count"`          // 阶段数量
}

// LLMCallStats LLM调用统计信息
type LLMCallStats struct {
	TotalCalls        int           `json:"total_calls"`        // 总调用次数
	SuccessCalls      int           `json:"success_calls"`      // 成功调用次数
	FailedCalls       int           `json:"failed_calls"`       // 失败调用次数
	TotalDuration     time.Duration `json:"total_duration"`     // 总耗时
	AverageDuration   time.Duration `json:"average_duration"`   // 平均耗时
	MinDuration       time.Duration `json:"min_duration"`       // 最小耗时
	MaxDuration       time.Duration `json:"max_duration"`       // 最大耗时
	StructureCalls    int           `json:"structure_calls"`    // 结构生成调用次数
	ContentCalls      int           `json:"content_calls"`      // 内容生成调用次数
	StructureDuration time.Duration `json:"structure_duration"` // 结构生成耗时
	ContentDuration   time.Duration `json:"content_duration"`   // 内容生成耗时
}

// NewLLMCallStats 创建新的LLM调用统计
func NewLLMCallStats() *LLMCallStats {
	return &LLMCallStats{
		MinDuration: time.Hour, // 初始化为较大值
	}
}

// RecordCall 记录一次LLM调用
func (ls *LLMCallStats) RecordCall(duration time.Duration, callType string, success bool) {
	ls.TotalCalls++
	ls.TotalDuration += duration

	if success {
		ls.SuccessCalls++
	} else {
		ls.FailedCalls++
	}

	// 更新最小/最大耗时
	if duration < ls.MinDuration {
		ls.MinDuration = duration
	}
	if duration > ls.MaxDuration {
		ls.MaxDuration = duration
	}

	// 根据调用类型统计
	switch callType {
	case "structure":
		ls.StructureCalls++
		ls.StructureDuration += duration
	case "content":
		ls.ContentCalls++
		ls.ContentDuration += duration
	}

	// 计算平均耗时
	if ls.SuccessCalls > 0 {
		ls.AverageDuration = ls.TotalDuration / time.Duration(ls.SuccessCalls)
	}
}

// GetSuccessRate 获取成功率
func (ls *LLMCallStats) GetSuccessRate() float64 {
	if ls.TotalCalls == 0 {
		return 0
	}
	return float64(ls.SuccessCalls) / float64(ls.TotalCalls) * 100
}

// String 返回LLM调用统计的字符串表示
func (ls *LLMCallStats) String() string {
	if ls.TotalCalls == 0 {
		return "LLM Call Stats: No call records"
	}

	successRate := ls.GetSuccessRate()
	return fmt.Sprintf("LLM Call Stats - Total Calls: %d, Success: %d, Failed: %d, Success Rate: %.1f%%, Total Duration: %v, Average Duration: %v, Min: %v, Max: %v, Structure Generation: %d calls/%v, Content Generation: %d calls/%v",
		ls.TotalCalls,
		ls.SuccessCalls,
		ls.FailedCalls,
		successRate,
		ls.TotalDuration,
		ls.AverageDuration,
		ls.MinDuration,
		ls.MaxDuration,
		ls.StructureCalls,
		ls.StructureDuration,
		ls.ContentCalls,
		ls.ContentDuration)
}

// NewPerformanceStats 创建新的性能统计
func NewPerformanceStats() *PerformanceStats {
	return &PerformanceStats{
		StartTime: time.Now(),
	}
}

// AddStageDuration 添加阶段耗时
func (ps *PerformanceStats) AddStageDuration(stage string, duration time.Duration) {
	switch stage {
	case "file_processing":
		ps.FileProcessing += duration
	case "structure_generation":
		ps.StructureGeneration += duration
	case "content_generation":
		ps.ContentGeneration += duration
	}
	ps.StageCount++
}

// Finish 完成性能统计
func (ps *PerformanceStats) Finish() {
	ps.EndTime = time.Now()
	ps.TotalDuration = ps.EndTime.Sub(ps.StartTime)
}

// GetAverageDuration 获取平均耗时
func (ps *PerformanceStats) GetAverageDuration() time.Duration {
	if ps.StageCount == 0 {
		return 0
	}
	return ps.TotalDuration / time.Duration(ps.StageCount)
}

// String 返回性能统计的字符串表示
func (ps *PerformanceStats) String() string {
	if ps.EndTime.IsZero() {
		return "Performance stats not completed"
	}

	avgDuration := ps.GetAverageDuration()
	return fmt.Sprintf("Performance Stats - Total Duration: %v, File Processing: %v, Structure Generation: %v, Content Generation: %v, Average Duration: %v, Stage Count: %d",
		ps.TotalDuration,
		ps.FileProcessing,
		ps.StructureGeneration,
		ps.ContentGeneration,
		avgDuration,
		ps.StageCount)
}

// LogProgressCallback 默认的日志进度回调函数
func LogProgressCallback(logger logger.Logger, progress float64, stage, message string) {
	switch stage {
	case "file_processing":
		logger.Info("File processing progress: %.1f%% - %s", progress*100, message)
	case "file_processing_complete":
		logger.Info("File processing completed: %.1f%% - %s", progress*100, message)
	case "structure_generation":
		logger.Info("Structure generation progress: %.1f%% - %s", progress*100, message)
	case "structure_generation_complete":
		logger.Info("Structure generation completed: %.1f%% - %s", progress*100, message)
	case "content_generation":
		logger.Info("Content generation progress: %.1f%% - %s", progress*100, message)
	case "complete":
		logger.Info("Wiki generation completed: %.1f%% - %s", progress*100, message)
	case "performance_stats":
		logger.Info("=== Performance Statistics Report ===")
		logger.Info("%s", message)
		logger.Info("===================")
	case "llm_stats":
		logger.Info("=== LLM Call Statistics Report ===")
		logger.Info("%s", message)
		logger.Info("====================")
	case "error":
		logger.Error("Generation error: %.1f%% - %s", progress*100, message, nil)
	default:
		logger.Info("Progress update: %.1f%% [%s] - %s", progress*100, stage, message)
	}
}
