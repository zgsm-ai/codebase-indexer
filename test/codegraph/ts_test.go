package codegraph

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"context"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

const TsProjectRootDir = "/tmp/projects/typescript"

func TestParseTsProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "typescript",
			Path:    filepath.Join(TsProjectRootDir, "typescript"),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
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
