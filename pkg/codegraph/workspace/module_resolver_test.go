package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockLogger 实现 Logger 接口，用于测试
type MockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
	fatalMessages []string
}

func NewMockLogger() *MockLogger {
	return &MockLogger{
		debugMessages: []string{},
		infoMessages:  []string{},
		warnMessages:  []string{},
		errorMessages: []string{},
		fatalMessages: []string{},
	}
}

func (m *MockLogger) Debug(format string, args ...any) {
	m.debugMessages = append(m.debugMessages, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Info(format string, args ...any) {
	m.infoMessages = append(m.infoMessages, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Warn(format string, args ...any) {
	m.warnMessages = append(m.warnMessages, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Error(format string, args ...any) {
	m.errorMessages = append(m.errorMessages, fmt.Sprintf(format, args...))
}

func (m *MockLogger) Fatal(format string, args ...any) {
	m.fatalMessages = append(m.fatalMessages, fmt.Sprintf(format, args...))
}

// TestNewModuleResolver 测试 NewModuleResolver 函数
func TestNewModuleResolver(t *testing.T) {
	mockLogger := NewMockLogger()
	
	resolver := NewModuleResolver(mockLogger)
	
	if resolver == nil {
		t.Fatal("NewModuleResolver 返回了 nil")
	}
	
	if resolver.logger != mockLogger {
		t.Error("Logger 未正确设置")
	}
}

// TestResolveProjectModules 测试 ResolveProjectModules 方法
func TestResolveProjectModules(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	// 测试传入 nil project 时的错误处理
	err := resolver.ResolveProjectModules(ctx, nil)
	if err == nil {
		t.Error("传入 nil project 时应该返回错误")
	}
	
	expectedErrMsg := "project cannot be nil"
	if err.Error() != expectedErrMsg {
		t.Errorf("错误消息不匹配，期望: %s, 实际: %s", expectedErrMsg, err.Error())
	}
	
	// 测试正常项目路径下的模块解析
	tempDir := t.TempDir()
	project := NewProject("test-project", tempDir)
	
	err = resolver.ResolveProjectModules(ctx, project)
	if err != nil {
		t.Errorf("解析项目模块时发生错误: %v", err)
	}
	
	// 验证日志消息
	found := false
	for _, msg := range mockLogger.infoMessages {
		if strings.Contains(msg, "开始解析项目模块信息") {
			found = true
			break
		}
	}
	if !found {
		t.Error("未找到开始解析的日志消息")
	}
}

// TestResolveProjectModulesWithVariousLanguages 测试各种语言包信息的解析
func TestResolveProjectModulesWithVariousLanguages(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	project := NewProject("test-project", tempDir)
	
	// 创建各种语言的配置文件
	createTestFiles(t, tempDir)
	
	err := resolver.ResolveProjectModules(ctx, project)
	if err != nil {
		t.Errorf("解析项目模块时发生错误: %v", err)
	}
	
	// 验证各种语言的包信息被正确解析
	// Go 模块
	if len(project.GoModules) == 0 {
		t.Error("Go 模块未被正确解析")
	}
	
	// Java 包前缀
	if len(project.JavaPackagePrefix) == 0 {
		t.Error("Java 包前缀未被正确解析")
	}
	
	// Python 包
	if len(project.PythonPackages) == 0 {
		t.Error("Python 包未被正确解析")
	}
	
	// C/C++ 头文件路径
	if len(project.CppIncludes) == 0 {
		t.Error("C/C++ 头文件路径未被正确解析")
	}
	
	// JavaScript/TypeScript 包
	if len(project.JsPackages) == 0 {
		t.Error("JavaScript/TypeScript 包未被正确解析")
	}
}

// TestResolveJavaPackagePrefixes 测试 resolveJavaPackagePrefixes 方法
func TestResolveJavaPackagePrefixes(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 测试 pom.xml 文件解析
	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <version>1.0.0</version>
</project>`
	
	pomPath := filepath.Join(tempDir, "pom.xml")
	err := os.WriteFile(pomPath, []byte(pomContent), 0644)
	if err != nil {
		t.Fatalf("创建 pom.xml 文件失败: %v", err)
	}
	
	prefixes, err := resolver.resolveJavaPackagePrefixes(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 Java 包前缀时发生错误: %v", err)
	}
	
	expectedPrefixes := []string{"com.example", "myapp", "com.example.myapp"}
	if !equalStringSlices(prefixes, expectedPrefixes) {
		t.Errorf("Java 包前缀不匹配，期望: %v, 实际: %v", expectedPrefixes, prefixes)
	}
	
	// 测试 Java 目录结构推断
	javaSrcPath := filepath.Join(tempDir, "src", "main", "java")
	comPath := filepath.Join(javaSrcPath, "com", "example", "myapp")
	err = os.MkdirAll(comPath, 0755)
	if err != nil {
		t.Fatalf("创建 Java 目录结构失败: %v", err)
	}
	
	// 创建一个 Java 文件
	javaFile := filepath.Join(comPath, "App.java")
	err = os.WriteFile(javaFile, []byte("package com.example.myapp;"), 0644)
	if err != nil {
		t.Fatalf("创建 Java 文件失败: %v", err)
	}
	
	prefixes, err = resolver.resolveJavaPackagePrefixes(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 Java 包前缀时发生错误: %v", err)
	}
	
	// 应该包含从 pom.xml 解析的前缀和从目录结构推断的前缀
	if !containsString(prefixes, "com.example.myapp") {
		t.Error("未包含从目录结构推断的包前缀")
	}
}

// TestResolvePythonPackages 测试 resolvePythonPackages 方法
func TestResolvePythonPackages(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 测试 setup.py 文件解析
	setupPyContent := `
from setuptools import setup
setup(
    name="myapp",
    packages=["myapp", "myapp.utils"]
)`
	
	setupPyPath := filepath.Join(tempDir, "setup.py")
	err := os.WriteFile(setupPyPath, []byte(setupPyContent), 0644)
	if err != nil {
		t.Fatalf("创建 setup.py 文件失败: %v", err)
	}
	
	packages, err := resolver.resolvePythonPackages(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 Python 包时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp") {
		t.Error("未从 setup.py 解析出包名")
	}
	
	// 测试 pyproject.toml 文件解析
	pyprojectContent := `[tool.poetry]
name = "myapp2"
`
	
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	err = os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644)
	if err != nil {
		t.Fatalf("创建 pyproject.toml 文件失败: %v", err)
	}
	
	packages, err = resolver.resolvePythonPackages(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 Python 包时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp") || !containsString(packages, "myapp2") {
		t.Error("未正确解析所有 Python 包")
	}
	
	// 测试 Python 包目录结构发现
	myappPath := filepath.Join(tempDir, "myapp")
	err = os.MkdirAll(myappPath, 0755)
	if err != nil {
		t.Fatalf("创建 Python 包目录失败: %v", err)
	}
	
	// 创建 __init__.py 文件
	initPath := filepath.Join(myappPath, "__init__.py")
	err = os.WriteFile(initPath, []byte("# Python package"), 0644)
	if err != nil {
		t.Fatalf("创建 __init__.py 文件失败: %v", err)
	}
	
	packages, err = resolver.resolvePythonPackages(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 Python 包时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp") {
		t.Error("未从目录结构发现 Python 包")
	}
}

// TestResolveCppIncludes 测试 resolveCppIncludes 方法
func TestResolveCppIncludes(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 创建 include 目录和头文件
	includePath := filepath.Join(tempDir, "include")
	utilsPath := filepath.Join(includePath, "utils")
	err := os.MkdirAll(utilsPath, 0755)
	if err != nil {
		t.Fatalf("创建 include 目录失败: %v", err)
	}
	
	// 创建头文件
	headerFile := filepath.Join(utilsPath, "utils.h")
	err = os.WriteFile(headerFile, []byte("#pragma once"), 0644)
	if err != nil {
		t.Fatalf("创建头文件失败: %v", err)
	}
	
	// 创建 src 目录和头文件
	srcPath := filepath.Join(tempDir, "src")
	err = os.MkdirAll(srcPath, 0755)
	if err != nil {
		t.Fatalf("创建 src 目录失败: %v", err)
	}
	
	srcHeader := filepath.Join(srcPath, "app.h")
	err = os.WriteFile(srcHeader, []byte("#pragma once"), 0644)
	if err != nil {
		t.Fatalf("创建 src 目录下的头文件失败: %v", err)
	}
	
	includes, err := resolver.resolveCppIncludes(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 C/C++ 头文件路径时发生错误: %v", err)
	}
	
	// 验证包含的头文件路径
	// 在 Windows 系统上，路径可能使用反斜杠
	hasUtils := false
	hasSrc := false
	
	for _, path := range includes {
		// 统一使用正斜杠进行比较
		normalizedPath := filepath.ToSlash(path)
		if normalizedPath == "include/utils" {
			hasUtils = true
		}
		if normalizedPath == "src" {
			hasSrc = true
		}
	}
	
	if !hasUtils {
		t.Errorf("未正确解析 include/utils 目录下的头文件路径: %v", includes)
	}
	
	if !hasSrc {
		t.Errorf("未正确解析 src 目录下的头文件路径: %v", includes)
	}
}

// TestResolveJsPackages 测试 resolveJsPackages 方法
func TestResolveJsPackages(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 测试 package.json 文件解析
	packageJsonContent := `{
    "name": "myapp",
    "private": false,
    "dependencies": {
        "react": "^18.0.0"
    }
}`
	
	packageJsonPath := filepath.Join(tempDir, "package.json")
	err := os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatalf("创建 package.json 文件失败: %v", err)
	}
	
	packages, err := resolver.resolveJsPackages(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 JavaScript/TypeScript 包时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp") {
		t.Error("未从 package.json 解析出包名")
	}
	
	// 测试 monorepo 结构
	monorepoContent := `{
    "name": "monorepo",
    "private": false,
    "workspaces": [
        "packages/*"
    ]
}`
	
	err = os.WriteFile(packageJsonPath, []byte(monorepoContent), 0644)
	if err != nil {
		t.Fatalf("更新 package.json 文件失败: %v", err)
	}
	
	// 创建 packages 目录和子包
	packagesPath := filepath.Join(tempDir, "packages")
	err = os.MkdirAll(packagesPath, 0755)
	if err != nil {
		t.Fatalf("创建 packages 目录失败: %v", err)
	}
	
	// 创建子包目录
	subPackagePath := filepath.Join(packagesPath, "subpackage")
	err = os.MkdirAll(subPackagePath, 0755)
	if err != nil {
		t.Fatalf("创建子包目录失败: %v", err)
	}
	
	// 创建子包的 package.json
	subPackageJsonPath := filepath.Join(subPackagePath, "package.json")
	err = os.WriteFile(subPackageJsonPath, []byte(`{"name": "subpackage"}`), 0644)
	if err != nil {
		t.Fatalf("创建子包的 package.json 文件失败: %v", err)
	}
	
	packages, err = resolver.resolveJsPackages(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 JavaScript/TypeScript 包时发生错误: %v", err)
	}
	
	if !containsString(packages, "subpackage") {
		t.Error("未从 monorepo 结构中解析出子包")
	}
}

// TestResolveGoModules 测试 resolveGoModules 方法
func TestResolveGoModules(t *testing.T) {
	ctx := context.Background()
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 创建 go.mod 文件
	goModContent := `module github.com/example/myapp

go 1.19
`
	
	goModPath := filepath.Join(tempDir, "go.mod")
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("创建 go.mod 文件失败: %v", err)
	}
	
	modules, err := resolver.resolveGoModules(ctx, tempDir)
	if err != nil {
		t.Errorf("解析 Go 模块时发生错误: %v", err)
	}
	
	if len(modules) != 1 || modules[0] != "github.com/example/myapp" {
		t.Errorf("Go 模块解析不正确，期望: [github.com/example/myapp], 实际: %v", modules)
	}
}

// TestDeduplicateStrings 测试 deduplicateStrings 方法
func TestDeduplicateStrings(t *testing.T) {
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	// 测试正常去重
	input := []string{"a", "b", "a", "c", "", "b", ""}
	result := resolver.deduplicateStrings(input)
	
	expected := []string{"a", "b", "c"}
	if !equalStringSlices(result, expected) {
		t.Errorf("去重结果不匹配，期望: %v, 实际: %v", expected, result)
	}
	
	// 测试空切片
	emptyInput := []string{}
	emptyResult := resolver.deduplicateStrings(emptyInput)
	
	if len(emptyResult) != 0 {
		t.Errorf("空切片去重结果应该为空，实际: %v", emptyResult)
	}
	
	// 测试只包含空字符串的切片
	emptyStrInput := []string{"", "", ""}
	emptyStrResult := resolver.deduplicateStrings(emptyStrInput)
	
	if len(emptyStrResult) != 0 {
		t.Errorf("只包含空字符串的切片去重结果应该为空，实际: %v", emptyStrResult)
	}
}

// TestParsePomXML 测试 parsePomXML 方法
func TestParsePomXML(t *testing.T) {
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 测试完整的 pom.xml
	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <version>1.0.0</version>
</project>`
	
	pomPath := filepath.Join(tempDir, "pom.xml")
	err := os.WriteFile(pomPath, []byte(pomContent), 0644)
	if err != nil {
		t.Fatalf("创建 pom.xml 文件失败: %v", err)
	}
	
	prefixes, err := resolver.parsePomXML(pomPath)
	if err != nil {
		t.Errorf("解析 pom.xml 文件时发生错误: %v", err)
	}
	
	expected := []string{"com.example", "myapp", "com.example.myapp"}
	if !equalStringSlices(prefixes, expected) {
		t.Errorf("解析结果不匹配，期望: %v, 实际: %v", expected, prefixes)
	}
	
	// 测试只有 groupId 的 pom.xml
	pomContent2 := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <groupId>com.example</groupId>
    <version>1.0.0</version>
</project>`
	
	pomPath2 := filepath.Join(tempDir, "pom2.xml")
	err = os.WriteFile(pomPath2, []byte(pomContent2), 0644)
	if err != nil {
		t.Fatalf("创建 pom2.xml 文件失败: %v", err)
	}
	
	prefixes, err = resolver.parsePomXML(pomPath2)
	if err != nil {
		t.Errorf("解析 pom2.xml 文件时发生错误: %v", err)
	}
	
	expected2 := []string{"com.example"}
	if !equalStringSlices(prefixes, expected2) {
		t.Errorf("解析结果不匹配，期望: %v, 实际: %v", expected2, prefixes)
	}
	
	// 测试不存在的文件
	nonexistentPath := filepath.Join(tempDir, "nonexistent.xml")
	_, err = resolver.parsePomXML(nonexistentPath)
	if err == nil {
		t.Error("解析不存在的文件应该返回错误")
	}
}

// TestParseSetupPy 测试 parseSetupPy 方法
func TestParseSetupPy(t *testing.T) {
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 测试包含 name 的 setup.py
	setupPyContent := `from setuptools import setup
setup(
    name="myapp",
    version="1.0.0"
)`
	
	setupPyPath := filepath.Join(tempDir, "setup.py")
	err := os.WriteFile(setupPyPath, []byte(setupPyContent), 0644)
	if err != nil {
		t.Fatalf("创建 setup.py 文件失败: %v", err)
	}
	
	packages, err := resolver.parseSetupPy(setupPyPath)
	if err != nil {
		t.Errorf("解析 setup.py 文件时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp") {
		t.Error("未解析出包名")
	}
	
	// 测试使用单引号的 setup.py
	setupPyContent2 := `from setuptools import setup
setup(
    name='myapp2',
    version='1.0.0'
)`
	
	setupPyPath2 := filepath.Join(tempDir, "setup2.py")
	err = os.WriteFile(setupPyPath2, []byte(setupPyContent2), 0644)
	if err != nil {
		t.Fatalf("创建 setup2.py 文件失败: %v", err)
	}
	
	packages, err = resolver.parseSetupPy(setupPyPath2)
	if err != nil {
		t.Errorf("解析 setup2.py 文件时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp2") {
		t.Error("未解析出包名")
	}
	
	// 测试不包含 name 的 setup.py
	setupPyContent3 := `from setuptools import setup
setup(
    version="1.0.0"
)`
	
	setupPyPath3 := filepath.Join(tempDir, "setup3.py")
	err = os.WriteFile(setupPyPath3, []byte(setupPyContent3), 0644)
	if err != nil {
		t.Fatalf("创建 setup3.py 文件失败: %v", err)
	}
	
	packages, err = resolver.parseSetupPy(setupPyPath3)
	if err != nil {
		t.Errorf("解析 setup3.py 文件时发生错误: %v", err)
	}
	
	if len(packages) != 0 {
		t.Errorf("不应解析出包名，实际: %v", packages)
	}
}

// TestParsePackageJson 测试 parsePackageJson 方法
func TestParsePackageJson(t *testing.T) {
	mockLogger := NewMockLogger()
	resolver := NewModuleResolver(mockLogger)
	
	tempDir := t.TempDir()
	
	// 测试正常的 package.json
	packageJsonContent := `{
    "name": "myapp",
    "private": false,
    "dependencies": {
        "react": "^18.0.0"
    }
}`
	
	packageJsonPath := filepath.Join(tempDir, "package.json")
	err := os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatalf("创建 package.json 文件失败: %v", err)
	}
	
	packages, err := resolver.parsePackageJson(packageJsonPath)
	if err != nil {
		t.Errorf("解析 package.json 文件时发生错误: %v", err)
	}
	
	if !containsString(packages, "myapp") {
		t.Error("未解析出包名")
	}
	
	// 测试 private 包
	packageJsonContent2 := `{
    "name": "myprivateapp",
    "private": true
}`
	
	packageJsonPath2 := filepath.Join(tempDir, "package2.json")
	err = os.WriteFile(packageJsonPath2, []byte(packageJsonContent2), 0644)
	if err != nil {
		t.Fatalf("创建 package2.json 文件失败: %v", err)
	}
	
	packages, err = resolver.parsePackageJson(packageJsonPath2)
	if err != nil {
		t.Errorf("解析 package2.json 文件时发生错误: %v", err)
	}
	
	if len(packages) != 0 {
		t.Errorf("private 包不应被解析，实际: %v", packages)
	}
	
	// 测试 monorepo 结构
	packageJsonContent3 := `{
    "name": "monorepo",
    "private": false,
    "workspaces": [
        "packages/*"
    ]
}`
	
	packageJsonPath3 := filepath.Join(tempDir, "package3.json")
	err = os.WriteFile(packageJsonPath3, []byte(packageJsonContent3), 0644)
	if err != nil {
		t.Fatalf("创建 package3.json 文件失败: %v", err)
	}
	
	// 创建 packages 目录和子包
	packagesPath := filepath.Join(tempDir, "packages")
	err = os.MkdirAll(packagesPath, 0755)
	if err != nil {
		t.Fatalf("创建 packages 目录失败: %v", err)
	}
	
	// 创建子包目录
	subPackagePath := filepath.Join(packagesPath, "subpackage")
	err = os.MkdirAll(subPackagePath, 0755)
	if err != nil {
		t.Fatalf("创建子包目录失败: %v", err)
	}
	
	packages, err = resolver.parsePackageJson(packageJsonPath3)
	if err != nil {
		t.Errorf("解析 package3.json 文件时发生错误: %v", err)
	}
	
	if !containsString(packages, "monorepo") || !containsString(packages, "subpackage") {
		t.Errorf("未正确解析 monorepo 结构，实际: %v", packages)
	}
}

// createTestFiles 创建测试用的各种语言配置文件
func createTestFiles(t *testing.T, tempDir string) {
	// 创建 go.mod 文件
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := `module github.com/example/myapp

go 1.19
`
	err := os.WriteFile(goModPath, []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("创建 go.mod 文件失败: %v", err)
	}
	
	// 创建 pom.xml 文件
	pomPath := filepath.Join(tempDir, "pom.xml")
	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <version>1.0.0</version>
</project>`
	err = os.WriteFile(pomPath, []byte(pomContent), 0644)
	if err != nil {
		t.Fatalf("创建 pom.xml 文件失败: %v", err)
	}
	
	// 创建 Java 目录结构
	javaSrcPath := filepath.Join(tempDir, "src", "main", "java")
	comPath := filepath.Join(javaSrcPath, "com", "example", "myapp")
	err = os.MkdirAll(comPath, 0755)
	if err != nil {
		t.Fatalf("创建 Java 目录结构失败: %v", err)
	}
	
	javaFile := filepath.Join(comPath, "App.java")
	err = os.WriteFile(javaFile, []byte("package com.example.myapp;"), 0644)
	if err != nil {
		t.Fatalf("创建 Java 文件失败: %v", err)
	}
	
	// 创建 setup.py 文件
	setupPyPath := filepath.Join(tempDir, "setup.py")
	setupPyContent := `
from setuptools import setup
setup(
    name="myapp",
    packages=["myapp", "myapp.utils"]
)`
	err = os.WriteFile(setupPyPath, []byte(setupPyContent), 0644)
	if err != nil {
		t.Fatalf("创建 setup.py 文件失败: %v", err)
	}
	
	// 创建 pyproject.toml 文件
	pyprojectPath := filepath.Join(tempDir, "pyproject.toml")
	pyprojectContent := `[tool.poetry]
name = "myapp2"
`
	err = os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644)
	if err != nil {
		t.Fatalf("创建 pyproject.toml 文件失败: %v", err)
	}
	
	// 创建 Python 包目录结构
	myappPath := filepath.Join(tempDir, "myapp")
	err = os.MkdirAll(myappPath, 0755)
	if err != nil {
		t.Fatalf("创建 Python 包目录失败: %v", err)
	}
	
	initPath := filepath.Join(myappPath, "__init__.py")
	err = os.WriteFile(initPath, []byte("# Python package"), 0644)
	if err != nil {
		t.Fatalf("创建 __init__.py 文件失败: %v", err)
	}
	
	// 创建 include 目录和头文件
	includePath := filepath.Join(tempDir, "include")
	utilsPath := filepath.Join(includePath, "utils")
	err = os.MkdirAll(utilsPath, 0755)
	if err != nil {
		t.Fatalf("创建 include 目录失败: %v", err)
	}
	
	headerFile := filepath.Join(utilsPath, "utils.h")
	err = os.WriteFile(headerFile, []byte("#pragma once"), 0644)
	if err != nil {
		t.Fatalf("创建头文件失败: %v", err)
	}
	
	// 创建 src 目录和头文件
	srcPath := filepath.Join(tempDir, "src")
	err = os.MkdirAll(srcPath, 0755)
	if err != nil {
		t.Fatalf("创建 src 目录失败: %v", err)
	}
	
	srcHeader := filepath.Join(srcPath, "app.h")
	err = os.WriteFile(srcHeader, []byte("#pragma once"), 0644)
	if err != nil {
		t.Fatalf("创建 src 目录下的头文件失败: %v", err)
	}
	
	// 创建 package.json 文件
	packageJsonPath := filepath.Join(tempDir, "package.json")
	packageJsonContent := `{
    "name": "myapp",
    "private": false,
    "dependencies": {
        "react": "^18.0.0"
    }
}`
	err = os.WriteFile(packageJsonPath, []byte(packageJsonContent), 0644)
	if err != nil {
		t.Fatalf("创建 package.json 文件失败: %v", err)
	}
}

// equalStringSlices 检查两个字符串切片是否相等（不考虑顺序）
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	
	aMap := make(map[string]bool)
	for _, s := range a {
		aMap[s] = true
	}
	
	for _, s := range b {
		if !aMap[s] {
			return false
		}
	}
	
	return true
}

// containsString 检查字符串切片是否包含指定字符串
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}