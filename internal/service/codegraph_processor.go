package service

import (
	"context"
	"fmt"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph"
	"codebase-indexer/pkg/logger"
)

// var _ EventProcessService = (*CodegraphProcessor)(nil)

type CodegraphProcessService interface {
	ProcessActiveWorkspaces() ([]*model.Workspace, error)
	ProcessAddFileEvent(event *model.Event) error
	ProcessModifyFileEvent(event *model.Event) error
	ProcessDeleteFileEvent(event *model.Event) error
	ProcessEvents() error
}

type CodegraphProcessor struct {
	indexer       *codegraph.Indexer
	workspaceRepo repository.WorkspaceRepository
	eventRepo     repository.EventRepository
	logger        logger.Logger
}

func NewCodegraphProcessor(
	indexer *codegraph.Indexer,
	workspaceRepo repository.WorkspaceRepository,
	eventRepo repository.EventRepository,
	logger logger.Logger,
) CodegraphProcessService {
	return &CodegraphProcessor{
		indexer:       indexer,
		workspaceRepo: workspaceRepo,
		eventRepo:     eventRepo,
		logger:        logger,
	}
}

// ProcessActiveWorkspaces 扫描活跃工作区
func (ep *CodegraphProcessor) ProcessActiveWorkspaces() ([]*model.Workspace, error) {
	workspaces, err := ep.workspaceRepo.GetActiveWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get active workspaces: %w", err)
	}

	var activeWorkspaces []*model.Workspace
	for _, workspace := range workspaces {
		if workspace.Active == "true" {
			activeWorkspaces = append(activeWorkspaces, workspace)
		}
	}

	return activeWorkspaces, nil
}

// ProcessAddFileEvent 处理添加文件事件
func (c *CodegraphProcessor) ProcessAddFileEvent(event *model.Event) error {
	c.logger.Info("processing add file event for codegraph: %s", event.SourceFilePath)

	// 创建代码图构建状态记录
	// state := &model.CodegraphState{
	// 	WorkspacePath: event.WorkspacePath,
	// 	FilePath:      event.SourceFilePath,
	// 	Status:        model.CodegraphStatusBuilding,
	// 	Message:       "文件等待构建代码图",
	// }

	// err := c.codegraphStateRepository.CreateCodegraphState(state)
	// if err != nil {
	// 	c.logger.Error("failed to create codegraph state for file %s: %v", event.SourceFilePath, err)
	// 	return fmt.Errorf("failed to create codegraph state: %w", err)
	// }

	// 使用索引器索引文件
	ctx := context.Background()
	err := c.indexer.IndexFiles(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err != nil {
		c.logger.Error("failed to index file %s: %v", event.SourceFilePath, err)
		// 更新状态为失败
		// updateErr := c.codegraphStateRepository.UpdateCodegraphStateStatus(event.WorkspacePath, event.SourceFilePath, model.CodegraphStatusFailed, fmt.Sprintf("索引失败: %v", err))
		// if updateErr != nil {
		// 	c.logger.Error("failed to update codegraph state status for file %s: %v", event.SourceFilePath, updateErr)
		// }
		return fmt.Errorf("failed to index file: %w", err)
	}

	// 更新状态为成功
	// err = c.codegraphStateRepository.UpdateCodegraphStateStatus(event.WorkspacePath, event.SourceFilePath, model.CodegraphStatusSuccess, "代码图构建成功")
	// if err != nil {
	// 	c.logger.Error("failed to update codegraph state status for file %s: %v", event.SourceFilePath, err)
	// 	return fmt.Errorf("failed to update codegraph state status: %w", err)
	// }

	// 删除已处理的事件
	err = c.eventRepo.DeleteEvent(event.ID)
	if err != nil {
		c.logger.Error("failed to delete processed event %d: %v", event.ID, err)
		return fmt.Errorf("failed to delete processed event: %w", err)
	}

	return nil
}

// ProcessModifyFileEvent 处理修改文件事件
func (c *CodegraphProcessor) ProcessModifyFileEvent(event *model.Event) error {
	c.logger.Info("processing modify file event for codegraph: %s", event.SourceFilePath)

	// 检查是否已存在代码图构建状态记录
	// state, err := c.codegraphStateRepository.GetCodegraphStateByFile(event.WorkspacePath, event.SourceFilePath)
	// if err != nil {
	// 	// 不存在记录，创建新的
	// 	state = &model.CodegraphState{
	// 		WorkspacePath: event.WorkspacePath,
	// 		FilePath:      event.SourceFilePath,
	// 		Status:        model.CodegraphStatusBuilding,
	// 		Message:       "文件修改，等待重新构建代码图",
	// 	}

	// 	err = c.codegraphStateRepository.CreateCodegraphState(state)
	// 	if err != nil {
	// 		c.logger.Error("failed to create codegraph state for modified file %s: %v", event.SourceFilePath, err)
	// 		return fmt.Errorf("failed to create codegraph state: %w", err)
	// 	}
	// } else {
	// 	// 已存在记录，更新状态为构建中
	// 	state.Status = model.CodegraphStatusBuilding
	// 	state.Message = "文件修改，等待重新构建代码图"

	// 	err = c.codegraphStateRepository.UpdateCodegraphState(state)
	// 	if err != nil {
	// 		c.logger.Error("failed to update codegraph state for modified file %s: %v", event.SourceFilePath, err)
	// 		return fmt.Errorf("failed to update codegraph state: %w", err)
	// 	}
	// }

	// 使用索引器重新索引文件
	ctx := context.Background()
	err := c.indexer.IndexFiles(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err != nil {
		c.logger.Error("failed to reindex modified file %s: %v", event.SourceFilePath, err)
		// 更新状态为失败
		// updateErr := c.codegraphStateRepository.UpdateCodegraphStateStatus(event.WorkspacePath, event.SourceFilePath, model.CodegraphStatusFailed, fmt.Sprintf("重新索引失败: %v", err))
		// if updateErr != nil {
		// 	c.logger.Error("failed to update codegraph state status for file %s: %v", event.SourceFilePath, updateErr)
		// }
		return fmt.Errorf("failed to reindex file: %w", err)
	}

	// 更新状态为成功
	// err = c.codegraphStateRepository.UpdateCodegraphStateStatus(event.WorkspacePath, event.SourceFilePath, model.CodegraphStatusSuccess, "代码图重新构建成功")
	// if err != nil {
	// 	c.logger.Error("failed to update codegraph state status for file %s: %v", event.SourceFilePath, err)
	// 	return fmt.Errorf("failed to update codegraph state status: %w", err)
	// }

	// 删除已处理的事件
	err = c.eventRepo.DeleteEvent(event.ID)
	if err != nil {
		c.logger.Error("failed to delete processed event %d: %v", event.ID, err)
		return fmt.Errorf("failed to delete processed event: %w", err)
	}

	return nil
}

// ProcessDeleteFileEvent 处理删除文件事件
func (c *CodegraphProcessor) ProcessDeleteFileEvent(event *model.Event) error {
	c.logger.Info("processing delete file event for codegraph: %s", event.SourceFilePath)

	// 使用索引器删除文件索引
	ctx := context.Background()
	err := c.indexer.RemoveIndexes(ctx, event.WorkspacePath, []string{event.SourceFilePath})
	if err != nil {
		c.logger.Error("failed to remove indexes for file %s: %v", event.SourceFilePath, err)
		// 继续执行，因为即使索引删除失败，我们也需要删除状态记录和事件
	}

	// 检查是否已存在代码图构建状态记录
	// _, err = c.codegraphStateRepository.GetCodegraphStateByFile(event.WorkspacePath, event.SourceFilePath)
	// if err == nil {
	// 	// 存在记录，删除它
	// 	err = c.codegraphStateRepository.DeleteCodegraphState(event.WorkspacePath, event.SourceFilePath)
	// 	if err != nil {
	// 		c.logger.Error("failed to delete codegraph state for deleted file %s: %v", event.SourceFilePath, err)
	// 		return fmt.Errorf("failed to delete codegraph state: %w", err)
	// 	}
	// }

	// 删除已处理的事件
	err = c.eventRepo.DeleteEvent(event.ID)
	if err != nil {
		c.logger.Error("failed to delete processed event %d: %v", event.ID, err)
		return fmt.Errorf("failed to delete processed event: %w", err)
	}

	return nil
}

// ProcessEvents 处理事件记录
func (c *CodegraphProcessor) ProcessEvents() error {
	// 获取待处理的添加文件事件
	events, err := c.eventRepo.GetEventsByType(model.EventTypeAddFile, 10, false)
	if err != nil {
		c.logger.Error("failed to get add file events: %v", err)
		return fmt.Errorf("failed to get add file events: %w", err)
	}

	// 处理添加文件事件
	for _, event := range events {
		err = c.ProcessAddFileEvent(event)
		if err != nil {
			c.logger.Error("failed to process add file event for codegraph: %v", err)
			continue
		}
	}

	// 获取修改文件事件
	modifyEvents, err := c.eventRepo.GetEventsByType(model.EventTypeModifyFile, 10, false)
	if err != nil {
		c.logger.Error("failed to get modify file events: %v", err)
		return fmt.Errorf("failed to get modify file events: %w", err)
	}

	// 处理修改文件事件
	for _, event := range modifyEvents {
		err = c.ProcessModifyFileEvent(event)
		if err != nil {
			c.logger.Error("failed to process modify file event for codegraph: %v", err)
			continue
		}
	}

	// 获取删除文件事件
	deleteEvents, err := c.eventRepo.GetEventsByType(model.EventTypeDeleteFile, 10, false)
	if err != nil {
		c.logger.Error("failed to get delete file events: %v", err)
		return fmt.Errorf("failed to get delete file events: %w", err)
	}

	// 处理删除文件事件
	for _, event := range deleteEvents {
		err = c.ProcessDeleteFileEvent(event)
		if err != nil {
			c.logger.Error("failed to process delete file event for codegraph: %v", err)
			continue
		}
	}

	return nil
}
