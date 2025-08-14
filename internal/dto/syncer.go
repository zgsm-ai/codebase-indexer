package dto

// CodebaseHashReq 代码库哈希请求
type CodebaseHashReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
}

// CodebaseHashResp 代码库哈希响应
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

// UploadReq 上传文件请求
type UploadReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
	RequestId    string `json:"requestId"`
	UploadToken  string `json:"uploadToken"`
}

// UploadTokenReq 获取上传令牌请求
type UploadTokenReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
}

// UploadTokenResp 获取上传令牌响应
type UploadTokenResp struct {
	Code    int                 `json:"code"`
	Message string              `json:"message"`
	Data    UploadTokenRespData `json:"data"`
}

type UploadTokenRespData struct {
	Token     string `json:"token"`
	UserCount int    `json:"userCount"`
}

// FileStatusReq 文件状态请求
type FileStatusReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
	SyncId       string `json:"syncId"`
}

// FileStatusResp 文件状态响应
type FileStatusResp struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    FileStatusRespData `json:"data"`
}

type FileStatusRespData struct {
	Process       string                       `json:"process"`
	TotalProcess  int                          `json:"totalProcess"`
	FileSuceesNum int                          `json:"fileSuceesNum"`
	FileList      []FileStatusRespFileListItem `json:"fileList"`
}

type FileStatusRespFileListItem struct {
	Path    string `json:"path"`
	Operate string `json:"operate"`
	Status  string `json:"status"`
}

// DeleteEmbeddingReq 删除嵌入请求
type DeleteEmbeddingReq struct {
	ClientId     string   `json:"clientId"`
	CodebasePath string   `json:"codebasePath"`
	FilePaths    []string `json:"filePaths"`
}

// DeleteEmbeddingResp 删除嵌入响应
type DeleteEmbeddingResp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}
