package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
			files, err := ListOnlyFiles(targetDir)

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

func TestEnsureTrailingSeparator(t *testing.T) {
	sep := string(filepath.Separator)
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"空路径", "", ""},
		{"无分隔符-单级", "dir", "dir" + sep},
		{"有分隔符-单级", "dir" + sep, "dir" + sep},
		{"无分隔符-多级", "a" + sep + "b", "a" + sep + "b" + sep},
		{"有分隔符-多级", "a" + sep + "b" + sep, "a" + sep + "b" + sep},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnsureTrailingSeparator(tt.input); got != tt.expected {
				t.Errorf("输入: %q\n期望: %q\n实际: %q", tt.input, tt.expected, got)
			}
		})
	}
}

func TestListSubDirs(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(t *testing.T) string
		teardown    func(t *testing.T, dir string)
		wantSubDirs []string
		wantErr     bool
	}{
		{
			name: "空目录",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			teardown:    func(t *testing.T, dir string) {},
			wantSubDirs: []string{},
			wantErr:     false,
		},
		{
			name: "包含普通目录和隐藏目录",
			setup: func(t *testing.T) string {
				rootDir := t.TempDir()

				// 创建普通目录
				os.Mkdir(filepath.Join(rootDir, "normal_dir1"), 0755)
				os.Mkdir(filepath.Join(rootDir, "normal_dir2"), 0755)

				// 创建隐藏目录（以.开头）
				os.Mkdir(filepath.Join(rootDir, ".hidden_dir1"), 0755)
				os.Mkdir(filepath.Join(rootDir, "_hidden_dir2"), 0755) // 非隐藏目录

				// 创建文件（不应被列出）
				f, _ := os.Create(filepath.Join(rootDir, "file.txt"))
				f.Close()

				return rootDir
			},
			teardown:    func(t *testing.T, dir string) {},
			wantSubDirs: []string{"normal_dir1", "normal_dir2", "_hidden_dir2"},
			wantErr:     false,
		},
		{
			name: "只有隐藏目录",
			setup: func(t *testing.T) string {
				rootDir := t.TempDir()
				os.Mkdir(filepath.Join(rootDir, ".hidden1"), 0755)
				os.Mkdir(filepath.Join(rootDir, ".hidden2"), 0755)
				return rootDir
			},
			teardown:    func(t *testing.T, dir string) {},
			wantSubDirs: []string{},
			wantErr:     false,
		},
		{
			name: "目录不存在",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "non_existent_dir")
			},
			teardown:    func(t *testing.T, dir string) {},
			wantSubDirs: []string{},
			wantErr:     true,
		},
		{
			name: "权限不足的目录",
			setup: func(t *testing.T) string {
				if runtime.GOOS == "windows" {
					t.Skip("Windows系统权限测试暂不支持")
				}

				rootDir := t.TempDir()
				restrictedDir := filepath.Join(rootDir, "restricted")
				os.Mkdir(restrictedDir, 0000) // 无任何权限

				return restrictedDir
			},
			teardown: func(t *testing.T, dir string) {
				// 恢复权限以便清理
				os.Chmod(dir, 0755)
			},
			wantSubDirs: []string{},
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setup(t)
			defer tc.teardown(t, dir)

			subDirs, err := ListSubDirs(dir)

			// 检查错误
			if (err != nil) != tc.wantErr {
				t.Errorf("错误预期: %v, 实际: %v", tc.wantErr, err)
				return
			}

			if tc.wantErr {
				return
			}

			// 提取目录名并排序
			var gotNames []string
			for _, d := range subDirs {
				gotNames = append(gotNames, filepath.Base(d))
			}
			sort.Strings(gotNames)
			sort.Strings(tc.wantSubDirs)

			// 比较数量
			if len(gotNames) != len(tc.wantSubDirs) {
				t.Errorf("目录数量不匹配: 预期 %d, 实际 %d, 结果: %v",
					len(tc.wantSubDirs), len(gotNames), gotNames)
				return
			}

			// 比较每个目录名
			for i := range gotNames {
				if gotNames[i] != tc.wantSubDirs[i] {
					t.Errorf("索引 %d 不匹配: 预期 %s, 实际 %s",
						i, tc.wantSubDirs[i], gotNames[i])
				}
			}
		})
	}
}

func TestIsSubdir(t *testing.T) {
	tests := []struct {
		name     string
		parent   string
		sub      string
		expected bool
	}{
		{
			name:     "NormalSubdir",
			parent:   "/home/user",
			sub:      "/home/user/documents",
			expected: true,
		},
		{
			name:     "SameDir",
			parent:   "/home/user",
			sub:      "/home/user",
			expected: false, // 相同目录不是子目录
		},
		{
			name:     "NotSubdir",
			parent:   "/home/user",
			sub:      "/var/log",
			expected: false,
		},
		{
			name:     "ParentTrailingSlash",
			parent:   "/home/user/",
			sub:      "/home/user/documents",
			expected: true,
		},
		{
			name:     "SubTrailingSlash",
			parent:   "/home/user",
			sub:      "/home/user/documents/",
			expected: true,
		},
		{
			name:     "BothTrailingSlash",
			parent:   "/home/user/",
			sub:      "/home/user/documents/",
			expected: true,
		},
		{
			name:     "DeepSubdir",
			parent:   "/a",
			sub:      "/a/b/c/d",
			expected: true,
		},
		{
			name:     "WindowsPath",
			parent:   "C:\\Users\\user",
			sub:      "C:\\Users\\user\\Downloads",
			expected: true,
		},
		{
			name:     "WindowsDifferentDrive",
			parent:   "C:\\Users",
			sub:      "D:\\Users",
			expected: false,
		},
		{
			name:     "WithDotSegments",
			parent:   "/home/user",
			sub:      "/home/user/../user/documents",
			expected: true, // 清理后是/home/user/documents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSubdir(tt.parent, tt.sub)
			if got != tt.expected {
				t.Errorf("IsSubdir(parent=%q, sub=%q) = %v, expected %v",
					tt.parent, tt.sub, got, tt.expected)
			}
		})
	}
}

func TestFindLongestExistingPath(t *testing.T) {
	// 创建临时测试目录结构
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	existingFile := filepath.Join(existingDir, "file.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 获取系统根目录（用于根目录测试）
	rootDir := filepath.VolumeName(tmpDir) + string(filepath.Separator)
	if rootDir == string(filepath.Separator) { // Unix 系统根目录
		rootDir = string(filepath.Separator)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
		// 测试前是否切换到临时目录（解决相对路径问题）
		chdirTemp bool
	}{
		{
			name:    "路径本身是存在的文件",
			path:    existingFile,
			want:    existingFile,
			wantErr: false,
		},
		{
			name:    "路径本身是存在的目录",
			path:    existingDir,
			want:    existingDir,
			wantErr: false,
		},
		{
			name:    "路径不存在，父目录存在",
			path:    filepath.Join(existingDir, "nonexist", "sub"),
			want:    existingDir,
			wantErr: false,
		},
		{
			name:    "多级父目录后存在",
			path:    filepath.Join(tmpDir, "a", "x", "y", "z"),
			want:    filepath.Join(tmpDir, "a"),
			wantErr: false,
		},
		{
			name:    "整个路径都不存在",
			path:    filepath.Join("/nonexist", "path"),
			wantErr: true, // 此时仅根目录存在，但原始路径不是根目录，返回错误
		},
		{
			name:    "根目录存在",
			path:    rootDir,
			want:    rootDir,
			wantErr: false,
		},
		{
			name:      "相对路径存在",
			path:      filepath.Join("a", "b"), // 相对路径，在临时目录下存在
			want:      filepath.Join(tmpDir, "a", "b"),
			wantErr:   false,
			chdirTemp: true, // 测试前切换到临时目录
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 若需要，切换到临时目录再处理相对路径
			var originalWd string
			if tt.chdirTemp {
				var err error
				originalWd, err = os.Getwd()
				if err != nil {
					t.Fatalf("获取当前工作目录失败: %v", err)
				}
				if err := os.Chdir(tmpDir); err != nil {
					t.Fatalf("切换到临时目录失败: %v", err)
				}
				defer os.Chdir(originalWd) // 测试结束后切回原目录
			}

			got, err := FindLongestExistingPath(tt.path)

			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不符: 实际=%v, 预期=%v", err, tt.wantErr)
			}

			if !tt.wantErr {
				gotClean := filepath.Clean(got)
				wantClean := filepath.Clean(tt.want)

				if gotClean != wantClean {
					t.Errorf("结果不符: 实际=%q, 预期=%q", gotClean, wantClean)
				}
			}
		})
	}
}
