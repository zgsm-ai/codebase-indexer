package handler

import (
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zgsm-ai/codebase-indexer/internal/logic"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func semanticSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SemanticSearchRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}

		l := logic.NewSemanticSearchLogic(r.Context(), svcCtx)
		resp, err := l.SemanticSearch(&req)
		if err != nil {
			response.Error(w, err)
		} else {
			response.Json(w, resp)
		}
	}
}
