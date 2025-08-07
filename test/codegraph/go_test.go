package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const GoProjectRootDir = "/tmp/projects/go"

func TestParseGoProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	indexer := createTestIndexer(env, types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".go"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "kubernetes",
			Path:    filepath.Join(GoProjectRootDir, "kubernetes"),
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

func TestIndexGoProjects(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	indexer := createTestIndexer(env, types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".go"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "在内存中全量索引kubernetes",
			Path:    filepath.Join(GoProjectRootDir, "kubernetes"),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err = indexer.IndexWorkspace(context.Background(), tc.Path)
			assert.NoError(t, err)
			summary, err := indexer.GetSummary(context.Background(), tc.Path)
			assert.NoError(t, err)
			t.Logf("index count: %d", summary.TotalFiles)
		})
	}
}

func TestWalkProjectCostTime(t *testing.T) {
	ctx := context.Background()
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	testCases := []struct {
		name  string
		path  string
		logic func(*testing.T, *testEnvironment, *types.WalkContext)
	}{
		{
			name: "do nothing",
			path: filepath.Join(GoProjectRootDir, "kubernetes"),
		},
		{
			name: "do index",
			path: filepath.Join(GoProjectRootDir, "kubernetes"),
			logic: func(t *testing.T, environment *testEnvironment, walkContext *types.WalkContext) {
				bytes, err := os.ReadFile(walkContext.Path)
				assert.NoError(t, err)
				_, err = environment.sourceFileParser.Parse(ctx, &types.SourceFile{
					Path:    walkContext.Path,
					Content: bytes,
				})
				if !lang.IsUnSupportedFileError(err) {
					assert.NoError(t, err)
				}
			},
		},
	}
	excludeDir := append([]string{}, defaultVisitPattern.ExcludeDirs...)
	excludeDir = append(excludeDir, "vendor")
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var fileCnt int
			start := time.Now()
			err = env.workspaceReader.WalkFile(ctx, tt.path, func(walkCtx *types.WalkContext) error {
				fileCnt++
				if tt.logic != nil {
					tt.logic(t, env, walkCtx)
				}
				return nil
			}, types.WalkOptions{IgnoreError: true, VisitPattern: types.VisitPattern{ExcludeDirs: excludeDir, IncludeExts: []string{".go"}}})
			assert.NoError(t, err)
			t.Logf("%s cost %d ms, %d files, avg %.2f ms/file", tt.name, time.Since(start).Milliseconds(), fileCnt,
				float32(time.Since(start).Milliseconds())/float32(fileCnt))
		})
	}

}
