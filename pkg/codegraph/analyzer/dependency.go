package analyzer

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/project"
	"context"
)

type ProjectSymbolTable struct {
	ProjectInfo      *project.ProjectInfo
	FileSymbolTables []*parser.FileSymbolTable
}

type DependencyAnalyzer struct {
}

func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{}
}

func (da *DependencyAnalyzer) Analyze(ctx context.Context, projectSymbolTable *ProjectSymbolTable) error {

	return nil
}
