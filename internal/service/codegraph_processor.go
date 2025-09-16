package service

import (
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/logger"
)

// var _ EventProcessService = (*CodegraphProcessor)(nil)

type CodegraphProcessService interface {
	ProcessActiveWorkspaces(ctx context.Context) ([]*model.Workspace, error)
	ProcessAddFileEvent(ctx context.Context, event *model.Event) error
	ProcessModifyFileEvent(ctx context.Context, event *model.Event) error
	ProcessDeleteFileEvent(ctx context.Context, event *model.Event) error
	ProcessRenameFileEvent(ctx context.Context, event *model.Event) error
	ProcessOpenWorkspaceEvent(ctx context.Context, event *model.Event) error
	ProcessRebuildWorkspaceEvent(ctx context.Context, event *model.Event) error
	ProcessEvents(ctx context.Context, workspacePaths []string) error
}

type CodegraphProcessor struct {
	indexer         Indexer
	workspaceReader workspace.WorkspaceReader
	workspaceRepo   repository.WorkspaceRepository
	eventRepo       repository.EventRepository
	logger          logger.Logger
}

func NewCodegraphProcessor(
	workspaceReader workspace.WorkspaceReader,
	indexer Indexer,
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	logger logger.Logger,
) CodegraphProcessService {
	return &CodegraphProcessor{
		workspaceReader: workspaceReader,
		indexer:         indexer,
		workspaceRepo:   workspaceRepo,
		eventRepo:       eventRepo,
		logger:          logger,
	}
}

// ProcessActiveWorkspaces 扫描活跃工作区
func (ep *CodegraphProcessor) ProcessActiveWorkspaces(ctx context.Context) ([]*model.Workspace, error) {
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
func (c *CodegraphProcessor) ProcessAddFileEvent(ctx context.Context, event *model.Event) error {
	fileInfo, err := c.workspaceReader.Stat(event.SourceFilePath)
	if errors.Is(err, workspace.ErrPathNotExists) {
		c.logger.Error("codegraph failed to process add event, file %s not exists.", event.SourceFilePath)
		if err = c.updateEventStatusFinally(event, err); err != nil {
			return fmt.Errorf("codegraph update add event %d err: %w", event.ID, err)
		}
		return err
	}

	if fileInfo.IsDir {
		c.logger.Error("codegraph add event, file %s is dir, not process.", event.SourceFilePath)
		if err = c.updateEventStatusFinally(event, nil); err != nil {
			return fmt.Errorf("codegraph update add event %d err: %w", event.ID, err)
		}
		return nil
	}

	// 使用索引器索引文件
	err = c.indexer.IndexFiles(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err = c.updateEventStatusFinally(event, err); err != nil {
		return fmt.Errorf("codegraph update add event %d err: %w", event.ID, err)
	}

	return nil
}

// ProcessModifyFileEvent 处理修改文件事件
func (c *CodegraphProcessor) ProcessModifyFileEvent(ctx context.Context, event *model.Event) error {
	fileInfo, err := c.workspaceReader.Stat(event.SourceFilePath)
	if errors.Is(err, workspace.ErrPathNotExists) {
		c.logger.Error("codegraph failed to process modify event, file %s not exists", event.SourceFilePath)
		if err = c.updateEventStatusFinally(event, err); err != nil {
			return fmt.Errorf("codegraph update modify event err: %w", err)
		}
		return err
	}

	if fileInfo.IsDir {
		c.logger.Error("codegraph modify event, file %s is dir, not process.", event.SourceFilePath)
		if err = c.updateEventStatusFinally(event, nil); err != nil {
			return fmt.Errorf("codegraph update modify event err: %w", err)
		}
		return nil
	}
	// 删除之前已经存在的索引
	err = c.indexer.RemoveIndexes(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err != nil {
		return fmt.Errorf("codegraph update modify event %d err: %w", event.ID, err)
	}
	// 使用索引器重新索引文件
	err = c.indexer.IndexFiles(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err = c.updateEventStatusFinally(event, err); err != nil {
		return fmt.Errorf("codegraph update modify event %d err: %w", event.ID, err)
	}

	return nil
}

// ProcessDeleteFileEvent 处理删除文件/目录事件
func (c *CodegraphProcessor) ProcessDeleteFileEvent(ctx context.Context, event *model.Event) error {
	// 使用索引器删除文件索引
	err := c.indexer.RemoveIndexes(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err = c.updateEventStatusFinally(event, err); err != nil {
		return fmt.Errorf("codegraph update delete event %d err: %w", event.ID, err)
	}
	return nil
}

// ProcessRenameFileEvent 处理重命名文件/目录事件
func (c *CodegraphProcessor) ProcessRenameFileEvent(ctx context.Context, event *model.Event) error {
	err := c.indexer.RenameIndexes(ctx, event.WorkspacePath, event.SourceFilePath, event.TargetFilePath)
	if err = c.updateEventStatusFinally(event, err); err != nil {
		return fmt.Errorf("codegraph update rename event %d err: %w", event.ID, err)
	}
	return nil
}

func (c *CodegraphProcessor) ProcessRebuildWorkspaceEvent(ctx context.Context, event *model.Event) error {

	if err := c.indexer.RemoveAllIndexes(ctx, event.WorkspacePath); err != nil {
		return err
	}

	if err := c.ProcessOpenWorkspaceEvent(ctx, event); err != nil {
		return err
	}
	return nil
}

func (c *CodegraphProcessor) ProcessOpenWorkspaceEvent(ctx context.Context, event *model.Event) error {
	// TODO 增加比对逻辑，如果构建过索引，进行比对。
	fileInfo, err := c.workspaceReader.Stat(event.WorkspacePath)
	if errors.Is(err, workspace.ErrPathNotExists) {
		c.logger.Error("codegraph failed to process open_workspace event event, workspace %s not exists",
			event.WorkspacePath)
		if err = c.updateEventStatusFinally(event, err); err != nil {
			return fmt.Errorf("codegraph update open_workspace event err: %w", err)
		}
		return err
	}

	if !fileInfo.IsDir {
		c.logger.Error("codegraph open_workspace event, %s is file, not process.",
			event.WorkspacePath)
		if err = c.updateEventStatusFinally(event, nil); err != nil {
			return fmt.Errorf("codegraph update open_workspace event err: %w", err)
		}
		return nil
	}

	// 更新进度为0，成功后再更新总进度。
	err = c.workspaceRepo.UpdateCodegraphInfo(event.WorkspacePath, 0, time.Now().Unix())
	if err != nil {
		c.logger.Error("codegraph failed to process open_workspace event event, workspace %s reset successful file num failed, err:%v",
			event.WorkspacePath, err)
		if err = c.updateEventStatusFinally(event, err); err != nil {
			return fmt.Errorf("codegraph update open_workspace event err: %w", err)
		}
		return err
	}
	// todo open_workspace过程中会更新进度，其余事件结束更新进度。
	_, err = c.indexer.IndexWorkspace(ctx, event.WorkspacePath)
	if err = c.updateEventStatusFinally(event, err); err != nil {
		return fmt.Errorf("codegraph update modify event %d err: %w", event.ID, err)
	}

	return nil
}

// ProcessEvents 处理事件记录
func (c *CodegraphProcessor) ProcessEvents(ctx context.Context, workspacePaths []string) error {

	codegraphStatuses := []int{
		model.CodegraphStatusInit,
	}
	// 重新构建事件
	rebuildEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeRebuildWorkspace}, workspacePaths, 10,
		false, nil, codegraphStatuses)

	if err != nil {
		c.logger.Error("failed to get rebuild_workspace events: %v", err)
		return fmt.Errorf("failed to get rebuild_workspace events: %w", err)
	}

	// 处理重新构建事件
	for _, event := range rebuildEvents {
		// 转换路径
		c.convertWorkspaceFilePathToAbs(event)
		c.logger.Info("codegraph start to process rebuild_workspace event: %s", event.WorkspacePath)
		err = c.ProcessRebuildWorkspaceEvent(ctx, event)
		if err != nil {
			c.logger.Error("failed to process rebuild_workspace event for codegraph: %v", err)
			continue
		}
		c.logger.Info("codegraph process rebuild_workspace event successfully: %s", event.WorkspacePath)
	}

	// 打开工作区事件
	openEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeOpenWorkspace}, workspacePaths, 10,
		false, nil, codegraphStatuses)

	if err != nil {
		c.logger.Error("failed to get open_workspace events: %v", err)
		return fmt.Errorf("failed to get open_workspace events: %w", err)
	}

	// 处理打开工作区事件
	for _, event := range openEvents {
		c.convertWorkspaceFilePathToAbs(event)
		c.logger.Info("codegraph start to process open_workspace event: %s", event.WorkspacePath)
		err = c.ProcessOpenWorkspaceEvent(ctx, event)
		if err != nil {
			c.logger.Error("failed to process open_workspace event for codegraph: %v", err)
			continue
		}
		c.logger.Info("codegraph process open_workspace event successfully: %s", event.WorkspacePath)
	}

	// 处理添加文件事件
	addEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeAddFile}, workspacePaths, 10,
		false, nil, codegraphStatuses)

	if err != nil {
		c.logger.Error("failed to get add file events: %v", err)
		return fmt.Errorf("failed to get add file events: %w", err)
	}

	// 处理添加文件事件
	for _, event := range addEvents {
		c.convertWorkspaceFilePathToAbs(event)
		c.logger.Info("codegraph start to process add_file event: %s", event.SourceFilePath)
		err = c.ProcessAddFileEvent(ctx, event)
		if err != nil {
			c.logger.Error("failed to process add file event for codegraph: %v", err)
			continue
		}
		c.logger.Info("codegraph process add_file event successfully: %s", event.SourceFilePath)
	}

	// 处理修改文件事件
	modifyEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeModifyFile}, workspacePaths, 10,
		false, nil, codegraphStatuses)

	if err != nil {
		c.logger.Error("failed to get modify file events: %v", err)
		return fmt.Errorf("failed to get modify file events: %w", err)
	}

	for _, event := range modifyEvents {
		c.convertWorkspaceFilePathToAbs(event)
		c.logger.Info("codegraph start to process modify_file event: %s", event.SourceFilePath)
		err = c.ProcessModifyFileEvent(ctx, event)
		if err != nil {
			c.logger.Error("failed to process modify file event for codegraph: %v", err)
			continue
		}
		c.logger.Info("codegraph process modify_file event successfully: %s", event.SourceFilePath)
	}

	// 处理删除文件事件
	deleteEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeDeleteFile}, workspacePaths, 10,
		false, nil, codegraphStatuses)

	if err != nil {
		c.logger.Error("failed to get delete file events: %v", err)
		return fmt.Errorf("failed to get delete file events: %w", err)
	}

	for _, event := range deleteEvents {
		c.convertWorkspaceFilePathToAbs(event)
		c.logger.Info("codegraph start to process delete_file event: %s", event.SourceFilePath)
		err = c.ProcessDeleteFileEvent(ctx, event)
		if err != nil {
			c.logger.Error("failed to process delete file event for codegraph: %v", err)
			continue
		}
		c.logger.Info("codegraph process delete_file event successfully: %s", event.SourceFilePath)
	}

	// 处理重命名事件
	renameEvents, err := c.eventRepo.GetEventsByTypeAndStatusAndWorkspaces([]string{model.EventTypeRenameFile}, workspacePaths, 10,
		false, nil, codegraphStatuses)

	if err != nil {
		c.logger.Error("failed to get rename file events: %v", err)
		return fmt.Errorf("failed to get rename file events: %w", err)
	}

	for _, event := range renameEvents {
		c.convertWorkspaceFilePathToAbs(event)
		c.logger.Info("codegraph start to process rename_file event: source %s target %s",
			event.SourceFilePath, event.TargetFilePath)
		err = c.ProcessRenameFileEvent(ctx, event)
		if err != nil {
			c.logger.Error("failed to process rename file event for codegraph: %v", err)
			continue
		}
		c.logger.Info("codegraph process rename_file event successfully: source %s target %s",
			event.SourceFilePath, event.TargetFilePath)

	}
	return nil
}

func (c *CodegraphProcessor) updateEventStatusFinally(event *model.Event, err error) error {
	updatedEvent := &model.Event{ID: event.ID}
	if err != nil {
		// 更新事件
		updatedEvent.CodegraphStatus = model.CodegraphStatusFailed
		updatedEvent.UpdatedAt = time.Now()
		updateErr := c.eventRepo.UpdateEvent(updatedEvent)
		if updateErr != nil {
			return fmt.Errorf("failed to update failed processed event. update err: %w, index err: %w", updateErr, err)
		}
		return err
	}

	// 更新状态为成功
	updatedEvent.CodegraphStatus = model.CodegraphStatusSuccess
	updatedEvent.UpdatedAt = time.Now()

	updateErr := c.eventRepo.UpdateEvent(updatedEvent)
	if updateErr != nil {
		return fmt.Errorf("failed to update success processed event. update err: %w", updateErr)
	}
	return nil
}

func (c *CodegraphProcessor) convertWorkspaceFilePathToAbs(event *model.Event) {
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
