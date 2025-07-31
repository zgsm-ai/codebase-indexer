package workspace

import (
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/types"
	"crypto/sha256"
	"encoding/hex"

	"google.golang.org/protobuf/proto"
)

// Project 项目基础配置信息
type Project struct {
	Name     string
	Path     string
	GoModule string
	Uuid     string
}

// generateUuid 生成项目UUID
func generateUuid(name, path string) string {
	if name == types.EmptyString {
		name = "unknown"
	}
	if path == types.EmptyString {
		path = "unknown"
	}

	hash := sha256.Sum256([]byte(path))
	return name + types.Underline + hex.EncodeToString(hash[:])
}

type FileElementTables []*codegraphpb.FileElementTable

func (l FileElementTables) Len() int { return len(l) }
func (l FileElementTables) Value(i int) proto.Message {
	return l[i]
}
func (l FileElementTables) Key(i int) string {
	return l[i].Path
}

type SymbolDefinitions []*codegraphpb.SymbolDefinition

func (l SymbolDefinitions) Len() int { return len(l) }
func (l SymbolDefinitions) Value(i int) proto.Message {
	return l[i]
}

func (l SymbolDefinitions) Key(i int) string {
	return l[i].Name
}
