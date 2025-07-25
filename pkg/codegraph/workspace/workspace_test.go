package workspace

import (
	"codebase-indexer/test/mocks"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createTestDir(t *testing.T, structure map[string]bool) string {
	dir := t.TempDir()
	for relPath, isDir := range structure {
		absPath := filepath.Join(dir, relPath)
		if isDir {
			if err := os.MkdirAll(absPath, 0755); err != nil {
				t.Fatalf("failed to create dir: %v", err)
			}
		} else {
			parent := filepath.Dir(absPath)
			if err := os.MkdirAll(parent, 0755); err != nil {
				t.Fatalf("failed to create parent dir: %v", err)
			}
			f, err := os.Create(absPath)
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}
			f.Close()
		}
	}
	return dir
}

func TestFindProjects(t *testing.T) {
	tests := []struct {
		name      string
		structure map[string]bool // 路径->是否为目录
		expectNum int
		expectHas []string // 期望包含的项目路径（相对路径）
	}{
		{
			name: "当前目录为git仓库",
			structure: map[string]bool{
				".git": true,
			},
			expectNum: 1,
			expectHas: []string{"."},
		},
		{
			name: "子目录为git仓库",
			structure: map[string]bool{
				"sub/.git": true,
			},
			expectNum: 1,
			expectHas: []string{"sub"},
		},
		{
			name: "无git仓库",
			structure: map[string]bool{
				"foo": true,
				"bar": true,
			},
			expectNum: 1,
			expectHas: []string{"."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := createTestDir(t, tt.structure)
			logger := &mocks.MockLogger{}
			logger.On("Info", mock.Anything, mock.Anything).Return()
			wr := NewWorkSpaceReader(logger)
			projects := wr.FindProjects(dir)
			assert.Equal(t, tt.expectNum, len(projects))
			for _, rel := range tt.expectHas {
				var found bool
				expPath := dir
				if rel != "." {
					expPath = filepath.Join(dir, rel)
				}
				for _, p := range projects {
					if filepath.Clean(p.Path) == filepath.Clean(expPath) {
						found = true
						break
					}
				}
				assert.True(t, found, "should find project: %s", expPath)
			}
		})
	}
}
