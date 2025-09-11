package lang

import (
	"codebase-indexer/pkg/codegraph/types"
	"errors"
	"fmt"
	"path/filepath"

	//sitterkotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	//sittercsharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"

	sittercpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	sitterjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
	sitterjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"

	//sitterphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
	sitterpython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	//sitterruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	//sitterrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	//sitterscala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	sittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Language represents a programming language.
type Language string

const (
	Java       Language = "java"
	Python     Language = "python"
	Go         Language = "go"
	JavaScript Language = "javascript"
	TypeScript Language = "typescript"
	Rust       Language = "rust"
	C          Language = "c"
	CPP        Language = "cpp"
	CSharp     Language = "csharp"
	Ruby       Language = "ruby"
	PHP        Language = "php"
	Kotlin     Language = "kotlin"
	Scala      Language = "scala"
)

// TreeSitterParser holds the configuration for a language
type TreeSitterParser struct {
	Language       Language
	SitterLanguage func() *sitter.Language
	SupportedExts  []string
}

// treeSitterParsers 定义了所有支持的语言配置
var treeSitterParsers = []*TreeSitterParser{
	{
		Language: Go,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittergo.Language())
		},
		SupportedExts: []string{".go"},
	},
	{
		Language: Java,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterjava.Language())
		},
		SupportedExts: []string{".java"},
	},
	{
		Language: Python,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterpython.Language())
		},
		SupportedExts: []string{".py"},
	},
	{
		Language: JavaScript,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sitterjavascript.Language())
		},
		SupportedExts: []string{".js", ".jsx", ".vue", ".Vue"},
	},
	{
		Language: TypeScript,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittertypescript.LanguageTypescript())
		},
		SupportedExts: []string{".ts", ".tsx"},
	},
	//{
	//	Language: Rust,
	//	SitterLanguage: func() *sitter.Language {
	//		return sitter.NewLanguage(sitterrust.Language())
	//	},
	//	SupportedExts: []string{".rs"},
	//},
	{
		Language: CPP,
		SitterLanguage: func() *sitter.Language {
			return sitter.NewLanguage(sittercpp.Language())
		},
		SupportedExts: []string{".cpp", ".cc", ".cxx", ".hpp", ".h", ".c"},
	},
	//{
	//	Language: CSharp,
	//	SitterLanguage: func() *sitter.Language {
	//		return sitter.NewLanguage(sittercsharp.Language())
	//	},
	//	SupportedExts: []string{".cs"},
	//},
	//{
	//	Language: Ruby,
	//	SitterLanguage: func() *sitter.Language {
	//		return sitter.NewLanguage(sitterruby.Language())
	//	},
	//	SupportedExts: []string{".rb"},
	//},
	//{
	//	Language: PHP,
	//	SitterLanguage: func() *sitter.Language {
	//		return sitter.NewLanguage(sitterphp.LanguagePHP())
	//	},
	//	SupportedExts: []string{".php", ".phtml"},
	//},
	//{
	//	Language: Kotlin,
	//	SitterLanguage: func() *sitter.Language {
	//		return sitter.NewLanguage(sitterkotlin.Language())
	//	},
	//	SupportedExts: []string{".kt", ".kts"},
	//},
	//{
	//	Language: Scala,
	//	SitterLanguage: func() *sitter.Language {
	//		return sitter.NewLanguage(sitterscala.Language())
	//	},
	//	SupportedExts: []string{".scala"},
	//},
}

// GetTreeSitterParsers 获取所有语言配置
func GetTreeSitterParsers() []*TreeSitterParser {
	return treeSitterParsers
}

// getSitterParserByExt 根据文件扩展名获取语言配置
func getSitterParserByExt(ext string) *TreeSitterParser {
	for _, tp := range treeSitterParsers {
		for _, supportedExt := range tp.SupportedExts {
			if supportedExt == ext {
				return tp
			}
		}
	}
	return nil
}

func InferLanguage(path string) (Language, error) {
	ext := filepath.Ext(path)
	if ext == types.EmptyString {
		return types.EmptyString, ErrFileExtNotFound
	}
	langConf := getSitterParserByExt(ext)
	if langConf == nil {
		return types.EmptyString, ErrLanguageParserNotFound
	}
	return langConf.Language, nil
}

func GetSitterParserByFilePath(path string) (*TreeSitterParser, error) {
	ext := filepath.Ext(path)
	if ext == types.EmptyString {
		return nil, ErrFileExtNotFound
	}
	langConf := getSitterParserByExt(ext)
	if langConf == nil {
		return nil, ErrLanguageParserNotFound
	}
	return langConf, nil
}

func IsUnSupportedFileError(err error) bool {
	return errors.Is(err, ErrFileExtNotFound) ||
		errors.Is(err, ErrLanguageParserNotFound) ||
		errors.Is(err, ErrUnSupportedLanguage)
}

func GetSitterParserByLanguage(language Language) (*TreeSitterParser, error) {
	if language == types.EmptyString {
		return nil, fmt.Errorf("get tree_sitter parser by language: language is empty")
	}
	for _, parser := range treeSitterParsers {
		if parser.Language == language {
			return parser, nil
		}
	}
	return nil, ErrLanguageParserNotFound
}

func ToLanguage(language string) (Language, error) {
	if language == types.EmptyString {
		return types.EmptyString, fmt.Errorf("language is empty")
	}
	for _, parser := range treeSitterParsers {
		if string(parser.Language) == language {
			return parser.Language, nil
		}
	}
	return types.EmptyString, ErrUnSupportedLanguage
}
func GetAllSupportedLanguages() []Language {
	var languages []Language
	for _, parser := range treeSitterParsers {
		languages = append(languages, parser.Language)
	}
	return languages
}