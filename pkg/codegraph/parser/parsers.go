package parser

import (
	"codebase-indexer/pkg/codegraph/parser/resolver"
	sitterkotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittercsharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	sitterc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	sittercpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	sitterjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
	sitterjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	sitterphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
	sitterpython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	sitterruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	sitterrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	sitterscala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	sittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"path/filepath"
)

// TreeSitterParser holds the configuration for a language
type TreeSitterParser struct {
	Language       resolver.Language
	SitterLanguage func() *sitter.Language
	SupportedExts  []string
}

// treeSitterParsers 定义了所有支持的语言配置
var treeSitterParsers = []*TreeSitterParser{
	{
		Language: resolver.Go,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittergo.Language())
		},
		SupportedExts: []string{".go"},
	},
	{
		Language: resolver.Java,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterjava.Language())
		},
		SupportedExts: []string{".java"},
	},
	{
		Language: resolver.Python,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterpython.Language())
		},
		SupportedExts: []string{".py"},
	},
	{
		Language: resolver.JavaScript,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterjavascript.Language())
		},
		SupportedExts: []string{".js", ".jsx"},
	},
	{
		Language: resolver.TypeScript,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittertypescript.LanguageTypescript())
		},
		SupportedExts: []string{".ts"},
	},
	{
		Language: resolver.TSX,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittertypescript.LanguageTSX())
		},
		SupportedExts: []string{".tsx"},
	},
	{
		Language: resolver.Rust,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterrust.Language())
		},
		SupportedExts: []string{".rs"},
	},
	{
		Language: resolver.C,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterc.Language())
		},
		SupportedExts: []string{".c", ".h"},
	},
	{
		Language: resolver.CPP,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittercpp.Language())
		},
		SupportedExts: []string{".cpp", ".cc", ".cxx", ".hpp"},
	},
	{
		Language: resolver.CSharp,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittercsharp.Language())
		},
		SupportedExts: []string{".cs"},
	},
	{
		Language: resolver.Ruby,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterruby.Language())
		},
		SupportedExts: []string{".rb"},
	},
	{
		Language: resolver.PHP,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterphp.LanguagePHP())
		},
		SupportedExts: []string{".php", ".phtml"},
	},
	{
		Language: resolver.Kotlin,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterkotlin.Language())
		},
		SupportedExts: []string{".kt", ".kts"},
	},
	{
		Language: resolver.Scala,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterscala.Language())
		},
		SupportedExts: []string{".scala"},
	},
}

// GetTreeSitterParsers 获取所有语言配置
func GetTreeSitterParsers() []*TreeSitterParser {
	return treeSitterParsers
}

// getTreeSitterParserByExt 根据文件扩展名获取语言配置
func getTreeSitterParserByExt(ext string) *TreeSitterParser {
	for _, config := range treeSitterParsers {
		for _, supportedExt := range config.SupportedExts {
			if supportedExt == ext {
				return config
			}
		}
	}
	return nil
}

func GetTreeSitterParserByFilePath(path string) (*TreeSitterParser, error) {
	ext := filepath.Ext(path)
	if ext == "" {
		return nil, ErrFileExtNotFound
	}
	langConf := getTreeSitterParserByExt(ext)
	if langConf == nil {
		return nil, ErrLangConfNotFound
	}
	return langConf, nil
}
