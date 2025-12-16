package indexer

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NormalizeLineRange 标准化行范围
func NormalizeLineRange(start, end, maxLimit int) (int, int) {
	// 确保最小为 1
	if start <= 0 {
		start = 1
	}
	if end <= 0 {
		end = 1
	}

	// 保证 end >= start
	if end < start {
		end = start
	}

	// 限制最大跨度
	if end-start+1 > maxLimit {
		end = start + maxLimit - 1
	}

	return start, end
}

// isValidRange 验证范围
func isValidRange(range_ []int32) bool {
	return len(range_) == 4
}

// isInLinesRange 是否在行范围内
func isInLinesRange(current, start, end int32) bool {
	return current >= start-1 && current <= end-1
}

// isSymbolExists 符号是否存在
func isSymbolExists(filePath string, ranges []int32, state map[string]bool) bool {
	key := symbolMapKey(filePath, ranges)
	_, ok := state[key]
	return ok
}

// symbolMapKey 符号映射键
func symbolMapKey(filePath string, ranges []int32) string {
	return filePath + "-" + utils.SliceToString(ranges)
}

// safeFilePath 安全文件路径，防止目录遍历攻击
func safeFilePath(workspace, relativeFilePath string) (string, error) {
	root, err := os.OpenInRoot(workspace, relativeFilePath)
	if err != nil {
		return "", err
	}
	defer root.Close()
	return filepath.Join(workspace, relativeFilePath), nil
}

// groupFilesByProject 根据项目对文件进行分组
func (idx *Indexer) groupFilesByProject(projects []*workspace.Project, filePaths []string) (map[string][]string, error) {
	projectFilesMap := make(map[string][]string)
	var errs []error

	for _, p := range projects {
		for _, filePath := range filePaths {
			if strings.HasPrefix(filePath, p.Path) {
				projectUuid := p.Uuid
				files, ok := projectFilesMap[projectUuid]
				if !ok {
					files = make([]string, 0)
				}
				files = append(files, filePath)
				projectFilesMap[projectUuid] = files
			}
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return projectFilesMap, nil
}

// findProjectForFile 查找文件所属的项目
func (idx *Indexer) findProjectForFile(projects []*workspace.Project, filePath string) (*workspace.Project, string, error) {
	for _, p := range projects {
		if strings.HasPrefix(filePath, p.Path) {
			return p, p.Uuid, nil
		}
	}

	return nil, types.EmptyString, fmt.Errorf("no project found for file path %s", filePath)
}

// GetProjectByFilePath 根据文件路径获取项目并检查项目索引是否存在
func (idx *Indexer) GetProjectByFilePath(ctx context.Context, workspace, filePath string) (*workspace.Project, error) {
	// 参数验证
	if workspace == "" {
		return nil, fmt.Errorf("workspace path cannot be empty")
	}
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// 获取项目信息
	project, err := idx.workspaceReader.GetProjectByFilePath(ctx, workspace, filePath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get project for workspace %s, file %s: %w", workspace, filePath, err)
	}

	// 验证项目索引是否存在
	exists, err := idx.storage.ProjectIndexExists(project.Uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to check workspace %s index existence: %w", workspace, err)
	}
	if !exists {
		return nil, fmt.Errorf("workspace %s index does not exist, project uuid: %s", workspace, project.Uuid)
	}
	return project, nil
}

// getFileElementTableByPath 通过路径获取FileElementTable
func (idx *Indexer) getFileElementTableByPath(ctx context.Context, projectUuid string, filePath string) (*codegraphpb.FileElementTable, error) {
	language, err := lang.InferLanguage(filePath)
	if err != nil {
		return nil, err
	}
	fileTableBytes, err := idx.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
	if errors.Is(err, store.ErrKeyNotFound) {
		return nil, fmt.Errorf("index not found for file %s", filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s index, err: %v", filePath, err)
	}
	var fileElementTable codegraphpb.FileElementTable
	if err = store.UnmarshalValue(fileTableBytes, &fileElementTable); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s index value, err: %v", filePath, err)
	}
	return &fileElementTable, nil
}

// getFileElementTable 根据文件路径获取文件元素表
func (idx *Indexer) getFileElementTable(ctx context.Context, projectUuid string, language lang.Language, filePath string) (*codegraphpb.FileElementTable, error) {
	fileTableBytes, err := idx.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
	if errors.Is(err, store.ErrKeyNotFound) {
		return nil, fmt.Errorf("index not found for file %s", filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s index, err: %v", filePath, err)
	}

	var fileElementTable codegraphpb.FileElementTable
	if err = store.UnmarshalValue(fileTableBytes, &fileElementTable); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s index value, err: %v", filePath, err)
	}

	return &fileElementTable, nil
}

// getSymbolOccurrenceByName 通过符号名获取符号出现
func (idx *Indexer) getSymbolOccurrenceByName(ctx context.Context, projectUuid string,
	language lang.Language, symbolName string) (*codegraphpb.SymbolOccurrence, error) {

	bytes, err := idx.storage.Get(ctx, projectUuid, store.SymbolNameKey{Language: language, Name: symbolName})
	if err != nil {
		return nil, err
	}
	var SymbolOccurrence codegraphpb.SymbolOccurrence
	err = store.UnmarshalValue(bytes, &SymbolOccurrence)
	return &SymbolOccurrence, err
}

// findSymbolInDocByRange 按范围查找符号
func (idx *Indexer) findSymbolInDocByRange(fileElementTable *codegraphpb.FileElementTable, symbolRange []int32) *codegraphpb.Element {
	//TODO 二分查找
	for _, s := range fileElementTable.Elements {
		// 开始行
		if len(s.Range) < 2 {
			idx.logger.Debug("findSymbolInDocByRange invalid range in doc:%s, less than 2: %v", s.Name, s.Range)
			continue
		}

		if s.Range[0] == symbolRange[0] {
			return s
		}
	}
	return nil
}

// findSymbolInDocByLineRange 按行范围查找符号
func (idx *Indexer) findSymbolInDocByLineRange(ctx context.Context,
	fileElementTable *codegraphpb.FileElementTable, startLine int32, endLine int32) []*codegraphpb.Element {
	var res []*codegraphpb.Element
	for _, s := range fileElementTable.Elements {
		// 开始行
		if len(s.Range) < 2 {
			idx.logger.Debug("findSymbolInDocByLineRange invalid range in fileElementTable:%s, less than 2: %v", s.Name, s.Range)
			continue
		}
		if s.Range[0] > endLine {
			break
		}
		// 开始行、(TODO 列一致)
		if s.Range[0] >= startLine && s.Range[0] <= endLine {
			res = append(res, s)
		}
	}
	return res
}

// findReferenceSymbolBelonging 查找引用符号归属
func (idx *Indexer) findReferenceSymbolBelonging(f *codegraphpb.FileElementTable,
	referenceElement *codegraphpb.Element) *codegraphpb.Element {
	if len(referenceElement.GetRange()) < 3 {
		idx.logger.Debug("find symbol belong %s invalid referenceElement range %s %s %v",
			f.Path, referenceElement.Name, referenceElement.Range)
		return nil
	}
	for _, e := range f.Elements {
		if !e.IsDefinition {
			continue
		}
		if len(e.GetRange()) < 3 {
			idx.logger.Debug("find symbol belong invalid range %s %s %v", f.Path, e.Name, e.Range)
			continue
		}
		// 判断行
		if referenceElement.Range[0] > e.Range[0] && referenceElement.Range[0] < e.Range[2] {
			return e
		}
	}
	return nil
}
