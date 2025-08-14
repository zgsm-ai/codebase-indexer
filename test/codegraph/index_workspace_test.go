package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
	"time"
)

const testRootDir = "G:\\tmp\\projects"

func getSupportedExtByLanguageTestHelper(language lang.Language) []string {
	parser, err := lang.GetSitterParserByLanguage(language)
	if err != nil {
		panic(err)
	}
	return parser.SupportedExts
}
func getAllSupportedExtTestHelper() []string {
	parsers := lang.GetTreeSitterParsers()
	ext := make([]string, 0, len(parsers))
	for _, parser := range parsers {
		ext = append(ext, parser.SupportedExts...)
	}
	return ext
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
			Exts:     getSupportedExtByLanguageTestHelper(lang.Go),
			wantErr:  nil,
		},
		{
			Name:     "spring-boot",
			Language: "java",
			Path:     filepath.Join(testRootDir, "java", "spring-boot"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.Java),
			wantErr:  nil,
		},
		{
			Name:     "django",
			Language: "python",
			Path:     filepath.Join(testRootDir, "python", "django"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.Python),
			wantErr:  nil,
		},
		{
			Name:     "vue-next",
			Language: "typescript",
			Path:     filepath.Join(testRootDir, "typescript", "vue-next"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.TypeScript),
			wantErr:  nil,
		},
		{
			Name:     "vue",
			Language: "javascript",
			Path:     filepath.Join(testRootDir, "javascript", "vue"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.JavaScript),
			wantErr:  nil,
		},
		{
			Name:     "redis",
			Language: "c",
			Path:     filepath.Join(testRootDir, "c", "redis"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.C),
			wantErr:  nil,
		},
		{
			Name:     "grpc",
			Language: "cpp",
			Path:     filepath.Join(testRootDir, "cpp", "grpc"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.CPP),
			wantErr:  nil,
		},
	}

	for _, tc := range testCases {
		ctx := context.Background()
		t.Run(fmt.Sprintf("single-index-%s--%s", tc.Language, tc.Name), func(t *testing.T) {
			err = initWorkspaceModel(env, tc.Path)
			err = initWorkspaceModel(env, tc.Path) // do again, first may fail.
			assert.NoError(t, err)
			start := time.Now()
			indexer := createTestIndexer(env, &types.VisitPattern{
				ExcludeDirs: defaultVisitPattern.ExcludeDirs,
				IncludeExts: tc.Exts,
			})
			metrics, err := indexer.IndexWorkspace(ctx, tc.Path)
			assert.NoError(t, err)
			t.Logf("===>single-index workspace %s, total files: %d, total failed: %d, cost: %d ms",
				tc.Name, metrics.TotalFiles, metrics.TotalFailedFiles, time.Since(start).Milliseconds())
		})
	}
}

func TestIndexMixedLanguages(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	setupPprof()
	defer teardownTestEnvironment(t, env)

	assert.NoError(t, err)
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: getAllSupportedExtTestHelper(),
	})

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
			Exts:     getSupportedExtByLanguageTestHelper(lang.Go),
			wantErr:  nil,
		},
		{
			Name:     "hadoop",
			Language: "java",
			Path:     filepath.Join(testRootDir, "java", "hadoop"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.Java),
			wantErr:  nil,
		},
		{
			Name:     "spring-boot",
			Language: "java",
			Path:     filepath.Join(testRootDir, "java", "spring-boot"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.Java),
			wantErr:  nil,
		},
		{
			Name:     "django",
			Language: "python",
			Path:     filepath.Join(testRootDir, "python", "django"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.Python),
			wantErr:  nil,
		},
		{
			Name:     "vue-next",
			Language: "typescript",
			Path:     filepath.Join(testRootDir, "typescript", "vue-next"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.TypeScript),
			wantErr:  nil,
		},
		{
			Name:     "vue",
			Language: "javascript",
			Path:     filepath.Join(testRootDir, "javascript", "vue"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.JavaScript),
			wantErr:  nil,
		},
		{
			Name:     "redis",
			Language: "c",
			Path:     filepath.Join(testRootDir, "c", "redis"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.C),
			wantErr:  nil,
		},
		{
			Name:     "grpc",
			Language: "cpp",
			Path:     filepath.Join(testRootDir, "cpp", "grpc"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.CPP),
			wantErr:  nil,
		},
	}

	for i := 0; i < 1000; i++ {

		for _, tc := range testCases {
			ctx := context.Background()
			t.Run(fmt.Sprintf("mixed-index-%s--%s", tc.Language, tc.Name), func(t *testing.T) {
				err = initWorkspaceModel(env, tc.Path)
				err = initWorkspaceModel(env, tc.Path) // do again, first may fail.
				assert.NoError(t, err)
				start := time.Now()
				metrics, err := indexer.IndexWorkspace(ctx, tc.Path)
				assert.NoError(t, err)
				t.Logf("===>mixed-index workspace %s, total files: %d, total failed: %d, cost: %d ms",
					tc.Name, metrics.TotalFiles, metrics.TotalFailedFiles, time.Since(start).Milliseconds())
			})
		}

	}

}
