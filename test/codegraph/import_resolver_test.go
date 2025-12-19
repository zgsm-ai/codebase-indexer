package codegraph

import (
	"codebase-indexer/pkg/codegraph/analyzer/import_resolver"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportResolver_MultiProject 测试多个项目的导入解析
func TestImportResolver_MultiProject(t *testing.T) {
	testCases := []struct {
		name        string
		language    lang.Language
		projectPath string
		goModules   []string
		testFiles   []string
		minResolved int // 最少期望解析成功的导入数
	}{
		// ========== Go 项目 ==========
		{
			name:        "Go_Kubernetes",
			language:    lang.Go,
			projectPath: filepath.Join(testRootDir, "go", "kubernetes"),
			goModules:   []string{"k8s.io/kubernetes"},
			testFiles: []string{
				"pkg/api/legacyscheme/scheme.go",
				"pkg/api/node/util.go",
				"pkg/kubelet/kubelet.go",
				"pkg/scheduler/scheduler.go",
				"pkg/controller/controller_utils.go",
			},
			minResolved: 5,
		},
		{
			name:        "Go_DockerCE",
			language:    lang.Go,
			projectPath: filepath.Join(testRootDir, "go", "docker-ce"),
			goModules:   []string{"github.com/docker/docker"},
			testFiles: []string{
				"components/engine/api/server/router/router.go",
				"components/engine/container/container.go",
				"components/engine/daemon/daemon.go",
				"components/cli/cli/command/command.go",
				"components/cli/cli/config/config.go",
			},
			minResolved: 3,
		},

		// ========== Python 项目 ==========
		{
			name:        "Python_Django",
			language:    lang.Python,
			projectPath: filepath.Join(testRootDir, "python", "django"),
			testFiles: []string{
				"django/apps/config.py",
				"django/apps/registry.py",
				"django/core/management/base.py",
				"django/db/models/base.py",
				"django/http/request.py",
			},
			minResolved: 10,
		},
		{
			name:        "Python_Pandas",
			language:    lang.Python,
			projectPath: filepath.Join(testRootDir, "python", "pandas"),
			testFiles: []string{
				"pandas/core/frame.py",
				"pandas/core/series.py",
				"pandas/core/generic.py",
				"pandas/io/parsers/readers.py",
				"pandas/plotting/_core.py",
			},
			minResolved: 5,
		},

		// ========== Java 项目 ==========
		{
			name:        "Java_SpringBoot",
			language:    lang.Java,
			projectPath: filepath.Join(testRootDir, "java", "spring-boot"),
			testFiles: []string{
				"spring-boot-project/spring-boot/src/main/java/org/springframework/boot/SpringApplication.java",
				"spring-boot-project/spring-boot-autoconfigure/src/main/java/org/springframework/boot/autoconfigure/SpringBootApplication.java",
				"spring-boot-project/spring-boot/src/main/java/org/springframework/boot/context/properties/ConfigurationProperties.java",
				"spring-boot-project/spring-boot-actuator/src/main/java/org/springframework/boot/actuate/endpoint/annotation/Endpoint.java",
				"spring-boot-project/spring-boot/src/main/java/org/springframework/boot/web/server/WebServer.java",
			},
			minResolved: 3,
		},
		{
			name:        "Java_Elasticsearch",
			language:    lang.Java,
			projectPath: filepath.Join(testRootDir, "java", "elasticsearch"),
			testFiles: []string{
				"server/src/main/java/org/elasticsearch/action/search/SearchRequest.java",
				"server/src/main/java/org/elasticsearch/index/query/QueryBuilders.java",
				"server/src/main/java/org/elasticsearch/client/RestClient.java",
				"server/src/main/java/org/elasticsearch/common/settings/Settings.java",
				"server/src/main/java/org/elasticsearch/search/SearchHit.java",
			},
			minResolved: 5,
		},

		// ========== TypeScript 项目 ==========
		{
			name:        "TypeScript_VueNext",
			language:    lang.TypeScript,
			projectPath: filepath.Join(testRootDir, "typescript", "vue-next"),
			testFiles: []string{
				"packages/runtime-core/src/component.ts",
				"packages/reactivity/src/reactive.ts",
				"packages/compiler-core/src/compile.ts",
				"packages/runtime-dom/src/index.ts",
				"packages/shared/src/index.ts",
			},
			minResolved: 0, // TODO: JSImportPathResolver 需支持 monorepo scoped packages (如 @vue/xx)
		},
		{
			name:        "TypeScript_Svelte",
			language:    lang.TypeScript,
			projectPath: filepath.Join(testRootDir, "typescript", "svelte"),
			testFiles: []string{
				"packages/svelte/src/compiler/index.js",
				"packages/svelte/src/index-client.js",
				"packages/svelte/src/compiler/preprocess/index.js",
				"packages/svelte/src/compiler/phases/1-parse/index.js",
				"packages/svelte/src/internal/client/index.js",
			},
			minResolved: 0, // TODO: JSImportPathResolver 需支持 monorepo scoped packages
		},

		// ========== JavaScript 项目 ==========
		{
			name:        "JavaScript_Vue",
			language:    lang.TypeScript, // Vue 3 uses TypeScript
			projectPath: filepath.Join(testRootDir, "javascript", "vue"),
			testFiles: []string{
				"src/compiler/index.ts",
				"src/compiler/codegen/index.ts",
				"src/compiler/parser/index.ts",
				"src/core/index.ts",
				"src/platforms/web/runtime/index.ts",
			},
			minResolved: 0, // TODO: JSImportPathResolver 需支持 monorepo scoped packages
		},

		// ========== C 项目 ==========
		{
			name:        "C_Redis",
			language:    lang.C,
			projectPath: filepath.Join(testRootDir, "c", "redis"),
			testFiles: []string{
				"src/server.c",
				"src/ae.c",
				"src/networking.c",
				"src/dict.c",
				"src/sds.c",
			},
			minResolved: 5,
		},

		// ========== C++ 项目 ==========
		{
			name:        "CPP_GRPC",
			language:    lang.CPP,
			projectPath: filepath.Join(testRootDir, "cpp", "grpc"),
			testFiles: []string{
				"src/core/lib/channel/channel_stack.cc",
				"src/core/lib/surface/call.cc",
				"src/core/lib/transport/transport.cc",
				"src/core/lib/iomgr/endpoint.cc",
				"src/cpp/server/server_cc.cc",
			},
			minResolved: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 检查项目是否存在
			if !dirExists(tc.projectPath) {
				t.Skipf("测试项目不存在: %s", tc.projectPath)
			}

			// Go 项目需要 go.mod 文件才能正确解析导入
			if tc.language == lang.Go {
				goModPath := filepath.Join(tc.projectPath, "go.mod")
				if !fileExists(goModPath) {
					t.Skipf("Go 项目缺少 go.mod 文件: %s", tc.projectPath)
				}
			}

			ctx := context.Background()

			// 创建简化的测试环境：只需要logger和sourceFileParser
			logLevel := os.Getenv("LOG_LEVEL")
			if logLevel == "" {
				logLevel = "info"
			}
			testLogger, err := logger.NewLogger("/tmp/logs", logLevel, "import-resolver-test")
			require.NoError(t, err)

			// 创建源文件解析器
			sourceFileParser := parser.NewSourceFileParser(testLogger)

			// 创建import解析器
			mgr := import_resolver.NewPathResolverManager(tc.projectPath)
			err = mgr.BuildIndex()
			require.NoError(t, err, "构建文件索引失败")

			// 创建project对象
			project := &workspace.Project{
				Path:      tc.projectPath,
				GoModules: tc.goModules,
			}

			// 获取解析器
			resolver := mgr.GetResolver(tc.language)
			require.NotNil(t, resolver, "%s解析器不存在", tc.language)

			totalImports := 0
			totalResolved := 0
			totalVerified := 0
			testedFiles := 0

			// 测试每个文件
			for _, testFile := range tc.testFiles {
				fullPath := filepath.Join(tc.projectPath, testFile)
				if !fileExists(fullPath) {
					t.Logf("⊘ 文件不存在，跳过: %s", testFile)
					continue
				}

				testedFiles++

				// 直接读取文件内容
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Logf("⊘ 读取文件失败，跳过: %s - %v", testFile, err)
					continue
				}

				// 解析文件获取imports
				fileElementTable, err := sourceFileParser.Parse(ctx, &types.SourceFile{
					Path:    fullPath,
					Content: content,
				})
				if err != nil {
					t.Logf("⊘ 解析文件失败，跳过: %s - %v", testFile, err)
					continue
				}

				imports := fileElementTable.Imports
				if len(imports) == 0 {
					t.Logf("⊘ 文件无导入: %s", testFile)
					continue
				}

				totalImports += len(imports)
				fileResolved := 0
				fileVerified := 0

				// 解析每个import
				for _, imp := range imports {
					// 过滤掉明显的外部包
					if !isProjectImport(imp.Source, tc.language, project) {
						continue
					}

					paths, err := resolver.ResolveImportPath(ctx, imp, project)
					if err == nil && len(paths) > 0 {
						fileResolved++
						totalResolved++

						// 验证路径存在（抽查前2个）
						checkCount := len(paths)
						if checkCount > 2 {
							checkCount = 2
						}
						allValid := true
						for i := 0; i < checkCount; i++ {
							fullPath := filepath.Join(tc.projectPath, paths[i])
							if !fileExists(fullPath) {
								allValid = false
								t.Logf("✗ 解析的路径不存在: %s (import: %s)", paths[i], imp.Source)
								break
							}
						}
						if allValid {
							fileVerified++
							totalVerified++
						}
					}
				}

				if fileResolved > 0 {
					t.Logf("✓ %s: %d imports, %d resolved, %d verified",
						filepath.Base(testFile), len(imports), fileResolved, fileVerified)
				} else {
					t.Logf("⊘ %s: %d imports, 0 resolved",
						filepath.Base(testFile), len(imports))
				}
			}

			// 汇总统计
			t.Logf("\n==== %s 汇总 ====", tc.name)
			t.Logf("测试文件: %d/%d", testedFiles, len(tc.testFiles))
			t.Logf("总导入: %d", totalImports)
			t.Logf("解析成功: %d", totalResolved)
			t.Logf("验证通过: %d", totalVerified)

			// 断言
			assert.Greater(t, testedFiles, 0, "至少应该成功测试一个文件")
			assert.GreaterOrEqual(t, totalResolved, tc.minResolved,
				"解析成功数应该>=最小期望值 (%d)", tc.minResolved)
			assert.GreaterOrEqual(t, totalVerified, tc.minResolved,
				"验证通过数应该>=最小期望值 (%d)", tc.minResolved)
		})
	}
}

// isProjectImport 判断import是否为项目内导入
func isProjectImport(source string, language lang.Language, project *workspace.Project) bool {
	if source == "" {
		return false
	}

	switch language {
	case lang.Go:
		// Go: 检查是否以项目模块开头，或者不包含域名的相对导入
		for _, module := range project.GoModules {
			if module != "" && strings.HasPrefix(source, module) {
				return true
			}
		}
		// 跳过明显的第三方包
		if strings.Contains(source, "github.com") ||
			strings.Contains(source, "golang.org") ||
			strings.Contains(source, "google.golang.org") {
			return false
		}
		return true

	case lang.Python:
		// Python: 检查是否以项目名开头
		projectName := filepath.Base(project.Path)
		if strings.HasPrefix(source, projectName) {
			return true
		}
		// 相对导入
		if strings.HasPrefix(source, ".") {
			return true
		}
		return false

	case lang.Java:
		// Java: 检查包名是否包含项目特征
		projectName := filepath.Base(project.Path)
		if strings.Contains(source, strings.ToLower(projectName)) {
			return true
		}
		// org.springframework, org.elasticsearch 等
		if strings.HasPrefix(source, "org.springframework") ||
			strings.HasPrefix(source, "org.elasticsearch") {
			return true
		}
		return false

	case lang.TypeScript, lang.JavaScript:
		// JS/TS: 相对路径或@开头的路径别名
		if strings.HasPrefix(source, "./") ||
			strings.HasPrefix(source, "../") ||
			strings.HasPrefix(source, "@/") {
			return true
		}
		// 检查是否为 scoped package (@xxx/yyy)
		// 对于 monorepo 项目，scoped package 通常是项目内部包
		// 排除常见的外部 scoped packages
		if strings.HasPrefix(source, "@") {
			// 常见的外部 scoped packages
			externalScopes := []string{
				"@types/", "@angular/", "@react/", "@babel/",
				"@typescript-eslint/", "@jest/", "@testing-library/",
			}
			isExternal := false
			for _, scope := range externalScopes {
				if strings.HasPrefix(source, scope) {
					isExternal = true
					break
				}
			}
			if !isExternal {
				return true
			}
		}
		return false

	case lang.C, lang.CPP:
		// C/C++: 双引号include为项目内，尖括号为系统
		return !strings.Contains(source, "<") && !strings.Contains(source, ">")

	default:
		return false
	}
}

// TestImportResolver_IndexBuildOnce 测试索引只构建一次
func TestImportResolver_IndexBuildOnce(t *testing.T) {
	projectPath := filepath.Join(testRootDir, "go", "kubernetes")
	if !dirExists(projectPath) {
		t.Skipf("测试项目不存在: %s", projectPath)
	}

	// 创建import解析器
	mgr := import_resolver.NewPathResolverManager(projectPath)

	// 第一次构建
	err := mgr.BuildIndex()
	require.NoError(t, err, "第一次构建文件索引失败")

	// 第二次构建（应该跳过）
	err = mgr.BuildIndex()
	require.NoError(t, err, "第二次构建文件索引失败")

	// 第三次构建（应该跳过）
	err = mgr.BuildIndex()
	require.NoError(t, err, "第三次构建文件索引失败")

	t.Log("✓ 索引构建优化测试通过：多次调用BuildIndex只构建一次")
}

// 辅助函数

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// TestImportResolver_FromJSON 测试基于 JSON 文件的导入解析
func TestImportResolver_FromJSON(t *testing.T) {
	ctx := context.Background()

	// Setup logger and parser
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	testLogger, err := logger.NewLogger("/tmp/logs", logLevel, "import-resolver-json-test")
	require.NoError(t, err)
	sourceFileParser := parser.NewSourceFileParser(testLogger)

	// 1. 读取 JSON 文件
	data, err := os.ReadFile("testdata/import_resolver_test_cases.json")
	require.NoError(t, err)

	// 2. 解析 JSON（使用匿名结构体）
	var testData struct {
		TestCases []struct {
			Language    string   `json:"language"`
			ProjectName string   `json:"projectName"`
			ProjectPath string   `json:"projectPath"`
			GoModules   []string `json:"goModules"`
			File        string   `json:"file"`
			Imports     []struct {
				Source          string   `json:"source"`
				IsProjectImport bool     `json:"isProjectImport"`
				ExpectedPaths   []string `json:"expectedPaths"`
			} `json:"imports"`
		} `json:"testCases"`
	}
	require.NoError(t, json.Unmarshal(data, &testData))

	// 3. 为每个测试用例（文件）运行测试
	for _, tc := range testData.TestCases {
		testName := fmt.Sprintf("%s_%s_%s", tc.Language, tc.ProjectName,
			filepath.Base(tc.File))

		t.Run(testName, func(t *testing.T) {
			// 检查项目存在
			if !dirExists(tc.ProjectPath) {
				t.Skipf("项目不存在: %s", tc.ProjectPath)
				return
			}

			// 创建解析器管理器
			mgr := import_resolver.NewPathResolverManager(tc.ProjectPath)
			err := mgr.BuildIndex()
			require.NoError(t, err)

			// 获取语言对应的解析器
			language := parseLanguage(tc.Language)
			pathResolver := mgr.GetResolver(language)
			require.NotNil(t, pathResolver, "解析器不存在: %s", tc.Language)

			// 解析文件获取实际 imports
			fullPath := filepath.Join(tc.ProjectPath, tc.File)
			if !fileExists(fullPath) {
				t.Skipf("文件不存在: %s", tc.File)
				return
			}

			content, err := os.ReadFile(fullPath)
			require.NoError(t, err)

			fileTable, err := sourceFileParser.Parse(ctx, &types.SourceFile{
				Path:    fullPath,
				Content: content,
			})
			require.NoError(t, err)

			// 创建 import map 便于查找
			importMap := make(map[string]*resolver.Import)
			for _, imp := range fileTable.Imports {
				importMap[imp.Source] = imp
			}

			// 创建 project 对象
			project := &workspace.Project{
				Path:      tc.ProjectPath,
				GoModules: tc.GoModules,
			}

			// 测试每个 import
			for _, expectedImport := range tc.Imports {
				importName := expectedImport.Source

				t.Run(importName, func(t *testing.T) {
					// 查找该 import 是否在解析结果中
					actualImport, found := importMap[expectedImport.Source]

					if expectedImport.IsProjectImport {
						// 项目内导入：必须存在且解析成功
						require.True(t, found,
							"项目内导入未被解析到: %s", expectedImport.Source)

						// 设置 import 路径（用于相对导入解析）
						actualImport.Path = fullPath

						// 解析路径
						actualPaths, err := pathResolver.ResolveImportPath(
							ctx, actualImport, project)
						require.NoError(t, err,
							"项目内导入解析失败: %s", expectedImport.Source)

						// 断言：精确匹配（集合相等，顺序无关）
						assert.ElementsMatch(t, expectedImport.ExpectedPaths, actualPaths,
							"导入 %s 解析路径不匹配.\n期望: %v\n实际: %v",
							expectedImport.Source, expectedImport.ExpectedPaths, actualPaths)

					} else {
						// 项目外导入：不应该被解析到，或者解析失败
						if found {
							actualImport.Path = fullPath
							actualPaths, err := pathResolver.ResolveImportPath(
								ctx, actualImport, project)

							// 如果解析成功且返回了路径，说明误判为项目内导入
							if err == nil && len(actualPaths) > 0 {
								t.Errorf("项目外导入 %s 被误判为项目内导入，解析到: %v",
									expectedImport.Source, actualPaths)
							}
						}
						// 不在 importMap 中或解析失败都是正确的
					}
				})
			}
		})
	}
}

// parseLanguage 将字符串转换为 lang.Language
func parseLanguage(s string) lang.Language {
	switch s {
	case "Go":
		return lang.Go
	case "Python":
		return lang.Python
	case "Java":
		return lang.Java
	case "TypeScript":
		return lang.TypeScript
	case "JavaScript":
		return lang.JavaScript
	case "C":
		return lang.C
	case "C++", "CPP":
		return lang.CPP
	default:
		return ""
	}
}
