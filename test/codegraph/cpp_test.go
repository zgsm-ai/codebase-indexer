package codegraph

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const CPPProjectRootDir = "../tmp/projects/cpp"

func TestParseCPPProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	indexer := createTestIndexer(env, types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".cpp", ".h", ".cc"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "grpc",
			Path:    filepath.Join(CPPProjectRootDir, "grpc"),
			wantErr: nil,
		},
		//{
		//	Name:    "protobuf",
		//	Path:    filepath.Join(CPPProjectRootDir, "protobuf"),
		//	wantErr: nil,
		//},
		//{
		//	Name:    "whisper.cpp",
		//	Path:    filepath.Join(CPPProjectRootDir, "whisper.cpp"),
		//	wantErr: nil,
		//},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			start := time.Now()
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := indexer.ParseProjectFiles(context.Background(), project)
			fmt.Println("err:", err)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
			duration := time.Since(start)
			fmt.Printf("测试用例 '%s' 执行时间: %v, 文件个数: %d\n", tc.Name, duration, len(fileElements))
			assert.NoError(t, err)
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, len(fileElements) > 0)
			for _, f := range fileElements {
				for _, e := range f.Elements {
					if !resolver.IsValidElement(e) {
						fmt.Printf("Type: %s Name: %s Path: %s\n",
							e.GetType(), e.GetName(), e.GetPath())
						fmt.Printf("  Range: %v Scope: %s\n",
							e.GetRange(), e.GetScope())

					}
					//assert.True(t, resolver.IsValidElement(e))
				}
				for _, e := range f.Imports {
					if !resolver.IsValidElement(e) {
						fmt.Printf("Type: %s Name: %s Path: %s\n",
							e.GetType(), e.GetName(), e.GetPath())
						fmt.Printf("  Range: %v Scope: %s\n",
							e.GetRange(), e.GetScope())

					}
				}
			}
		})
	}
}
