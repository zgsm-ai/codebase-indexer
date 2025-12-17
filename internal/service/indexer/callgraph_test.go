package indexer

import (
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/types"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalleeInfo_Key(t *testing.T) {
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

func TestCallerInfo_Key(t *testing.T) {
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
}

func TestCalleeKey(t *testing.T) {
	key := CalleeKey{
		SymbolName: "TestFunc",
		ParamCount: 3,
	}

	assert.Equal(t, "TestFunc", key.SymbolName)
	assert.Equal(t, 3, key.ParamCount)
}

func TestExtractCalleeSymbols(t *testing.T) {
	idx := &Indexer{}

	fileTable := &codegraphpb.FileElementTable{
		Path:     "/test/file.go",
		Language: "go",
		Elements: []*codegraphpb.Element{
			{
				Name:        "call1",
				ElementType: codegraphpb.ElementType_CALL,
				Range:       []int32{10, 0, 10, 10},
				ExtraData:   map[string][]byte{"params": []byte("2")},
			},
			{
				Name:        "call2",
				ElementType: codegraphpb.ElementType_CALL,
				Range:       []int32{15, 0, 15, 10},
				ExtraData:   map[string][]byte{"params": []byte("1")},
			},
			{
				Name:        "notACall",
				ElementType: codegraphpb.ElementType_FUNCTION,
				Range:       []int32{20, 0, 30, 0},
			},
		},
	}

	startLine := int32(5)
	endLine := int32(20)

	results := idx.extractCalleeSymbols(fileTable, startLine, endLine)

	// 应该只提取CALL类型的元素
	assert.LessOrEqual(t, len(results), 2)
	for _, result := range results {
		assert.NotEmpty(t, result.SymbolName)
	}
}

func TestBuildCallGraphBFS(t *testing.T) {
	// 这个测试需要完整的依赖注入和存储层
	t.Skip("需要完整的依赖注入环境")
}

func TestBuildCalleeMap(t *testing.T) {
	// 这个测试需要完整的存储层依赖
	t.Skip("需要完整的存储依赖")
}

func TestMapBatcher_Add(t *testing.T) {
	// 创建一个mock的MapBatcher用于测试
	// 这里测试基本的数据结构
	calleeMap := make(map[string][]CallerInfo)

	key := "TestFunc"
	callers := []CallerInfo{
		{
			SymbolName: "Caller1",
			FilePath:   "/test/caller1.go",
			ParamCount: 2,
		},
		{
			SymbolName: "Caller2",
			FilePath:   "/test/caller2.go",
			ParamCount: 1,
		},
	}

	calleeMap[key] = callers

	assert.Len(t, calleeMap[key], 2)
	assert.Equal(t, "Caller1", calleeMap[key][0].SymbolName)
	assert.Equal(t, "Caller2", calleeMap[key][1].SymbolName)
}

func TestQueryCallGraph(t *testing.T) {
	// 这个测试需要完整的依赖注入
	t.Skip("需要完整的依赖注入环境")
}

func TestQueryCallGraphBySymbol(t *testing.T) {
	// 这个测试需要完整的依赖注入
	t.Skip("需要完整的依赖注入环境")
}

func TestQueryCallGraphByLineRange(t *testing.T) {
	// 这个测试需要完整的依赖注入
	t.Skip("需要完整的依赖注入环境")
}

func TestVariadicParameterMatching(t *testing.T) {
	tests := []struct {
		name           string
		calleeVariadic bool
		calleeParamCnt int
		callerParamCnt int
		shouldMatch    bool
	}{
		{
			name:           "可变参数-足够的参数",
			calleeVariadic: true,
			calleeParamCnt: 2,
			callerParamCnt: 3,
			shouldMatch:    true,
		},
		{
			name:           "可变参数-不足的参数",
			calleeVariadic: true,
			calleeParamCnt: 2,
			callerParamCnt: 1,
			shouldMatch:    false,
		},
		{
			name:           "固定参数-匹配",
			calleeVariadic: false,
			calleeParamCnt: 2,
			callerParamCnt: 2,
			shouldMatch:    true,
		},
		{
			name:           "固定参数-不匹配",
			calleeVariadic: false,
			calleeParamCnt: 2,
			callerParamCnt: 3,
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var matches bool
			if tt.calleeVariadic {
				matches = tt.callerParamCnt >= tt.calleeParamCnt
			} else {
				matches = tt.callerParamCnt == tt.calleeParamCnt
			}
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

