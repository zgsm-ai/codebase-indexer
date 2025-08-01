package service

import (
	"fmt"
	"math/rand"
	"time"

	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/logger"
)

// StatusService 状态检查服务接口
type EmbeddingStatusService interface {
	CheckFileBuildingStatus(state *model.EmbeddingState) bool
	CheckAllBuildingStates() error
}

// embeddingStatusService 状态检查服务实现
type embeddingStatusService struct {
	embeddingStateRepo repository.EmbeddingStateRepository
	workspaceRepo      repository.WorkspaceRepository
	logger             logger.Logger
}

// NewEmbeddingStatusService 创建状态检查服务
func NewEmbeddingStatusService(
	embeddingStateRepo repository.EmbeddingStateRepository,
	workspaceRepo repository.WorkspaceRepository,
	logger logger.Logger,
) EmbeddingStatusService {
	return &embeddingStatusService{
		embeddingStateRepo: embeddingStateRepo,
		workspaceRepo:      workspaceRepo,
		logger:             logger,
	}
}

// CheckFileBuildingStatus 检查文件构建状态
func (sc *embeddingStatusService) CheckFileBuildingStatus(state *model.EmbeddingState) bool {
	// 计算当前时间与状态更新时间的差值
	now := time.Now()
	duration := now.Sub(state.UpdatedAt)

	// 如果时间差超过5秒，则进行状态更新
	if duration.Seconds() > 5 {
		sc.logger.Info("updating file building status: %s (duration: %v)", state.FilePath, duration)

		// 生成一个随机数，模拟80%的概率构建成功
		random := rand.Intn(100)
		if random < 80 {
			// 构建成功
			state.Status = model.EmbeddingStatusSuccess
			state.Message = "构建成功"
		} else {
			// 构建失败
			state.Status = model.EmbeddingStatusBuildFailed
			state.Message = "构建失败"
		}

		// 更新状态记录
		err := sc.embeddingStateRepo.UpdateEmbeddingState(state)
		if err != nil {
			sc.logger.Error("failed to update embedding state: %v", err)
			return false
		}

		// 更新工作区信息
		nowTime := now.Unix()
		states, err := sc.embeddingStateRepo.GetEmbeddingStatesByWorkspace(state.WorkspacePath)
		if err != nil {
			sc.logger.Error("failed to get embedding states: %v", err)
			return true
		}

		fileNum := len(states)
		err = sc.workspaceRepo.UpdateEmbeddingInfo(state.WorkspacePath, fileNum, nowTime)
		if err != nil {
			sc.logger.Error("failed to update workspace embedding info: %v", err)
			return false
		}

		sc.logger.Info("updated file building status: %s -> %d", state.FilePath, state.Status)
		return true
	}

	return false
}

// CheckAllBuildingStates 检查所有building状态
func (sc *embeddingStatusService) CheckAllBuildingStates() error {
	// 获取所有building状态的记录
	buildingStates, err := sc.embeddingStateRepo.GetEmbeddingStatesByStatus(model.EmbeddingStatusBuilding)
	if err != nil {
		sc.logger.Error("failed to get building states: %v", err)
		return fmt.Errorf("failed to get building states: %w", err)
	}

	if len(buildingStates) == 0 {
		return nil
	}

	sc.logger.Info("checking %d building states", len(buildingStates))
	// 检查每个building状态
	for _, state := range buildingStates {
		updated := sc.CheckFileBuildingStatus(state)
		if updated {
			sc.logger.Info("file building status updated: %s", state.FilePath)
		}
	}
	sc.logger.Info("checking %d building states done", len(buildingStates))

	return nil
}
