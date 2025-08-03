package workspace

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"crypto/sha256"
	"encoding/hex"

	"google.golang.org/protobuf/proto"
)

// Project 项目基础配置信息
type Project struct {
	Name string
	Path string
	Uuid string

	GoModules []string
	// JavaPackagePrefix Java 项目包前缀（如 com.example.myapp）
	JavaPackagePrefix []string
	// PythonPackages Python 项目包列表（如 myapp, myapp.utils）
	PythonPackages []string
	// CppIncludes C/C++ 项目头文件路径（相对路径）
	CppIncludes []string
	// JsPackages JavaScript/TypeScript 项目包列表（如 myapp, @myapp/utils）
	JsPackages []string
}

func NewProject(name, path string) *Project {
	return &Project{
		Name:              name,
		Path:              path,
		Uuid:              generateUuid(name, path),
		GoModules:         []string{},
		JavaPackagePrefix: []string{}, // 默认为空字符串
		PythonPackages:    []string{}, // 默认为空切片
		CppIncludes:       []string{}, // 默认为空切片
		JsPackages:        []string{}, // 默认为空切片
	}
}

// generateUuid 生成缩短的项目UUID，保持唯一性同时减少长度
func generateUuid(name, path string) string {
	if name == types.EmptyString {
		name = "empty"
	}
	if path == types.EmptyString {
		path = "empty"
	}

	// 计算路径的SHA-256哈希
	hash := sha256.Sum256([]byte(path))

	// 截取前16字节（32个十六进制字符），原长度为32字节（64个字符）
	// 16字节哈希提供2^128的可能组合，足以保证唯一性
	shortHash := hex.EncodeToString(hash[:16])

	return name + types.Underline + shortHash
}

type FileElementTables []*codegraphpb.FileElementTable

func (l FileElementTables) Len() int { return len(l) }
func (l FileElementTables) Value(i int) proto.Message {
	return l[i]
}
func (l FileElementTables) Key(i int) store.Key {
	return store.ElementPathKey{Language: lang.Language(l[i].Language), Path: l[i].Path}
}

type SymbolDefinitions []*codegraphpb.SymbolDefinition

func (l SymbolDefinitions) Len() int { return len(l) }
func (l SymbolDefinitions) Value(i int) proto.Message {
	return l[i]
}

func (l SymbolDefinitions) Key(i int) store.Key {
	return store.SymbolNameKey{Language: lang.Language(l[i].Language), Name: l[i].Name}
}
