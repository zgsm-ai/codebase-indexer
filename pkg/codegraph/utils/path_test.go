package utils

import (
	"os"
	"path/filepath"
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

// 测试用例结构体
type listFilesTestCase struct {
	name        string               // 测试用例名称
	setupFunc   func(tempDir string) // 测试环境 setup 函数
	wantErr     bool                 // 是否期望返回错误
	wantFileCnt int                  // 期望返回的文件数量
}

// TestListFiles 表格驱动测试
func TestListFiles(t *testing.T) {
	// 定义测试用例
	testCases := []listFilesTestCase{
		{
			name: "正常场景：包含文件和子目录",
			setupFunc: func(tempDir string) {
				// 创建测试文件
				os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content"), 0644)
				os.WriteFile(filepath.Join(tempDir, "image.png"), []byte("binary"), 0644)

				// 创建子目录及其中的文件（不应被列出）
				subDir := filepath.Join(tempDir, "subdir")
				os.Mkdir(subDir, 0755)
				os.WriteFile(filepath.Join(subDir, "subfile.txt"), []byte("subcontent"), 0644)
			},
			wantErr:     false,
			wantFileCnt: 2,
		},
		{
			name: "空目录场景",
			setupFunc: func(tempDir string) {
				// 不创建任何文件
			},
			wantErr:     false,
			wantFileCnt: 0,
		},
		{
			name: "目录不存在场景",
			setupFunc: func(tempDir string) {
				// 不做任何 setup，使用不存在的子目录
			},
			wantErr:     true,
			wantFileCnt: 0,
		},
		{
			name: "只有子目录的场景",
			setupFunc: func(tempDir string) {
				// 创建多个子目录
				for i := 0; i < 3; i++ {
					subDir := filepath.Join(tempDir, "subdir"+string(rune(i+48)))
					os.Mkdir(subDir, 0755)
				}
			},
			wantErr:     false,
			wantFileCnt: 0,
		},
		{
			name: "包含隐藏文件的场景",
			setupFunc: func(tempDir string) {
				// 创建隐藏文件（Unix-like 系统）
				os.WriteFile(filepath.Join(tempDir, ".hidden"), []byte("secret"), 0644)
				// 创建普通文件
				os.WriteFile(filepath.Join(tempDir, "visible.txt"), []byte("public"), 0644)
			},
			wantErr:     false,
			wantFileCnt: 2,
		},
	}

	// 执行测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建临时目录
			tempDir := t.TempDir()

			// 确定测试目标目录
			targetDir := tempDir
			if tc.name == "目录不存在场景" {
				// 构造不存在的目录路径
				targetDir = filepath.Join(tempDir, "nonexistent")
			} else {
				// 执行 setup 函数
				tc.setupFunc(tempDir)
			}

			// 调用被测试函数
			files, err := ListFiles(targetDir)

			// 验证错误是否符合预期
			if (err != nil) != tc.wantErr {
				t.Fatalf("错误验证失败: 期望错误=%v, 实际错误=%v", tc.wantErr, err)
			}
			if tc.wantErr {
				return // 错误场景无需继续验证文件数量
			}

			// 验证文件数量是否符合预期
			if len(files) != tc.wantFileCnt {
				t.Errorf("文件数量验证失败: 期望=%d, 实际=%d, 文件列表=%v",
					tc.wantFileCnt, len(files), files)
			}
		})
	}
}
