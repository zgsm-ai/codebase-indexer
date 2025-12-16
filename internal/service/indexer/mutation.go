package indexer

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RemoveIndexes 根据工作区路径、文件路径/文件夹路径前缀，批量删除索引
func (idx *Indexer) RemoveIndexes(ctx context.Context, workspacePath string, filePaths []string) error {
	start := time.Now()
	idx.logger.Info("start to remove workspace %s files: %v", workspacePath, filePaths)

	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}
	workspaceModel, err := idx.workspaceRepository.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return err
	}
	var errs []error
	projectFilesMap, err := idx.groupFilesByProject(projects, filePaths)
	if err != nil {
		return fmt.Errorf("group files by project failed: %w", err)
	}
	totalRemoved := 0
	for projectUuid, files := range projectFilesMap {
		pStart := time.Now()
		idx.logger.Info("start to remove project %s files index", projectUuid)

		removed, err := idx.removeIndexByFilePaths(ctx, projectUuid, files)
		if err != nil {
			errs = append(errs, err)
		}
		totalRemoved += removed
		idx.logger.Info("remove project %s files index end, cost %d ms, removed %d index.", projectUuid,
			time.Since(pStart).Milliseconds(), removed)
	}
	// 更新为删除后的值
	if workspaceModel != nil {
		if err := idx.workspaceRepository.UpdateCodegraphInfo(workspacePath,
			workspaceModel.CodegraphFileNum-totalRemoved, time.Now().Unix()); err != nil {
			return errors.Join(append(errs, err)...)
		}
	}
	err = errors.Join(errs...)
	idx.logger.Info("remove workspace %s files index successfully, cost %d ms, removed %d index, errors: %v",
		workspacePath, time.Since(start).Milliseconds(), totalRemoved, utils.TruncateError(err))
	return err
}

// removeIndexByFilePaths 删除单个项目的索引
func (idx *Indexer) removeIndexByFilePaths(ctx context.Context, projectUuid string, filePaths []string) (int, error) {
	// 1. 查询path相应的file_table
	deleteFileTables, err := idx.searchFileElementTablesByPath(ctx, projectUuid, filePaths)
	if err != nil {
		return 0, fmt.Errorf("get file tables for deletion failed: %w", err)
	}
	deletePaths := make(map[string]any)
	for _, v := range deleteFileTables {
		deletePaths[v.Path] = nil
	}

	// 2. 清理符号定义
	if err = idx.cleanupSymbolOccurrences(ctx, projectUuid, deleteFileTables, deletePaths); err != nil {
		return 0, fmt.Errorf("cleanup symbol definitions failed: %w", err)
	}

	// 3. 删除path索引
	deleted, err := idx.deleteFileIndexes(ctx, projectUuid, deletePaths)
	if err != nil {
		return 0, fmt.Errorf("delete file indexes failed: %w", err)
	}

	return deleted, nil
}

// searchFileElementTablesByPath 获取待删除的文件表和路径（包括文件夹）
func (idx *Indexer) searchFileElementTablesByPath(ctx context.Context, puuid string, filePaths []string) ([]*codegraphpb.FileElementTable, error) {
	var deleteFileTables []*codegraphpb.FileElementTable
	var errs []error

	for _, filePath := range filePaths {
		language, err := lang.InferLanguage(filePath)
		var fileTable []byte
		if err == nil {
			key := store.ElementPathKey{Language: language, Path: filePath}
			fileTable, err = idx.storage.Get(ctx, puuid, key)
		}

		if lang.IsUnSupportedFileError(err) || errors.Is(err, store.ErrKeyNotFound) {
			// 精确匹配不到，使用前缀模糊匹配
			idx.logger.Debug("indexer delete index, key path %s not found in store, use prefix search", filePath)
			tables, errors_ := idx.searchFileElementTablesByPathPrefix(ctx, puuid, filePath)
			if len(errors_) > 0 {
				errs = append(errs, errors_...)
			}
			if len(tables) > 0 {
				deleteFileTables = append(deleteFileTables, tables...)
			}
			continue
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}
		ft := new(codegraphpb.FileElementTable)
		if err = store.UnmarshalValue(fileTable, ft); err != nil {
			errs = append(errs, err)
			continue
		}
		deleteFileTables = append(deleteFileTables, ft)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return deleteFileTables, nil
}

// searchFileElementTablesByPathPrefix 按路径前缀搜索
func (idx *Indexer) searchFileElementTablesByPathPrefix(ctx context.Context, projectUuid string, path string) (
	[]*codegraphpb.FileElementTable, []error) {
	var errs []error
	tables := make([]*codegraphpb.FileElementTable, 0)
	iter := idx.storage.Iter(ctx, projectUuid)
	for iter.Next() {
		if !store.IsElementPathKey(iter.Key()) {
			continue
		}
		pathKey, err := store.ToElementPathKey(iter.Key())
		if err != nil {
			idx.logger.Debug("indexer delete index, parse element path key %s err:%v", iter.Key(), err)
			continue
		}
		// path 可能包含分隔符，也可能不包含，统一处理
		pathPrefix := utils.EnsureTrailingSeparator(path)
		if strings.HasPrefix(pathKey.Path, pathPrefix) {
			ft := new(codegraphpb.FileElementTable)
			if err = store.UnmarshalValue(iter.Value(), ft); err != nil {
				errs = append(errs, err)
				continue
			}
			tables = append(tables, ft)
		}
	}
	err := iter.Close()
	if err != nil {
		idx.logger.Error("indexer close graph_store err:%v", err)
	}
	return tables, errs
}

// cleanupSymbolOccurrences 清理符号定义
func (idx *Indexer) cleanupSymbolOccurrences(ctx context.Context, projectUuid string,
	deleteFileTables []*codegraphpb.FileElementTable, deletedPaths map[string]interface{}) error {
	var errs []error

	for _, ft := range deleteFileTables {
		for _, e := range ft.Elements {
			if e.IsDefinition {
				language := lang.Language(ft.Language)
				sym, err := idx.storage.Get(ctx, projectUuid, store.SymbolNameKey{Language: language, Name: e.GetName()})
				if err != nil && !errors.Is(err, store.ErrKeyNotFound) {
					errs = append(errs, err)
					continue
				}
				symDefs := new(codegraphpb.SymbolOccurrence)
				if err = store.UnmarshalValue(sym, symDefs); err != nil {
					return fmt.Errorf("unmarshal SymbolOccurrence error:%w", err)
				}

				newSymDefs := &codegraphpb.SymbolOccurrence{
					Name:        e.GetName(),
					Language:    ft.Language,
					Occurrences: make([]*codegraphpb.Occurrence, 0),
				}
				for _, d := range symDefs.Occurrences {
					if _, ok := deletedPaths[d.Path]; ok {
						continue
					}
					newSymDefs.Occurrences = append(newSymDefs.Occurrences, d)
				}
				// 如果新的为0，就无需再写入，并删除旧的
				if len(newSymDefs.Occurrences) == 0 {
					if err := idx.storage.Delete(ctx, projectUuid, store.SymbolNameKey{Language: language,
						Name: newSymDefs.Name}); err != nil {
						errs = append(errs, err)
					}
					continue
				}

				// 保存更新后的符号表
				if err := idx.storage.Put(ctx, projectUuid, &store.Entry{Key: store.SymbolNameKey{Language: language,
					Name: newSymDefs.Name}, Value: newSymDefs}); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// deleteFileIndexes 删除文件索引
func (idx *Indexer) deleteFileIndexes(ctx context.Context, puuid string, deletePaths map[string]any) (int, error) {
	var errs []error
	deleted := 0
	for fp := range deletePaths {
		// 删除path索引
		language, err := lang.InferLanguage(fp)
		if err != nil {
			continue
		}
		if err = idx.storage.Delete(ctx, puuid, store.ElementPathKey{Language: language, Path: fp}); err != nil {
			errs = append(errs, err)
		} else {
			deleted++
		}
	}

	if len(errs) > 0 {
		return deleted, errors.Join(errs...)
	}

	return deleted, nil
}

// RemoveAllIndexes 删除工作区的所有索引
func (idx *Indexer) RemoveAllIndexes(ctx context.Context, workspacePath string) error {
	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		idx.logger.Info("found no projects in workspace %s", workspacePath)
		return nil
	}
	var errs []error
	for _, p := range projects {
		errs = append(errs, idx.storage.DeleteAll(ctx, p.Uuid))
	}
	// 将数据库数据置为0
	if err := idx.workspaceRepository.UpdateCodegraphInfo(workspacePath, 0, time.Now().Unix()); err != nil {
		return errors.Join(append(errs, fmt.Errorf("update codegraph info err:%v", err))...)
	}
	return errors.Join(errs...)
}

// RenameIndexes 重命名索引，根据路径（文件或文件夹）
func (idx *Indexer) RenameIndexes(ctx context.Context, workspacePath string, sourceFilePath string, targetFilePath string) error {
	//TODO 查出来source，删除、重命名相关path、写入，更新symbol中指向source的路径为target（迭代式进行）
	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		idx.logger.Info("found no projects in workspace %s", workspacePath)
		return nil
	}
	var sourceProject *workspace.Project
	var targetProject *workspace.Project
	// rename 后，原文件（目录）已经不存在了。
	for _, p := range projects {
		if strings.HasPrefix(sourceFilePath, p.Path) {
			sourceProject = p
		}
		if strings.HasPrefix(targetFilePath, p.Path) {
			targetProject = p
		}
	}
	if sourceProject == nil {
		return fmt.Errorf("could not find source project in workspace %s for file %s", workspacePath, sourceFilePath)
	}
	if targetProject == nil {
		return fmt.Errorf("could not find target project in workspace %s for file %s", workspacePath, targetFilePath)
	}

	sourceProjectUuid, targetProjectUuid := sourceProject.Uuid, targetProject.Uuid
	// 可能是文件，也可能是目录
	sourceTables, err := idx.searchFileElementTablesByPath(ctx, sourceProjectUuid, []string{sourceFilePath})
	if err != nil {
		return fmt.Errorf("search source element tables by path %s err:%v", sourceFilePath, err)
	}
	if len(sourceTables) == 0 {
		idx.logger.Debug("found no index by source path %s", sourceFilePath)
		return nil
	}
	// 统一去掉最后的分隔符（如果有），防止一个有，另一个没有
	trimmedSourcePath, trimmedTargetPath := utils.TrimLastSeparator(sourceFilePath), utils.TrimLastSeparator(targetFilePath)
	// 将source删除、key重命名为target，更新source相关的symbol 为target
	for _, st := range sourceTables {
		oldPath := st.Path
		oldLanguage := st.Language
		// 删除
		if err = idx.storage.Delete(ctx, sourceProjectUuid, store.ElementPathKey{Language: lang.Language(st.Language), Path: st.Path}); err != nil {
			idx.logger.Debug("delete index %s %s err:%v", st.Language, st.Path, err)
		}
		// 将path中 sourceFilePath 重命名为targetFilePath，
		newPath := strings.ReplaceAll(st.Path, trimmedSourcePath, trimmedTargetPath)
		newLanguage, err := lang.InferLanguage(newPath)
		if err != nil {
			idx.logger.Debug("unsupported language for new path %s", newPath)
			// TODO 删除symbol 中的source_path、reference中的指向它的relation
			continue
		}
		st.Path = newPath
		st.Language = string(newLanguage)
		// 保存target
		if err = idx.storage.Put(ctx, targetProjectUuid, &store.Entry{Key: store.ElementPathKey{
			Language: newLanguage, Path: newPath}, Value: st}); err != nil {
			idx.logger.Debug("save new index %s err:%v ", newPath, err)
		}

		// 更新符号定义，找到相关符号，将它的path由old改为new
		for _, e := range st.Elements {
			if !e.IsDefinition {
				continue
			}
			SymbolOccurrence, err := idx.getSymbolOccurrenceByName(ctx, sourceProjectUuid, lang.Language(oldLanguage), e.Name)
			if err != nil {
				idx.logger.Debug("get symbol definition by name %s %s err:%v", oldLanguage, e.Name, err)
				continue
			}

			// 语言相同则更新，语言不同，则删除新增
			sameLanguage := oldLanguage == string(newLanguage)
			definitions := make([]*codegraphpb.Occurrence, 0, len(SymbolOccurrence.Occurrences))
			for _, d := range SymbolOccurrence.Occurrences {
				if d.Path == oldPath {
					if sameLanguage {
						d.Path = newPath
					}
				} else {
					definitions = append(definitions, d)
				}
			}
			// 保存
			if err = idx.storage.Put(ctx, sourceProjectUuid, &store.Entry{
				Key: store.SymbolNameKey{
					Language: lang.Language(SymbolOccurrence.Language),
					Name:     SymbolOccurrence.Name},
				Value: SymbolOccurrence,
			}); err != nil {
				idx.logger.Debug("save SymbolOccurrence %s err:%v", e.Name, err)
			}
			// 不同语言，保存新的
			if !sameLanguage {
				newSymbolDefinition := &codegraphpb.SymbolOccurrence{
					Name:     e.Name,
					Language: string(newLanguage),
					Occurrences: []*codegraphpb.Occurrence{
						{
							Path:        newPath,
							Range:       e.Range,
							ElementType: e.ElementType,
						},
					},
				}
				if err = idx.storage.Put(ctx, sourceProjectUuid, &store.Entry{
					Key: store.SymbolNameKey{
						Language: lang.Language(SymbolOccurrence.Language),
						Name:     SymbolOccurrence.Name},
					Value: newSymbolDefinition,
				}); err != nil {
					idx.logger.Debug("save SymbolOccurrence %s err:%v", e.Name, err)
				}
			}

		}

	}

	return nil
}
