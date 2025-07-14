package resolver

//
//import (
//	"reflect"
//	"testing"
//)
//
//func TestJavaResolver_Basic(t *testing.T) {
//	cfg := NewProjectConfig(Java, "src/main/java", []string{
//		"src/main/java/a/b/c/Class1.java",
//		"src/main/java/a/b/c/Class2.java",
//		"src/main/java/a/b/d/Class3.java",
//	})
//
//	resolver := &JavaResolver{}
//
//	// 测试包导入
//	imp := &Import{BaseElement: &BaseElement{Name: "a.b.c.*"}}
//	err := resolver.Resolve(imp, "src/main/java/a/b/c/Any.java", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/main/java/a/b/c/Class1.java", "src/main/java/a/b/c/Class2.java"}) {
//		t.Errorf("Java 包导入失败: %v, %v", err, imp.FilePaths)
//	}
//
//	// 测试类导入
//	imp = &Import{BaseElement: &BaseElement{Name: "a.b.c.Class1"}}
//	err = resolver.Resolve(imp, "src/main/java/a/b/c/Any.java", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/main/java/a/b/c/Class1.java"}) {
//		t.Errorf("Java 类导入失败: %v, %v", err, imp.FilePaths)
//	}
//}
//
//func TestGoResolver_Basic(t *testing.T) {
//	sourceRoot := "github.com/zgsm-ai"
//	config := NewProjectConfig(Go, sourceRoot, []string{
//		"pkg/foo/bar.go",
//		"pkg/foo/baz.go",
//		"pkg/other.go",
//	})
//
//	resolver := &GoResolver{}
//
//	// 测试包导入
//	imp := &Import{BaseElement: &BaseElement{Name: sourceRoot + "/pkg/" + "foo"}}
//	err := resolver.Resolve(imp, "pkg/other.go", config)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"pkg/foo/bar.go", "pkg/foo/baz.go"}) {
//		t.Errorf("Go 包导入失败: %v, %v", err, imp.FilePaths)
//	}
//
//	// 测试文件导入
//	imp = &Import{BaseElement: &BaseElement{Name: sourceRoot + "/pkg/" + "foo/bar"}}
//	err = resolver.Resolve(imp, "pkg/other.go", config)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"pkg/foo/bar.go"}) {
//		t.Errorf("Go 文件导入失败: %v, %v", err, imp.FilePaths)
//	}
//}
//
//func TestPythonResolver_Basic(t *testing.T) {
//	cfg := &ProjectInfo{}
//	cfg.buildIndex([]string{
//		"src/foo/__init__.py",
//		"src/foo/bar.py",
//	})
//	resolver := &PythonResolver{}
//
//	// 测试绝对导入
//	imp := &Import{BaseElement: &BaseElement{Name: "src.foo.bar"}}
//	err := resolver.Resolve(imp, "src/main.py", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/foo/bar.py"}) {
//		t.Errorf("Python 绝对导入失败: %v, %v", err, imp.FilePaths)
//	}
//}
//
//func TestCppResolver_Basic(t *testing.T) {
//	cfg := &ProjectInfo{}
//	cfg.buildIndex([]string{
//		"include/foo.h",
//		"include/bar.h",
//	})
//	resolver := &CppResolver{}
//
//	// 测试相对路径导入
//	imp := &Import{BaseElement: &BaseElement{Name: "foo.h"}}
//	err := resolver.Resolve(imp, "include/main.cpp", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"include/foo.h"}) {
//		t.Errorf("C/C++ 头文件导入失败: %v, %v", err, imp.FilePaths)
//	}
//}
//
//func TestJavaScriptResolver_Basic(t *testing.T) {
//	cfg := &ProjectInfo{}
//	cfg.buildIndex([]string{
//		"src/foo.js",
//		"src/bar/index.js",
//	})
//	resolver := &JavaScriptResolver{}
//
//	// 测试相对路径导入
//	imp := &Import{BaseElement: &BaseElement{Name: "./foo"}}
//	err := resolver.Resolve(imp, "src/main.js", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/foo.js"}) {
//		t.Errorf("JS 相对路径导入失败: %v, %v", err, imp.FilePaths)
//	}
//
//	// 测试绝对路径导入
//	imp = &Import{BaseElement: &BaseElement{Name: "bar"}}
//	err = resolver.Resolve(imp, "src/main.js", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/bar/index.js"}) {
//		t.Errorf("JS 绝对路径导入失败: %v, %v", err, imp.FilePaths)
//	}
//}
//
//func TestRustResolver_Basic(t *testing.T) {
//	cfg := &ProjectInfo{}
//	cfg.buildIndex([]string{
//		"src/foo.rs",
//		"src/bar/mod.rs",
//	})
//	resolver := &RustResolver{}
//
//	// 测试模块导入
//	imp := &Import{BaseElement: &BaseElement{Name: "foo"}}
//	err := resolver.Resolve(imp, "src/main.rs", cfg)
//	if err != nil || !reflect.DeepEqual(imp.FilePaths, []string{"src/foo.rs"}) {
//		t.Errorf("Rust 模块导入失败: %v, %v", err, imp.FilePaths)
//	}
//}
