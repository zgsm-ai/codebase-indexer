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
	assert.NoError(t, err)
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".go"},
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
	if err != nil {
		t.Logf("setupTestEnvironment error: %v", err)
		return
	}
	defer teardownTestEnvironment(t, env)

	// 使用codebase-indexer-main项目作为测试数据
	workspacePath, err := filepath.Abs("../../")
	// 初始化工作空间数据库记录
	if err = initWorkspaceModel(env, workspacePath); err != nil {
		t.Logf("initWorkspaceModel error: %v", err)
		return
	}

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".go"}, // 只索引Go文件
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
		CodeSnippet     []byte             // 代码片段内容
	}

	// 使用您提供的10个解析出来的元素作为测试用例
	testCases := []QueryTestCase{
		{
			Name:          "查询createTestIndexer函数调用",
			ElementName:   "createTestIndexer",
			FilePath:      filepath.Join(workspacePath, "test/codegraph/ts_test.go"),
			StartLine:     67,
			EndLine:       70,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "createTestIndexer", Path: "test_utils.go", Range: []int32{109, 0, 109, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询StripSpaces函数调用",
			ElementName:   "StripSpaces",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/resolver/java.go"),
			StartLine:     33,
			EndLine:       33,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "StripSpaces", Path: "common.go", Range: []int32{306, 0, 306, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询symbolMapKey函数调用",
			ElementName:   "symbolMapKey",
			FilePath:      filepath.Join(workspacePath, "internal/service/indexer.go"),
			StartLine:     1647,
			EndLine:       1647,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "symbolMapKey", Path: "indexer.go", Range: []int32{1650, 0, 1650, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询makeQueryPath函数调用",
			ElementName:   "makeQueryPath",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/parser/scm.go"),
			StartLine:     58,
			EndLine:       58,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "makeQueryPath", Path: "scm.go", Range: []int32{69, 0, 69, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询NewTaskPool函数调用",
			ElementName:   "NewTaskPool",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/pool/task_pool_test.go"),
			StartLine:     19,
			EndLine:       19,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "NewTaskPool", Path: "task_pool.go", Range: []int32{28, 0, 28, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询parseBaseClassClause函数调用",
			ElementName:   "parseBaseClassClause",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/resolver/cpp.go"),
			StartLine:     133,
			EndLine:       133,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "parseBaseClassClause", Path: "cpp.go", Range: []int32{348, 0, 348, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询NewReference函数调用",
			ElementName:   "NewReference",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/resolver/go.go"),
			StartLine:     241,
			EndLine:       241,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "NewReference", Path: "common.go", Range: []int32{149, 0, 149, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询findAllTypeIdentifiers函数调用",
			ElementName:   "findAllTypeIdentifiers",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/resolver/cpp.go"),
			StartLine:     225,
			EndLine:       225,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "findAllTypeIdentifiers", Path: "common.go", Range: []int32{239, 0, 239, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询CreateTestValues函数调用",
			ElementName:   "CreateTestValues",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/store/leveldb_test.go"),
			StartLine:     409,
			EndLine:       409,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "CreateTestValues", Path: "test_utils.go", Range: []int32{69, 0, 69, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询containsModifier函数调用",
			ElementName:   "containsModifier",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/resolver/javascript.go"),
			StartLine:     301,
			EndLine:       301,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "containsModifier", Path: "javascript.go", Range: []int32{312, 0, 312, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询NewModuleResolver函数调用",
			ElementName:   "NewModuleResolver",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/workspace/workspace.go"),
			StartLine:     68,
			EndLine:       68,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "NewModuleResolver", Path: "module_resolver.go", Range: []int32{35, 0, 35, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询Definition结构体",
			ElementName:   "Definition",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/types/index.go"),
			StartLine:     31,
			EndLine:       31,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "Definition", Path: "index.go", Range: []int32{33, 0, 33, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询SourceFile结构体",
			ElementName:   "SourceFile",
			FilePath:      filepath.Join(workspacePath, "internal/service/indexer.go"),
			StartLine:     1469,
			EndLine:       1469,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "SourceFile", Path: "element.go", Range: []int32{258, 0, 258, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询GraphNode结构体",
			ElementName:   "GraphNode",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/types/indexer.go"),
			StartLine:     60,
			EndLine:       60,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "GraphNode", Path: "indexer.go", Range: []int32{40, 0, 40, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ScannerInterface结构体",
			ElementName:   "ScannerInterface",
			FilePath:      filepath.Join(workspacePath, "internal/service/indexer.go"),
			StartLine:     59,
			EndLine:       59,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ScannerInterface", Path: "scanner.go", Range: []int32{24, 0, 24, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询CodeGraphSummary结构体",
			ElementName:   "CodeGraphSummary",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/types/indexer.go"),
			StartLine:     1274,
			EndLine:       1274,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "CodeGraphSummary", Path: "index.go", Range: []int32{62, 0, 62, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询VersionRequest结构体",
			ElementName:   "VersionRequest",
			FilePath:      filepath.Join(workspacePath, "api/codebase_syncer.pb.go"),
			StartLine:     454,
			EndLine:       454,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "VersionRequest", Path: "codebase_syncer.pb.go", Range: []int32{445, 0, 445, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ConfigServer结构体",
			ElementName:   "ConfigServer",
			FilePath:      filepath.Join(workspacePath, "internal/config/config.go"),
			StartLine:     43,
			EndLine:       43,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ConfigServer", Path: "config.go", Range: []int32{11, 0, 11, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询DefinitionData结构体",
			ElementName:   "DefinitionData",
			FilePath:      filepath.Join(workspacePath, "internal/service/codebase.go"),
			StartLine:     431,
			EndLine:       431,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "DefinitionData", Path: "backend.go", Range: []int32{78, 0, 78, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询JavaClassifier结构体",
			ElementName:   "JavaClassifier",
			FilePath:      filepath.Join(workspacePath, "pkg/codegraph/analyzer/package_classifier/java_classifier.go"),
			StartLine:     15,
			EndLine:       15,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "JavaClassifier", Path: "java_classifier.go", Range: []int32{8, 0, 8, 0}},
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

func TestFindDefinitionsForAllElementsGo(t *testing.T) {
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
		IncludeExts: []string{".go"},
	})

	// 先清除所有已有的索引，确保强制重新索引
	fmt.Println("清除工作空间的所有索引...")
	err = indexer.RemoveAllIndexes(context.Background(), workspacePath)
	assert.NoError(t, err)
	fmt.Println("索引清除完成")

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
