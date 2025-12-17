package indexer

import (
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/workspace"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLineRange(t *testing.T) {
	tests := []struct {
		name      string
		start     int
		end       int
		maxLimit  int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "正常范围",
			start:     10,
			end:       20,
			maxLimit:  200,
			wantStart: 10,
			wantEnd:   20,
		},
		{
			name:      "起始值为0",
			start:     0,
			end:       10,
			maxLimit:  200,
			wantStart: 1,
			wantEnd:   10,
		},
		{
			name:      "结束值为0",
			start:     5,
			end:       0,
			maxLimit:  200,
			wantStart: 5,
			wantEnd:   5,
		},
		{
			name:      "结束值小于起始值",
			start:     20,
			end:       10,
			maxLimit:  200,
			wantStart: 20,
			wantEnd:   20,
		},
		{
			name:      "超过最大限制",
			start:     1,
			end:       300,
			maxLimit:  200,
			wantStart: 1,
			wantEnd:   200,
		},
		{
			name:      "负数起始值",
			start:     -5,
			end:       10,
			maxLimit:  200,
			wantStart: 1,
			wantEnd:   10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := NormalizeLineRange(tt.start, tt.end, tt.maxLimit)
			assert.Equal(t, tt.wantStart, gotStart, "start line mismatch")
			assert.Equal(t, tt.wantEnd, gotEnd, "end line mismatch")
		})
	}
}

func TestIsValidRange(t *testing.T) {
	tests := []struct {
		name   string
		range_ []int32
		want   bool
	}{
		{
			name:   "有效范围",
			range_: []int32{1, 2, 3, 4},
			want:   true,
		},
		{
			name:   "范围太短",
			range_: []int32{1, 2, 3},
			want:   false,
		},
		{
			name:   "空范围",
			range_: []int32{},
			want:   false,
		},
		{
			name:   "nil范围",
			range_: nil,
			want:   false,
		},
		{
			name:   "范围正好为4",
			range_: []int32{1, 2, 3, 4},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidRange(tt.range_)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGroupFilesByProject(t *testing.T) {
	// Mock indexer
	idx := &Indexer{}

	projects := []*workspace.Project{
		{
			Uuid: "project1",
			Path: "/path/to/project1",
		},
		{
			Uuid: "project2",
			Path: "/path/to/project2",
		},
	}

	filePaths := []string{
		"/path/to/project1/file1.go",
		"/path/to/project1/file2.go",
		"/path/to/project2/file3.go",
		"/path/to/project2/subdir/file4.go",
	}

	result, err := idx.groupFilesByProject(projects, filePaths)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Len(t, result["project1"], 2)
	assert.Len(t, result["project2"], 2)
	assert.Contains(t, result["project1"], "/path/to/project1/file1.go")
	assert.Contains(t, result["project2"], "/path/to/project2/file3.go")
}

func TestFindProjectForFile(t *testing.T) {
	idx := &Indexer{}

	projects := []*workspace.Project{
		{
			Uuid: "project1",
			Path: "/path/to/project1",
		},
		{
			Uuid: "project2",
			Path: "/path/to/project2",
		},
	}

	tests := []struct {
		name     string
		filePath string
		wantUuid string
		wantErr  bool
	}{
		{
			name:     "找到项目1",
			filePath: "/path/to/project1/file.go",
			wantUuid: "project1",
			wantErr:  false,
		},
		{
			name:     "找到项目2",
			filePath: "/path/to/project2/subdir/file.go",
			wantUuid: "project2",
			wantErr:  false,
		},
		{
			name:     "未找到项目",
			filePath: "/other/path/file.go",
			wantUuid: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, uuid, err := idx.findProjectForFile(projects, tt.filePath)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, project)
				assert.Empty(t, uuid)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				assert.Equal(t, tt.wantUuid, uuid)
			}
		})
	}
}

func TestIsInLinesRange(t *testing.T) {
	tests := []struct {
		name    string
		current int32
		start   int32
		end     int32
		want    bool
	}{
		{
			name:    "在范围内",
			current: 10,
			start:   10,
			end:     20,
			want:    true,
		},
		{
			name:    "等于起始值-1",
			current: 9,
			start:   10,
			end:     20,
			want:    true,
		},
		{
			name:    "等于结束值-1",
			current: 19,
			start:   10,
			end:     20,
			want:    true,
		},
		{
			name:    "小于范围",
			current: 8,
			start:   10,
			end:     20,
			want:    false,
		},
		{
			name:    "大于范围",
			current: 20,
			start:   10,
			end:     20,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInLinesRange(tt.current, tt.start, tt.end)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSymbolMapKey(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		ranges   []int32
		want     string
	}{
		{
			name:     "正常情况",
			filePath: "/path/to/file.go",
			ranges:   []int32{1, 2, 3, 4},
			want:     "/path/to/file.go-1,2,3,4",
		},
		{
			name:     "空范围",
			filePath: "/path/to/file.go",
			ranges:   []int32{},
			want:     "/path/to/file.go-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := symbolMapKey(tt.filePath, tt.ranges)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsSymbolExists(t *testing.T) {
	state := map[string]bool{
		"/path/file.go-1,2,3,4": true,
	}

	tests := []struct {
		name     string
		filePath string
		ranges   []int32
		want     bool
	}{
		{
			name:     "符号存在",
			filePath: "/path/file.go",
			ranges:   []int32{1, 2, 3, 4},
			want:     true,
		},
		{
			name:     "符号不存在",
			filePath: "/path/file.go",
			ranges:   []int32{5, 6, 7, 8},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSymbolExists(tt.filePath, tt.ranges, state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindSymbolInDocByLineRange(t *testing.T) {
	idx := &Indexer{}

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
		startLine int32
		endLine   int32
		wantCount int
		wantNames []string
	}{
		{
			name:      "查找第一个函数",
			startLine: 5,
			endLine:   10,
			wantCount: 1,
			wantNames: []string{"func1"},
		},
		{
			name:      "查找多个函数",
			startLine: 5,
			endLine:   20,
			wantCount: 2,
			wantNames: []string{"func1", "func2"},
		},
		{
			name:      "查找所有函数",
			startLine: 0,
			endLine:   30,
			wantCount: 3,
			wantNames: []string{"func1", "func2", "func3"},
		},
		{
			name:      "未找到",
			startLine: 35,
			endLine:   40,
			wantCount: 0,
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idx.findSymbolInDocByLineRange(nil, fileTable, tt.startLine, tt.endLine)
			assert.Len(t, result, tt.wantCount)
			for i, name := range tt.wantNames {
				assert.Equal(t, name, result[i].Name)
			}
		})
	}
}

