package parser

import (
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/project"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func initLogger() logger.Logger {
	logger, err := logger.NewLogger(utils.LogsDir, "info")
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logging system: %v\n", err))
	}
	return logger
}

func TestGoBaseParse(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := project.NewProjectInfo(lang.Go, "github.com/hashicorp", []string{"pkg/go-uuid/uuid.go"})
	testCases := []struct {
		name       string
		sourceFile *types.SourceFile
		wantErr    error
	}{
		{
			name: "Go",
			sourceFile: &types.SourceFile{
				Path:    "test.go",
				Content: readFile("testdata/test.go"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
			assert.NotNil(t, res.Package)
			assert.NotEmpty(t, res.Imports)
			assert.NotEmpty(t, res.Elements)

		})
	}
}

func readFile(path string) []byte {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return bytes
}
