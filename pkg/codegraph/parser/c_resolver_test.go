package parser

import (
	"context"
	"fmt"
	"testing"

	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"

	"github.com/stretchr/testify/assert"
)

func TestCResolver(t *testing.T) {

}
func TestCResolver_ResolveImport(t *testing.T) {
	logger := initLogger()                // 如果有日志初始化
	parser := NewSourceFileParser(logger) // 假设有类似 Java 的解析器
	prj := workspace.NewProjectInfo(lang.C, "pkg/codegraph/parser/testdata", []string{"test.c"})

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		description string
	}{
		{
			name: "正常头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ImportTest.c",
				Content: []byte(`#include "test.h"
#include <stdio.h>
#include "utils/helper.h"
`),
			},
			wantErr:     nil,
			description: "测试正常的C头文件导入解析",
		},
		{
			name: "系统头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/SystemImportTest.c",
				Content: []byte(`#include <stdlib.h>
#include <string.h>
`),
			},
			wantErr:     nil,
			description: "测试系统头文件导入解析",
		},
		{
			name: "相对路径头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/RelativeImportTest.c",
				Content: []byte(`#include "./local.h"
#include "../common.h"
`),
			},
			wantErr:     nil,
			description: "测试相对路径头文件导入解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)
			// fmt.Println("res",res)
			if err == nil {
				// 验证导入解析
				fmt.Println(len(res.Imports))
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s, FilePaths: %v\n", importItem.GetName(), importItem.FilePaths)
					assert.NotEmpty(t, importItem.GetName())
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
				}
			}
		})
	}
}

func TestCResolver_ResolveFunction(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.C, "pkg/codegraph/parser/testdata", []string{"functions.c"})

	cCode := `#include <stdio.h>
#include <stdlib.h>
#include <stdarg.h>
#include <string.h>

typedef struct { int x, y; } Point;
typedef enum { STATUS_OK, STATUS_ERROR } Status;
typedef int (*BinaryOp)(int, int);

int add(int a, int b) { return a + b; }
void print_message(const char *msg) { printf("Message: %s\n", msg); }
char* get_greeting() { return "Hello from C!"; }
void* create_buffer(size_t size) { return malloc(size); }
Point make_point(int x, int y) { Point p = {x, y}; return p; }
BinaryOp get_operator(char op) { int add(int a, int b) { return a + b; } int sub(int a, int b) { return a - b; } return (op == '-') ? sub : add; }
Status check_value(int val) { return val >= 0 ? STATUS_OK : STATUS_ERROR; }
void say_hello(void) { puts("Hello!"); }
void update_value(int *x) { *x += 1; }
void print_array(int arr[], int len) { for (int i = 0; i < len; i++) { printf("%d ", arr[i]); } puts(""); }
void move_point(Point p) { printf("Moving to (%d, %d)\n", p.x, p.y); }
void scale_point(Point *p, int factor) { p->x *= factor; p->y *= factor; }
void apply(int a, int b, BinaryOp op) { printf("Result: %d\n", op(a, b)); }
void logf(const char *fmt, ...) { va_list args; va_start(args, fmt); vprintf(fmt, args); va_end(args); }`

	sourceFile := &types.SourceFile{
		Path:    "testdata/functions.c",
		Content: []byte(cCode),
	}

	res, err := parser.Parse(context.Background(), sourceFile, prj)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	funcCount := 0
	fmt.Println("res.Elements", len(res.Elements))
	for _, elem := range res.Elements {
		if elem.GetType() == types.ElementTypeFunction {
			funcCount++
			fn, ok := elem.(*resolver.Function)
			assert.True(t, ok)
			fmt.Printf("Function: %s, ReturnType: %v, Parameters: %v\n", fn.GetName(), fn.Declaration.ReturnType, fn.Declaration.Parameters)
			assert.NotEmpty(t, fn.GetName())
			assert.Equal(t, types.ElementTypeFunction, fn.GetType())
			// 可根据需要添加更细致的断言
		}
	}
	fmt.Println("函数数量:", funcCount)
}
