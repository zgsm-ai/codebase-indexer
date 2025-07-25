package utils

import (
	"testing"
)

func TestIsSameParentDir(t *testing.T) {
	tests := []struct {
		name     string
		pathA    string
		pathB    string
		expected bool
	}{
		// 保留之前的正确测试用例...
		{
			name:     "unix same parent absolute",
			pathA:    "/home/user/docs/file1.txt",
			pathB:    "/home/user/docs/file2.txt",
			expected: true,
		},
		{
			name:     "unix different parent absolute",
			pathA:    "/home/user/docs/file1.txt",
			pathB:    "/home/user/pics/file2.txt",
			expected: false,
		},
		// ...其他原有测试用例

		// 修正相对路径测试用例
		{
			name:     "relative paths with same parent",
			pathA:    "./current/dir/fileA",
			pathB:    "./current/dir/fileB",
			expected: true, // 相同相对路径下的文件
		},
		{
			name:     "relative paths with different parents",
			pathA:    "./current/dir/fileA",
			pathB:    "../current/dir/fileB",
			expected: false, // 不同相对路径下的文件
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSameParentDir(tt.pathA, tt.pathB)
			if result != tt.expected {
				t.Errorf("test %q failed: got %v, expected %v (paths: %q and %q)",
					tt.name, result, tt.expected, tt.pathA, tt.pathB)
			}
		})
	}
}
