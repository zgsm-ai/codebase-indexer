package import_resolver

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// 辅助函数：创建测试Import
func createTestImport(name, path, source string) *resolver.Import {
	imp := &resolver.Import{
		BaseElement: resolver.NewBaseElement(0),
		Source:      source,
	}
	if name != "" {
		imp.SetName(name)
	}
	if path != "" {
		imp.SetPath(path)
	}
	return imp
}

func TestGoImportPathResolver(t *testing.T) {
	// 创建测试项目结构
	tmpDir := t.TempDir()

	// 创建Go项目结构
	pkgDir := filepath.Join(tmpDir, "pkg", "utils")
	os.MkdirAll(pkgDir, 0755)

	// 创建测试文件
	os.WriteFile(filepath.Join(pkgDir, "helper.go"), []byte("package utils"), 0644)
	os.WriteFile(filepath.Join(pkgDir, "types.go"), []byte("package utils"), 0644)
	os.WriteFile(filepath.Join(pkgDir, "helper_test.go"), []byte("package utils"), 0644)

	// 创建解析器
	searcher := NewFileSearcher(tmpDir)
	searcher.BuildIndex()
	resolver := NewGoImportPathResolver(searcher)

	// 创建测试导入
	imp := createTestImport("utils", "main.go", "github.com/test/project/pkg/utils")

	// 创建测试项目
	project := &workspace.Project{
		Path:      tmpDir,
		GoModules: []string{"github.com/test/project"},
	}

	// 执行解析
	ctx := context.Background()
	paths, err := resolver.ResolveImportPath(ctx, imp, project)

	if err != nil {
		t.Fatalf("ResolveImportPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected resolved paths, got none")
	}

	// 验证结果（应该包含2个.go文件，排除测试文件）
	expectedCount := 2
	if len(paths) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(paths), paths)
	}

	// 验证不包含测试文件
	for _, path := range paths {
		if filepath.Base(path) == "helper_test.go" {
			t.Errorf("Result should not include test file: %s", path)
		}
	}
}

func TestPythonImportPathResolver_Absolute(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建Python项目结构
	pkgDir := filepath.Join(tmpDir, "myapp", "utils")
	os.MkdirAll(pkgDir, 0755)

	os.WriteFile(filepath.Join(pkgDir, "__init__.py"), []byte(""), 0644)
	os.WriteFile(filepath.Join(pkgDir, "helper.py"), []byte(""), 0644)

	searcher := NewFileSearcher(tmpDir)
	searcher.BuildIndex()
	resolver := NewPythonImportPathResolver(searcher)

	imp := createTestImport("", "main.py", "myapp.utils")

	project := &workspace.Project{Path: tmpDir}

	ctx := context.Background()
	paths, err := resolver.ResolveImportPath(ctx, imp, project)

	if err != nil {
		t.Fatalf("ResolveImportPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected resolved paths, got none")
	}
}

func TestPythonImportPathResolver_Relative(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建目录结构
	pkgDir := filepath.Join(tmpDir, "myapp")
	os.MkdirAll(pkgDir, 0755)

	utilsDir := filepath.Join(pkgDir, "utils")
	os.MkdirAll(utilsDir, 0755)

	// 创建 __init__.py 和 helper.py
	os.WriteFile(filepath.Join(utilsDir, "__init__.py"), []byte(""), 0644)
	os.WriteFile(filepath.Join(utilsDir, "helper.py"), []byte(""), 0644)

	searcher := NewFileSearcher(tmpDir)
	searcher.BuildIndex()
	resolver := NewPythonImportPathResolver(searcher)

	// 相对导入：from ..utils import helper
	imp := createTestImport("", filepath.Join("myapp", "submodule", "main.py"), "..utils")

	project := &workspace.Project{Path: tmpDir}

	ctx := context.Background()
	paths, err := resolver.ResolveImportPath(ctx, imp, project)

	if err != nil {
		t.Fatalf("ResolveImportPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected resolved paths, got none")
	}
}

func TestJSImportPathResolver_Relative(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建JS项目结构
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	os.WriteFile(filepath.Join(srcDir, "utils.js"), []byte(""), 0644)
	os.WriteFile(filepath.Join(srcDir, "main.js"), []byte(""), 0644)

	searcher := NewFileSearcher(tmpDir)
	searcher.BuildIndex()
	resolver := NewJSImportPathResolver(searcher)

	imp := createTestImport("", filepath.Join("src", "main.js"), "./utils")

	project := &workspace.Project{Path: tmpDir}

	ctx := context.Background()
	paths, err := resolver.ResolveImportPath(ctx, imp, project)

	if err != nil {
		t.Fatalf("ResolveImportPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected resolved paths, got none")
	}
}

func TestJavaImportPathResolver_SingleClass(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建Java项目结构
	javaDir := filepath.Join(tmpDir, "src", "main", "java", "com", "example")
	os.MkdirAll(javaDir, 0755)

	os.WriteFile(filepath.Join(javaDir, "MyClass.java"), []byte(""), 0644)

	searcher := NewFileSearcher(tmpDir)
	searcher.BuildIndex()
	resolver := NewJavaImportPathResolver(searcher)

	imp := createTestImport("com.example.MyClass", "", "")

	project := &workspace.Project{Path: tmpDir}

	ctx := context.Background()
	paths, err := resolver.ResolveImportPath(ctx, imp, project)

	if err != nil {
		t.Fatalf("ResolveImportPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected resolved paths, got none")
	}
}

func TestCppImportPathResolver(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建C++项目结构
	includeDir := filepath.Join(tmpDir, "include")
	os.MkdirAll(includeDir, 0755)

	os.WriteFile(filepath.Join(includeDir, "utils.h"), []byte(""), 0644)

	searcher := NewFileSearcher(tmpDir)
	searcher.BuildIndex()
	resolver := NewCppImportPathResolver(searcher)

	imp := createTestImport(`"utils.h"`, "src/main.cpp", "")

	project := &workspace.Project{Path: tmpDir}

	ctx := context.Background()
	paths, err := resolver.ResolveImportPath(ctx, imp, project)

	if err != nil {
		t.Fatalf("ResolveImportPath failed: %v", err)
	}

	if len(paths) == 0 {
		t.Fatal("Expected resolved paths, got none")
	}
}

func TestPathResolverManager(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建多语言项目结构
	os.MkdirAll(filepath.Join(tmpDir, "pkg"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "pkg", "utils.go"), []byte(""), 0644)

	mgr := NewPathResolverManager(tmpDir)

	// 测试获取Go解析器
	goResolver := mgr.GetResolver(lang.Go)
	if goResolver == nil {
		t.Fatal("Expected Go resolver, got nil")
	}

	// 测试获取Python解析器
	pyResolver := mgr.GetResolver(lang.Python)
	if pyResolver == nil {
		t.Fatal("Expected Python resolver, got nil")
	}

	// 测试构建索引
	err := mgr.BuildIndex()
	if err != nil {
		t.Fatalf("BuildIndex failed: %v", err)
	}
}
