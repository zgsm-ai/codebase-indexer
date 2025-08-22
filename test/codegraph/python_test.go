package codegraph

import (
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const PythonProjectRootDir = "/tmp/projects/python/fastapi"

func TestParsePythonProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "fastapi",
			Path:    filepath.Join(PythonProjectRootDir),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			start := time.Now()
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
			duration := time.Since(start)
			fmt.Printf("测试用例 '%s' 执行时间: %v, 文件个数: %d\n", tc.Name, duration, len(fileElements))
			assert.NoError(t, err)
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, len(fileElements) > 0)
			for _, f := range fileElements {
				for _, e := range f.Elements {
					if !resolver.IsValidElement(e) {
						fmt.Printf("Type: %s Name: %s Path: %s\n",
							e.GetType(), e.GetName(), e.GetPath())
						fmt.Printf("  Range: %v Scope: %s\n",
							e.GetRange(), e.GetScope())

					}
					//assert.True(t, resolver.IsValidElement(e))
				}
				for _, e := range f.Imports {
					if !resolver.IsValidElement(e) {
						fmt.Printf("Type: %s Name: %s Path: %s\n",
							e.GetType(), e.GetName(), e.GetPath())
						fmt.Printf("  Range: %v Scope: %s\n",
							e.GetRange(), e.GetScope())
					}
				}
			}
			fmt.Println("-------------------------------------------------")
		})
	}
}

func TestQueryPython(t *testing.T) {
	// 设置测试环境
	// 设置测试环境
	env, err := setupTestEnvironment()
	if err != nil {
		t.Logf("setupTestEnvironment error: %v", err)
		return
	}
	defer teardownTestEnvironment(t, env)

	// 使用codebase-indexer-main项目作为测试数据
	workspacePath := "e:\\tmp\\projects\\python\\fastapi"

	if err = initWorkspaceModel(env, workspacePath); err != nil {
		t.Logf("initWorkspaceModel error: %v", err)
		return
	}

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".py"},
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
		ExpectedCount   int                // 期望的定义数量
		ExpectedNames   []string           // 期望找到的定义名称
		ShouldFindDef   bool               // 是否应该找到定义
		wantDefinitions []types.Definition // 期望的详细定义结果
		wantErr         error              // 期望的错误
	}

	// 使用您提供的10个解析出来的元素作为测试用例
	testCases := []QueryTestCase{
		{
			Name:          "查询get_websocket_app函数调用",
			ElementName:   "get_websocket_app",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\fastapi\\routing.py",
			StartLine:     415,
			EndLine:       419,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "get_websocket_app", Path: "routing.py", Range: []int32{360, 0, 385, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询get_authorization_scheme_param函数调用",
			ElementName:   "get_authorization_scheme_param",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\fastapi\\security\\oauth2.py",
			StartLine:     490,
			EndLine:       490,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "get_authorization_scheme_param", Path: "utils.py", Range: []int32{3, 0, 9, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询_get_flat_fields_from_params函数调用",
			ElementName:   "_get_flat_fields_from_params",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\fastapi\\openapi\\utils.py",
			StartLine:     107,
			EndLine:       107,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "_get_flat_fields_from_params", Path: "utils.py", Range: []int32{211, 0, 211, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询add_task函数调用",
			ElementName:   "add_task",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\fastapi\\background.py",
			StartLine:     59,
			EndLine:       59,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "add_task", Path: "background.py", Range: []int32{8, 0, 8, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询get_lang_paths函数调用",
			ElementName:   "get_lang_paths",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\scripts\\docs.py",
			StartLine:     71,
			EndLine:       71,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "get_lang_paths", Path: "docs.py", Range: []int32{57, 0, 57, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询get_graphql_translation_discussions函数调用",
			ElementName:   "get_graphql_translation_discussions",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\scripts\\notify_translations.py",
			StartLine:     350,
			EndLine:       350,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "get_graphql_translation_discussions", Path: "notify_translations.py", Range: []int32{238, 0, 238, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询include_router方法调用",
			ElementName:   "include_router",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\bigger_applications\\app\\main.py",
			StartLine:     12,
			EndLine:       18,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "include_router", Path: "applications.py", Range: []int32{1254, 0, 1254, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询Cookie方法调用",
			ElementName:   "Cookie",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\websockets\\tutorial002_an.py",
			StartLine:     69,
			EndLine:       69,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "Cookie", Path: "param_functions.py", Range: []int32{958, 0, 958, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询startswith方法调用",
			ElementName:   "startswith",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\scripts\\notify_translations.py",
			StartLine:     342,
			EndLine:       342,
			ElementType:   "call.function",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询mkdir方法调用",
			ElementName:   "mkdir",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\scripts\\translate.py",
			StartLine:     105,
			EndLine:       105,
			ElementType:   "call.function",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询Item引用",
			ElementName:   "Item",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\body_multiple_params\\tutorial001_an_py310.py",
			StartLine:     105,
			EndLine:       105,
			ElementType:   "reference",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询Settings引用",
			ElementName:   "Settings",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\conditional_openapi\\tutorial001.py",
			StartLine:     9,
			EndLine:       9,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "Settings", Path: "tutorial001.py", Range: []int32{4, 0, 4, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询GzipRequest引用",
			ElementName:   "GzipRequest",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\custom_request_and_route\\tutorial001.py",
			StartLine:     23,
			EndLine:       23,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "GzipRequest", Path: "tutorial001.py", Range: []int32{7, 0, 7, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询Annotated引用",
			ElementName:   "Annotated",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\body_multiple_params\\tutorial004_an_py39.py",
			StartLine:     26,
			EndLine:       26,
			ElementType:   "reference",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询SecurityRequirement引用",
			ElementName:   "SecurityRequirement",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\fastapi\\dependencies\\utils.py",
			StartLine:     159,
			EndLine:       161,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "SecurityRequirement", Path: "models.py", Range: []int32{8, 0, 8, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询LinkData引用",
			ElementName:   "LinkData",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\scripts\\deploy_docs_status.py",
			StartLine:     93,
			EndLine:       93,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "LinkData", Path: "deploy_docs_status.py", Range: []int32{17, 0, 17, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询FastAPI引用",
			ElementName:   "FastAPI",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\docs_src\\websockets\\tutorial002_an_py39.py",
			StartLine:     14,
			EndLine:       14,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "FastAPI", Path: "applications.py", Range: []int32{47, 0, 47, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询TypeVar引用",
			ElementName:   "TypeVar",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\fastapi\\concurrency.py",
			StartLine:     12,
			EndLine:       12,
			ElementType:   "reference",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询Author引用",
			ElementName:   "Author",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\scripts\\contributors.py",
			StartLine:     74,
			EndLine:       74,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "Author", Path: "contributors.py", Range: []int32{58, 0, 58, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询APIRouter引用",
			ElementName:   "APIRouter",
			FilePath:      "e:\\tmp\\projects\\python\\fastapi\\tests\\test_custom_middleware_exception.py",
			StartLine:     10,
			EndLine:       10,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "APIRouter", Path: "routing.py", Range: []int32{595, 0, 595, 0}},
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
				Workspace: workspacePath,
				StartLine: tc.StartLine,
				EndLine:   tc.EndLine,
				FilePath:  tc.FilePath,
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

func TestFindDefinitionsForAllElementsPython(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用项目自身的代码作为测试数据
	workspacePath, err := filepath.Abs(PythonProjectRootDir) // 指向项目根目录
	assert.NoError(t, err)

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器并索引工作空间
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "test", ".git"),
		IncludeExts: []string{".py"}, // 只索引python文件
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

	// 计算统计数据
	successRate := 0.0
	if testedElements > 0 {
		successRate = float64(foundDefinitions) / float64(testedElements) * 100
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
	// 断言检查：确保基本的成功率
	assert.GreaterOrEqual(t, successRate, 20.0,
		"元素定义查找的成功率应该至少达到20%")

	// 确保没有过多的查询错误
	errorRate := float64(queryErrors) / float64(testedElements) * 100
	assert.LessOrEqual(t, errorRate, 10.0,
		"查询错误率不应超过10%")

	// 确保至少测试了一定数量的元素
	assert.GreaterOrEqual(t, testedElements, 50,
		"应该至少测试50个元素")
}

func TestIterPythonProjectKeys(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用fastapi项目作为测试数据
	workspacePath := "/tmp/projects/python/fastapi"

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "git"),
		IncludeExts: []string{".py"}, // 只索引Python文件
	})

	// 先索引工作空间，确保有数据可查询
	fmt.Println("开始索引fastapi项目...")
	indexResult, err := indexer.IndexWorkspace(context.Background(), workspacePath)
	assert.NoError(t, err)
	fmt.Printf("工作空间索引完成，索引了 %d 个文件\n", indexResult.TotalFiles)
	fmt.Printf("失败的文件数: %d\n", indexResult.TotalFailedFiles)

	// 获取项目列表和实际的UUID
	projects := env.workspaceReader.FindProjects(context.Background(), workspacePath, true, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "git"),
		IncludeExts: []string{".py"},
	})

	fmt.Printf("\n发现的项目数量: %d\n", len(projects))
	for i, project := range projects {
		fmt.Printf("项目 %d: 名称=%s, 路径=%s, UUID=%s\n", i+1, project.Name, project.Path, project.Uuid)

		// 检查每个项目的索引数据
		dataSize := env.storage.Size(context.Background(), project.Uuid, "")
		fmt.Printf("  索引数据大小: %d\n", dataSize)

		if dataSize > 0 {
			fmt.Printf("  开始迭代项目 %s 的索引key...\n", project.Uuid)
			iter := env.storage.Iter(context.Background(), project.Uuid)
			defer iter.Close()

			keyCount := 0
			symbolKeys := 0
			pathKeys := 0

			fmt.Printf("  前20个索引Key:\n")
			for iter.Next() && keyCount < 50 {
				key := iter.Key()
				keyCount++

				if keyCount <= 20 {
					fmt.Printf("    %d. %s\n", keyCount, key)
				}

				// 统计key类型
				if strings.HasPrefix(key, "@sym:") {
					symbolKeys++
					if symbolKeys <= 5 { // 显示前5个符号key的详细信息
						fmt.Printf("      -> 符号Key: %s\n", key)
					}
				} else if strings.HasPrefix(key, "@path:") {
					pathKeys++

					// 检查特定路径的内容
					if strings.Contains(key, "fastapi/routing.py") {
						fmt.Printf("      -> 找到目标文件路径Key: %s\n", key)

						// 尝试获取这个路径的数据
						if data, err := env.storage.Get(context.Background(), project.Uuid, store.ElementPathKey{
							Language: "python",
							Path:     "/tmp/projects/python/fastapi/fastapi/routing.py",
						}); err == nil {
							fmt.Printf("         文件数据大小: %d 字节\n", len(data))

							// 尝试解析文件元素表
							var fileTable codegraphpb.FileElementTable
							if err := proto.Unmarshal(data, &fileTable); err == nil {
								fmt.Printf("         元素数量: %d\n", len(fileTable.Elements))
								fmt.Printf("         导入数量: %d\n", len(fileTable.Imports))

								// 显示前几个元素
								for j, element := range fileTable.Elements {
									if j < 3 {
										fmt.Printf("           元素%d: 名称=%s, 类型=%s, 是否定义=%t\n",
											j+1, element.Name, element.GetElementType(), element.IsDefinition)
									}
								}
							} else {
								fmt.Printf("         解析文件元素表失败: %v\n", err)
							}
						} else {
							fmt.Printf("         获取文件数据失败: %v\n", err)
						}
					}
				}
			}

			fmt.Printf("  总Key数量: %d, 符号Key: %d, 路径Key: %d\n", keyCount, symbolKeys, pathKeys)
			fmt.Println("  " + strings.Repeat("-", 60))
		}
	}

	// 测试 QueryDefinitions 使用正确的项目信息
	if len(projects) > 0 {
		mainProject := projects[0]
		fmt.Printf("\n使用主项目进行查询测试: %s (UUID: %s)\n", mainProject.Name, mainProject.Uuid)

		// 测试一个简单的查询
		testFilePath := "/tmp/projects/python/fastapi/fastapi/routing.py"

		// 验证文件是否存在并且在项目范围内
		if strings.HasPrefix(testFilePath, mainProject.Path) {
			fmt.Printf("测试文件 %s 属于项目 %s\n", testFilePath, mainProject.Path)

			// 先检查文件是否在索引中
			exists, err := env.storage.Exists(context.Background(), mainProject.Uuid, store.ElementPathKey{
				Language: "python",
				Path:     testFilePath,
			})
			fmt.Printf("文件是否在索引中: %t, 错误: %v\n", exists, err)

			// 尝试查询定义
			definitions, err := indexer.QueryDefinitions(context.Background(), &types.QueryDefinitionOptions{
				Workspace: workspacePath,
				StartLine: 415,
				EndLine:   419,
				FilePath:  testFilePath,
			})

			if err != nil {
				fmt.Printf("查询错误: %v\n", err)
			} else {
				fmt.Printf("查询成功，找到 %d 个定义\n", len(definitions))
				for i, def := range definitions {
					fmt.Printf("  定义%d: 名称=%s, 类型=%s, 路径=%s, 范围=%v\n",
						i+1, def.Name, def.Type, def.Path, def.Range)
				}
			}
		} else {
			fmt.Printf("警告: 测试文件 %s 不在项目 %s 范围内\n", testFilePath, mainProject.Path)
		}
	}
}
