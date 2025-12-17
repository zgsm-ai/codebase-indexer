package codegraph

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TsProjectRootDir = "/tmp/projects/typescript"

func TestParseTsProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "vue-next",
			Path:    filepath.Join(TsProjectRootDir, "vue-next"),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, len(fileElements) > 0)
			for _, f := range fileElements {
				for _, e := range f.Elements {
					if !resolver.IsValidElement(e) {
						t.Logf("error element: %s %s %v", e.GetName(), e.GetPath(), e.GetRange())
					}
				}
			}
		})
	}
}

func TestFindDefinitionsForAllElementsTypeScript(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用项目自身的代码作为测试数据
	workspacePath, err := filepath.Abs(TsProjectRootDir) // 指向项目根目录
	assert.NoError(t, err)

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器并索引工作空间
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "test", ".git"),
		IncludeExts: []string{".ts", ".tsx"},
	})

	project := NewTestProject(workspacePath, env.logger)
	fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
	assert.NoError(t, err)

	// 先索引所有文件到数据库
	_, err = indexer.IndexWorkspace(context.Background(), workspacePath)
	assert.NoError(t, err)

	// 统计变量
	var (
		totalElements       = 0
		testedElements      = 0
		foundDefinitions    = 0
		notFoundDefinitions = 0
		queryErrors         = 0
		skippedElements     = 0
		skippedVariables    = 0
	)

	// 定义需要跳过测试的元素类型（基于types.ElementType的实际值）
	skipElementTypes := map[string]bool{
		"import":         true, // 导入语句通常不需要查找定义
		"import.name":    true, // 导入名称
		"import.alias":   true, // 导入别名
		"import.path":    true, // 导入路径
		"import.source":  true, // 导入源
		"package":        true, // 包声明
		"package.name":   true, // 包名
		"namespace":      true, // 命名空间
		"namespace.name": true, // 命名空间名称
		"undefined":      true, // 未定义类型
	}

	// 详细的元素类型统计
	elementTypeStats := make(map[string]int)
	elementTypeSuccessStats := make(map[string]int)

	// 遍历每个文件的元素
	for _, fileElement := range fileElements {
		for _, element := range fileElement.Elements {
			elementType := string(element.GetType())
			totalElements++
			elementTypeStats[elementType]++

			// 跳过某些类型的元素
			if skipElementTypes[elementType] {
				skippedElements++
				continue
			}

			elementName := element.GetName()
			elementRange := element.GetRange()

			// 如果元素名称为空或者范围无效，跳过
			if elementName == "" || len(elementRange) != 4 {
				skippedElements++
				continue
			}
			if elementType == "variable" && element.GetScope() == types.ScopeFunction {
				skippedVariables++
				continue
			}
			testedElements++

			// 尝试查找该元素的定义
			definitions, err := indexer.QueryDefinitions(context.Background(), &types.QueryDefinitionOptions{
				Workspace: workspacePath,
				StartLine: int(elementRange[0]) + 1,
				EndLine:   int(elementRange[2]) + 1,
				FilePath:  fileElement.Path,
			})

			if err != nil {
				queryErrors++
				continue
			}

			if len(definitions) > 0 {
				foundDefinitions++
				elementTypeSuccessStats[elementType]++
			} else {
				notFoundDefinitions++
			}
		}
	}

	// 输出各类型元素的统计信息
	fmt.Println("\n📈 各类型元素统计:")
	fmt.Println(strings.Repeat("-", 60))
	for elementType, count := range elementTypeStats {
		successCount := elementTypeSuccessStats[elementType]
		rate := 0.0
		if count > 0 {
			rate = float64(successCount) / float64(count) * 100
		}
		if elementType == "variable" {
			fmt.Println("跳过的变量数量", skippedVariables)
			rate = float64(successCount) / float64(count-skippedVariables) * 100
		}
		fmt.Printf("%-15s: %4d 个 (成功找到定义: %4d, 成功率: %5.1f%%)\n",
			elementType, count, successCount, rate)
	}

}

func TestQueryTypeScript(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	if err != nil {
		t.Logf("setupTestEnvironment error: %v", err)
		return
	}
	defer teardownTestEnvironment(t, env)

	workspacePath := filepath.Join(TsProjectRootDir, "vue-next")
	// 初始化工作空间数据库记录
	if err = initWorkspaceModel(env, workspacePath); err != nil {
		t.Logf("initWorkspaceModel error: %v", err)
		return
	}

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".ts", ".tsx"},
	})

	// 先清除所有已有的索引，确保强制重新索引
	if err = indexer.RemoveAllIndexes(context.Background(), workspacePath); err != nil {
		t.Logf("remove indexes error: %v", err)
		return
	}

	// 先索引工作空间，确保有数据可查询
	if _, err = indexer.IndexWorkspace(context.Background(), workspacePath); err != nil {
		t.Logf("index workspace error: %v", err)
		return
	}

	// 定义查询测试用例结构
	type QueryTestCase struct {
		Name            string             // 测试用例名称
		ElementName     string             // 元素名称
		FilePath        string             // 查询的文件路径
		StartLine       int                // 开始行号
		EndLine         int                // 结束行号
		ElementType     string             // 元素类型
		CodeSnippet     []byte             // 代码片段内容
		ExpectedCount   int                // 期望的定义数量
		ExpectedNames   []string           // 期望找到的定义名称
		ShouldFindDef   bool               // 是否应该找到定义
		wantDefinitions []types.Definition // 期望的详细定义结果
		wantErr         error              // 期望的错误
	}

	testCases := []QueryTestCase{
		{
			Name:          "查询compileCode函数调用",
			ElementName:   "compileCode",
			FilePath:      filepath.Join(workspacePath, "packages-private", "template-explorer", "src", "index.ts"),
			StartLine:     142,
			EndLine:       142,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "compileCode", Path: "index.ts", Range: []int32{75, 0, 75, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询directive函数调用",
			ElementName:   "directive",
			FilePath:      filepath.Join(workspacePath, "packages-private", "dts-test", "appDirective.test-d.ts"),
			StartLine:     6,
			EndLine:       19,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "directive", Path: "apiCreateApp.ts", Range: []int32{56, 0, 56, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ssrCodegenTransform函数调用",
			ElementName:   "ssrCodegenTransform",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-ssr", "src", "index.ts"),
			StartLine:     89,
			EndLine:       89,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ssrCodegenTransform", Path: "ssrCodegenTransform.ts", Range: []int32{37, 0, 37, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询onError函数调用",
			ElementName:   "onError",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-core", "src", "validateExpression.ts"),
			StartLine:     56,
			EndLine:       63,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "onError", Path: "options.ts", Range: []int32{18, 0, 18, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询processExpression函数调用",
			ElementName:   "processExpression",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-ssr", "src", "ssrCodegenTransform.ts"),
			StartLine:     49,
			EndLine:       49,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "processExpression", Path: "transformExpression.ts", Range: []int32{103, 0, 103, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询createSimpleExpression函数",
			ElementName:   "createSimpleExpression",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-core", "src", "transforms", "vOn.ts"),
			StartLine:     59,
			EndLine:       59,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "createSimpleExpression", Path: "ast.ts", Range: []int32{684, 0, 684, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询isSimpleIdentifier函数",
			ElementName:   "isSimpleIdentifier",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-core", "src", "parser.ts"),
			StartLine:     994,
			EndLine:       994,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "isSimpleIdentifier", Path: "utils.ts", Range: []int32{66, 0, 66, 0}},
			},
			wantErr: nil,
		},
		{
			Name:        "查询isFnExpression函数",
			ElementName: "isFnExpression",
			FilePath:    filepath.Join(workspacePath, "packages", "compiler-core", "src", "transforms", "vOn.ts"),
			StartLine:   85,
			EndLine:     85,
			ElementType: "call.function",
			//CodeSnippet:   []byte(`const isInlineStatement = !(isMemberExp || isFnExpression(exp, context))`), // 添加包含函数调用的代码片段
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "isFnExpression", Path: "utils.ts", Range: []int32{227, 0, 227, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询DebuggerEvent引用",
			ElementName:   "DebuggerEvent",
			FilePath:      filepath.Join(workspacePath, "packages", "runtime-core", "__tests__", "apiLifecycle.spec.ts"),
			StartLine:     341,
			EndLine:       341,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "DebuggerEvent", Path: "effect.ts", Range: []int32{9, 0, 9, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询SFCTemplateBlock引用",
			ElementName:   "SFCTemplateBlock",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-sfc", "src", "parse.ts"),
			StartLine:     75,
			EndLine:       75,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "SFCTemplateBlock", Path: "parse.ts", Range: []int32{44, 0, 44, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询CompilerOptions引用",
			ElementName:   "CompilerOptions",
			FilePath:      filepath.Join(workspacePath, "packages", "runtime-core", "src", "parse.ts"),
			StartLine:     1020,
			EndLine:       1020,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "CompilerOptions", Path: "options.ts", Range: []int32{348, 0, 348, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ReactiveEffect引用",
			ElementName:   "ReactiveEffect",
			FilePath:      filepath.Join(workspacePath, "packages", "reactivity", "src", "effectScope.ts"),
			StartLine:     18,
			EndLine:       18,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ReactiveEffect", Path: "effect.ts", Range: []int32{86, 0, 86, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询SIMPLE_EXPRESSION引用",
			ElementName:   "SIMPLE_EXPRESSION",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-core", "src", "transforms", "vModel.ts"),
			StartLine:     36,
			EndLine:       36,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "SIMPLE_EXPRESSION", Path: "ast.ts", Range: []int32{33, 0, 33, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询CodegenResult引用",
			ElementName:   "CodegenResult",
			FilePath:      filepath.Join(workspacePath, "packages", "compiler-core", "src", "compile.ts"),
			StartLine:     68,
			EndLine:       68,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "CodegenResult", Path: "codegen.ts", Range: []int32{107, 0, 107, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询MockInstance引用",
			ElementName:   "MockInstance",
			FilePath:      filepath.Join(workspacePath, "scripts", "setup-vitest.ts"),
			StartLine:     81,
			EndLine:       81,
			ElementType:   "reference",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询RootHydrateFunction引用",
			ElementName:   "RootHydrateFunction",
			FilePath:      filepath.Join(workspacePath, "packages", "runtime-core", "src", "hydration.ts"),
			StartLine:     119,
			EndLine:       119,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "RootHydrateFunction", Path: "hydration.ts", Range: []int32{46, 0, 46, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询Dep引用",
			ElementName:   "Dep",
			FilePath:      filepath.Join(workspacePath, "packages", "reactivity", "src", "ref.ts"),
			StartLine:     291,
			EndLine:       291,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "Dep", Path: "dep.ts", Range: []int32{66, 0, 66, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询DevtoolsHook引用",
			ElementName:   "DevtoolsHook",
			FilePath:      filepath.Join(workspacePath, "packages", "runtime-core", "src", "devtools.ts"),
			StartLine:     38,
			EndLine:       38,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "DevtoolsHook", Path: "devtools.ts", Range: []int32{23, 0, 23, 0}},
			},
			wantErr: nil,
		},
	}

	// 统计变量
	totalCases := len(testCases)
	correctCases := 0

	// 执行每个测试用例
	for i, tc := range testCases {
		tc := tc // 捕获循环变量
		t.Run(tc.Name, func(t *testing.T) {
			t.Logf("test case %d/%d: %s", i+1, totalCases, tc.Name)
			// 检查文件是否存在
			if _, err := os.Stat(tc.FilePath); os.IsNotExist(err) {
				t.Logf("file not exist: %s", tc.FilePath)
				return
			}

			// 检查行号范围是否有效
			if tc.StartLine < 0 || tc.EndLine < 0 {
				t.Logf("invalid line range: %d-%d", tc.StartLine, tc.EndLine)
				if !tc.ShouldFindDef {
					correctCases++
					t.Logf("expect invalid range, test pass")
				} else {
					t.Logf("expect find definition but range is invalid, test fail")
				}
				return
			}

			// 调用QueryDefinitions接口
			definitions, err := indexer.QueryDefinitions(context.Background(), &types.QueryDefinitionOptions{
				Workspace:   workspacePath,
				StartLine:   tc.StartLine,
				EndLine:     tc.EndLine,
				FilePath:    tc.FilePath,
				CodeSnippet: tc.CodeSnippet, // 添加代码片段参数
			})

			foundDefinitions := len(definitions)

			if err != nil {
				t.Logf("query failed: %v", err)
			} else {
				t.Logf("found %d definitions", foundDefinitions)

				if foundDefinitions > 0 {
					t.Logf("query result detail:")
					for j, def := range definitions {
						t.Logf(
							"  [%d] name: '%s' type: '%s' range: %v path: '%s' fullPath: '%s'", j+1, def.Name, def.Type, def.Range, def.Path, filepath.Dir(def.Path))

						// 如果有期望的定义，进行匹配度分析
						if len(tc.wantDefinitions) > 0 {
							for _, wantDef := range tc.wantDefinitions {
								if def.Name != wantDef.Name {
									t.Logf("name not match: expect '%s' actual '%s'", wantDef.Name, def.Name)
								}
								if def.Name == wantDef.Name {
									nameMatch := "✓"
									lineMatch := "✗"
									pathMatch := "✗"

									if wantDef.Range[0] == def.Range[0] {
										lineMatch = "✓"
									}
									if wantDef.Path == "" || strings.Contains(def.Path, wantDef.Path) {
										pathMatch = "✓"
									}

									t.Logf("match analysis: name %s line %s path %s", nameMatch, lineMatch, pathMatch)
								}
							}
						}
					}
				} else {
					t.Logf("no definition found")
				}

				// 输出查询总结
				t.Logf("query summary: expect find=%v, actual find=%d",
					tc.ShouldFindDef, foundDefinitions)

			}

			// 计算当前用例是否正确
			caseCorrect := false
			if tc.wantErr != nil {
				caseCorrect = err != nil
				if !caseCorrect {
					t.Logf("expect error %v but got nil", tc.wantErr)
				}
			} else if len(tc.wantDefinitions) > 0 {
				if err != nil {
					t.Logf("unexpected error: %v", err)
					caseCorrect = false
				} else {
					allFound := true
					for _, wantDef := range tc.wantDefinitions {
						found := false
						for _, actualDef := range definitions {
							nameMatch := actualDef.Name == wantDef.Name
							lineMatch := wantDef.Range[0] == actualDef.Range[0]
							pathMatch := wantDef.Path == "" || strings.Contains(actualDef.Path, wantDef.Path)

							if nameMatch && pathMatch && lineMatch {
								found = true
								break
							}
						}
						if !found {
							allFound = false
							t.Logf("missing expected definition: name='%s' line='%d' path='%s'",
								wantDef.Name, wantDef.Range[0], wantDef.Path)
						}
					}
					caseCorrect = allFound
				}
			} else {
				should := tc.ShouldFindDef
				actual := foundDefinitions > 0
				caseCorrect = (should == actual)
			}

			if caseCorrect {
				correctCases++
				t.Logf("✓ %s: pass", tc.Name)
			} else {
				t.Logf("✗ %s: fail", tc.Name)
			}
		})
	}

	accuracy := 0.0
	if totalCases > 0 {
		accuracy = float64(correctCases) / float64(totalCases) * 100
	}
	t.Logf("TestQueryTypeScript summary: total=%d, correct=%d, accuracy=%.2f%%", totalCases, correctCases, accuracy)

}
