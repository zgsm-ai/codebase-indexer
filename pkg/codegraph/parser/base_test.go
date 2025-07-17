package parser

import (
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func initLogger() logger.Logger {
	logger, err := logger.NewLogger(utils.LogsDir, "info")
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logging system: %v\n", err))
	}
	return logger
}

func TestGoBaseParse(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Go, "github.com/hashicorp", []string{"pkg/go-uuid/uuid.go"})
	testCases := []struct {
		name       string
		sourceFile *types.SourceFile
		wantErr    error
	}{
		{
			name: "Go",
			sourceFile: &types.SourceFile{
				Path:    "test.go",
				Content: readFile("testdata/test.go"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			if res != nil {
				if res.Package != nil {
					fmt.Printf("  name: %s\n", res.Package.BaseElement.GetName())
					fmt.Printf("  Content: %s\n", res.Package.BaseElement.GetContent())
					fmt.Printf("  RootIndex: %v\n", res.Package.BaseElement.GetRootIndex())
				}

				if res.Imports != nil {
					fmt.Println("\nImports详情:")
					for i, imp := range res.Imports {
						fmt.Printf("[%d]\n", i)
						if imp != nil {
							fmt.Printf("  name: %s\n", imp.BaseElement.GetName())
							fmt.Printf("  Content: %s\n", imp.BaseElement.GetContent())
							fmt.Printf("  RootIndex: %v\n", imp.BaseElement.GetRootIndex())
							fmt.Printf("  type: %s\n", imp.BaseElement.GetType())
							fmt.Printf("  Alias: %s\n", imp.Alias)
							fmt.Printf("  FilePaths: %v\n", imp.FilePaths)

						}
					}
				}

				if res.Elements != nil {
					fmt.Println("\nElements详情:")
					for i, elem := range res.Elements {
						if elem == nil {
							continue
						}
						fmt.Printf("[%d] %s (Type: %v)\n", i, elem.GetName(), elem.GetType())
						fmt.Printf("  Range: %v\n", elem.GetRange())

						if base, ok := elem.(*resolver.BaseElement); ok {
							fmt.Printf("  Content: %s\n", string(base.Content))
						}

						// 根据元素类型打印详细信息
						switch v := elem.(type) {
						case *resolver.Function:
							fmt.Printf("    详细内容(Function) 名称: %s, 作用域: %s, 返回类型: %s, 参数数量: %d\n",
								v.Declaration.Name, v.Scope, v.ReturnType, len(v.Parameters))
							fmt.Printf("  Parameters: %v\n", v.Parameters)
							fmt.Printf("  ReturnType: %s\n", v.ReturnType)
						case *resolver.Method:
							fmt.Printf("    详细内容(Method) 名称: %s, 拥有者: %s, 作用域: %s, 返回类型: %s, 参数数量: %d\n",
								v.Declaration.Name, v.Owner, v.Scope, v.ReturnType, len(v.Parameters))
							fmt.Printf("  Parameters: %v\n", v.Parameters)
							fmt.Printf("  ReturnType: %s\n", v.ReturnType)
						case *resolver.Call:
							fmt.Printf("    详细内容(Call) 名称: %s, 所有者: %s, 参数数量: %d\n",
								elem.GetName(), v.Owner, len(v.Parameters))
							for _, param := range v.Parameters {
								fmt.Printf("    参数: %s, 类型: %s\n", param.Name, param.Type)
							}
						case *resolver.Package:
							fmt.Printf("    详细内容(Package) 名称: %s\n", elem.GetName())
						case *resolver.Import:
							fmt.Printf("    详细内容(Import) 源: %s, 别名: %s\n", v.Source, v.Alias)
						case *resolver.Class:
							fmt.Printf("    详细内容(Class) 名称: %s, 作用域: %s, 字段数量: %d, 方法数量: %d\n",
								elem.GetName(), v.Scope, len(v.Fields), len(v.Methods))
							for _, field := range v.Fields {
								fmt.Println(field.Modifier, field.Type, field.Name)
							}
						case *resolver.Interface:
							fmt.Printf("    详细内容(Interface) 名称: %s, 作用域: %s, 方法数量: %d\n",
								elem.GetName(), v.Scope, len(v.Methods))
							for _, method := range v.Methods {
								fmt.Printf("    方法: %s, 参数: %v, 返回类型: %s\n",
									method.Name, method.Parameters, method.ReturnType)
							}
						case *resolver.Variable:
							fmt.Printf("    详细内容(Variable) 名称: %s, 类型: %s, 作用域: %s, 范围: %v, 内容: %s\n",
								elem.GetName(), elem.GetType(), v.Scope, elem.GetRange(), string(elem.GetContent()))

						default:
							fmt.Printf("    详细内容(其他类型) 类型: %T\n", elem)
						}
					}
				}
			}
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
			assert.NotNil(t, res.Package)
			assert.NotEmpty(t, res.Imports)
			assert.NotEmpty(t, res.Elements)

		})
	}
}

func TestJavaBaseParse(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.Java, "pkg/codegraph/parser/testdata", []string{"com/example/test/TestClass.java"})
	testCases := []struct {
		name       string
		sourceFile *types.SourceFile
		wantErr    error
	}{
		{
			name: "Java",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestClass.java",
				Content: readFile("testdata/com/example/test/TestClass.java"),
			},
			wantErr: nil,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
			assert.NotNil(t, res.Package)
			// assert.NotEmpty(t, res.Imports)
			// for _, ipt := range res.Imports {
			// 	fmt.Println("import:", ipt.GetName())
			// 	fmt.Println("import file paths:", ipt.FilePaths)
			// }
			fmt.Println("package:", res.Package.GetName())
			// Java 文件未必有 Imports，但一般有 Elements
			assert.NotEmpty(t, res.Elements)
			for _, element := range res.Elements {

				cls, ok := element.(*resolver.Class)
				fmt.Println("--------------------------------")
				if ok {
					fmt.Println(cls.GetName())
					fmt.Println(cls.GetType())
					for _, field := range cls.Fields {
						fmt.Println(field.Modifier, field.Type, field.Name)
					}
					for _, method := range cls.Methods {
						fmt.Println(method.Declaration.Modifier, method.Declaration.ReturnType,
							method.Declaration.Name, method.Declaration.Parameters)
						fmt.Println("owner:", method.Owner)
					}
				}
				variable, ok := element.(*resolver.Variable)
				if ok {
					fmt.Println(variable.GetName())
					fmt.Println(variable.GetType())
				}

			}
		})
	}
}

func readFile(path string) []byte {
	bytes, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return bytes
}

func TestGoBaseParse_MatchesDebug(t *testing.T) {
	// logger := initLogger()
	// parser := NewSourceFileParser(logger)
	// prj := project.NewProjectInfo(lang.Go, "github.com/hashicorp", []string{"pkg/go-uuid/uuid.go"})

	sourceFile := &types.SourceFile{
		Path:    "test.java",
		Content: readFile("testdata/test.java"),
	}

	// 1. 获取语言解析器
	langParser, err := lang.GetSitterParserByFilePath(sourceFile.Path)
	if err != nil {
		t.Fatalf("lang parser error: %v", err)
	}
	sitterParser := sitter.NewParser()
	sitterLanguage := langParser.SitterLanguage()
	if err := sitterParser.SetLanguage(sitterLanguage); err != nil {
		t.Fatalf("set language error: %v", err)
	}
	content := sourceFile.Content
	tree := sitterParser.Parse(content, nil)
	if tree == nil {
		t.Fatalf("parse tree error")
	}
	defer tree.Close()

	queryScm, ok := BaseQueries[langParser.Language]
	if !ok {
		t.Fatalf("query not found")
	}
	// TODO: 巨坑err1，变量遮蔽（shadowing）
	query, err1 := sitter.NewQuery(sitterLanguage, queryScm)
	if err1 != nil {
		t.Fatalf("new query error: %v", err1)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()
	matches := qc.Matches(query, tree.RootNode(), content)

	names := query.CaptureNames()
	fmt.Println("CaptureNames:", names)
	// 打印前15个match的内容
	for i := 0; ; i++ {
		match := matches.Next()
		if match == nil {
			break
		}
		fmt.Printf("Match #%d:\n", i+1)
		for _, cap := range match.Captures {
			// 层级结构，从上到下
			//Capture: name=import, text=import java.util.List;, start=3:0, end=3:22
			//Capture: name=import.name, text=java.util.List, start=3:7, end=3:21
			fmt.Printf("  Capture: name=%s, text=%s, start=%d:%d, end=%d:%d\n",
				query.CaptureNames()[cap.Index],
				cap.Node.Utf8Text(content),
				cap.Node.StartPosition().Row, cap.Node.StartPosition().Column,
				cap.Node.EndPosition().Row, cap.Node.EndPosition().Column,
			)
		}
	}
}
