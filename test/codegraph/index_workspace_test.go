package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

const testRootDir = "G:\\tmp\\projects"

func getSupportedExtTestHelper(language lang.Language) []string {
	parser, err := lang.GetSitterParserByLanguage(language)
	if err != nil {
		panic(err)
	}
	return parser.SupportedExts
}

func TestIndexLanguages(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	setupPprof()
	defer teardownTestEnvironment(t, env)

	assert.NoError(t, err)

	testCases := []struct {
		Name     string
		Language string
		Exts     []string
		Path     string
		wantErr  error
	}{
		{
			Name:     "kubernetes",
			Language: "go",
			Path:     filepath.Join(testRootDir, "go", "kubernetes"),
			Exts:     getSupportedExtTestHelper(lang.Go),
			wantErr:  nil,
		},
		{
			Name:     "spring-boot",
			Language: "java",
			Path:     filepath.Join(testRootDir, "java", "spring-boot"),
			Exts:     getSupportedExtTestHelper(lang.Java),
			wantErr:  nil,
		},
		{
			Name:     "django",
			Language: "python",
			Path:     filepath.Join(testRootDir, "python", "django"),
			Exts:     getSupportedExtTestHelper(lang.Python),
			wantErr:  nil,
		},
		{
			Name:     "vue-next",
			Language: "typescript",
			Path:     filepath.Join(testRootDir, "typescript", "vue-next"),
			Exts:     getSupportedExtTestHelper(lang.TypeScript),
			wantErr:  nil,
		},
		{
			Name:     "vue",
			Language: "javascript",
			Path:     filepath.Join(testRootDir, "javascript", "vue"),
			Exts:     getSupportedExtTestHelper(lang.JavaScript),
			wantErr:  nil,
		},
		{
			Name:     "redis",
			Language: "c",
			Path:     filepath.Join(testRootDir, "c", "redis"),
			Exts:     getSupportedExtTestHelper(lang.C),
			wantErr:  nil,
		},
		{
			Name:     "grpc",
			Language: "cpp",
			Path:     filepath.Join(testRootDir, "cpp", "grpc"),
			Exts:     getSupportedExtTestHelper(lang.CPP),
			wantErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run("init-database-"+tc.Name, func(t *testing.T) {
			err = initWorkspaceModel(env, tc.Path)
			assert.NoError(t, err)
		})
	}

	for _, tc := range testCases {
		ctx := context.Background()
		t.Run(fmt.Sprintf("index-%s--%s", tc.Language, tc.Name), func(t *testing.T) {
			indexer := createTestIndexer(env, &types.VisitPattern{
				ExcludeDirs: defaultVisitPattern.ExcludeDirs,
				IncludeExts: tc.Exts,
			})
			_, err = indexer.IndexWorkspace(ctx, tc.Path)
			assert.NoError(t, err)
		})
	}
}
