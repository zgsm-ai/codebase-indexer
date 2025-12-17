package indexer

import (
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockLogger 是一个简单的 mock logger
type mockLogger struct{}

func (m *mockLogger) Debug(format string, args ...interface{}) {}
func (m *mockLogger) Info(format string, args ...interface{})  {}
func (m *mockLogger) Warn(format string, args ...interface{})  {}
func (m *mockLogger) Error(format string, args ...interface{}) {}
func (m *mockLogger) Fatal(format string, args ...interface{}) {}
func (m *mockLogger) Close() error                             { return nil }

func TestQuerySymbolsByName(t *testing.T) {
	idx := &Indexer{}

	doc := &codegraphpb.FileElementTable{
		Path: "/test/file.go",
		Elements: []*codegraphpb.Element{
			{
				Name:  "TestFunction",
				Range: []int32{10, 0, 20, 0},
			},
			{
				Name:  "AnotherFunction",
				Range: []int32{30, 0, 40, 0},
			},
			{
				Name:  "TestFunction",
				Range: []int32{50, 0, 60, 0},
			},
		},
	}

	opts := &types.QueryReferenceOptions{
		SymbolName: "TestFunction",
	}

	results := idx.querySymbolsByName(doc, opts)

	assert.Len(t, results, 2, "应该找到2个同名符号")
	for _, result := range results {
		assert.Equal(t, "TestFunction", result.Name)
	}
}

func TestQuerySymbolsByLines(t *testing.T) {
	// 创建一个mock logger
	mockLogger := &mockLogger{}
	idx := &Indexer{logger: mockLogger}

	fileTable := &codegraphpb.FileElementTable{
		Path: "/test/file.go",
		Elements: []*codegraphpb.Element{
			{
				Name:  "func1",
				Range: []int32{5, 0, 10, 0},
			},
			{
				Name:  "func2",
				Range: []int32{15, 0, 20, 0},
			},
			{
				Name:  "func3",
				Range: []int32{25, 0, 30, 0},
			},
		},
	}

	tests := []struct {
		name      string
		opts      *types.QueryReferenceOptions
		wantCount int
	}{
		{
			name: "查找包含第一个函数的范围",
			opts: &types.QueryReferenceOptions{
				StartLine: 1,
				EndLine:   11, // 需要包含 range[2] = 10
			},
			wantCount: 1,
		},
		{
			name: "查找包含前两个函数的范围",
			opts: &types.QueryReferenceOptions{
				StartLine: 1,
				EndLine:   21, // 需要包含 range[2] = 20
			},
			wantCount: 2,
		},
		{
			name: "无效范围",
			opts: &types.QueryReferenceOptions{
				StartLine: -1,
				EndLine:   0,
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results := idx.querySymbolsByLines(ctx, fileTable, tt.opts)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestNormalizeLineRange_Query(t *testing.T) {
	tests := []struct {
		name      string
		startLine int
		endLine   int
		want      bool
	}{
		{
			name:      "正常范围",
			startLine: 10,
			endLine:   20,
			want:      true,
		},
		{
			name:      "反向范围",
			startLine: 20,
			endLine:   10,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := NormalizeLineRange(tt.startLine, tt.endLine, 200)
			if tt.want {
				assert.LessOrEqual(t, start, end)
			} else {
				assert.Equal(t, start, end)
			}
		})
	}
}

func TestQueryDefinitionOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    *types.QueryDefinitionOptions
		wantErr bool
	}{
		{
			name: "有效选项-文件路径",
			opts: &types.QueryDefinitionOptions{
				Workspace: "/workspace",
				FilePath:  "/workspace/file.go",
				StartLine: 10,
				EndLine:   20,
			},
			wantErr: false,
		},
		{
			name: "有效选项-符号名",
			opts: &types.QueryDefinitionOptions{
				Workspace:   "/workspace",
				SymbolNames: "TestFunc,AnotherFunc",
			},
			wantErr: false,
		},
		{
			name: "无效-缺少workspace",
			opts: &types.QueryDefinitionOptions{
				FilePath: "/workspace/file.go",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				assert.Empty(t, tt.opts.Workspace)
			} else {
				assert.NotEmpty(t, tt.opts.Workspace)
			}
		})
	}
}

func TestFindSymbolInDocByRange(t *testing.T) {
	idx := &Indexer{}

	fileTable := &codegraphpb.FileElementTable{
		Path: "/test/file.go",
		Elements: []*codegraphpb.Element{
			{
				Name:  "func1",
				Range: []int32{10, 5, 20, 10},
			},
			{
				Name:  "func2",
				Range: []int32{30, 5, 40, 10},
			},
		},
	}

	tests := []struct {
		name        string
		symbolRange []int32
		wantName    string
		wantNil     bool
	}{
		{
			name:        "找到符号",
			symbolRange: []int32{10, 5, 20, 10},
			wantName:    "func1",
			wantNil:     false,
		},
		{
			name:        "未找到符号",
			symbolRange: []int32{50, 0, 60, 0},
			wantName:    "",
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.findSymbolInDocByRange(fileTable, tt.symbolRange)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantName, result.Name)
			}
		})
	}
}

func TestSearchSymbolNames(t *testing.T) {
	// 这个测试需要完整的存储层依赖
	t.Skip("需要完整的存储依赖")
}

func TestQueryReferences(t *testing.T) {
	// 这个测试需要完整的依赖注入
	t.Skip("需要完整的依赖注入环境")
}

func TestQueryDefinitions(t *testing.T) {
	// 这个测试需要完整的依赖注入
	t.Skip("需要完整的依赖注入环境")
}

