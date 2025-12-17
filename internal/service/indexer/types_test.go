package indexer

import (
	"codebase-indexer/pkg/codegraph/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Defaults(t *testing.T) {
	config := &Config{}

	// 测试默认值初始化
	initConfig(config)

	assert.Greater(t, config.MaxConcurrency, 0, "MaxConcurrency should be set to default")
	assert.Greater(t, config.MaxBatchSize, 0, "MaxBatchSize should be set to default")
	assert.Greater(t, config.MaxProjects, 0, "MaxProjects should be set to default")
	assert.Greater(t, config.CacheCapacity, 0, "CacheCapacity should be set to default")
}

func TestBatchProcessParams(t *testing.T) {
	params := &BatchProcessParams{
		ProjectUuid: "test-project",
		SourceFiles: []*types.FileWithModTimestamp{
			{Path: "/test/file1.go", ModTime: 123456},
			{Path: "/test/file2.go", ModTime: 123457},
		},
		BatchStart: 0,
		BatchEnd:   2,
		BatchSize:  2,
		TotalFiles: 10,
	}

	assert.Equal(t, "test-project", params.ProjectUuid)
	assert.Len(t, params.SourceFiles, 2)
	assert.Equal(t, 2, params.BatchSize)
	assert.Equal(t, 10, params.TotalFiles)
}

func TestProgressInfo(t *testing.T) {
	progress := &ProgressInfo{
		Total:         100,
		Processed:     50,
		PreviousNum:   10,
		WorkspacePath: "/test/workspace",
	}

	assert.Equal(t, 100, progress.Total)
	assert.Equal(t, 50, progress.Processed)
	assert.Equal(t, 10, progress.PreviousNum)
	assert.Equal(t, "/test/workspace", progress.WorkspacePath)
}

func TestCalleeKeyStruct(t *testing.T) {
	key := CalleeKey{
		SymbolName: "TestFunction",
		ParamCount: 3,
	}

	assert.Equal(t, "TestFunction", key.SymbolName)
	assert.Equal(t, 3, key.ParamCount)
}

func TestCalleeInfoKey(t *testing.T) {
	info := &CalleeInfo{
		FilePath:   "/test/file.go",
		SymbolName: "TestFunc",
		ParamCount: 2,
		Position: types.Position{
			StartLine:   10,
			StartColumn: 5,
			EndLine:     15,
			EndColumn:   10,
		},
		IsVariadic: false,
	}

	key := info.Key()
	expectedKey := "TestFunc::/test/file.go::10:5:15:10"
	assert.Equal(t, expectedKey, key)
}

func TestCallerInfoKey(t *testing.T) {
	info := &CallerInfo{
		SymbolName: "CallerFunc",
		FilePath:   "/test/caller.go",
		Position: types.Position{
			StartLine:   20,
			StartColumn: 1,
			EndLine:     25,
			EndColumn:   5,
		},
		ParamCount: 1,
		IsVariadic: false,
		CalleeKey: CalleeKey{
			SymbolName: "CalleeFunc",
			ParamCount: 1,
		},
		Score: 0.95,
	}

	key := info.Key()
	expectedKey := "CallerFunc::/test/caller.go::20:1:25:5"
	assert.Equal(t, expectedKey, key)
	assert.Equal(t, 0.95, info.Score)
}

func TestBatchProcessingParams(t *testing.T) {
	params := &BatchProcessingParams{
		ProjectUuid:          "test-uuid",
		NeedIndexSourceFiles: []*types.FileWithModTimestamp{},
		TotalFilesCnt:        100,
		PreviousFileNum:      50,
		WorkspacePath:        "/workspace",
		Concurrency:          4,
		BatchSize:            20,
	}

	assert.Equal(t, "test-uuid", params.ProjectUuid)
	assert.Equal(t, 100, params.TotalFilesCnt)
	assert.Equal(t, 50, params.PreviousFileNum)
	assert.Equal(t, 4, params.Concurrency)
	assert.Equal(t, 20, params.BatchSize)
}

func TestConstants(t *testing.T) {
	// 测试导出的常量值的合理性
	assert.Equal(t, 1600, MaxCalleeMapCacheCapacity)

	// 测试私有常量通过配置来验证
	config := &Config{}
	initConfig(config)
	assert.Greater(t, config.MaxConcurrency, 0)
	assert.Greater(t, config.MaxBatchSize, 0)
}

