package workspace

import (
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/types"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"google.golang.org/protobuf/proto"
)

// Project 项目基础配置信息
type Project struct {
	Name     string
	Path     string
	GoModule string
}

func (p *Project) Uuid() (string, error) {
	if p.Name == types.EmptyString {
		return types.EmptyString, fmt.Errorf("get_uuid project %s %s missing name", p.Name, p.Path)
	}

	if p.Path == types.EmptyString {
		return types.EmptyString, fmt.Errorf("get_uuid project %s %s missing path", p.Name, p.Path)
	}

	hash := sha256.Sum256([]byte(p.Path))

	return p.Name + types.Underline + hex.EncodeToString(hash[:]), nil
}

type FileElementTables []*parser.FileElementTable

func (l FileElementTables) Len() int { return len(l) }
func (l FileElementTables) Value(i int) proto.Message {
	return l[i]
}
func (l FileElementTables) Key(i int) string {
	return l[i].Path
}

type SymbolDefinitions []*analyzer.SymbolDefinition

func (l SymbolDefinitions) Len() int { return len(l) }
func (l SymbolDefinitions) Value(i int) proto.Message {
	return l[i].Definitions
}

func (l SymbolDefinitions) Key(i int) string {
	return l[i].Name
}
