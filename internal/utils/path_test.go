package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRootDir(t *testing.T) {
	// 保存原始环境
	originalEnv := map[string]string{
		"USERPROFILE":     os.Getenv("USERPROFILE"),
		"APPDATA":         os.Getenv("APPDATA"),
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
	}

	tests := []struct {
		name    string
		env     map[string]string
		appName string
		want    string
		wantErr bool
	}{
		// 测试当前平台下的正常路径处理
		{
			name:    "basic path test",
			appName: "testapp",
			want:    filepath.Join(os.Getenv("USERPROFILE"), ".testapp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got, err := GetRootDir(tt.appName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRootDir() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				// 验证目录是否创建
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetRootDir() = %v, path does not exist", got)
				}
				// 验证全局变量
				if AppRootDir != got {
					t.Errorf("AppRootDir = %v, want %v", AppRootDir, got)
				}
			}

			// 恢复环境
			for k := range tt.env {
				os.Unsetenv(k)
			}
		})
	}

	// 恢复全局环境
	for k, v := range originalEnv {
		os.Setenv(k, v)
	}
}

func TestGetLogDir(t *testing.T) {
	tests := []struct {
		name        string
		rootPath    string
		wantErr     bool
		prepareFunc func(string) error
		cleanupFunc func(string)
	}{
		{
			name:     "non-existent root path",
			rootPath: "/nonexistent/path",
			wantErr:  true,
		},
		{
			name:     "read-only directory",
			rootPath: filepath.Join(os.TempDir(), "readonly"),
			wantErr:  true,
			prepareFunc: func(path string) error {
				_ = os.MkdirAll(path, 0444) // 只读权限
				return nil
			},
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
		{
			name:     "normal case",
			rootPath: filepath.Join(os.TempDir(), "normal"),
			wantErr:  false,
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备测试环境
			if tt.prepareFunc != nil {
				if err := tt.prepareFunc(tt.rootPath); err != nil {
					t.Fatalf("prepare failed: %v", err)
				}
			}

			// 执行测试
			got, err := GetLogDir(tt.rootPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLogDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// 验证返回值
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetLogDir() = %v, path does not exist", got)
				}
				// 验证目录权限
				if fi, err := os.Stat(got); err == nil {
					if fi.Mode().Perm() != 0755 {
						t.Errorf("GetLogDir() created directory has wrong permissions: %v", fi.Mode().Perm())
					}
				}
			}

			// 清理
			if tt.cleanupFunc != nil {
				tt.cleanupFunc(tt.rootPath)
			}
		})
	}
}

func TestGetCacheDir(t *testing.T) {
	tests := []struct {
		name        string
		rootPath    string
		wantErr     bool
		prepareFunc func(string) error
		cleanupFunc func(string)
	}{
		{
			name:     "non-existent root path",
			rootPath: "/nonexistent/path",
			wantErr:  true,
		},
		{
			name:     "read-only directory",
			rootPath: filepath.Join(os.TempDir(), "readonly_cache"),
			wantErr:  true,
			prepareFunc: func(path string) error {
				_ = os.MkdirAll(path, 0444) // 只读权限
				return nil
			},
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
		{
			name:     "normal case",
			rootPath: filepath.Join(os.TempDir(), "normal_cache"),
			wantErr:  false,
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备测试环境
			if tt.prepareFunc != nil {
				if err := tt.prepareFunc(tt.rootPath); err != nil {
					t.Fatalf("prepare failed: %v", err)
				}
			}

			got, err := GetCacheDir(tt.rootPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCacheDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// 验证返回值
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetCacheDir() = %v, path does not exist", got)
				}
				// 验证目录权限
				if fi, err := os.Stat(got); err == nil {
					if fi.Mode().Perm() != 0755 {
						t.Errorf("GetCacheDir() created directory has wrong permissions: %v", fi.Mode().Perm())
					}
				}
				// 验证全局变量
				if CacheDir != got {
					t.Errorf("CacheDir global variable = %v, want %v", CacheDir, got)
				}
			}

			// 清理
			if tt.cleanupFunc != nil {
				tt.cleanupFunc(tt.rootPath)
			}
		})
	}
}

func TestGetUploadTmpDir(t *testing.T) {
	tests := []struct {
		name        string
		rootPath    string
		wantErr     bool
		prepareFunc func(string) error
		cleanupFunc func(string)
	}{
		{
			name:     "non-existent root path",
			rootPath: "/nonexistent/path",
			wantErr:  true,
		},
		{
			name:     "read-only directory",
			rootPath: filepath.Join(os.TempDir(), "readonly_upload"),
			wantErr:  true,
			prepareFunc: func(path string) error {
				_ = os.MkdirAll(path, 0444) // 只读权限
				return nil
			},
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
		{
			name:     "normal case",
			rootPath: filepath.Join(os.TempDir(), "normal_upload"),
			wantErr:  false,
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 准备测试环境
			if tt.prepareFunc != nil {
				if err := tt.prepareFunc(tt.rootPath); err != nil {
					t.Fatalf("prepare failed: %v", err)
				}
			}

			got, err := GetUploadTmpDir(tt.rootPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUploadTmpDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// 验证返回值
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetUploadTmpDir() = %v, path does not exist", got)
				}
				// 验证目录权限
				if fi, err := os.Stat(got); err == nil {
					if fi.Mode().Perm() != 0755 {
						t.Errorf("GetUploadTmpDir() created directory has wrong permissions: %v", fi.Mode().Perm())
					}
				}
				// 验证全局变量
				if UploadTmpDir != got {
					t.Errorf("UploadTmpDir global variable = %v, want %v", UploadTmpDir, got)
				}
			}

			// 清理
			if tt.cleanupFunc != nil {
				tt.cleanupFunc(tt.rootPath)
			}
		})
	}
}
