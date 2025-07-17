package resolver

import (

	"fmt"
	"testing"
	"github.com/stretchr/testify/assert"

)

func TestParseParameters(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantParams  int
		wantNames   []string
		wantTypes   []string
		description string
	}{
		{
			name:        "无参数",
			content:     "()",
			wantParams:  0,
			description: "测试无参数解析",
		},
		{
			name:        "单个参数",
			content:     "(String name)",
			wantParams:  1,
			wantNames:   []string{"name"},
			wantTypes:   []string{"String"},
			description: "测试单个参数解析",
		},
		{
			name:        "多个参数",
			content:     "(int id, String name, boolean active)",
			wantParams:  3,
			wantNames:   []string{"id", "name", "active"},
			wantTypes:   []string{"int", "String", "boolean"},
			description: "测试多个参数解析",
		},
		{
			name:        "泛型参数",
			content:     "(List<String> items)",
			wantParams:  1,
			wantNames:   []string{"items"},
			wantTypes:   []string{"List<String>"},
			description: "测试泛型参数解析",
		},
		{
			name:        "带注解参数",
			content:     "(@NotNull String name)",
			wantParams:  1,
			wantNames:   []string{"name"},
			wantTypes:   []string{"String"},
			description: "测试带注解参数解析",
		},
		{
			name:        "可变参数",
			content:     "(int... numbers)",
			wantParams:  1,
			wantNames:   []string{"numbers"},
			wantTypes:   []string{"int..."},
			description: "测试可变参数解析",
		},
		{
			name:        "多个参数",
			content:     "(int a, Function<String, Integer> func, Runnable r, List<String[]> arrs,int... nums)",
			wantParams:  5,
			wantNames:   []string{"a", "func", "r", "arrs", "nums"},
			wantTypes:   []string{"int", "Function<String, Integer>", "Runnable", "List<String[]>", "int..."},
			description: "测试多个参数解析",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := parseParameters(tt.content)

			assert.Len(t, params, tt.wantParams)

			if tt.wantNames != nil {
				for i, param := range params {
					assert.Equal(t, tt.wantNames[i], param.Name)
					assert.Equal(t, tt.wantTypes[i], param.Type)
					fmt.Println(param)
				}
			}
		})
	}
}

func TestParseSingleParameter(t *testing.T) {
	tests := []struct {
		name        string
		paramStr    string
		wantName    string
		wantType    string
		description string
	}{
		{
			name:        "简单参数",
			paramStr:    "String name",
			wantName:    "name",
			wantType:    "String",
			description: "测试简单参数解析",
		},
		{
			name:        "带注解参数",
			paramStr:    "@NotNull String name",
			wantName:    "name",
			wantType:    "String",
			description: "测试带注解参数解析",
		},
		{
			name:        "泛型参数",
			paramStr:    "List<String> items",
			wantName:    "items",
			wantType:    "List<String>",
			description: "测试泛型参数解析",
		},
		{
			name:        "数组参数",
			paramStr:    "String[] names",
			wantName:    "names",
			wantType:    "String[]",
			description: "测试数组参数解析",
		},
		{
			name:        "可变参数",
			paramStr:    "int... numbers",
			wantName:    "numbers",
			wantType:    "int...",
			description: "测试可变参数解析",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := parseSingleParameter(tt.paramStr)
			assert.Equal(t, tt.wantName, param.Name)
			assert.Equal(t, tt.wantType, param.Type)
		})
	}
}

func TestParseParameters_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantParams  int
		wantNames   []string
		description string
	}{
		{
			name:        "缺少左括号但能解析",
			content:     "int a, String b)",
			wantParams:  2,
			wantNames:   []string{"a", "b"},
			description: "测试缺少左括号但能解析参数",
		},
		{
			name:        "缺少右括号",
			content:     "(int a, String b",
			wantParams:  2,
			wantNames:   []string{"a", "b"},
			description: "测试缺少右括号的参数解析",
		},
		{
			name:        "没有括号但能解析",
			content:     "int a, String b",
			wantParams:  2,
			wantNames:   []string{"a", "b"},
			description: "测试没有括号但能解析参数",
		},
		// ... 其他测试用例
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := parseParameters(tt.content)
			assert.Len(t, params, tt.wantParams)
			if tt.wantNames != nil {
				for i, param := range params {
					assert.Equal(t, tt.wantNames[i], param.Name)
				}
			}
		})
	}
}
