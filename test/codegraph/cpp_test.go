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

const CPPProjectRootDir = "/tmp/projects/cpp/grpc"

func TestParseCPPProjectFiles(t *testing.T) {
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "grpc",
			Path:    filepath.Join(CPPProjectRootDir, ""),
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

func TestQueryCPP(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	workspacePath := "e:\\tmp\\projects\\cpp\\grpc"
	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", ".git"),
		IncludeExts: []string{".cpp", ".cc", ".cxx", ".hpp", ".h"},
	})

	// 先索引工作空间，确保有数据可查询
	fmt.Println("开始索引CProjectRootDir工作空间...")
	_, err = indexer.IndexWorkspace(context.Background(), workspacePath)
	assert.NoError(t, err)
	fmt.Println("工作空间索引完成")

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
			Name:          "查询grpc_channel_destroy_internal函数调用",
			ElementName:   "grpc_channel_destroy_internal",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\channel.cc",
			StartLine:     96,
			EndLine:       96,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_channel_destroy_internal", Path: "channel.h", Range: []int32{153, 0, 153, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_channel_stack_type_is_client函数调用",
			ElementName:   "grpc_channel_stack_type_is_client",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\legacy_channel.cc",
			StartLine:     67,
			EndLine:       67,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_channel_stack_type_is_client", Path: "channel_stack_type.cc", Range: []int32{22, 0, 22, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_call_details_init函数调用",
			ElementName:   "grpc_call_details_init",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\cpp\\server\\server_cc.cc",
			StartLine:     607,
			EndLine:       607,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_call_details_init", Path: "call_details.cc", Range: []int32{26, 0, 26, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询run_in_call_combiner函数调用",
			ElementName:   "run_in_call_combiner",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\channel\\connected_channel.cc",
			StartLine:     104,
			EndLine:       104,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "run_in_call_combiner", Path: "connected_channel.cc", Range: []int32{96, 0, 96, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询FromTopElem函数调用",
			ElementName:   "FromTopElem",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\filter_stack_call.cc",
			StartLine:     1175,
			EndLine:       1175,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "FromTopElem", Path: "filter_stack_call.h", Range: []int32{81, 0, 81, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_metadata_array_init函数调用",
			ElementName:   "grpc_metadata_array_init",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\load_balancing\\grpclb\\grpclb.cc",
			StartLine:     907,
			EndLine:       907,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_metadata_array_init", Path: "metadata_array.cc", Range: []int32{25, 0, 25, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询GrpcLbLoadReportRequestCreate函数调用",
			ElementName:   "GrpcLbLoadReportRequestCreate",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\load_balancing\\grpclb\\grpclb.cc",
			StartLine:     1066,
			EndLine:       1066,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "GrpcLbLoadReportRequestCreate", Path: "load_balancer_api.cc", Range: []int32{81, 0, 81, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ReadPolicyFromFile函数调用",
			ElementName:   "ReadPolicyFromFile",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\security\\authorization\\grpc_authorization_policy_provider.cc",
			StartLine:     143,
			EndLine:       143,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ReadPolicyFromFile", Path: "grpc_authorization_policy_provider.cc", Range: []int32{62, 0, 62, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询run_test函数调用",
			ElementName:   "run_test",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\cpp\\codegen\\golden_file_test.cc",
			StartLine:     54,
			EndLine:       55,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "run_test", Path: "golden_file_test.cc", Range: []int32{34, 0, 34, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_chttp2_transport_start_reading函数调用",
			ElementName:   "grpc_chttp2_transport_start_reading",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\ext\\transport\\chttp2\\server\\chttp2_server.cc",
			StartLine:     249,
			EndLine:       250,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_chttp2_transport_start_reading", Path: "chttp2_transport.cc", Range: []int32{3477, 0, 3477, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询FromStaticString方法调用",
			ElementName:   "FromStaticString",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\ext\\transport\\chttp2\\transport\\hpack_encoder.cc",
			StartLine:     421,
			EndLine:       421,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "FromStaticString", Path: "chttp2_transport.cc", Range: []int32{117, 0, 120, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询StartBatch方法调用",
			ElementName:   "StartBatch",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\call.cc",
			StartLine:     489,
			EndLine:       489,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "StartBatch", Path: "filter_stack_call.cc", Range: []int32{745, 0, 745, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询CancelWithError方法调用",
			ElementName:   "CancelWithError",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\call.cc",
			StartLine:     421,
			EndLine:       422,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "CancelWithError", Path: "filter_stack_call.cc", Range: []int32{332, 0, 332, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询GetInfo方法调用",
			ElementName:   "GetInfo",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\channel.cc",
			StartLine:     165,
			EndLine:       165,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "GetInfo", Path: "legacy_channel.cc", Range: []int32{376, 0, 376, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询channel_init方法调用",
			ElementName:   "channel_init",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\init.cc",
			StartLine:     74,
			EndLine:       76,
			ElementType:   "call.method",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "channel_init", Path: "core_configuration.h", Range: []int32{76, 0, 76, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询HttpProxyMapper类的调用",
			ElementName:   "HttpProxyMapper",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\core\\handshake\\http_proxy_mapper_test.cc",
			StartLine:     209,
			EndLine:       209,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "HttpProxyMapper", Path: "http_proxy_mapper.h", Range: []int32{34, 0, 45, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询FuzzingEndpoint类的调用",
			ElementName:   "FuzzingEndpoint",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\core\\event_engine\\fuzzing_event_engine\\fuzzing_event_engine.cc",
			StartLine:     684,
			EndLine:       684,
			ElementType:   "call.function",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "FuzzingEndpoint", Path: "fuzzing_event_engine.h", Range: []int32{266, 0, 266, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ScopedEnvVar类的调用",
			ElementName:   "ScopedEnvVar",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\core\\handshake\\http_proxy_mapper_test.cc",
			StartLine:     63,
			EndLine:       63,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ScopedEnvVar", Path: "scoped_env_var.h", Range: []int32{26, 0, 26, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询ScopedExperimentalEnvVar类的调用",
			ElementName:   "ScopedExperimentalEnvVar",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\core\\xds\\file_watcher_certificate_provider_factory_test.cc",
			StartLine:     132,
			EndLine:       132,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "ScopedExperimentalEnvVar", Path: "scoped_env_var.h", Range: []int32{38, 0, 38, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询SocketUseAfterCloseDetector类的调用",
			ElementName:   "SocketUseAfterCloseDetector",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\cpp\\naming\\cancel_ares_query_test.cc",
			StartLine:     361,
			EndLine:       362,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "SocketUseAfterCloseDetector", Path: "socket_use_after_close_detector.h", Range: []int32{41, 0, 41, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_call_credentials结构体的调用",
			ElementName:   "grpc_call_credentials",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\test\\core\\test_util\\test_call_creds.cc",
			StartLine:     43,
			EndLine:       43,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_call_credentials", Path: "credentials.h", Range: []int32{36, 0, 36, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_auth_context结构体的调用",
			ElementName:   "grpc_auth_context",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\include\\grpc\\grpc_security.cc",
			StartLine:     37,
			EndLine:       37,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_auth_context", Path: "credentials.h", Range: []int32{37, 0, 37, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_transport_stream_op_batch结构体的调用",
			ElementName:   "grpc_transport_stream_op_batch",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\filter_stack_call.cc",
			StartLine:     352,
			EndLine:       353,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_transport_stream_op_batch", Path: "transport.h", Range: []int32{258, 0, 258, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_transport_stream_op_batch结构体的调用",
			ElementName:   "grpc_transport_stream_op_batch",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\surface\\filter_stack_call.cc",
			StartLine:     352,
			EndLine:       353,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_transport_stream_op_batch", Path: "transport.h", Range: []int32{258, 0, 258, 0}},
			},
			wantErr: nil,
		},
		{
			Name:          "查询grpc_closure结构体的调用",
			ElementName:   "grpc_closure",
			FilePath:      "e:\\tmp\\projects\\cpp\\grpc\\src\\core\\lib\\transport\\transport.h",
			StartLine:     279,
			EndLine:       279,
			ElementType:   "reference",
			ShouldFindDef: true,
			wantDefinitions: []types.Definition{
				{Name: "grpc_closure", Path: "closure.h", Range: []int32{35, 0, 35, 0}},
			},
			wantErr: nil,
		},
	}

	// 统计变量
	totalCases := len(testCases)
	correctCases := 0

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

			foundDefinitions := len(definitions)

			fmt.Printf("查询结果: ")
			if err != nil {
				fmt.Printf("查询失败 - %v\n", err)
			} else {
				fmt.Printf("找到 %d 个定义\n", foundDefinitions)

				if foundDefinitions > 0 {
					fmt.Println("📋 查询结果详情:")
					for j, def := range definitions {
						fmt.Printf("  [%d] 名称: '%s'\n", j+1, def.Name)
						fmt.Printf("      类型: '%s'\n", def.Type)
						fmt.Printf("      范围: %v\n", def.Range)
						fmt.Printf("      文件: '%s'\n", filepath.Base(def.Path))
						fmt.Printf("      完整路径: '%s'\n", def.Path)

						// 如果有期望的定义，进行匹配度分析
						if len(tc.wantDefinitions) > 0 {
							for _, wantDef := range tc.wantDefinitions {
								if def.Name != wantDef.Name {
									fmt.Printf("      ❌ 名称不匹配: 期望 '%s' 实际 '%s'\n", wantDef.Name, def.Name)
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

									fmt.Printf("      匹配分析: 名称%s 行号%s 路径%s\n", nameMatch, lineMatch, pathMatch)
								}
							}
						}
						fmt.Println("      " + strings.Repeat("-", 40))
					}
				} else {
					fmt.Println("  ❌ 未找到任何定义")
				}

				// 输出查询总结
				fmt.Printf("📊 查询总结: 期望找到=%v, 实际找到=%d\n",
					tc.ShouldFindDef, foundDefinitions)

				if tc.ShouldFindDef && foundDefinitions == 0 {
					fmt.Println("  ⚠️  警告: 期望找到定义但未找到")
				} else if !tc.ShouldFindDef && foundDefinitions > 0 {
					fmt.Println("  ⚠️  警告: 期望不找到定义但找到了")
				} else {
					fmt.Println("  ✅ 查询结果符合预期")
				}
			}

			// 使用结构化的期望结果进行验证（类似js_resolver_test.go格式）
			if len(tc.wantDefinitions) > 0 || tc.wantErr != nil {
				// 使用新的结构化验证
				assert.Equal(t, tc.wantErr, err, fmt.Sprintf("%s: 错误应该匹配", tc.Name))

				if tc.wantErr == nil {
					// 当返回多个定义时，验证期望的定义是否都存在
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
						assert.True(t, found,
							fmt.Sprintf("%s: 应该找到名为 '%s' 行号为'%d'路径包含 '%s' 的定义",
								tc.Name, wantDef.Name, wantDef.Range[0], wantDef.Path))
					}

				}
			} else {
				// 对于空的wantDefinitions，直接判断正确
				correctCases++
				fmt.Printf("✓ %s: wantDefinitions为空，测试通过\n", tc.Name)
			}
		})
	}

}

func TestFindDefinitionsForAllElementsCPP(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 使用项目自身的代码作为测试数据
	workspacePath, err := filepath.Abs(CPPProjectRootDir) // 指向项目根目录
	assert.NoError(t, err)

	// 初始化工作空间数据库记录
	err = initWorkspaceModel(env, workspacePath)
	assert.NoError(t, err)

	// 创建索引器并索引工作空间
	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: append(defaultVisitPattern.ExcludeDirs, "vendor", "test", ".git"),
		IncludeExts: []string{".cpp", ".cc", ".cxx", ".hpp", ".h"}, // 只索引cpp文件
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
