package parser

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
)

func TestCPPResolver(t *testing.T) {

}
func TestCPPResolver_ResolveImport(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	prj := workspace.NewProjectInfo(lang.CPP, "pkg/codegraph/parser/testdata", []string{
		"test.cpp", "utils/helper.hpp", "local.hpp", "common.hpp", "nested/dir/deep.hpp", "special-@file.hpp", "space file.hpp",
	})

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		description string
	}{
		{
			name: "普通头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ImportTest.cpp",
				Content: []byte(`#include "test.h"
#include "utils/helper.hpp"
`),
			},
			wantErr:     nil,
			description: "测试普通C++头文件导入",
		},
		{
			name: "系统头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/SystemImportTest.cpp",
				Content: []byte(`#include <vector>
#include <string>
`),
			},
			wantErr:     nil,
			description: "测试系统头文件导入",
		},
		{
			name: "相对路径头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/RelativeImportTest.cpp",
				Content: []byte(`#include "./local.hpp"
#include "../common.hpp"
`),
			},
			wantErr:     nil,
			description: "测试相对路径头文件导入",
		},
		{
			name: "嵌套目录头文件导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/NestedImportTest.cpp",
				Content: []byte(`#include "nested/dir/deep.hpp"
`),
			},
			wantErr:     nil,
			description: "测试嵌套目录头文件导入",
		},
		{
			name: "混合引号和尖括号",
			sourceFile: &types.SourceFile{
				Path: "testdata/MixedImportTest.cpp",
				Content: []byte(`#include "test.h"
#include <map>
`),
			},
			wantErr:     nil,
			description: "测试混合引号和尖括号导入",
		},
		{
			name: "using声明导入",
			sourceFile: &types.SourceFile{		
				Path: "testdata/UsingImportTest.cpp",
				Content: []byte(`using namespace std;
using std::vector;
using myns::MyClass;
using myns::MyClass2;
`),
			},
			wantErr:     nil,
			description: "测试using声明导入",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile, prj)
			if tt.wantErr != nil {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
			assert.NotNil(t, res)
			if err == nil {
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s, FilePaths: %v\n", importItem.GetName(), importItem.FilePaths)
					assert.NotEmpty(t, importItem.GetName())
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
				}
			}
		})
	}
}


func TestCPPResolver_ResolveFunction(t *testing.T) {

}
