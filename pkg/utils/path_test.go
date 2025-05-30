package utils

import (
	"os"
	"path/filepath"
	"testing"
)

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
