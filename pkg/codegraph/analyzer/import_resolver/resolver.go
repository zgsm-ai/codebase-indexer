package import_resolver

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"sync"
)

// ImportPathResolver 导入路径解析器接口
type ImportPathResolver interface {
	// ResolveImportPath 解析import的文件路径
	// 返回可能的文件路径列表（某些情况下可能匹配多个文件）
	ResolveImportPath(ctx context.Context, imp *resolver.Import, project *workspace.Project) ([]string, error)
}

// PathResolverManager 路径解析器管理器
type PathResolverManager struct {
	resolvers  map[lang.Language]ImportPathResolver
	searcher   *FileSearcher
	indexBuilt bool       // 索引是否已构建
	mu         sync.Mutex // 保护indexBuilt
}

// NewPathResolverManager 创建路径解析器管理器
func NewPathResolverManager(projectPath string) *PathResolverManager {
	searcher := NewFileSearcher(projectPath)

	mgr := &PathResolverManager{
		resolvers: make(map[lang.Language]ImportPathResolver),
		searcher:  searcher,
	}

	// 注册各语言解析器
	mgr.RegisterResolver(lang.Go, NewGoImportPathResolver(searcher))
	mgr.RegisterResolver(lang.Python, NewPythonImportPathResolver(searcher))

	jsResolver := NewJSImportPathResolver(searcher)
	mgr.RegisterResolver(lang.JavaScript, jsResolver)
	mgr.RegisterResolver(lang.TypeScript, jsResolver) // JS和TS使用相同解析器

	mgr.RegisterResolver(lang.Java, NewJavaImportPathResolver(searcher))

	cppResolver := NewCppImportPathResolver(searcher)
	mgr.RegisterResolver(lang.CPP, cppResolver)
	mgr.RegisterResolver(lang.C, cppResolver) // C和CPP使用相同解析器

	return mgr
}

// RegisterResolver 注册语言解析器
func (m *PathResolverManager) RegisterResolver(language lang.Language, resolver ImportPathResolver) {
	m.resolvers[language] = resolver
}

// GetResolver 获取指定语言的解析器
func (m *PathResolverManager) GetResolver(language lang.Language) ImportPathResolver {
	return m.resolvers[language]
}

// BuildIndex 构建文件索引（只构建一次）
func (m *PathResolverManager) BuildIndex() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 已构建过索引，跳过
	if m.indexBuilt {
		return nil
	}

	// 构建索引
	err := m.searcher.BuildIndex()
	if err == nil {
		m.indexBuilt = true
	}

	return err
}

// ResolveImportPath 解析import路径
func (m *PathResolverManager) ResolveImportPath(
	ctx context.Context,
	language lang.Language,
	imp *resolver.Import,
	project *workspace.Project,
) ([]string, error) {
	// 获取解析器
	resolver := m.GetResolver(language)
	if resolver == nil {
		return nil, nil // 不支持的语言，返回空列表
	}

	// 解析路径
	return resolver.ResolveImportPath(ctx, imp, project)
}
