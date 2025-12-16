package indexer

import (
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
)

// 常量定义
const (
	MaxQueryLineLimit         = 200
	DefaultConcurrency        = 1
	DefaultBatchSize          = 50
	DefaultMapBatchSize       = 5
	DefaultMaxFiles           = 10000
	DefaultMaxProjects        = 3
	DefaultCacheCapacity      = 100000 // 假定单个文件平均10个元素,1万个文件
	DefaultTopN               = 10
	MaxCalleeMapCacheCapacity = 1600
	VarVariadic               = "..."
	DefaultMaxLayer           = 3
)

// Config 索引器配置
type Config struct {
	MaxConcurrency int
	MaxBatchSize   int
	MaxFiles       int
	MaxProjects    int
	VisitPattern   *types.VisitPattern
	CacheCapacity  int
}

// CalleeKey 表示被调用的符号信息
type CalleeKey struct {
	SymbolName string
	ParamCount int
}

// CalleeInfo 被调用者信息
type CalleeInfo struct {
	FilePath   string         `json:"filePath,omitempty"`
	Position   types.Position `json:"range,omitempty"`
	SymbolName string         `json:"symbolName,omitempty"`
	ParamCount int            `json:"paramCount,omitempty"`
	IsVariadic bool           `json:"isVariadic,omitempty"`
}

// Key 生成被调用者唯一键
func (c *CalleeInfo) Key() string {
	return fmt.Sprintf("%s::%s::%d:%d:%d:%d",
		c.SymbolName,
		c.FilePath,
		c.Position.StartLine,
		c.Position.StartColumn,
		c.Position.EndLine,
		c.Position.EndColumn,
	)
}

// CallerInfo 表示调用者信息
type CallerInfo struct {
	SymbolName string
	FilePath   string
	Position   types.Position
	ParamCount int
	IsVariadic bool
	CalleeKey  CalleeKey
	Score      float64 // 起到排序的作用
}

// Key 生成调用者唯一键
func (c *CallerInfo) Key() string {
	return fmt.Sprintf("%s::%s::%d:%d:%d:%d",
		c.SymbolName,
		c.FilePath,
		c.Position.StartLine,
		c.Position.StartColumn,
		c.Position.EndLine,
		c.Position.EndColumn,
	)
}

// MapBatcher 批量映射处理器
type MapBatcher struct {
	storage     store.GraphStorage // 存储
	logger      logger.Logger
	projectUuid string

	batchSize int // 批量写入的大小限制
	calleeMap map[string][]CallerInfo
}

// NewMapBatcher 创建批量映射处理器
func NewMapBatcher(storage store.GraphStorage, logger logger.Logger, projectUuid string, batchSize int) *MapBatcher {
	mb := &MapBatcher{
		storage:     storage,
		logger:      logger,
		projectUuid: projectUuid,
		batchSize:   batchSize,
		calleeMap:   make(map[string][]CallerInfo),
	}
	return mb
}

// Add 添加映射项
func (mb *MapBatcher) Add(key string, val []CallerInfo, merge bool) {
	if merge {
		mb.calleeMap[key] = append(mb.calleeMap[key], val...)
	} else {
		mb.calleeMap[key] = val
	}
	// 达到批次立即推送
	if len(mb.calleeMap) >= mb.batchSize {
		tempCalleeMap := mb.calleeMap
		mb.calleeMap = make(map[string][]CallerInfo)
		mb.flush(tempCalleeMap)
	}
}

// flush 批量写入数据库
func (mb *MapBatcher) flush(tempCalleeMap map[string][]CallerInfo) {
	if len(tempCalleeMap) == 0 {
		return
	}
	items := make([]*codegraphpb.CalleeMapItem, 0, len(tempCalleeMap))

	for calleeName, callers := range tempCalleeMap {
		item := &codegraphpb.CalleeMapItem{
			CalleeName: calleeName,
			Callers:    make([]*codegraphpb.CallerInfo, 0, len(callers)),
		}
		for _, c := range callers {
			item.Callers = append(item.Callers, &codegraphpb.CallerInfo{
				SymbolName: c.SymbolName,
				FilePath:   c.FilePath,
				Position: &codegraphpb.Position{
					StartLine:   int32(c.Position.StartLine),
					StartColumn: int32(c.Position.StartColumn),
					EndLine:     int32(c.Position.EndLine),
					EndColumn:   int32(c.Position.EndColumn),
				},
				ParamCount: int32(c.ParamCount),
				CalleeKey: &codegraphpb.CalleeKey{
					SymbolName: c.CalleeKey.SymbolName,
					ParamCount: int32(c.CalleeKey.ParamCount),
				},
				IsVariadic: c.IsVariadic,
				Score:      c.Score,
			})
		}

		// 合并旧数据
		old, _ := mb.storage.Get(context.Background(), mb.projectUuid,
			store.CalleeMapKey{SymbolName: calleeName})
		if old != nil {
			var oldItem codegraphpb.CalleeMapItem
			if err := store.UnmarshalValue(old, &oldItem); err == nil {
				item.Callers = append(item.Callers, oldItem.Callers...)
			}
		}
		items = append(items, item)
	}

	if err := mb.storage.BatchSave(context.Background(), mb.projectUuid,
		workspace.CalleeMapItems(items)); err != nil {
		mb.logger.Error("batch save failed: %v", err)
	}
}

// Flush 手动刷盘
func (mb *MapBatcher) Flush() {
	mb.flush(mb.calleeMap)
}
