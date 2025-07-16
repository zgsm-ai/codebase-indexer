package analyzer

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/project"
	"context"
)

type ProjectElementTable struct {
	ProjectInfo       *project.ProjectInfo
	FileElementTables []*parser.FileElementTable
}

type DependencyAnalyzer struct {
}

func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{}
}

func (da *DependencyAnalyzer) Analyze(ctx context.Context, projectSymbolTable *ProjectElementTable) error {
	// 1、遍历所有文件，构建符号表
	//var projectSymbols
	return nil
}
