package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/codegraph/definition"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxReadLine = 5000

type StructureLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFileDefinitionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StructureLogic {
	return &StructureLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StructureLogic) ParseFileDefinitions(req *types.FileDefinitionParseRequest) (resp *types.FileDefinitionResponseData, err error) {
	clientId := req.ClientId
	clientPath := req.CodebasePath
	filePath := req.FilePath

	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}

	//TODO check param
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))

	bytes, err := l.svcCtx.CodebaseStore.Read(ctx, codebase.Path, filePath, types.ReadOptions{EndLine: maxReadLine})
	if err != nil {
		return nil, err
	}

	parsed, err := l.svcCtx.FileDefinitionParser.Parse(ctx, &types.SourceFile{
		CodebasePath: codebase.Path,
		Path:         filePath,
		Content:      bytes,
	}, definition.ParseOptions{IncludeContent: true})
	if err != nil {
		return nil, err
	}
	resp = new(types.FileDefinitionResponseData)
	for _, d := range parsed.Definitions {
		resp.List = append(resp.List, &types.FileDefitnionItem{
			Name:     d.Name,
			ItemType: d.Type,
			Position: types.ToPosition(d.Range),
			Content:  string(d.Content),
		})
	}
	return resp, nil
}
