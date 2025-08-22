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

const CProjectRootDir = "/tmp/projects/c/zstd-dev"

func TestParseCProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "zstd-dev",
			Path:    filepath.Join(CProjectRootDir, ""),
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fmt.Println("tc.Path", tc.Path)
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
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
		})
	}
}

func TestQueryC(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	if err != nil {
		t.Logf("setupTestEnvironment error: %v", err)
		return
	}
	defer teardownTestEnvironment(t, env)

	workspacePath := "e:\\tmp\\projects\\c\\zstd-dev"
	// 初始化工作空间数据库记录
	if err = initWorkspaceModel(env, workspacePath); err != nil {
		t.Logf("initWorkspaceModel error: %v", err)
		return
	}

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".h", ".c"},
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

	testCases := []QueryTestCase{
		{
			Name:          "查询checkLibVersion函数调用",
			ElementName:   "checkLibVersion",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\zstdcli.c",
			StartLine:     927,
			EndLine:       927,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "checkLibVersion", Path: "zstdcli.c", Range: []int32{114, 0, 114, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询lastNameFromPath函数调用",
			ElementName:   "lastNameFromPath",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\zstdcli.c",
			StartLine:     932,
			EndLine:       932,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "lastNameFromPath", Path: "zstdcli.c", Range: []int32{333, 0, 333, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询exeNameMatch函数调用",
			ElementName:   "exeNameMatch",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\zstdcli.c",
			StartLine:     935,
			EndLine:       935,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "exeNameMatch", Path: "zstdcli.c", Range: []int32{129, 0, 129, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询defaultCoverParams函数调用",
			ElementName:   "defaultCoverParams",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\zstdcli.c",
			StartLine:     917,
			EndLine:       917,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "defaultCoverParams", Path: "zstdcli.c", Range: []int32{563, 0, 563, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询defaultFastCoverParams函数调用",
			ElementName:   "defaultFastCoverParams",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\zstdcli.c",
			StartLine:     918,
			EndLine:       918,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "defaultFastCoverParams", Path: "zstdcli.c", Range: []int32{575, 0, 575, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询FIO_checkFilenameCollisions函数调用",
			ElementName:   "FIO_checkFilenameCollisions",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\fileio.c",
			StartLine:     3142,
			EndLine:       3142,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "FIO_checkFilenameCollisions", Path: "fileio.c", Range: []int32{879, 0, 879, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询extractFilename函数调用",
			ElementName:   "extractFilename",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\fileio.c",
			StartLine:     938,
			EndLine:       938,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "extractFilename", Path: "fileio.c", Range: []int32{911, 0, 911, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询BMK_benchCLevels函数调用",
			ElementName:   "BMK_benchCLevels",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\benchzstd.c",
			StartLine:     1015,
			EndLine:       1015,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "BMK_benchCLevels", Path: "benchzstd.c", Range: []int32{920, 0, 920, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询UTIL_allocateFileNamesTable函数调用",
			ElementName:   "UTIL_allocateFileNamesTable",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\zstdcli.c",
			StartLine:     900,
			EndLine:       900,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "UTIL_allocateFileNamesTable", Path: "util.c", Range: []int32{824, 0, 824, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询UTIL_prepareFileList函数调用",
			ElementName:   "UTIL_prepareFileList",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\programs\\util.c",
			StartLine:     950,
			EndLine:       950,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "UTIL_prepareFileList", Path: "util.c", Range: []int32{908, 0, 908, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ZSTD_localDict结构体调用",
			ElementName:   "ZSTD_localDict",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\lib\\compress\\zstd_compress_internal.c",
			StartLine:     1270,
			EndLine:       1270,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ZSTD_localDict", Path: "util.h", Range: []int32{54, 0, 60, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询cdict_collection_t结构体调用",
			ElementName:   "cdict_collection_t",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\contrib\\comprlargeNbDictsss\\largeNbDicts.c",
			StartLine:     441,
			EndLine:       441,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "cdict_collection_t", Path: "largeNbDicts.c", Range: []int32{435, 0, 438, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ZSTD_eDist_match结构体调用",
			ElementName:   "ZSTD_eDist_match",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\contrib\\match_finders\\zstd_edist.c",
			StartLine:     64,
			EndLine:       64,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ZSTD_eDist_match", Path: "zstd_edist.c", Range: []int32{48, 0, 52, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询job结构体调用",
			ElementName:   "job",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\contrib\\seekable_format\\examples\\parallel_compression.c",
			StartLine:     88,
			EndLine:       88,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "job", Path: "parallel_compression.c", Range: []int32{85, 0, 96, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ZSTDv05_parameters结构体调用",
			ElementName:   "ZSTDv05_parameters",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\lib\\legacy\\zstd_v05.c",
			StartLine:     2603,
			EndLine:       2603,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ZSTDv05_parameters", Path: "zstd_v05.h", Range: []int32{85, 0, 90, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询BMK_benchParams_t结构体调用",
			ElementName:   "BMK_benchParams_t",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\tests\\paramgrill.c",
			StartLine:     1591,
			EndLine:       1591,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "BMK_benchParams_t", Path: "benchfn.h", Range: []int32{61, 0, 80, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ZWRAP_DCtx结构体调用",
			ElementName:   "ZWRAP_DCtx",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\zlibWrapper\\zstd_zlibwrapper.c",
			StartLine:     636,
			EndLine:       636,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ZWRAP_DCtx", Path: "zstd_zlibwrapper.c", Range: []int32{515, 0, 530, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询data_type_t结构体调用",
			ElementName:   "data_type_t",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\tests\\regression\\data.h",
			StartLine:     31,
			EndLine:       31,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "data_type_t", Path: "data.h", Range: []int32{16, 0, 19, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询data_type_t结构体调用",
			ElementName:   "data_type_t",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\tests\\regression\\data.h",
			StartLine:     31,
			EndLine:       31,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "data_type_t", Path: "data.h", Range: []int32{16, 0, 19, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ZSTDv07_customMem结构体调用",
			ElementName:   "ZSTDv07_customMem",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\lib\\legacy\\zstd_v07.c",
			StartLine:     2795,
			EndLine:       2795,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ZSTDv07_customMem", Path: "zstd_v07.c", Range: []int32{67, 0, 67, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询config_t结构体调用",
			ElementName:   "config_t",
			FilePath:      "e:\\tmp\\projects\\c\\zstd-dev\\tests\\regression\\config.c",
			StartLine:     170,
			EndLine:       170,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "config_t", Path: "config.h", Range: []int32{33, 0, 60, 0}},
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

func TestFindDefinitionsForAllElementsC(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用项目自身的代码作为测试数据
	workspacePath, err := filepath.Abs(CProjectRootDir)
	assert.NoError(t, err)

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器并索引工作空间
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "test", ".git"),
		IncludeExts: []string{".c", ".h"}, // 只索引cpp文件
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
