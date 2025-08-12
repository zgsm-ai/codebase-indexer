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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const JavaProjectRootDir = "../tmp/projects/java"

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

	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".java"},
	})
	testCases := []struct {
		Name    string
		Path    string
		wantErr error
	}{
		{
			Name:    "hadoop",
			Path:    filepath.Join(JavaProjectRootDir, "hadoop"),
			wantErr: nil,
		},
		{
			Name:    "mall",
			Path:    filepath.Join(JavaProjectRootDir, "mall"),
			wantErr: nil,
		},
		{
			Name:    "maven",
			Path:    filepath.Join(JavaProjectRootDir, "maven"),
			wantErr: nil,
		},
		{
			Name:    "elasticsearch",
			Path:    filepath.Join(JavaProjectRootDir, "elasticsearch"),
			wantErr: nil,
		},
		{
			Name:    "kafka",
			Path:    filepath.Join(JavaProjectRootDir, "kafka"),
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
			fileElements, _, err := indexer.ParseProjectFiles(context.Background(), project)
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

// 添加一个专门的性能基准测试
func BenchmarkParseJavaProject(b *testing.B) {
	env, err := setupTestEnvironment()
	if err != nil {
		b.Fatal(err)
	}
	defer teardownTestEnvironment(nil, env)

	indexer := createTestIndexer(env, &types.VisitPattern{
		ExcludeDirs: defaultVisitPattern.ExcludeDirs,
		IncludeExts: []string{".java"},
	})

	// 选择一个中等大小的项目进行基准测试
	projectPath := filepath.Join(JavaProjectRootDir, "kafka")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		project := NewTestProject(projectPath, env.logger)
		fileElements, _, err := indexer.ParseProjectFiles(context.Background(), project)
		if err != nil {
			b.Fatal(err)
		}
		_ = fileElements
	}
}
