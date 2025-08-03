package codegraph

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

const JsProjectRootDir = "/tmp/projects/javascript"

func TestParseJsProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	indexer := createTestIndexer(env, types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".js"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "vue",
			Path:    filepath.Join(JsProjectRootDir, "vue"),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := indexer.ParseProjectFiles(context.Background(), project)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, len(fileElements) > 0)
			for _, f := range fileElements {
				for _, e := range f.Elements {
					assert.True(t, resolver.IsValidElement(e))
				}
			}
		})
	}
}
