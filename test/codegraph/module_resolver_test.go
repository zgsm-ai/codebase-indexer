package codegraph

import (
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
	"time"
)

const testProjectRoot = "G:\\tmp\\projects"

func TestModuleResolver(t *testing.T) {
	env, err := setupTestEnvironment()
	if err != nil {
		panic(err)
	}
	testCases := []struct {
		Name            string
		Workspace       string
		ExpectedModules []string
		WantError       bool
	}{
		{
			Name:            "go kubernetes",
			Workspace:       filepath.Join(testProjectRoot, "go", "kubernetes"),
			ExpectedModules: []string{"k8s.io"},
			WantError:       false,
		},
	}

	for _, testCase := range testCases {
		start := time.Now()
		resolver := workspace.NewModuleResolver(env.logger)
		project := &workspace.Project{Name: testCase.Name,
			Path: testCase.Workspace,
		}
		err := resolver.ResolveProjectModules(context.Background(), project, testCase.Workspace, 3)
		assert.NoError(t, err)
		assert.True(t, sliceEqual(testCase.ExpectedModules, project.GoModules))
		cost := time.Since(start)
		assert.True(t, cost.Milliseconds() < 50, "module resolve cost should less than %d ms", 50)
	}

}

func sliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
