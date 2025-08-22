package codegraph

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const JavaProjectRootDir = "/tmp/projects/java/mall"

// 添加性能分析辅助函数
func setupProfiling() (func(), error) {
	// CPU Profile
	cpuFile, err := os.Create("cpu.profile")
	if err != nil {
		return nil, fmt.Errorf("创建CPU profile文件失败: %v", err)
	}
	pprof.StartCPUProfile(cpuFile)

	// Memory Profile
	memFile, err := os.Create("memory.profile")
	if err != nil {
		cpuFile.Close()
		pprof.StopCPUProfile()
		return nil, fmt.Errorf("创建内存profile文件失败: %v", err)
	}

	// Goroutine Profile
	goroutineFile, err := os.Create("goroutine.profile")
	if err != nil {
		cpuFile.Close()
		memFile.Close()
		pprof.StopCPUProfile()
		return nil, fmt.Errorf("创建goroutine profile文件失败: %v", err)
	}

	// Trace Profile
	traceFile, err := os.Create("trace.out")
	if err != nil {
		cpuFile.Close()
		memFile.Close()
		goroutineFile.Close()
		pprof.StopCPUProfile()
		return nil, fmt.Errorf("创建trace文件失败: %v", err)
	}
	trace.Start(traceFile)

	cleanup := func() {
		// 停止CPU profile
		pprof.StopCPUProfile()
		cpuFile.Close()

		// 停止trace
		trace.Stop()
		traceFile.Close()

		// 写入内存profile
		pprof.WriteHeapProfile(memFile)
		memFile.Close()

		// 写入goroutine profile
		pprof.Lookup("goroutine").WriteTo(goroutineFile, 0)
		goroutineFile.Close()

		// 打印运行时统计信息
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		fmt.Printf("\n=== 运行时统计信息 ===\n")
		fmt.Printf("总分配内存: %d MB\n", m.TotalAlloc/1024/1024)
		fmt.Printf("系统内存: %d MB\n", m.Sys/1024/1024)
		fmt.Printf("堆内存: %d MB\n", m.HeapAlloc/1024/1024)
		fmt.Printf("堆系统内存: %d MB\n", m.HeapSys/1024/1024)
		fmt.Printf("GC次数: %d\n", m.NumGC)
		fmt.Printf("当前goroutine数量: %d\n", runtime.NumGoroutine())
		fmt.Printf("========================\n")
	}

	return cleanup, nil
}

func TestParseJavaProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 设置性能分析
	cleanup, err := setupProfiling()
	assert.NoError(t, err)
	defer cleanup()

	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "mall",
			Path:    filepath.Join(JavaProjectRootDir),
			wantErr: nil,
		},
	}

	// 记录总体开始时间
	totalStart := time.Now()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// 记录每个测试用例开始前的内存状态
			var mBefore runtime.MemStats
			runtime.ReadMemStats(&mBefore)

			start := time.Now()
			project := NewTestProject(tc.Path, env.logger)
			fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
			fmt.Println("err:", err)
			err = exportFileElements(defaultExportDir, tc.Name, fileElements)
			duration := time.Since(start)

			// 记录每个测试用例结束后的内存状态
			var mAfter runtime.MemStats
			runtime.ReadMemStats(&mAfter)

			fmt.Printf("测试用例 '%s' 执行时间: %v, 文件个数: %d\n", tc.Name, duration, len(fileElements))
			fmt.Printf("内存变化: 分配 +%d MB, 系统 +%d MB\n",
				(mAfter.TotalAlloc-mBefore.TotalAlloc)/1024/1024,
				(mAfter.Sys-mBefore.Sys)/1024/1024)

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

	// 打印总体执行时间
	totalDuration := time.Since(totalStart)
	fmt.Printf("\n=== 总体执行时间: %v ===\n", totalDuration)
}

func TestQueryJava(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	if err != nil {
		t.Logf("setupTestEnvironment error: %v", err)
		return
	}
	defer teardownTestEnvironment(t, env)

	workspacePath := "e:\\tmp\\projects\\java\\mall"
	// 初始化工作空间数据库记录
	if err = initWorkspaceModel(env, workspacePath); err != nil {
		t.Logf("initWorkspaceModel error: %v", err)
		return
	}

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".java"},
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
			Name:          "查询success方法调用",
			ElementName:   "success",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-admin\\src\\main\\java\\com\\macro\\mall\\controller\\SmsHomeNewProductController.java",
			StartLine:     34,
			EndLine:       34,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "success", Path: "CommonResult.java", Range: []int32{34, 0, 34, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询createBrand方法调用",
			ElementName:   "createBrand",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-demo\\src\\main\\java\\com\\macro\\mall\\demo\\controller\\DemoController.java",
			StartLine:     45,
			EndLine:       45,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "createBrand", Path: "DemoService.java", Range: []int32{14, 0, 14, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询deleteBrand方法调用",
			ElementName:   "deleteBrand",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-demo\\src\\main\\java\\com\\macro\\mall\\demo\\controller\\DemoController.java",
			StartLine:     76,
			EndLine:       76,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "deleteBrand", Path: "DemoService.java", Range: []int32{18, 0, 18, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ApiException方法调用",
			ElementName:   "ApiException",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-common\\src\\main\\java\\com\\macro\\mall\\common\\exception\\Asserts.java",
			StartLine:     15,
			EndLine:       15,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ApiException", Path: "ApiException.java", Range: []int32{8, 0, 11, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询validateFailed方法调用",
			ElementName:   "validateFailed",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-common\\src\\main\\java\\com\\macro\\mall\\common\\exception\\GlobalExceptionHandler.java",
			StartLine:     56,
			EndLine:       56,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "validateFailed", Path: "CommonResult.java", Range: []int32{91, 0, 91, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询cancelOrder方法调用",
			ElementName:   "cancelOrder",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-portal\\src\\main\\java\\com\\macro\\mall\\portal\\component\\CancelOrderReceiver.java",
			StartLine:     23,
			EndLine:       23,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "cancelOrder", Path: "OmsPortalOrderService.java", Range: []int32{42, 0, 43, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询getUserNameFromToken方法调用",
			ElementName:   "getUserNameFromToken",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-security\\src\\main\\java\\com\\macro\\mall\\security\\component\\JwtAuthenticationTokenFilter.java",
			StartLine:     43,
			EndLine:       43,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "getUserNameFromToken", Path: "JwtTokenUtil.java", Range: []int32{75, 0, 75, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询getCode方法调用",
			ElementName:   "getCode",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-common\\src\\main\\java\\com\\macro\\mall\\common\\api\\CommonResult.java",
			StartLine:     36,
			EndLine:       36,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "getCode", Path: "ResultCode.java", Range: []int32{20, 0, 20, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询list方法调用",
			ElementName:   "list",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-admin\\src\\main\\java\\com\\macro\\mall\\controller\\UmsAdminController.java",
			StartLine:     122,
			EndLine:       122,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "list", Path: "UmsAdminService.java", Range: []int32{49, 0, 49, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询getLogger方法调用", //调用系统包
			ElementName:   "getLogger",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-portal\\src\\main\\java\\com\\macro\\mall\\portal\\component\\CancelOrderReceiver.java",
			StartLine:     18,
			EndLine:       18,
			ElementType:   "call.method",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询UmsMemberLevelService引用",
			ElementName:   "UmsMemberLevelService",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-admin\\src\\main\\java\\com\\macro\\mall\\controller\\UmsMemberLevelController.java",
			StartLine:     28,
			EndLine:       28,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "UmsMemberLevelService", Path: "UmsMemberLevelService.java", Range: []int32{10, 0, 16, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询WebLog引用",
			ElementName:   "WebLog",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-common\\src\\main\\java\\com\\macro\\mall\\common\\log\\WebLogAspect.java",
			StartLine:     61,
			EndLine:       61,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "WebLog", Path: "WebLog.java", Range: []int32{9, 0, 11, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询Criteria引用",
			ElementName:   "Criteria",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-mbg\\src\\main\\java\\com\\macro\\mall\\model\\CmsPrefrenceAreaExample.java",
			StartLine:     56,
			EndLine:       56,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "Criteria", Path: "CmsPrefrenceAreaExample.java", Range: []int32{427, 0, 427, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询PmsPortalBrandService引用",
			ElementName:   "PmsPortalBrandService",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-portal\\src\\main\\java\\com\\macro\\mall\\portal\\controller\\PmsPortalBrandController.java",
			StartLine:     28,
			EndLine:       28,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "PmsPortalBrandService", Path: "PmsPortalBrandService.java", Range: []int32{12, 0, 12, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询AlipayConfig引用",
			ElementName:   "AlipayConfig",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-portal\\src\\main\\java\\com\\macro\\mall\\portal\\service\\impl\\AlipayServiceImpl.java",
			StartLine:     33,
			EndLine:       33,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "AlipayConfig", Path: "AlipayConfig.java", Range: []int32{13, 0, 17, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询JSONObject引用", //调用系统包
			ElementName:   "JSONObject",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-portal\\src\\main\\java\\com\\macro\\mall\\portal\\service\\impl\\AlipayServiceImpl.java",
			StartLine:     52,
			EndLine:       52,
			ElementType:   "reference",
			ShouldFindDef: false,
			wantErr:       nil,
		},
		{
			Name:          "查询MethodSignature引用", //调用系统包
			ElementName:   "MethodSignature",
			FilePath:      "e:\\tmp\\projects\\java\\mall\\mall-security\\src\\main\\java\\com\\macro\\mall\\security\\aspect\\RedisCacheAspect.java",
			StartLine:     34,
			EndLine:       34,
			ElementType:   "reference",
			ShouldFindDef: false,
			wantErr:       nil,
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

// 添加一个专门的性能基准测试
func BenchmarkParseJavaProject(b *testing.B) {
	env, err := setupTestEnvironment()
	if err != nil {
		b.Fatal(err)
	}
	defer teardownTestEnvironment(nil, env)

	// 选择一个中等大小的项目进行基准测试
	projectPath := filepath.Join(JavaProjectRootDir, "kafka")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		project := NewTestProject(projectPath, env.logger)
		fileElements, _, err := ParseProjectFiles(context.Background(), env, project)
		if err != nil {
			b.Fatal(err)
		}
		_ = fileElements
	}
}

func TestFindDefinitionsForAllElementsJava(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用项目自身的代码作为测试数据
	workspacePath, err := filepath.Abs(JavaProjectRootDir) // 指向项目根目录
	assert.NoError(t, err)

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器并索引工作空间
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "test", ".git"),
		IncludeExts: []string{".java"}, // 只索引java文件
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
