package analyzer

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
)

type ProjectElementTable struct {
	ProjectInfo       *workspace.ProjectInfo
	FileElementTables []*parser.FileElementTable
}

type DependencyAnalyzer struct {
}

func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{}
}

func (da *DependencyAnalyzer) Analyze(ctx context.Context, projectSymbolTable *ProjectElementTable) error {
	// 1、第一次遍历所有文件，构建定义符号表
	//var projectDefinitionSymbols map[string][]Element

	// 2、 第二次遍历所有文件，针对引用类型， 根据 符号名 + 符号所属文件位置 + 符号所属文件的import， 找到定义，构建关系。

	return nil
}
