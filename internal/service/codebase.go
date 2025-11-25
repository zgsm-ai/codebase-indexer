package service

import (
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/errs"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph/definition"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/response"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"codebase-indexer/internal/config"
	"codebase-indexer/pkg/logger"
)

// CodebaseService 处理代码库相关的业务逻辑
type CodebaseService interface {
	// FindCodebasePaths 查找指定路径下的代码库配置
	FindCodebasePaths(ctx context.Context, basePath, baseName string) ([]config.CodebaseConfig, error)

	// IsGitRepository 检查路径是否为Git仓库
	IsGitRepository(ctx context.Context, path string) bool

	// GenerateCodebaseID 生成代码库唯一ID
	GenerateCodebaseID(name, path string) string

	// GetFileContent 读取文件内容
	GetFileContent(ctx context.Context, req *dto.GetFileContentRequest) ([]byte, error)

	// GetCodebaseDirectoryTree 获取代码库目录树结构
	GetCodebaseDirectoryTree(ctx context.Context, req *dto.GetCodebaseDirectoryRequest) (*dto.DirectoryData, error)

	// ParseFileDefinitions 解析文件中的定义信息（如函数、类等）
	ParseFileDefinitions(ctx context.Context, req *dto.GetFileStructureRequest) (*dto.FileStructureData, error)

	// QueryDefinition 查询代码定义（支持按行号或代码片段检索）
	QueryDefinition(ctx context.Context, req *dto.SearchDefinitionRequest) (*dto.DefinitionData, error)

	// QueryReference 查询代码间的关系（如调用、引用等）
	QueryReference(ctx context.Context, req *dto.SearchReferenceRequest) (*dto.ReferenceData, error)

	// QueryCallGraph 查询代码片段内部元素或单符号的调用链及其里面的元素定义，支持代码片段检索
	QueryCallGraph(ctx context.Context, req *dto.SearchCallGraphRequest) (*dto.CallGraphData, error)

	// Summarize 获取代码库索引摘要信息
	Summarize(ctx context.Context, req *dto.GetIndexSummaryRequest) (*dto.IndexSummary, error)

	// DeleteIndex 删除代码库的索引（支持按类型删除）
	DeleteIndex(ctx context.Context, req *dto.DeleteIndexRequest) error
	ExportIndex(c *gin.Context, d *dto.ExportIndexRequest) error
	ReadCodeSnippets(c *gin.Context, d *dto.ReadCodeSnippetsRequest) (*dto.CodeSnippetsData, error)

	// GetFileSkeleton 获取文件骨架（FileElementTable）
	GetFileSkeleton(ctx context.Context, req *dto.GetFileSkeletonRequest) (*dto.FileSkeletonData, error)
}

const maxReadLine = 5000
const definitionFillContentNodeLimit = 20
const definitionFillContentLineLimit = 200
const DefaultMaxCodeSnippetLines = 500
const DefaultMaxCodeSnippets = 200

// NewCodebaseService 创建新的代码库服务
func NewCodebaseService(
	manager repository.StorageInterface,
	logger logger.Logger,
	workspaceReader workspace.WorkspaceReader,
	workspaceRepository repository.WorkspaceRepository,
	fileDefinitionParser *definition.DefParser,
	indexer Indexer) CodebaseService {
	return &codebaseService{
		manager:              manager,
		logger:               logger,
		workspaceReader:      workspaceReader,
		workspaceRepository:  workspaceRepository,
		fileDefinitionParser: fileDefinitionParser,
		indexer:              indexer,
	}
}

type codebaseService struct {
	manager              repository.StorageInterface
	logger               logger.Logger
	workspaceReader      workspace.WorkspaceReader
	workspaceRepository  repository.WorkspaceRepository
	fileDefinitionParser *definition.DefParser
	indexer              Indexer
	mu                   sync.Mutex
}

func (s *codebaseService) checkPath(_ context.Context, workspacePath string, filePaths []string) error {
	for _, filePath := range filePaths {
		if filePath != types.EmptyString && !utils.IsSubdir(workspacePath, filePath) {
			return fmt.Errorf("cannot access path %s which not in workspace %s", filePath, workspacePath)
		}
	}
	_, err := s.workspaceRepository.GetWorkspaceByPath(workspacePath)
	return err
}

func (s *codebaseService) ExportIndex(c *gin.Context, d *dto.ExportIndexRequest) error {
	projects := s.workspaceReader.FindProjects(c, d.CodebasePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return fmt.Errorf("can not find project in workspace %s", d.CodebasePath)
	}
	downloader := response.NewDownloader(c, fmt.Sprintf("%s-index.json", d.CodebasePath))
	defer downloader.Finish()
	for _, project := range projects {
		summary, _ := s.indexer.GetSummary(c, d.CodebasePath)
		s.logger.Debug("workspace %s has %d indexes", d.CodebasePath, summary.TotalFiles)
		iter := s.indexer.IndexIter(c, project.Uuid)
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			if store.IsElementPathKey(key) {
				var fileTable codegraphpb.FileElementTable
				if err := store.UnmarshalValue(value, &fileTable); err != nil {
					return err
				} else {
					bytes, err := json.Marshal(&fileTable)
					if err == nil {
						_ = downloader.Write(bytes)
						_ = downloader.Write([]byte("\n"))
					}
				}
			} else if store.IsSymbolNameKey(key) {
				var sym codegraphpb.SymbolOccurrence
				if err := store.UnmarshalValue(value, &sym); err != nil {
					return err
				} else {
					bytes, err := json.Marshal(&sym)
					if err == nil {
						_ = downloader.Write(bytes)
						_ = downloader.Write([]byte("\n"))
					}
				}
			}
		}
		_ = iter.Close()
	}
	return nil
}

// FindCodebasePaths 查找指定路径下的代码库配置
func (s *codebaseService) FindCodebasePaths(ctx context.Context, basePath, baseName string) ([]config.CodebaseConfig, error) {
	var configs []config.CodebaseConfig

	if s.IsGitRepository(ctx, basePath) {
		s.logger.Info("path %s is a git repository", basePath)
		configs = append(configs, config.CodebaseConfig{
			CodebasePath: basePath,
			CodebaseName: baseName,
		})
		return configs, nil
	}

	subDirs, err := os.ReadDir(basePath)
	if err != nil {
		s.logger.Error("failed to read directory %s: %v", basePath, err)
		return nil, fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}

	foundSubRepo := false
	for _, entry := range subDirs {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			subDirPath := filepath.Join(basePath, entry.Name())
			if s.IsGitRepository(ctx, subDirPath) {
				configs = append(configs, config.CodebaseConfig{
					CodebasePath: subDirPath,
					CodebaseName: entry.Name(),
				})
				foundSubRepo = true
			}
		}
	}

	if !foundSubRepo {
		configs = append(configs, config.CodebaseConfig{
			CodebasePath: basePath,
			CodebaseName: baseName,
		})
	}

	return configs, nil
}

// IsGitRepository 检查路径是否为Git仓库
func (s *codebaseService) IsGitRepository(ctx context.Context, path string) bool {
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		return true
	}

	// 检查是否为子模块（.git文件）
	gitFile := filepath.Join(path, ".git")
	if info, err := os.Stat(gitFile); err == nil && !info.IsDir() {
		return true
	}

	return false
}

// GenerateCodebaseID 生成代码库唯一ID
func (s *codebaseService) GenerateCodebaseID(name, path string) string {
	// 使用MD5哈希生成唯一ID，结合名称和路径
	return fmt.Sprintf("%s_%x", name, md5.Sum([]byte(path)))
}

func (l *codebaseService) GetFileContent(ctx context.Context, req *dto.GetFileContentRequest) ([]byte, error) {
	// 读取文件
	filePath := req.FilePath
	clientPath := req.CodebasePath
	if err := l.checkPath(ctx, clientPath, []string{filePath}); err != nil {
		return nil, err
	}

	if clientPath == types.EmptyString {
		return nil, errors.New("codebase path is empty")
	}

	return l.workspaceReader.ReadFile(ctx, filePath, types.ReadOptions{StartLine: req.StartLine, EndLine: req.EndLine})
}

func (l *codebaseService) ReadCodeSnippets(ctx *gin.Context, req *dto.ReadCodeSnippetsRequest) (*dto.CodeSnippetsData, error) {
	workspacePath := req.WorkspacePath
	snippets := req.CodeSnippets

	// 添加空数组验证
	if len(snippets) == 0 {
		return nil, fmt.Errorf("codeSnippets array cannot be empty")
	}

	if len(snippets) > DefaultMaxCodeSnippets {
		snippets = snippets[:DefaultMaxCodeSnippets]
	}
	filePaths := make([]string, 0)
	for i, snippet := range snippets {
		// 如果开始行小于等于0，让它等于1；
		// 如果结束行小于等于开始行， 让它等于开始行；
		// 如果结束行 - 开始行 > 默认最大值，让它等于最大值；
		if snippet.StartLine <= 0 {
			snippets[i].StartLine = 1
		}
		if snippet.EndLine <= snippet.StartLine {
			snippets[i].EndLine = snippet.StartLine
		}
		if snippet.EndLine-snippet.StartLine > DefaultMaxCodeSnippetLines {
			snippets[i].EndLine = snippet.StartLine + DefaultMaxCodeSnippetLines
		}

		filePaths = append(filePaths, snippet.FilePath)
	}

	if err := l.checkPath(ctx, workspacePath, filePaths); err != nil {
		return nil, err
	}

	// 2. 从索引库查询代码片段
	codeSnippets := make([]*dto.CodeSnippet, 0)
	for _, snippet := range snippets {
		bytes, err := l.workspaceReader.ReadFile(ctx, snippet.FilePath, types.ReadOptions{StartLine: snippet.StartLine, EndLine: snippet.EndLine})
		if err != nil {
			l.logger.Error("failed to read code snippet %s %v: %v", snippet.FilePath, snippet.StartLine, snippet.EndLine, err)
			continue
		}
		codeSnippets = append(codeSnippets, &dto.CodeSnippet{
			FilePath:  snippet.FilePath,
			StartLine: snippet.StartLine,
			EndLine:   snippet.EndLine,
			Content:   string(bytes),
		})
	}

	return &dto.CodeSnippetsData{
		CodeSnippets: codeSnippets,
	}, nil

}

func (l *codebaseService) GetCodebaseDirectoryTree(ctx context.Context, req *dto.GetCodebaseDirectoryRequest) (
	resp *dto.DirectoryData, err error) {

	if err = l.checkPath(ctx, req.CodebasePath, []string{}); err != nil {
		return nil, err
	}

	// 1. 从数据库查询 codebase 信息
	treeOpts := types.TreeOptions{
		MaxDepth: req.Depth,
	}

	nodes, err := l.workspaceReader.Tree(ctx, req.CodebasePath, req.SubDir, treeOpts)
	if err != nil {
		l.logger.Error("failed to get directory tree: %v", err)
		return nil, err
	}

	// 3. 计算文件统计信息
	var totalFiles int
	var totalSize int64
	if len(nodes) > 0 {
		countFilesAndSize(nodes, &totalFiles, &totalSize, req.IncludeFiles)
	}

	resp = &dto.DirectoryData{
		RootPath:      req.CodebasePath,
		TotalFiles:    totalFiles,
		TotalSize:     totalSize,
		DirectoryTree: nodes,
	}

	return resp, nil
}

// countFilesAndSize 统计文件数量和总大小
func countFilesAndSize(nodes []*types.TreeNode, totalFiles *int, totalSize *int64, includeFiles bool) {
	if len(nodes) == 0 {
		return
	}

	for _, node := range nodes {
		if node == nil {
			continue
		}

		if !node.IsDir {
			if includeFiles {
				*totalFiles++
				*totalSize += node.Size
			}
			continue
		}

		// 递归处理子节点
		countFilesAndSize(node.Children, totalFiles, totalSize, includeFiles)
	}
}

func (l *codebaseService) ParseFileDefinitions(ctx context.Context, req *dto.GetFileStructureRequest) (resp *dto.FileStructureData, err error) {
	filePath := req.FilePath
	bytes, err := l.workspaceReader.ReadFile(ctx, req.FilePath, types.ReadOptions{EndLine: maxReadLine})

	if err != nil {
		return nil, err
	}

	parsed, err := l.fileDefinitionParser.Parse(ctx, &types.SourceFile{
		Path:    filePath,
		Content: bytes,
	}, definition.ParseOptions{IncludeContent: true})
	if err != nil {
		return nil, err
	}
	resp = new(dto.FileStructureData)
	for _, d := range parsed.Definitions {
		resp.List = append(resp.List, &dto.FileStructureInfo{
			Name:     d.Name,
			Type:     d.Type,
			Position: dto.ToPosition(d.Range),
			Content:  string(d.Content),
		})
	}
	return resp, nil
}

func (l *codebaseService) QueryDefinition(ctx context.Context, req *dto.SearchDefinitionRequest) (resp *dto.DefinitionData, err error) {
	// 参数验证
	// 支持三种检索方式：（FilePaths 必传）
	// 查询优先顺序：
	// 1. 根据符号名 (SymbolNames) 批量查询
	// 2. 根据代码片段 (CodeSnippet) 模糊检索（解析出其中的符号）
	// 3. 根据行号范围查询

	// 索引是否关闭
	if l.manager.GetCodebaseEnv().Switch == dto.SwitchOff {
		return nil, errs.ErrIndexDisabled
	}

	// codebasePath不能为空
	if req.CodebasePath == types.EmptyString {
		return nil, fmt.Errorf("missing param: codebasePath")
	}

	nodes, err := l.indexer.QueryDefinitions(ctx, &types.QueryDefinitionOptions{
		Workspace:   req.CodebasePath,
		StartLine:   req.StartLine,
		EndLine:     req.EndLine,
		FilePath:    req.FilePath,
		CodeSnippet: []byte(req.CodeSnippet),
		SymbolNames: req.SymbolNames,
	})
	if err != nil {
		return nil, err
	}

	// 填充content，控制层数和节点数
	definitions, err := l.convert2DefinitionInfo(ctx, nodes, definitionFillContentNodeLimit, definitionFillContentLineLimit)
	if err != nil {
		l.logger.Error("fill definition query contents err:%v", err)
	}

	return &dto.DefinitionData{List: definitions}, nil
}

func (l *codebaseService) convert2DefinitionInfo(ctx context.Context, nodes []*types.Definition, nodeLimit int, lineLimit int) ([]*dto.DefinitionInfo, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	definitions := make([]*dto.DefinitionInfo, 0, len(nodes))
	defNodeCountMap := make(map[string]int)
	defNodeCapMap := make(map[string]int)
	// 每个符号根据实际数量占比情况，(外部限制了一个batch查询的符号数量)
	for _, node := range nodes {
		defNodeCountMap[node.Name]++
	}
	for key, val := range defNodeCountMap {
		defNodeCapMap[key] = int(float64(val)/float64(len(nodes))*float64(nodeLimit) + 0.5) // 四舍五入
	}
	for _, node := range nodes {
		position := dto.ToPosition(node.Range)
		def := &dto.DefinitionInfo{
			FilePath: node.Path,
			Name:     node.Name,
			Type:     node.Type,
			Position: position,
		}
		definitions = append(definitions, def)
		startLine := position.StartLine
		endLine := position.EndLine
		// 通过定义个数和行数限制填充内容
		if defNodeCapMap[node.Name] <= 0 || endLine-startLine > lineLimit {
			continue
		}
		// 读取文件内容
		content, err := l.workspaceReader.ReadFile(ctx, node.Path, types.ReadOptions{
			StartLine: startLine,
			EndLine:   endLine,
		})
		if err != nil {
			l.logger.Error("read file content failed: %v", err)
			continue
		}
		defNodeCapMap[node.Name]--
		// 设置节点内容
		def.Content = string(content)
	}

	return definitions, nil
}

const relationFillContentLayerLimit = 2
const relationFillContentLayerNodeLimit = 10

func (l *codebaseService) QueryReference(ctx context.Context, req *dto.SearchReferenceRequest) (resp *dto.ReferenceData, err error) {

	if l.manager.GetCodebaseEnv().Switch == dto.SwitchOff {
		return nil, errs.ErrIndexDisabled
	}

	if req.CodebasePath == types.EmptyString {
		return nil, errs.NewMissingParamError("codebasePath")
	}

	nodes, err := l.indexer.QueryReferences(ctx, &types.QueryReferenceOptions{
		Workspace:  req.CodebasePath,
		FilePath:   req.FilePath,
		StartLine:  req.StartLine,
		EndLine:    req.EndLine,
		SymbolName: req.SymbolName,
	})
	if err != nil {
		return nil, err
	}
	// 如果filePath为空，且symbolName不为空，则根据symbolName查询引用
	if req.FilePath == types.EmptyString && req.SymbolName != types.EmptyString {
		if len(nodes) == 0 {
			return &dto.ReferenceData{List: nodes}, nil
		}
		// 填充content，控制层数和节点数，只填充子节点内容，不填充根节点内容
		if err = l.fillContent(ctx, nodes[0].Children, relationFillContentLayerLimit, relationFillContentLayerNodeLimit, defaultLineLimit); err != nil {
			l.logger.Error("fill graph query contents err:%v", err)
		}
		return &dto.ReferenceData{List: nodes}, nil
	}
	// 如果filePath不为空，则根据filePath查询引用
	// 填充content，控制层数和节点数
	if err = l.fillContent(ctx, nodes, relationFillContentLayerLimit, relationFillContentLayerNodeLimit, defaultLineLimit); err != nil {
		l.logger.Error("fill graph query contents err:%v", err)
	}
	return &dto.ReferenceData{List: nodes}, nil
}

const defaultMaxLayerLimit = 10
const defaultMaxLayer = 5
const maxLayerNodeLimit = 8
const defaultLineLimit = 200

func (l *codebaseService) QueryCallGraph(ctx context.Context, req *dto.SearchCallGraphRequest) (resp *dto.CallGraphData, err error) {
	// 二次校验
	if req.CodebasePath == types.EmptyString {
		return nil, errs.NewMissingParamError("codebasePath")
	}
	if req.FilePath == types.EmptyString {
		return nil, errs.NewMissingParamError("filePath")
	}
	if req.MaxLayer <= 0 {
		req.MaxLayer = defaultMaxLayer
	}
	if req.MaxLayer > defaultMaxLayerLimit {
		req.MaxLayer = defaultMaxLayerLimit
	}
	// 保证同一时间只有一个查询调用，避免内存过高
	l.mu.Lock()
	defer l.mu.Unlock()
	nodes, err := l.indexer.QueryCallGraph(ctx, &types.QueryCallGraphOptions{
		Workspace:  req.CodebasePath,
		FilePath:   req.FilePath,
		LineRange:  req.LineRange,
		SymbolName: req.SymbolName,
		MaxLayer:   req.MaxLayer,
	})
	if err != nil {
		return nil, err
	}
	// 填充content，控制层数和节点数（noContent=1时跳过填充）
	if req.NoContent != 1 {
		if err = l.fillContent(ctx, nodes, req.MaxLayer, maxLayerNodeLimit, defaultLineLimit); err != nil {
			l.logger.Error("fill graph query contents err:%v", err)
		}
	}
	return &dto.CallGraphData{
		List: nodes,
	}, nil
}

func (l *codebaseService) fillContent(ctx context.Context, nodes []*types.RelationNode, layerLimit, layerNodeLimit, lineLimit int) error {
	if len(nodes) == 0 {
		return nil
	}
	// 处理当前层的节点
	for i, node := range nodes {
		// 如果超过每层节点限制，跳过剩余节点
		if i >= layerNodeLimit {
			break
		}
		if node.Position.EndLine-node.Position.StartLine <= lineLimit {
			// 读取文件内容
			content, err := l.workspaceReader.ReadFile(ctx, node.FilePath, types.ReadOptions{
				StartLine: node.Position.StartLine,
				EndLine:   node.Position.EndLine,
			})

			if err != nil {
				l.logger.Error("read file content failed: %v", err)
				continue
			}

			// 设置节点内容
			node.Content = string(content)
		}

		// 如果还没有达到层级限制且有子节点，递归处理子节点
		if layerLimit > 1 && len(node.Children) > 0 {
			if err := l.fillContent(ctx, node.Children, layerLimit-1, layerNodeLimit, lineLimit); err != nil {
				l.logger.Error("fill children content failed: %v", err)
			}
		}
	}

	return nil
}

func (l *codebaseService) Summarize(ctx context.Context, req *dto.GetIndexSummaryRequest) (*dto.IndexSummary, error) {
	if l.manager.GetCodebaseEnv().Switch == dto.SwitchOff {
		return nil, errs.ErrIndexDisabled
	}
	// 从存储获取数量
	summary, err := l.indexer.GetSummary(ctx, req.CodebasePath)
	if err != nil {
		return nil, err
	}
	status := model.CodegraphStatusInit
	if summary.TotalFiles > 0 {
		status = model.CodegraphStatusSuccess
	}
	resp := &dto.IndexSummary{
		Codegraph: dto.CodegraphInfo{
			Status:     convertStatus(status),
			TotalFiles: summary.TotalFiles,
		},
	}

	return resp, nil
}

func (l *codebaseService) DeleteIndex(ctx context.Context, req *dto.DeleteIndexRequest) error {
	indexType := req.IndexType
	codebasePath := req.CodebasePath
	if indexType == types.EmptyString {
		return errs.NewMissingParamError("indexType")
	}
	if codebasePath == types.EmptyString {
		return errs.NewMissingParamError("codebasePath")
	}
	l.logger.Info("start to delete %s index for workspace %s", req.IndexType, codebasePath)
	// 根据索引类型删除对应的索引
	switch indexType {
	case dto.Embedding:
		return fmt.Errorf("deleting embedding index is not supported")
	case dto.Codegraph:
		if err := l.indexer.RemoveAllIndexes(ctx, codebasePath); err != nil {
			return fmt.Errorf("failed to delete graph index, err:%w", err)
		}
	case dto.All:
		return fmt.Errorf("deleting all index is not supported")
	default:
		return errs.NewInvalidParamErr("indexType", indexType)
	}
	l.logger.Info("delete all index successfully for workspace %s", codebasePath)
	return nil
}

func (l *codebaseService) GetFileSkeleton(ctx context.Context, req *dto.GetFileSkeletonRequest) (*dto.FileSkeletonData, error) {
	// 参数校验
	if req.WorkspacePath == types.EmptyString {
		return nil, errs.NewMissingParamError("workspacePath")
	}
	if req.FilePath == types.EmptyString {
		return nil, errs.NewMissingParamError("filePath")
	}

	// 处理相对/绝对路径
	filePath := req.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(req.WorkspacePath, filePath)
	}

	// 验证路径是否在 workspace 内
	if err := l.checkPath(ctx, req.WorkspacePath, []string{filePath}); err != nil {
		return nil, err
	}

	// 获取文件元素表
	fileElementTable, err := l.indexer.GetFileElementTable(ctx, req.WorkspacePath, filePath)
	if err != nil {
		return nil, err
	}

	// 根据 filteredBy 参数过滤 Elements
	if req.FilteredBy != types.EmptyString {
		filteredElements := make([]*codegraphpb.Element, 0)
		switch req.FilteredBy {
		case "definition":
			for _, elem := range fileElementTable.Elements {
				if elem.IsDefinition {
					filteredElements = append(filteredElements, elem)
				}
			}
		case "reference":
			for _, elem := range fileElementTable.Elements {
				if !elem.IsDefinition {
					filteredElements = append(filteredElements, elem)
				}
			}
		default:
			// 不识别的过滤类型，不进行过滤
			filteredElements = fileElementTable.Elements
		}
		fileElementTable.Elements = filteredElements
	}

	return fileElementTable, nil
}

func convertStatus(status int) string {
	var indexStatus string
	switch status {
	case model.CodegraphStatusBuilding:
		indexStatus = "running"
	case model.CodegraphStatusSuccess:
		indexStatus = "success"
	case model.CodegraphStatusInit:
		indexStatus = "pending"
	default:
		indexStatus = "failed"
	}
	return indexStatus
}
