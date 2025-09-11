package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// g:\tmp\projects\c\linux-master
// g:\tmp\projects\c\netdata
// g:\tmp\projects\c\openssl
// g:\tmp\projects\c\redis
// g:\tmp\projects\c\sqlite
// g:\tmp\projects\cpp\grpc
// g:\tmp\projects\cpp\opencv
// g:\tmp\projects\cpp\protobuf
// g:\tmp\projects\go\docker-ce
// g:\tmp\projects\go\go
// g:\tmp\projects\go\kubernetes
// g:\tmp\projects\go\weaviate
// g:\tmp\projects\java\elasticsearch
// g:\tmp\projects\java\hadoop
// g:\tmp\projects\java\mall
// g:\tmp\projects\javascript\angular
// g:\tmp\projects\javascript\react
// g:\tmp\projects\javascript\vue
// g:\tmp\projects\python\django
// g:\tmp\projects\python\pandas
// g:\tmp\projects\python\scikit-learn
// g:\tmp\projects\typescript\svelte
// g:\tmp\projects\typescript\TypeScript
// g:\tmp\projects\typescript\vue-next

const testRootDir = "/home/kcx/codeWorkspace/testProjects"

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
			Exts:     getSupportedExtByLanguageTestHelper(lang.CPP),
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

	testCases := []struct {
		Name       string
		Language   string
		FilesLimit int
		Exts       []string
		Path       string
		wantErr    error
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
			Exts:     getSupportedExtByLanguageTestHelper(lang.CPP),
			wantErr:  nil,
		},
		{ // 220.97s 223.75s 228s 38s 65s
			Name:     "grpc",
			Language: "cpp",
			Path:     filepath.Join(testRootDir, "cpp", "grpc"),
			Exts:     getSupportedExtByLanguageTestHelper(lang.CPP),
			wantErr:  nil,
		},
		/////////////////////// 耗时测试
		{
			Name:       "100 文件项目",
			Language:   "java",
			FilesLimit: 100,
			Path:       filepath.Join(testRootDir, "java", "spring-boot"),
			Exts:       getSupportedExtByLanguageTestHelper(lang.Java),
			wantErr:    nil,
		},
		{
			Name:       "1000 文件项目",
			Language:   "java",
			FilesLimit: 1000,
			Path:       filepath.Join(testRootDir, "java", "hadoop"),
			Exts:       getSupportedExtByLanguageTestHelper(lang.TypeScript),
			wantErr:    nil,
		},
		{
			Name:       "5000 文件项目",
			Language:   "typescript",
			FilesLimit: 5000,
			Path:       filepath.Join(testRootDir, "typescript", "TypeScript"),
			Exts:       getSupportedExtByLanguageTestHelper(lang.TypeScript),
			wantErr:    nil,
		},
		{
			Name:       "10000 文件项目",
			Language:   "go",
			FilesLimit: 10000,
			Path:       filepath.Join(testRootDir, "go", "kubernetes"),
			Exts:       getSupportedExtByLanguageTestHelper(lang.Go),
			wantErr:    nil,
		},
		{
			Name:       "50000 文件项目",
			Language:   "typescript",
			FilesLimit: 50000,
			Path:       filepath.Join(testRootDir, "typescript", "TypeScript"),
			Exts:       getSupportedExtByLanguageTestHelper(lang.TypeScript),
			wantErr:    nil,
		},
	}
	cost := make([]string, 0)
	for i := 0; i < 1; i++ {

		for _, tc := range testCases {
			ctx := context.Background()
			t.Run(fmt.Sprintf("mixed-index-%s--%s", tc.Language, tc.Name), func(t *testing.T) {
				err = initWorkspaceModel(env, tc.Path)
				err = initWorkspaceModel(env, tc.Path) // do again, first may fail.
				assert.NoError(t, err)

				if tc.FilesLimit > 0 {
					err := os.Setenv("MAX_FILES", strconv.Itoa(tc.FilesLimit))
					if err != nil {
						panic(err)
					}
				}

				indexer := createTestIndexer(env, &types.VisitPattern{
					ExcludeDirs: defaultVisitPattern.ExcludeDirs,
					IncludeExts: getAllSupportedExtTestHelper(),
				})

				// clean
				err = indexer.RemoveAllIndexes(ctx, tc.Path)
				if err != nil {
					panic(err)
				}
				summary, err := indexer.GetSummary(ctx, tc.Path)
				if err != nil {
					panic(err)
				}
				assert.Equal(t, summary.TotalFiles, 0)

				start := time.Now()
				metrics, err := indexer.IndexWorkspace(ctx, tc.Path)
				assert.NoError(t, err)
				if tc.FilesLimit > 0 {
					summary, err := indexer.GetSummary(ctx, tc.Path)
					if err != nil {
						panic(err)
					}
					assert.Equal(t, summary.TotalFiles, tc.FilesLimit)
				}
				cost = append(cost, fmt.Sprintf("===>workspace %s, total files: %d, total failed: %d, cost: %d ms",
					tc.Name, metrics.TotalFiles, metrics.TotalFailedFiles, time.Since(start).Milliseconds()))
			})
		}
		t.Logf("###############################耗时统计#####################################")
		t.Log(strings.Join(cost, "\n"))
		fmt.Println("###############################耗时统计#####################################")
		fmt.Println(strings.Join(cost, "\n"))
		for {
			time.Sleep(1 * time.Second)
		}
	}

}
