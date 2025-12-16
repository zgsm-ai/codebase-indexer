package indexer

import (
	"codebase-indexer/pkg/codegraph/proto"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	defaultMaxLayer = 3
)

// QueryCallGraph 获取符号定义代码块里面的调用图
func (idx *Indexer) QueryCallGraph(ctx context.Context, opts *types.QueryCallGraphOptions) ([]*types.RelationNode, error) {
	startTime := time.Now()

	// 参数验证
	if opts.MaxLayer <= 0 {
		opts.MaxLayer = defaultMaxLayer // 默认最大层数
	}
	// 支持相对路径，防止目录遍历攻击
	if !filepath.IsAbs(opts.FilePath) {
		absFilePath, err := safeFilePath(opts.Workspace, opts.FilePath)
		if err != nil {
			return nil, fmt.Errorf("file path %s is not in workspace %s: %w", opts.FilePath, opts.Workspace, err)
		}
		opts.FilePath = absFilePath
	}

	project, err := idx.GetProjectByFilePath(ctx, opts.Workspace, opts.FilePath)
	if err != nil {
		return nil, err
	}
	projectUuid := project.Uuid
	defer func() {
		idx.logger.Info("query callgraph cost %d ms", time.Since(startTime).Milliseconds())
	}()

	var results []*types.RelationNode

	if opts.LineRange != "" {
		// 解析行范围
		lineRange := strings.Split(opts.LineRange, "-")
		if len(lineRange) != 2 {
			return nil, fmt.Errorf("line range format error: %s", opts.LineRange)
		}
		startLine, err := strconv.Atoi(strings.TrimSpace(lineRange[0]))
		if err != nil {
			return nil, fmt.Errorf("line number format error: %s", opts.LineRange)
		}
		endLine, err := strconv.Atoi(strings.TrimSpace(lineRange[1]))
		if err != nil {
			return nil, fmt.Errorf("line number format error: %s", opts.LineRange)
		}
		// 查询组合1：文件路径+行范围
		startLine, endLine = NormalizeLineRange(startLine, endLine, 1000)
		results, err = idx.queryCallGraphByLineRange(ctx, projectUuid, opts.Workspace, opts.FilePath, startLine, endLine, opts.MaxLayer)
		return results, err
	}
	opts.SymbolName = strings.TrimSpace(opts.SymbolName)
	// 根据查询类型处理
	if opts.SymbolName != "" {
		// 查询组合2：文件路径+符号名(类、函数)
		results, err = idx.queryCallGraphBySymbol(ctx, projectUuid, opts.Workspace, opts.FilePath, opts.SymbolName, opts.MaxLayer)
		return results, err
	}

	return nil, fmt.Errorf("invalid query callgraph options: missing symbol name or invalid line range")
}

// queryCallGraphBySymbol 根据符号名查询调用链
func (idx *Indexer) queryCallGraphBySymbol(ctx context.Context, projectUuid string, workspace, filePath, symbolName string, maxLayer int) ([]*types.RelationNode, error) {
	// 查找符号定义
	fileTable, err := idx.getFileElementTableByPath(ctx, projectUuid, filePath)
	if err != nil {
		return nil, err
	}
	// 查询符号
	foundSymbols := idx.querySymbolsByName(fileTable, &types.QueryReferenceOptions{SymbolName: symbolName})

	// 检索符号定义
	var definitions []*types.RelationNode
	var calleeElements []*CalleeInfo
	// 找定义节点，如函数、方法
	for _, symbol := range foundSymbols {
		// 根节点只能是函数、方法的定义
		if !symbol.IsDefinition {
			continue
		}
		if symbol.ElementType != codegraphpb.ElementType_METHOD &&
			symbol.ElementType != codegraphpb.ElementType_FUNCTION {
			continue
		}
		params, err := proto.GetParametersFromExtraData(symbol.ExtraData)
		if err != nil {
			idx.logger.Error("failed to get parameters from extra data, err: %v", err)
			continue
		}
		isVariadic := false
		paramCount := len(params)
		if len(params) > 0 {
			lastParam := params[paramCount-1]
			if strings.Contains(lastParam.Name, VarVariadic) {
				isVariadic = true
				paramCount = paramCount - 1
			}
		}
		position := types.ToPosition(symbol.Range)
		node := &types.RelationNode{
			SymbolName: symbol.Name,
			FilePath:   filePath,
			NodeType:   string(types.NodeTypeDefinition),
			Position:   &position,
			Children:   make([]*types.RelationNode, 0),
		}
		callee := &CalleeInfo{
			SymbolName: symbol.Name,
			FilePath:   filePath,
			ParamCount: paramCount,
			IsVariadic: isVariadic,
			Position:   types.ToPosition(symbol.Range),
		}
		definitions = append(definitions, node)
		calleeElements = append(calleeElements, callee)
	}
	visited := make(map[string]struct{})
	idx.buildCallGraphBFS(ctx, projectUuid, workspace, definitions, calleeElements, maxLayer, visited)
	return definitions, nil
}

// queryCallGraphByLineRange 根据行范围查询调用链
func (idx *Indexer) queryCallGraphByLineRange(ctx context.Context, projectUuid string, workspace string, filePath string, startLine, endLine, maxLayer int) ([]*types.RelationNode, error) {
	// 获取文件元素表
	fileTable, err := idx.getFileElementTableByPath(ctx, projectUuid, filePath)
	if err != nil {
		return nil, err
	}

	// 查找范围内的符号
	queryStartLine := int32(startLine - 1)
	queryEndLine := int32(endLine - 1)
	foundSymbols := idx.findSymbolInDocByLineRange(ctx, fileTable, queryStartLine, queryEndLine)

	// 提取调用函数或方法调用，构建调用图
	var definitions []*types.RelationNode
	var calleeElements []*CalleeInfo
	for _, symbol := range foundSymbols {
		// 根节点只能是函数、方法的定义
		if !symbol.IsDefinition {
			continue
		}
		if symbol.ElementType != codegraphpb.ElementType_METHOD &&
			symbol.ElementType != codegraphpb.ElementType_FUNCTION {
			continue
		}
		params, err := proto.GetParametersFromExtraData(symbol.ExtraData)
		if err != nil {
			idx.logger.Error("failed to get parameters from extra data, err: %v", err)
			continue
		}
		isVariadic := false
		paramCount := len(params)
		if len(params) > 0 {
			lastParam := params[paramCount-1]
			if strings.Contains(lastParam.Name, VarVariadic) {
				isVariadic = true
				paramCount = paramCount - 1
			}
		}
		position := types.ToPosition(symbol.Range)
		node := &types.RelationNode{
			SymbolName: symbol.Name,
			FilePath:   filePath,
			NodeType:   string(types.NodeTypeDefinition),
			Position:   &position,
			Children:   make([]*types.RelationNode, 0),
		}
		callee := &CalleeInfo{
			SymbolName: symbol.Name,
			FilePath:   filePath,
			ParamCount: paramCount,
			IsVariadic: isVariadic,
			Position:   types.ToPosition(symbol.Range),
		}
		definitions = append(definitions, node)
		calleeElements = append(calleeElements, callee)
	}
	visited := make(map[string]struct{})
	idx.buildCallGraphBFS(ctx, projectUuid, workspace, definitions, calleeElements, maxLayer, visited)

	return definitions, nil
}

// buildCallGraphBFS 使用BFS层次遍历构建调用链
func (idx *Indexer) buildCallGraphBFS(ctx context.Context, projectUuid string, workspace string, rootNodes []*types.RelationNode, calleeInfos []*CalleeInfo, maxLayer int, visited map[string]struct{}) {
	if len(rootNodes) == 0 || maxLayer <= 0 {
		return
	}

	// 初始化队列，存储当前层的节点和对应的被调用元素
	type layerNode struct {
		node   *types.RelationNode
		callee *CalleeInfo
	}

	currentLayerNodes := make([]*layerNode, 0)

	// 初始化第一层
	for k, node := range rootNodes {
		calleeInfo := calleeInfos[k]
		key := calleeInfo.Key()
		// visited 用于防止递归的情况，同时可以进行剪枝，避免重复计算，提高性能
		if _, ok := visited[key]; !ok {
			visited[key] = struct{}{}
			currentLayerNodes = append(currentLayerNodes, &layerNode{
				node:   node,
				callee: calleeInfo,
			})
		}
	}
	// 构建反向索引映射：callee -> []caller
	err := idx.buildCalleeMap(ctx, projectUuid)
	if err != nil {
		idx.logger.Error("failed to build callee map for write, err: %v", err)
		return
	}
	calleeMap, err := lru.New[string, []CallerInfo](MaxCalleeMapCacheCapacity / 2)
	if err != nil {
		idx.logger.Error("failed to create callee map cache, err: %v", err)
		return
	}
	// BFS层次遍历
	for layer := 0; layer < maxLayer && len(currentLayerNodes) > 0; layer++ {
		nextLayerNodes := make([]*layerNode, 0)
		// 使用反向索引直接查找调用者
		for _, ln := range currentLayerNodes {
			// 构建callee的key
			calleeKey := ln.callee.SymbolName

			// 从反向索引中获取调用者列表
			callers, exists := calleeMap.Get(calleeKey)
			if !exists {
				// 从数据库查询
				dbCallers, err := idx.queryCallersFromDB(ctx, projectUuid, calleeKey)
				if err != nil {
					// idx.logger.Debug("query callers from db for callee %s,err: %v", calleeKey.SymbolName, err)
					continue
				}
				callers = dbCallers
				// 更新缓存
				calleeMap.Add(calleeKey, callers)
			}
			realCallers := make([]CallerInfo, 0, len(callers))
			for i := range len(callers) {
				// 根据可变参数，过滤掉不符合条件的调用者
				if ln.callee.IsVariadic && callers[i].CalleeKey.ParamCount < ln.callee.ParamCount {
					// 调用者传入的参数少于被调用者的固定参数个数（可变参数）
					continue
				}
				if !ln.callee.IsVariadic && callers[i].CalleeKey.ParamCount != ln.callee.ParamCount {
					// 调用者传入的参数不等于被调用者的固定参数个数（固定参数）
					continue
				}
				// 可以保留递归情况的层次信息，但是不继续遍历下去
				if _, ok := visited[callers[i].Key()]; ok {
					// 防止循环引用
					continue
				}

				fileElementTable, err := idx.getFileElementTableByPath(ctx, projectUuid, callers[i].FilePath)
				if err != nil {
					idx.logger.Error("failed to get file element table by path, err: %v", err)
					continue
				}
				imports := fileElementTable.Imports
				// 计算匹配分数
				score := idx.analyzer.CalculateSymbolMatchScore(workspace, imports, callers[i].FilePath, ln.callee.FilePath,
					ln.callee.SymbolName, callers[i].SymbolName)
				callers[i].Score = float64(score)
				realCallers = append(realCallers, callers[i])
			}

			sort.Slice(realCallers, func(i, j int) bool {
				return realCallers[i].Score > realCallers[j].Score
			})
			// 第一层不限制，其他层限制
			if layer != 0 && len(realCallers) > DefaultTopN {
				realCallers = realCallers[:DefaultTopN]
			}

			for i := range len(realCallers) {
				// 创建对应的被调用元素
				calleeInfo := &CalleeInfo{
					FilePath:   realCallers[i].FilePath,
					SymbolName: realCallers[i].SymbolName,
					ParamCount: realCallers[i].ParamCount,
					Position:   realCallers[i].Position,
					IsVariadic: realCallers[i].IsVariadic,
				}
				// 创建调用者节点
				callerNode := &types.RelationNode{
					FilePath:   realCallers[i].FilePath,
					SymbolName: realCallers[i].SymbolName,
					Position:   &realCallers[i].Position,
					NodeType:   string(types.NodeTypeReference),
					Children:   make([]*types.RelationNode, 0),
				}
				// 将调用者添加到当前节点的children中
				ln.node.Children = append(ln.node.Children, callerNode)
				// 标记为已访问，并添加到下一层
				visited[calleeInfo.Key()] = struct{}{}

				nextLayerNodes = append(nextLayerNodes, &layerNode{
					node:   callerNode,
					callee: calleeInfo,
				})
			}
		}

		// 移动到下一层
		currentLayerNodes = nextLayerNodes
	}
	// 清除数据库
	err = idx.storage.DeleteAllWithPrefix(ctx, projectUuid, store.CalleeMapKeySystemPrefix)
	if err != nil {
		idx.logger.Error("failed to delete callee map for project %s, err: %v", projectUuid, err)
		return
	}
}

// buildCalleeMap 构建反向索引映射：callee -> []caller
func (idx *Indexer) buildCalleeMap(ctx context.Context, projectUuid string) error {
	// 创建batcher实例
	batcher := NewMapBatcher(idx.storage, idx.logger, projectUuid, DefaultMapBatchSize)
	defer batcher.Flush()

	// 用name作为key
	calleeMap, err := lru.NewWithEvict(MaxCalleeMapCacheCapacity, func(key string, value []CallerInfo) {
		batcher.Add(key, value, true)
	})
	if err != nil {
		return fmt.Errorf("failed to create variadic map cache, err: %v", err)
	}
	iter := idx.storage.Iter(ctx, projectUuid)
	defer iter.Close()

	for iter.Next() {
		key := iter.Key()
		if store.IsSymbolNameKey(key) {
			continue
		}

		var elementTable codegraphpb.FileElementTable
		if err := store.UnmarshalValue(iter.Value(), &elementTable); err != nil {
			idx.logger.Error("failed to unmarshal file %s element_table value, err: %v", elementTable.Path, err)
			continue
		}

		// 遍历所有函数/方法定义
		for _, element := range elementTable.Elements {
			if !element.IsDefinition ||
				(element.ElementType != codegraphpb.ElementType_FUNCTION &&
					element.ElementType != codegraphpb.ElementType_METHOD) {
				continue
			}

			// 获取调用者（函数/方法）参数个数
			callerParams, err := proto.GetParametersFromExtraData(element.ExtraData)
			if err != nil {
				idx.logger.Debug("parse caller parameters from extra data, err: %v", err)
				continue
			}
			callerParamCount := len(callerParams)
			isVariadic := false
			if callerParamCount > 0 {
				lastParam := callerParams[callerParamCount-1]
				if strings.Contains(lastParam.Name, VarVariadic) {
					callerParamCount = callerParamCount - 1
					isVariadic = true
				}
			}
			// 查找该函数内部的所有调用
			calleeKeys := idx.extractCalleeSymbols(&elementTable, element.Range[0], element.Range[2])

			// 为每个被调用的符号添加调用者信息
			for _, calleeKey := range calleeKeys {
				callerInfo := CallerInfo{
					SymbolName: element.Name,
					FilePath:   elementTable.Path,
					Position:   types.ToPosition(element.Range),
					ParamCount: callerParamCount,
					IsVariadic: isVariadic,
					CalleeKey:  calleeKey,
				}
				// 添加到缓存
				if val, ok := calleeMap.Get(calleeKey.SymbolName); ok {
					calleeMap.Add(calleeKey.SymbolName, append(val, callerInfo))
				} else {
					calleeMap.Add(calleeKey.SymbolName, []CallerInfo{callerInfo})
				}
			}
		}
	}

	// 清空缓存，必须全部写到数据库里面去，保证数据库是最新的
	calleeMap.Purge()
	return nil
}

// extractCalleeSymbols 提取函数定义范围内的所有被调用符号
func (idx *Indexer) extractCalleeSymbols(fileTable *codegraphpb.FileElementTable, startLine, endLine int32) []CalleeKey {
	var calleeKeys []CalleeKey

	// 直接遍历元素，避免调用 findSymbolInDocByLineRange
	for _, element := range fileTable.Elements {
		if len(element.Range) < 4 {
			continue
		}

		// 检查是否在指定范围内
		if !(element.Range[0] >= startLine && element.Range[2] <= endLine) {
			continue
		}
		// 只处理调用类型的元素
		if element.ElementType != codegraphpb.ElementType_CALL {
			continue
		}
		// 获取参数个数
		params, err := proto.GetParametersFromExtraData(element.ExtraData)
		if err != nil {
			idx.logger.Debug("failed to get parameters from extra data, err: %v", err)
			continue
		}
		// 被调用的符号
		calleeKeys = append(calleeKeys, CalleeKey{
			SymbolName: element.Name,
			ParamCount: len(params),
		})
	}

	return calleeKeys
}

// queryCallersFromDB 从数据库查询指定符号的调用者列表
func (idx *Indexer) queryCallersFromDB(ctx context.Context, projectUuid string, calleeName string) ([]CallerInfo, error) {
	var item codegraphpb.CalleeMapItem
	result, err := idx.storage.Get(ctx, projectUuid, store.CalleeMapKey{
		SymbolName: calleeName,
	})
	if err != nil {
		return nil, fmt.Errorf("storage query failed: %w", err)
	}

	if err := store.UnmarshalValue(result, &item); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	callers := make([]CallerInfo, 0, len(item.Callers))
	for _, c := range item.Callers {
		callers = append(callers, CallerInfo{
			SymbolName: c.SymbolName,
			FilePath:   c.FilePath,
			Position: types.Position{
				StartLine:   int(c.Position.StartLine),
				StartColumn: int(c.Position.StartColumn),
				EndLine:     int(c.Position.EndLine),
				EndColumn:   int(c.Position.EndColumn),
			},
			ParamCount: int(c.ParamCount),
			CalleeKey:  CalleeKey{SymbolName: c.CalleeKey.SymbolName, ParamCount: int(c.CalleeKey.ParamCount)},
			Score:      c.Score,
		})
	}
	return callers, nil
}
