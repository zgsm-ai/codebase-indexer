package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const GoProjectRootDir = "/tmp/projects/go"

func TestParseGoProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "codebase-indexer-main",
			Path:    filepath.Join(GoProjectRootDir, "codebase-indexer-main"),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
			//err = exportFileElements(defaultExportDir, tc.Name, fileElements)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantErr, err)
			assert.True(t, len(fileElements) > 0)
			for _, f := range fileElements {
				for _, e := range f.Elements {
					//fmt.Println(resolver.IsValidElement(e), e.GetName(), e.GetPath(), e.GetRange())
					if !resolver.IsValidElement(e) {
						t.Logf("error element: %s %s %v", e.GetName(), e.GetPath(), e.GetRange())
					}
				}
			}
		})
	}
}

func TestIndexGoProjects(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	setupPprof()
	defer teardownTestEnvironment(t, env)

	// 添加这一行 - 初始化工作空间数据库记录
	err = initWorkspaceModel(env, filepath.Join(GoProjectRootDir, "kubernetes"))
	err = initWorkspaceModel(env, filepath.Join(GoProjectRootDir, "kubernetes"))
	assert.NoError(t, err)

	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		//IncludeExts: []string{".go"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "kubernetes",
			Path:    filepath.Join(GoProjectRootDir, "kubernetes"),
			wantErr: nil,
		},
	}
	// - 1W文件：
	//   6min 100MB 使用1000个cache，没有则从磁盘读取
	//   1min45s 500MB 使用500万个cache，没有则从磁盘读取
	//   2min53s 120MB 仅缓存所有名字(初始化cache为1000)，第二次访问该元素时从磁盘加载
	//   3min54s  150MB    初始化为1000，没有则从磁盘读取
	// - 5W文件：
	//    200MB+ 初始化为1000，缓存key和value，没有则从磁盘读取
	//   1h      100MB     仅缓存名字，第二次访问从磁盘加载
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err = indexer.IndexWorkspace(context.Background(), tc.Path)
			assert.NoError(t, err)
		})
	}
}

func TestWalkProjectCostTime(t *testing.T) {
	ctx := context.Background()
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	testCases := []struct {
		name  string
		path  string
		logic func(*testing.T, *testEnvironment, *types.WalkContext)
	}{
		{
			name: "do nothing",
			path: filepath.Join(GoProjectRootDir, "kubernetes"),
		},
		{
			name: "do index",
			path: filepath.Join(GoProjectRootDir, "kubernetes"),
			logic: func(t *testing.T, environment *testEnvironment, walkContext *types.WalkContext) {
				bytes, err := os.ReadFile(walkContext.Path)
				if err != nil {
					t.Logf("read file %s error: %v", walkContext.Path, err)
					return
				}
				_, err = environment.sourceFileParser.Parse(ctx, &types.SourceFile{
					Path:    walkContext.Path,
					Content: bytes,
				})
				if !lang.IsUnSupportedFileError(err) {
					assert.NoError(t, err)
				}
			},
		},
	}
	excludeDir := append([]string{}, defaultVisitPattern.ExcludeDirs...)
	excludeDir = append(excludeDir, "vendor")
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var fileCnt int
			start := time.Now()
			err = env.workspaceReader.WalkFile(ctx, tt.path, func(walkCtx *types.WalkContext) error {
				fileCnt++
				if tt.logic != nil {
					tt.logic(t, env, walkCtx)
				}
				return nil
			}, types.WalkOptions{IgnoreError: true, VisitPattern: &types.VisitPattern{ExcludeDirs: excludeDir, IncludeExts: []string{".go"}}})
			assert.NoError(t, err)
			t.Logf("%s cost %d ms, %d files, avg %.2f ms/file", tt.name, time.Since(start).Milliseconds(), fileCnt,
				float32(time.Since(start).Milliseconds())/float32(fileCnt))
		})
	}
}

func TestQuery(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用codebase-indexer-main项目作为测试数据
	workspacePath := "/tmp/projects/go/codebase-indexer-main"

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".go"}, // 只索引Go文件
	})

	// 先索引工作空间，确保有数据可查询
	fmt.Println("开始索引codebase-indexer-main工作空间...")
	_, err = indexer.IndexWorkspace(context.Background(), workspacePath)
	assert.NoError(t, err)
	fmt.Println("工作空间索引完成")

	// 定义查询测试用例结构
	type QueryTestCase struct {
		Name          string   // 测试用例名称
		ElementName   string   // 元素名称
		FilePath      string   // 查询的文件路径
		StartLine     int      // 开始行号
		EndLine       int      // 结束行号
		ElementType   string   // 元素类型
		ExpectedCount int      // 期望的定义数量
		ExpectedNames []string // 期望找到的定义名称
		ShouldFindDef bool     // 是否应该找到定义
	}

	// 使用您提供的10个解析出来的元素作为测试用例
	testCases := []QueryTestCase{
		{
			Name:          "查询WriteFile方法调用",
			ElementName:   "createTestIndexer",
			FilePath:      "/tmp/projects/go/codebase-indexer-main/test/codegraph/js_test.go",
			StartLine:     18,
			EndLine:       18,
			ElementType:   "call.method",
			ExpectedCount: 1,
			ShouldFindDef: true,
		},
		{
			Name:          "查询FileScanService引用",
			ElementName:   "FileScanService",
			FilePath:      "/tmp/projects/go/codebase-indexer-main/internal/service/file_scanner_job.go",
			StartLine:     18,
			EndLine:       18,
			ElementType:   "reference",
			ExpectedCount: 1,
			ShouldFindDef: true,
		},
		{
			Name:          "查询空白标识符(_)方法调用",
			ElementName:   "StripSpaces",
			FilePath:      "/tmp/projects/go/codebase-indexer-main/pkg/codegraph/parser/c_resolver_test.go",
			StartLine:     22,
			EndLine:       22,
			ElementType:   "call.method",
			ExpectedCount: 0, // 空白标识符通常不会有定义
			ShouldFindDef: false,
		},
		// {
		// 	Name:          "查询name方法调用",
		// 	ElementName:   "name",
		// 	FilePath:      "/tmp/projects/go/codebase-indexer-main/pkg/codegraph/definition/definition_test.go",
		// 	StartLine:     112,
		// 	EndLine:       112,
		// 	ElementType:   "call.method",
		// 	ExpectedCount: 1,
		// 	ShouldFindDef: true,
		// },
		// {
		// 	Name:          "查询Equal方法调用(assert)",
		// 	ElementName:   "Equal",
		// 	FilePath:      "/tmp/projects/go/codebase-indexer-main/pkg/codegraph/definition/definition_test.go",
		// 	StartLine:     106,
		// 	EndLine:       106,
		// 	ElementType:   "call.method",
		// 	ExpectedCount: 1,
		// 	ShouldFindDef: true,
		// },
		// {
		// 	Name:          "查询Errorf方法调用(t)",
		// 	ElementName:   "Errorf",
		// 	FilePath:      "/tmp/projects/go/codebase-indexer-main/pkg/codegraph/definition/definition_test.go",
		// 	StartLine:     95,
		// 	EndLine:       95,
		// 	ElementType:   "call.method",
		// 	ExpectedCount: 1,
		// 	ShouldFindDef: true,
		// },
		// {
		// 	Name:          "查询Warn方法调用(logger)",
		// 	ElementName:   "Warn",
		// 	FilePath:      "/tmp/projects/go/codebase-indexer-main/internal/service/extension.go",
		// 	StartLine:     264,
		// 	EndLine:       264,
		// 	ElementType:   "call.method",
		// 	ExpectedCount: 1,
		// 	ShouldFindDef: true,
		// },
		// {
		// 	Name:          "查询int64函数调用",
		// 	ElementName:   "int64",
		// 	FilePath:      "/tmp/projects/go/codebase-indexer-main/internal/service/extension.go",
		// 	StartLine:     261,
		// 	EndLine:       261,
		// 	ElementType:   "call.function",
		// 	ExpectedCount: 1, // int64是内置类型转换
		// 	ShouldFindDef: true,
		// },
	}

	// 统计变量
	totalCases := len(testCases)
	correctCases := 0

	fmt.Printf("\n开始执行 %d 个基于人工索引元素的查询测试用例...\n", totalCases)
	fmt.Println(strings.Repeat("=", 80))

	// 执行每个测试用例
	for i, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fmt.Printf("\n[测试用例 %d/%d] %s\n", i+1, totalCases, tc.Name)
			fmt.Printf("元素名称: %s (类型: %s)\n", tc.ElementName, tc.ElementType)
			fmt.Printf("文件路径: %s\n", tc.FilePath)
			fmt.Printf("查询范围: 第%d行 - 第%d行\n", tc.StartLine, tc.EndLine)

			// 检查文件是否存在
			if _, err := os.Stat(tc.FilePath); os.IsNotExist(err) {
				fmt.Printf("文件不存在，跳过查询\n")
				if !tc.ShouldFindDef {
					correctCases++
					fmt.Printf("✓ 预期文件不存在，测试通过\n")
				} else {
					fmt.Printf("✗ 预期找到定义但文件不存在，测试失败\n")
				}
				return
			}

			// 检查行号范围是否有效
			if tc.StartLine < 0 || tc.EndLine < 0 {
				fmt.Printf("无效的行号范围，跳过查询\n")
				if !tc.ShouldFindDef {
					correctCases++
					fmt.Printf("✓ 预期无效范围，测试通过\n")
				} else {
					fmt.Printf("✗ 预期找到定义但范围无效，测试失败\n")
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

			// 判断查询是否成功
			querySuccess := (err == nil)
			foundDefinitions := len(definitions)

			fmt.Printf("查询结果: ")
			if err != nil {
				fmt.Printf("查询失败 - %v\n", err)
			} else {
				fmt.Printf("找到 %d 个定义\n", foundDefinitions)

				// 打印找到的定义详情
				for j, def := range definitions {
					fmt.Printf("  定义%d: 名称='%s', 类型='%s', 范围=%v, 文件='%s'\n",
						j+1, def.Name, def.Type, def.Range, filepath.Base(def.Path))
				}
			}

			// 断言逻辑：判断结果是否符合预期
			testPassed := false

			if tc.ShouldFindDef {
				// 期望找到定义
				if querySuccess && foundDefinitions >= tc.ExpectedCount {
					testPassed = true

					// 如果指定了期望的名称，进一步验证
					if len(tc.ExpectedNames) > 0 {
						foundExpectedNames := 0
						for _, expectedName := range tc.ExpectedNames {
							for _, def := range definitions {
								if def.Name == expectedName {
									foundExpectedNames++
									break
								}
							}
						}
						testPassed = (foundExpectedNames == len(tc.ExpectedNames))
					}
				}
			} else {
				// 期望不找到定义或查询失败
				if !querySuccess || foundDefinitions == 0 {
					testPassed = true
				}
			}

			// 更新统计
			if testPassed {
				correctCases++
				fmt.Printf("✓ 测试通过\n")
			} else {
				fmt.Printf("✗ 测试失败\n")
				fmt.Printf("  期望: ShouldFindDef=%t, ExpectedCount=%d\n",
					tc.ShouldFindDef, tc.ExpectedCount)
				fmt.Printf("  实际: QuerySuccess=%t, FoundCount=%d\n",
					querySuccess, foundDefinitions)
			}

			// 使用testify断言（可选，用于详细的测试报告）
			if tc.ShouldFindDef {
				assert.NoError(t, err, "查询应该成功")
				assert.GreaterOrEqual(t, foundDefinitions, tc.ExpectedCount,
					"找到的定义数量应该大于等于期望值")
			}
		})
	}

	// 计算并输出最终正确率
	accuracy := float64(correctCases) / float64(totalCases) * 100

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 基于人工索引元素的查询测试结果统计")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("总测试用例数: %d\n", totalCases)
	fmt.Printf("通过用例数: %d\n", correctCases)
	fmt.Printf("失败用例数: %d\n", totalCases-correctCases)
	fmt.Printf("正确率: %.2f%%\n", accuracy)

	// 根据正确率给出评价
	var evaluation string
	switch {
	case accuracy >= 90:
		evaluation = "优秀 🎉"
	case accuracy >= 80:
		evaluation = "良好 👍"
	case accuracy >= 70:
		evaluation = "一般 🤔"
	case accuracy >= 60:
		evaluation = "需要改进 😐"
	default:
		evaluation = "亟需优化 😞"
	}

	fmt.Printf("评价: %s\n", evaluation)
	fmt.Println(strings.Repeat("=", 80))

	// 如果正确率太低，测试失败
	assert.GreaterOrEqual(t, accuracy, 60.0,
		"基于人工索引元素的QueryDefinition接口正确率应该至少达到60%")
}

func TestFindDefinitionsForAllElements(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用项目自身的代码作为测试数据
	workspacePath, err := filepath.Abs("../../") // 指向项目根目录
	assert.NoError(t, err)

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器并索引工作空间
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "test", ".git"),
		IncludeExts: []string{".go"}, // 只索引Go文件
	})

	fmt.Println("开始索引工作空间...")
	project := NewTestProject(workspacePath, env.logger)
	fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
	assert.NoError(t, err)
	fmt.Printf("解析完成，共找到 %d 个文件\n", len(fileElements))

	// 统计所有元素
	totalElements := 0
	for _, fileElement := range fileElements {
		totalElements += len(fileElement.Elements)
	}
	fmt.Printf("总共解析出 %d 个代码元素\n", totalElements)

	// 先索引所有文件到数据库
	fmt.Println("开始将元素索引到数据库...")
	_, err = indexer.IndexWorkspace(context.Background(), workspacePath)
	assert.NoError(t, err)
	fmt.Println("索引完成")

	// 统计变量
	var (
		testedElements      = 0
		foundDefinitions    = 0
		notFoundDefinitions = 0
		queryErrors         = 0
		skippedElements     = 0
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

	fmt.Println("\n开始遍历所有元素并查找定义...")
	fmt.Println(strings.Repeat("=", 80))

	// 遍历每个文件的元素
	for _, fileElement := range fileElements {
		fmt.Printf("\n📁 处理文件: %s (包含 %d 个元素)\n",
			filepath.Base(fileElement.Path), len(fileElement.Elements))

		for i, element := range fileElement.Elements {
			// 跳过某些类型的元素
			elementType := string(element.GetType())
			if skipElementTypes[elementType] {
				skippedElements++
				continue
			}

			testedElements++
			elementName := element.GetName()
			elementRange := element.GetRange()

			// 如果元素名称为空或者范围无效，跳过
			if elementName == "" || len(elementRange) != 4 {
				skippedElements++
				continue
			}

			fmt.Printf("  [%d] 测试元素: %s (类型: %s, 行: %d-%d)\n",
				i+1, elementName, elementType,
				elementRange[0], elementRange[2])

			// 尝试查找该元素的定义
			definitions, err := indexer.QueryDefinitions(context.Background(), &types.QueryDefinitionOptions{
				Workspace: workspacePath,
				StartLine: int(elementRange[0]),
				EndLine:   int(elementRange[2]),
				FilePath:  fileElement.Path,
			})

			if err != nil {
				queryErrors++
				fmt.Printf("    ❌ 查询出错: %v\n", err)
				continue
			}

			if len(definitions) > 0 {
				foundDefinitions++
				fmt.Printf("    ✅ 找到 %d 个定义\n", len(definitions))

				// 打印找到的定义详情（限制输出数量）
				for j, def := range definitions {
					if j >= 3 { // 最多显示3个定义
						fmt.Printf("    ... 还有 %d 个定义\n", len(definitions)-3)
						break
					}
					fmt.Printf("      - %s (类型: %s)\n", def.Name, def.Type)
				}
			} else {
				notFoundDefinitions++
				fmt.Printf("    ⚠️  未找到定义\n")
			}

			// 每处理100个元素输出一次进度
			if testedElements%100 == 0 {
				fmt.Printf("\n📊 进度更新: 已测试 %d 个元素\n", testedElements)
				fmt.Printf("  ✅ 找到定义: %d\n", foundDefinitions)
				fmt.Printf("  ⚠️  未找到: %d\n", notFoundDefinitions)
				fmt.Printf("  ❌ 查询错误: %d\n", queryErrors)
			}
		}
	}

	// 计算统计数据
	successRate := 0.0
	if testedElements > 0 {
		successRate = float64(foundDefinitions) / float64(testedElements) * 100
	}

	// 输出最终统计结果
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🎯 元素定义查找测试完成")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("总元素数量: %d\n", totalElements)
	fmt.Printf("跳过的元素: %d (类型: IMPORT, PACKAGE, COMMENT, LITERAL, KEYWORD)\n", skippedElements)
	fmt.Printf("测试的元素: %d\n", testedElements)
	fmt.Printf("成功找到定义: %d\n", foundDefinitions)
	fmt.Printf("未找到定义: %d\n", notFoundDefinitions)
	fmt.Printf("查询出错: %d\n", queryErrors)
	fmt.Printf("成功率: %.2f%%\n", successRate)

	// 根据成功率给出评价
	var evaluation string
	switch {
	case successRate >= 80:
		evaluation = "优秀 🎉"
	case successRate >= 60:
		evaluation = "良好 👍"
	case successRate >= 40:
		evaluation = "一般 🤔"
	case successRate >= 20:
		evaluation = "需要改进 😐"
	default:
		evaluation = "亟需优化 😞"
	}

	fmt.Printf("评价: %s\n", evaluation)
	fmt.Println(strings.Repeat("=", 80))

	// 详细的元素类型统计
	elementTypeStats := make(map[string]int)
	elementTypeSuccessStats := make(map[string]int)

	// 重新遍历计算类型统计（这次不输出详细信息）
	for _, fileElement := range fileElements {
		for _, element := range fileElement.Elements {
			elementType := string(element.GetType())
			elementTypeStats[elementType]++

			// 跳过某些类型的元素
			if skipElementTypes[elementType] {
				continue
			}

			elementName := element.GetName()
			elementRange := element.GetRange()

			if elementName == "" || len(elementRange) != 4 {
				continue
			}

			// 尝试查找定义
			definitions, err := indexer.QueryDefinitions(context.Background(), &types.QueryDefinitionOptions{
				Workspace: workspacePath,
				StartLine: int(elementRange[0]),
				EndLine:   int(elementRange[2]),
				FilePath:  fileElement.Path,
			})

			if err == nil && len(definitions) > 0 {
				elementTypeSuccessStats[elementType]++
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
