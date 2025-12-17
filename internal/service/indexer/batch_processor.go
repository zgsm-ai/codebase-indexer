package indexer

import (
	"codebase-indexer/pkg/codegraph/cache"
	"codebase-indexer/pkg/codegraph/proto"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"errors"
	"fmt"
	"time"
)

// BatchProcessParams 批处理参数
type BatchProcessParams struct {
	ProjectUuid string
	SourceFiles []*types.FileWithModTimestamp
	BatchStart  int
	BatchEnd    int
	BatchSize   int
	TotalFiles  int
	Project     *workspace.Project
}

// BatchProcessResult 批处理结果
type BatchProcessResult struct {
	ElementTablesCnt int
	Metrics          *types.IndexTaskMetrics
	Duration         time.Duration
}

// BatchProcessingParams 批处理阶段参数
type BatchProcessingParams struct {
	ProjectUuid          string
	NeedIndexSourceFiles []*types.FileWithModTimestamp
	TotalFilesCnt        int
	PreviousFileNum      int
	Project              *workspace.Project
	WorkspacePath        string
	Concurrency          int
	BatchSize            int
}

// BatchProcessingResult 批处理阶段结果
type BatchProcessingResult struct {
	ParsedFilesCount int
	ProjectMetrics   *types.IndexTaskMetrics
	Duration         time.Duration
}

// ProgressInfo 进度信息
type ProgressInfo struct {
	Total         int
	Processed     int
	PreviousNum   int
	WorkspacePath string
}

// processBatch 处理单个批次的文件
func (idx *Indexer) processBatch(ctx context.Context, batchId int, params *BatchProcessParams,
	symbolCache *cache.LRUCache[*codegraphpb.SymbolOccurrence]) (*types.IndexTaskMetrics, error) {
	batchStartTime := time.Now()

	idx.logger.Info("batch-%d start, [%d:%d]/%d, batch_size %d",
		batchId, params.BatchStart, params.BatchEnd, params.TotalFiles, params.BatchSize)

	// 解析文件
	elementTables, metrics, err := idx.parseFiles(ctx, params.SourceFiles)
	if err != nil {
		return nil, fmt.Errorf("parse files failed: %w", err)
	}
	if len(elementTables) == 0 {
		return metrics, nil
	}

	idx.logger.Info("batch-%d [%d:%d]/%d parse files end, cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(batchStartTime).Milliseconds())

	// 项目符号表存储
	symbolStart := time.Now()

	symbolMetrics, err := idx.analyzer.SaveSymbolOccurrences(ctx, params.ProjectUuid, params.TotalFiles, elementTables, symbolCache)
	metrics.TotalSymbols += symbolMetrics.TotalSymbols
	metrics.TotalSavedSymbols += symbolMetrics.TotalSavedSymbols
	metrics.TotalVariables += symbolMetrics.TotalVariables
	metrics.TotalSavedVariables += symbolMetrics.TotalSavedVariables
	if err != nil {
		return nil, fmt.Errorf("save symbol definitions failed: %w", err)
	}
	idx.logger.Info("batch-%d batch [%d:%d]/%d save symbols end, cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(symbolStart).Milliseconds())

	// 预处理import
	if err := idx.preprocessImports(ctx, elementTables, params.Project); err != nil {
		idx.logger.Error("batch-%d preprocess import error: %v", utils.TruncateError(err))
	}

	// element存储，后面依赖分析，基于磁盘，避免大型项目占用太多内存
	protoElementTables := proto.FileElementTablesToProto(elementTables)
	batchSaveStart := time.Now()
	// 关系索引存储
	if err = idx.storage.BatchSave(ctx, params.ProjectUuid, workspace.FileElementTables(protoElementTables)); err != nil {
		metrics.TotalFailedFiles += params.BatchSize
		for _, f := range params.SourceFiles {
			metrics.FailedFilePaths = append(metrics.FailedFilePaths, f.Path)
		}
		return nil, fmt.Errorf("batch save element tables failed: %w", err)
	}

	idx.logger.Info("batch-%d [%d:%d]/%d save element_tables end, cost %d ms, batch cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(batchSaveStart).Milliseconds(),
		time.Since(batchStartTime).Milliseconds())

	return metrics, nil
}

// indexFilesInBatches 批量处理文件
func (idx *Indexer) indexFilesInBatches(ctx context.Context, params *BatchProcessingParams) (*BatchProcessingResult, error) {

	idx.logger.Info("%s, concurrency: %d, batch_size: %d cache_capacity: %d",
		params.Project.Path, idx.config.MaxConcurrency, idx.config.MaxBatchSize, idx.config.CacheCapacity)

	startTime := time.Now()
	totalNeedIndexFiles := len(params.NeedIndexSourceFiles)

	// 基于文件数量预分配切片容量，优化内存使用
	var errs []error

	projectMetrics := &types.IndexTaskMetrics{
		TotalFiles:      totalNeedIndexFiles,
		FailedFilePaths: make([]string, 0, totalNeedIndexFiles/4), // 预估失败文件数约为文件数的5%
	}
	// 缓存
	symbolCache := cache.NewLRUCache[*codegraphpb.SymbolOccurrence](1000, idx.config.CacheCapacity)
	defer symbolCache.Purge()

	var processedFilesCnt int
	var batchId int
	// 处理批次
	for m := 0; m < totalNeedIndexFiles; {
		batch := utils.Min(totalNeedIndexFiles-m, params.BatchSize)
		batchStart, batchEnd := m, m+batch
		sourceFilesBatch := params.NeedIndexSourceFiles[batchStart:batchEnd]
		batchId++
		// 构建批处理参数
		batchParams := &BatchProcessParams{
			ProjectUuid: params.ProjectUuid,
			SourceFiles: sourceFilesBatch,
			BatchStart:  batchStart,
			BatchEnd:    batchEnd,
			BatchSize:   batch,
			TotalFiles:  totalNeedIndexFiles,
			Project:     params.Project,
		}

		// 提交任务
		err := func(ctx context.Context, taskID int) error {
			batchStartTime := time.Now()
			metrics, err := idx.processBatch(ctx, taskID, batchParams, symbolCache)
			if err != nil {
				idx.logger.Debug("batch-%d process batch err:%v", taskID, err)
				return fmt.Errorf("process batch err:%w", err)
			}

			processedFilesCnt += metrics.TotalFiles - metrics.TotalFailedFiles
			projectMetrics.TotalFailedFiles += metrics.TotalFailedFiles
			projectMetrics.TotalSymbols += metrics.TotalSymbols
			projectMetrics.TotalSavedSymbols += metrics.TotalSavedSymbols
			projectMetrics.TotalVariables += metrics.TotalVariables
			projectMetrics.TotalSavedVariables += metrics.TotalSavedVariables
			projectMetrics.FailedFilePaths = append(projectMetrics.FailedFilePaths, metrics.FailedFilePaths...)
			//TODO 更新进度
			batchUpdateStart := time.Now()
			if err := idx.updateProgress(ctx, &ProgressInfo{
				Total:         totalNeedIndexFiles,
				Processed:     processedFilesCnt,
				PreviousNum:   params.PreviousFileNum,
				WorkspacePath: params.WorkspacePath,
			}); err != nil {
				return fmt.Errorf("update progress failed: %w", err)
			}

			idx.logger.Info("update batch-%d workspace %s successful, file num %d/%d, cache size %d, cost %d ms, batch %d cost %d ms",
				taskID, params.WorkspacePath, processedFilesCnt+params.PreviousFileNum,
				totalNeedIndexFiles, symbolCache.Len(), time.Since(batchUpdateStart).Milliseconds(), batch, time.Since(batchStartTime).Milliseconds())
			return nil
		}(ctx, batchId)
		if err != nil {
			idx.logger.Debug("%s submit task err:%v", params.ProjectUuid, err)
		}

		m += batch
	}

	// 最终更新进度
	if err := idx.updateProgress(ctx, &ProgressInfo{
		Total:         totalNeedIndexFiles,
		Processed:     processedFilesCnt,
		PreviousNum:   params.PreviousFileNum,
		WorkspacePath: params.WorkspacePath,
	}); err != nil {
		idx.logger.Debug("%s update progress failed: %v", params.ProjectUuid, err)
	}

	return &BatchProcessingResult{
		ParsedFilesCount: processedFilesCnt,
		ProjectMetrics:   projectMetrics,
		Duration:         time.Since(startTime),
	}, errors.Join(errs...)
}
