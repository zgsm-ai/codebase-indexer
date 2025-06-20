package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"gorm.io/gorm"

	codebasestore "github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"crypto/sha256"
	"encoding/hex"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
)

type CodebaseHash struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCompareCodebaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CodebaseHash {
	return &CodebaseHash{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CodebaseHash) GetCodebaseHash(req *types.CodebaseHashRequest) (resp *types.CodebaseHashResponseData, err error) {
	clientPath := req.CodebasePath
	clientId := req.ClientId
	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logx.Errorf("clientId: %s, clientPath: %s, codebase not exists in database", clientId, clientPath)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	codebasePath := codebase.Path
	if utils.IsBlank(codebasePath) {
		return nil, errors.New("illegal database data, codebase path is empty")
	}

	// 构建响应数据
	resp = &types.CodebaseHashResponseData{
		CodebaseHash: make([]*types.CodebaseFileHashItem, 0),
	}
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))
	// 使用 Walk 方法递归遍历目录树
	err = l.svcCtx.CodebaseStore.Walk(ctx, codebasePath, "", func(walkCtx *codebasestore.WalkContext, reader io.ReadCloser) error {
		if walkCtx == nil || reader == nil {
			return nil
		}
		// 跳过目录和隐藏文件
		if walkCtx.Info.IsDir {
			if walkCtx.Info.IsDir {
				return codebasestore.SkipDir
			}
			return nil
		}

		// 读取文件内容
		content, err := io.ReadAll(reader)
		if err != nil {
			l.Logger.Errorf("failed to read file %s: %v", walkCtx.RelativePath, err)
			return nil
		}

		// 计算文件内容的 SHA-256 哈希
		hash := sha256.Sum256(content)
		hashStr := hex.EncodeToString(hash[:])

		// 添加到响应数据中
		resp.CodebaseHash = append(resp.CodebaseHash, &types.CodebaseFileHashItem{
			Path: walkCtx.RelativePath,
			Hash: hashStr,
		})

		return nil
	}, codebasestore.WalkOptions{IgnoreError: true, ExcludePrefixes: []string{"."}})

	// 不存在，返回空即可
	if err != nil && !errors.Is(err, codebasestore.ErrCodebasePathNotExists) {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return resp, nil
}
