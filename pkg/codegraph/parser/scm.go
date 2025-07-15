package parser

import (
	"codebase-indexer/pkg/codegraph/lang"
	"embed"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"path/filepath"
)

//go:embed queries/*/*.scm
var scmFS embed.FS

const queryDir = "queries"
const defSubdir = "def"
const baseSubDir = "base"
const queryExt = ".scm"

var DefinitionQueries = make(map[lang.Language]string)
var BaseQueries = make(map[lang.Language]string)

func init() {
	if err := loadScm(); err != nil {
		panic(fmt.Errorf("tree_sitter parser load scm queries err:%v", err))
	}
}

func loadScm() error {
	configs := lang.GetTreeSitterParsers()
	for _, l := range configs {
		// 校验query
		langParser := sitter.NewParser()
		sitterLang := l.SitterLanguage()
		err := langParser.SetLanguage(sitterLang)
		if err != nil {
			return fmt.Errorf("failed to init language parser %s: %w", l.Language, err)
		}

		baseQueryContent, err := loadLanguageScm(l, baseSubDir, sitterLang)
		if err != nil {
			return err
		}

		defQueryContent, err := loadLanguageScm(l, defSubdir, sitterLang)
		if err != nil {
			return err
		}

		langParser.Close()
		BaseQueries[l.Language] = string(baseQueryContent)
		DefinitionQueries[l.Language] = string(defQueryContent)
	}
	return nil
}

func loadLanguageScm(l *lang.TreeSitterParser, scmDir string, sitterLang *sitter.Language) ([]byte, error) {
	var err error
	baseQuery := makeQueryPath(l.Language, scmDir)
	baseQueryContent, err := scmFS.ReadFile(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to read base query file %s for %s: %w", baseQuery, l.Language, err)
	}
	query, queryError := sitter.NewQuery(sitterLang, string(baseQueryContent))
	if queryError != nil && lang.IsRealQueryErr(queryError) {
		return nil, fmt.Errorf("failed to parse base query file %s: %w", baseQuery, queryError)
	}
	query.Close()
	return baseQueryContent, nil
}

func makeQueryPath(lang lang.Language, subdir string) string {
	return filepath.ToSlash(filepath.Join(queryDir, subdir, string(lang)+queryExt))
}
