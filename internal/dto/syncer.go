package dto

type CodebaseHashReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
}

type CodebaseHashResp struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    CodebaseHashRespData `json:"data"`
}

type CodebaseHashRespData struct {
	List []HashItem `json:"list"`
}

type HashItem struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

type UploadReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
}
