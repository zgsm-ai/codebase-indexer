package service

import (
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/wiki"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type WikiProcessService interface {
	ProcessActiveWorkspaces(ctx context.Context) ([]*model.Workspace, error)
	ProcessAddFileEvent(ctx context.Context, event *model.Event) error
	ProcessModifyFileEvent(ctx context.Context, event *model.Event) error
	ProcessDeleteFileEvent(ctx context.Context, event *model.Event) error
	ProcessRenameFileEvent(ctx context.Context, event *model.Event) error
	ProcessOpenWorkspaceEvent(ctx context.Context, event *model.Event) error
	ProcessRebuildWorkspaceEvent(ctx context.Context, event *model.Event) error
	ProcessEvents(ctx context.Context, workspacePaths []string) error
}

type WikiProcessor struct {
	wiki            *wiki.DocumentManager
	workspaceReader workspace.WorkspaceReader
	workspaceRepo   repository.WorkspaceRepository
	eventRepo       repository.EventRepository
	logger          logger.Logger
}

func NewWikiProcessor(
	workspaceReader workspace.WorkspaceReader,
	wiki *wiki.DocumentManager,
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	logger logger.Logger,
) WikiProcessService {
	return &WikiProcessor{
		workspaceReader: workspaceReader,
		wiki:            wiki,
		workspaceRepo:   workspaceRepo,
		eventRepo:       eventRepo,
		logger:          logger,
	}
}

// ProcessActiveWorkspaces 扫描活跃工作区
func (ep *WikiProcessor) ProcessActiveWorkspaces(ctx context.Context) ([]*model.Workspace, error) {
	workspaces, err := ep.workspaceRepo.GetActiveWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get active workspaces: %w", err)
	}

	var activeWorkspaces []*model.Workspace
	for _, w := range workspaces {
		if w.Active == model.True {
			activeWorkspaces = append(activeWorkspaces, w)
		}
	}

	return activeWorkspaces, nil
}

// ProcessAddFileEvent 处理添加文件事件
func (c *WikiProcessor) ProcessAddFileEvent(ctx context.Context, event *model.Event) error {
	return nil
}

// ProcessModifyFileEvent 处理修改文件事件
func (c *WikiProcessor) ProcessModifyFileEvent(ctx context.Context, event *model.Event) error {
	return nil
}

// ProcessDeleteFileEvent 处理删除文件/目录事件
func (c *WikiProcessor) ProcessDeleteFileEvent(ctx context.Context, event *model.Event) error {
	return nil
}

// ProcessRenameFileEvent 处理重命名文件/目录事件
func (c *WikiProcessor) ProcessRenameFileEvent(ctx context.Context, event *model.Event) error {
	return nil
}

func (c *WikiProcessor) ProcessRebuildWorkspaceEvent(ctx context.Context, event *model.Event) error {

	return nil
}

func (c *WikiProcessor) ProcessOpenWorkspaceEvent(ctx context.Context, event *model.Event) error {
	var errs []error
	fileInfo, err := c.workspaceReader.Stat(event.WorkspacePath)
	if errors.Is(err, workspace.ErrPathNotExists) {
		c.logger.Error("wiki failed to process open_workspace event event, workspace %s not exists",
			event.WorkspacePath)
		//if err = c.updateEventStatusFinally(event, err); err != nil {
		//	return fmt.Errorf("wiki update open_workspace event err: %w", err)
		//}
		return err
	}

	if !fileInfo.IsDir {
		c.logger.Error("wiki open_workspace event, %s is file, not process.",
			event.WorkspacePath)
		//if err = c.updateEventStatusFinally(event, nil); err != nil {
		//	return fmt.Errorf("wiki update open_workspace event err: %w", err)
		//}
		return nil
	}

	// 更新进度为0，成功后再更新总进度。
	//err = c.workspaceRepo.UpdateCodegraphInfo(event.WorkspacePath, 0, time.Now().Unix())
	//if err != nil {
	//	c.logger.Error("wiki failed to process open_workspace event event, workspace %s reset successful file num failed, err:%v",
	//		event.WorkspacePath, err)
	//	//if err = c.updateEventStatusFinally(event, err); err != nil {
	//	//	return fmt.Errorf("wiki update open_workspace event err: %w", err)
	//	//}
	//	return err
	//}

	// rules 存在则跳过
	if exists, err := c.workspaceReader.Exists(ctx, filepath.Join(event.WorkspacePath, ".roo", "rules-code", "generated-rules.md")); !exists {
		c.logger.Info("wiki open_workspace event, %s code rules not exists, generate it. err:%v", event.WorkspacePath, err)
		if _, err := c.wiki.GenerateCodeRules(ctx, event.WorkspacePath); err != nil {
			c.logger.Error("failed to generate code rules for wiki: %v", err)
			errs = append(errs, err)
		} else {
			codeRulesPath := filepath.Join(event.WorkspacePath, ".roo", "rules-code")
			c.logger.Info("wiki open_workspace event, %s code rules generated successfully, start to export to %s",
				event.WorkspacePath, codeRulesPath)
			if err := c.wiki.ExportCodeRules(event.WorkspacePath, codeRulesPath, "markdown", "single", "generated-rules.md"); err != nil {
				c.logger.Error("failed to export code rules for wiki: %v", err)
				errs = append(errs, err)
			} else {
				c.logger.Info("wiki %s code rules export successfully to path %s",
					event.WorkspacePath, codeRulesPath)
			}
		}
	} else {
		c.logger.Info("wiki open_workspace event, %s code rules exists, skip.", event.WorkspacePath)
	}

	// wiki 存在则跳过
	if exists, err := c.workspaceReader.Exists(ctx, filepath.Join(event.WorkspacePath, ".costrict", "wiki", "index.md")); !exists {
		c.logger.Info("wiki open_workspace event, %s wiki not exists, generate. err:%v", event.WorkspacePath, err)
		if _, err = c.wiki.GenerateWiki(ctx, event.WorkspacePath); err != nil {
			c.logger.Error("wiki %s generate err: %w", event.WorkspacePath, err)
			errs = append(errs, err)
		} else {
			wikiPath := filepath.Join(".costrict", "wiki")
			c.logger.Info("wiki %s generate successfully, start to export to path %s", event.WorkspacePath, wikiPath)
			// 导出到workspace的输出目录
			if err = c.wiki.ExportWiki(event.WorkspacePath, wikiPath, "markdown", "multi", ""); err != nil {
				c.logger.Error("export wiki for workspace %s err:%w", event.WorkspacePath, err)
				errs = append(errs, err)
			} else {
				c.logger.Info("wiki %s export successfully to path %s ", event.WorkspacePath, wikiPath)
			}
		}

	} else {
		c.logger.Info("wiki open_workspace event, %s wiki exists, skip.",
			event.WorkspacePath)
	}

	return errors.Join(errs...)
}

// ProcessEvents 处理事件记录
func (c *WikiProcessor) ProcessEvents(ctx context.Context, workspacePaths []string) error {

	//codegraphStatuses := []int{
	//	model.CodegraphStatusInit,
	//	model.CodegraphStatusSuccess,
	//	model.CodegraphStatusFailed,
	//}

	// 打开工作区事件
	//openEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeOpenWorkspace}, workspacePaths, 10,
	//	false, nil, codegraphStatuses)
	//
	//if err != nil {
	//	c.logger.Error("failed to get open_workspace events: %v", err)
	//	return fmt.Errorf("failed to get open_workspace events: %w", err)
	//}

	// TODO 处理打开工作区事件
	//for _, event := range openEvents {
	//	c.convertWorkspaceFilePathToAbs(event)
	//	c.logger.Info("wiki start to process open_workspace event: %s", event.WorkspacePath)
	//	err = c.ProcessOpenWorkspaceEvent(ctx, event)
	//	if err != nil {
	//		c.logger.Error("failed to process open_workspace event for codegraph: %v", err)
	//		continue
	//	}
	//	c.logger.Info("wiki process open_workspace event successfully: %s", event.WorkspacePath)
	//}
	// TODO
	for _, workspacePath := range workspacePaths {
		c.logger.Info("wiki start to process open_workspace event: %s", workspacePath)
		err := c.ProcessOpenWorkspaceEvent(ctx, &model.Event{
			WorkspacePath: workspacePath,
		})
		if err != nil {
			c.logger.Error("failed to process open_workspace event for codegraph: %v", err)
			continue
		}
		c.logger.Info("wiki process open_workspace event successfully: %s", workspacePath)
	}

	return nil
}

func (c *WikiProcessor) convertWorkspaceFilePathToAbs(event *model.Event) {
	sourcePath := event.SourceFilePath
	targetPath := event.TargetFilePath
	workspacePath := event.WorkspacePath
	if !utils.IsSubdir(workspacePath, sourcePath) {
		event.SourceFilePath = filepath.Join(workspacePath, event.SourceFilePath)
	}
	if !utils.IsSubdir(workspacePath, targetPath) {
		event.TargetFilePath = filepath.Join(workspacePath, event.TargetFilePath)
	}
}
