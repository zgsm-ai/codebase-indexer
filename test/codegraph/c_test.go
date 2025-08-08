package codegraph

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const CProjectRootDir = "../tmp/projects/c"

func TestParseCProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".c", ".h"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "redis",
			Path:    filepath.Join(CProjectRootDir, "redis"),
			wantErr: nil,
		},
		{
			Name:    "sqlite",
			Path:    filepath.Join(CProjectRootDir, "sqlite"),
			wantErr: nil,
		},
		{
			Name:    "openssl",
			Path:    filepath.Join(CProjectRootDir, "openssl"),
			wantErr: nil,
		},
		{
			Name:    "netdata",
			Path:    filepath.Join(CProjectRootDir, "netdata"),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fmt.Println("tc.Path", tc.Path)
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := indexer.ParseProjectFiles(context.Background(), project)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
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
