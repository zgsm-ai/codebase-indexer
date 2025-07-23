package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetRootDir(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"USERPROFILE":     os.Getenv("USERPROFILE"),
		"APPDATA":         os.Getenv("APPDATA"),
		"XDG_CONFIG_HOME": os.Getenv("XDG_CONFIG_HOME"),
	}

	tests := []struct {
		name        string
		env         map[string]string
		appName     string
		want        string
		wantErr     bool
		cleanupfunc func()
	}{
		// Test normal path handling for current platform
		{
			name:    "basic path test",
			appName: "testapp",
			want:    filepath.Join(os.Getenv("USERPROFILE"), ".testapp"),
			cleanupfunc: func() {
				_ = os.RemoveAll(filepath.Join(os.Getenv("USERPROFILE"), ".testapp"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got, err := GetRootDir(tt.appName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRootDir() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				// Verify directory creation
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetRootDir() = %v, path does not exist", got)
				}
				// Verify global variables
				if AppRootDir != got {
					t.Errorf("AppRootDir = %v, want %v", AppRootDir, got)
				}
			}

			// Restore environment
			for k := range tt.env {
				os.Unsetenv(k)
			}

			if tt.cleanupfunc != nil {
				tt.cleanupfunc()
			}
		})
	}

	// Restore global environment
	for k, v := range originalEnv {
		os.Setenv(k, v)
	}
}

func TestGetLogDir(t *testing.T) {
	normalPath := filepath.Join(os.TempDir(), "normal_log")
	if err := os.MkdirAll(normalPath, 0755); err != nil {
		t.Fatal("Failed to create normal_log directory", normalPath)
	}

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
			name:     "normal case",
			rootPath: normalPath,
			wantErr:  false,
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare test environment
			if tt.prepareFunc != nil {
				if err := tt.prepareFunc(tt.rootPath); err != nil {
					t.Fatalf("prepare failed: %v", err)
				}
			}

			// Run test
			got, err := GetLogDir(tt.rootPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLogDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				// Verify return value
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetLogDir() = %v, path does not exist", got)
				}
				// Verify directory permissions (Windows doesn't enforce strict permissions)
				if runtime.GOOS != "windows" {
					if fi, err := os.Stat(got); err == nil {
						if fi.Mode().Perm() != 0755 {
							t.Errorf("GetLogDir() created directory has wrong permissions: %v", fi.Mode().Perm())
						}
					}
				}
				// Verify global variables
				if LogsDir != got {
					t.Errorf("LogDir global variable = %v, want %v", LogsDir, got)
				}
			}

			// Clean up
			if tt.cleanupFunc != nil {
				tt.cleanupFunc(tt.rootPath)
			}
		})
	}
}

func TestGetCacheDir(t *testing.T) {
	normalPath := filepath.Join(os.TempDir(), "normal_cache")
	if err := os.MkdirAll(normalPath, 0755); err != nil {
		t.Fatal("Failed to create normal_cache directory", normalPath)
	}
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
			name:     "normal case",
			rootPath: normalPath,
			wantErr:  false,
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
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
				// Verify return value
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetCacheDir() = %v, path does not exist", got)
				}
				// Verify directory permissions
				if runtime.GOOS != "windows" {
					if fi, err := os.Stat(got); err == nil {
						if fi.Mode().Perm() != 0755 {
							t.Errorf("GetCacheDir() created directory has wrong permissions: %v", fi.Mode().Perm())
						}
					}
				}
				// Verify global variables
				if CacheDir != got {
					t.Errorf("CacheDir global variable = %v, want %v", CacheDir, got)
				}
			}

			// Clean up
			if tt.cleanupFunc != nil {
				tt.cleanupFunc(tt.rootPath)
			}
		})
	}
}

func TestGetUploadTmpDir(t *testing.T) {
	normalPath := filepath.Join(os.TempDir(), "normal_upload")
	if err := os.MkdirAll(normalPath, 0755); err != nil {
		t.Fatal("Failed to create normal_upload directory", normalPath)
	}
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
			name:     "normal case",
			rootPath: normalPath,
			wantErr:  false,
			cleanupFunc: func(path string) {
				_ = os.RemoveAll(path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare test environment
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
				// Verify return value
				if _, err := os.Stat(got); os.IsNotExist(err) {
					t.Errorf("GetUploadTmpDir() = %v, path does not exist", got)
				}
				// Verify directory permissions
				if runtime.GOOS != "windows" {
					if fi, err := os.Stat(got); err == nil {
						if fi.Mode().Perm() != 0755 {
							t.Errorf("GetUploadTmpDir() created directory has wrong permissions: %v", fi.Mode().Perm())
						}
					}
				}
				// Verify global variables
				if UploadTmpDir != got {
					t.Errorf("UploadTmpDir global variable = %v, want %v", UploadTmpDir, got)
				}
			}

			// Clean up
			if tt.cleanupFunc != nil {
				tt.cleanupFunc(tt.rootPath)
			}
		})
	}
}
