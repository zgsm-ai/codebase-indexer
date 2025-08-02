package parser

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJavaResolver_ResolveImport(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		description string
	}{
		{
			name: "正常类导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ImportTest.java",
				Content: []byte(`package com.example;
import java.util.List;
import java.util.ArrayList;
import static java.lang.Math.PI;
import com.example.utils.*;`),
			},
			wantErr:     nil,
			description: "测试正常的Java导入解析",
		},
		{
			name: "包通配符导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/WildcardImportTest.java",
				Content: []byte(`package com.example;
import java.util.*;
import java.io.*;`),
			},
			wantErr:     nil,
			description: "测试包通配符导入解析",
		},
		{
			name: "静态导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/StaticImportTest.java",
				Content: []byte(`package com.example;
import static java.lang.Math.PI;
import static java.lang.Math.abs;
import static java.util.Collections.emptyList;`),
			},
			wantErr:     nil,
			description: "测试静态导入解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 验证导入解析
				fmt.Println(len(res.Imports))
				for _, importItem := range res.Imports {
					fmt.Printf("Import: %s", importItem.GetName())
					assert.NotEmpty(t, importItem.GetName())
					assert.Equal(t, types.ElementTypeImport, importItem.GetType())
				}
			}
		})
	}
}

func TestJavaResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	sourceFile := &types.SourceFile{
		Path:    "testdata/com/example/test/TestClass.java",
		Content: readFile("testdata/com/example/test/TestClass.java"),
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// 1. 平铺输出所有类的详细信息
	fmt.Println("\n【所有类详细信息】")
	for _, element := range res.Elements {
		cls, ok := element.(*resolver.Class)
		if !ok {
			continue
		}
		fmt.Printf("类名: %s\n", cls.GetName())
		fmt.Printf("  作用域: %v\n", cls.BaseElement.Scope)
		if len(cls.SuperClasses) > 0 {
			fmt.Printf("  父类: %v\n", cls.SuperClasses)
		} else {
			fmt.Printf("  父类: 无\n")
		}
		if len(cls.SuperInterfaces) > 0 {
			fmt.Printf("  实现接口: %v\n", cls.SuperInterfaces)
		} else {
			fmt.Printf("  实现接口: 无\n")
		}
		if len(cls.Fields) > 0 {
			fmt.Println("  字段:")
			for _, field := range cls.Fields {
				fmt.Printf("    %s %s %s\n", field.Modifier, field.Type, field.Name)
			}
		} else {
			fmt.Println("  字段: 无")
		}
		if len(cls.Methods) > 0 {
			fmt.Println("  方法:")
			for _, method := range cls.Methods {
				fmt.Printf("    %s %s %s(", method.Declaration.Modifier, method.Declaration.ReturnType, method.Declaration.Name)
				for i, param := range method.Declaration.Parameters {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%s %s", param.Type, param.Name)
				}
				fmt.Println(")")
			}
		} else {
			fmt.Println("  方法: 无")
		}
		fmt.Println("--------------------------------")
	}

	// 2. 断言所有类的结构和内容
	// 期望类信息
	expectedClasses := map[string]struct {
		Scope        types.Scope
		SuperClasses []string
		SuperIfaces  []string
	}{
		"ReportGenerator": {
			Scope:        types.ScopePackage,
			SuperClasses: nil,
			SuperIfaces:  []string{"Printable", "Savable"},
		},
		"ReportDetails": {
			Scope:        types.ScopeProject,
			SuperClasses: nil,
			SuperIfaces:  nil,
		},
		"InternalReview": {
			Scope:        types.ScopeClass,
			SuperClasses: nil,
			SuperIfaces:  nil,
		},
		"ReportMetadata": {
			Scope:        types.ScopePackage,
			SuperClasses: nil,
			SuperIfaces:  nil,
		},
		"User": {
			Scope:        types.ScopePackage,
			SuperClasses: nil,
			SuperIfaces:  nil,
		},
		"FinancialReport": {
			Scope:        types.ScopeProject, // 若有 types.ScopePublic 则用，否则用 ScopePackage
			SuperClasses: []string{"User"},
			SuperIfaces:  []string{"Printable", "Savable"},
		},
		"UserServiceImpl": {
			Scope:        types.ScopeProject,
			SuperClasses: []string{"BaseService"},
			SuperIfaces:  []string{"UserApi","String", "Loggable", "Serializable"},
		},
	}

	// 遍历所有期望类，逐一断言
	for className, want := range expectedClasses {
		found := false
		for _, element := range res.Elements {
			cls, ok := element.(*resolver.Class)
			if !ok {
				continue
			}
			if cls.GetName() != className {
				continue
			}
			found = true
			assert.Equal(t, want.Scope, cls.BaseElement.Scope, "类 %s 作用域不匹配", className)
			// 父类
			if want.SuperClasses != nil {
				assert.ElementsMatch(t, want.SuperClasses, cls.SuperClasses, "类 %s 父类不匹配", className)
			}
			// 接口
			if want.SuperIfaces != nil {
				assert.ElementsMatch(t, want.SuperIfaces, cls.SuperInterfaces, "类 %s 实现接口不匹配", className)
			}
		}
		assert.True(t, found, "未找到类: %s", className)
	}
}

func TestJavaResolver_ResolveVariable(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantVariables []resolver.Variable
		description   string
	}{
		{
			name: "TestVar.java 全变量类型校验",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestVar.java",
				Content: readFile("testdata/com/example/test/TestVar.java"),
			},
			wantErr: nil,
			wantVariables: []resolver.Variable{
				// 字段（成员变量）
				{BaseElement: &resolver.BaseElement{Name: "userId", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "isVerified", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "retryCount", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "loginAttempts", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "currentUser", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "shoppingCart", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},
				{BaseElement: &resolver.BaseElement{Name: "guestUser", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "tempUser", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "favoriteProducts", Type: types.ElementTypeVariable}, VariableType: []string{"Product"}},
				{BaseElement: &resolver.BaseElement{Name: "assignedRoles", Type: types.ElementTypeVariable}, VariableType: []string{"Role"}},
				{BaseElement: &resolver.BaseElement{Name: "userMap", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "adminMap", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "customerProfile", Type: types.ElementTypeVariable}, VariableType: []string{"Customer"}},
				{BaseElement: &resolver.BaseElement{Name: "sessionResult", Type: types.ElementTypeVariable}, VariableType: []string{"Result", "Session"}},
				{BaseElement: &resolver.BaseElement{Name: "accessToken", Type: types.ElementTypeVariable}, VariableType: []string{"Token"}},
				{BaseElement: &resolver.BaseElement{Name: "refreshToken", Type: types.ElementTypeVariable}, VariableType: []string{"Token"}},
				{BaseElement: &resolver.BaseElement{Name: "MAX_LOGIN_ATTEMPTS", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "auditLogger", Type: types.ElementTypeVariable}, VariableType: []string{"Logger"}},
				{BaseElement: &resolver.BaseElement{Name: "MIN_PASSWORD_LENGTH", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "MAX_PASSWORD_LENGTH", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "userRatings", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "categoryTree", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "onlineUsers", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "offlineUsers", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "jsonUserId", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "emailService", Type: types.ElementTypeVariable}, VariableType: []string{"EmailService"}},
				{BaseElement: &resolver.BaseElement{Name: "notificationService", Type: types.ElementTypeVariable}, VariableType: []string{"NotificationService"}},
				{BaseElement: &resolver.BaseElement{Name: "messagingService", Type: types.ElementTypeVariable}, VariableType: []string{"NotificationService"}},
				{BaseElement: &resolver.BaseElement{Name: "loginAttemptsCounter", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "isProcessing", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "successCount", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "failureCount", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "userSerializer", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "premiumProduct", Type: types.ElementTypeVariable}, VariableType: []string{"Product"}},
				{BaseElement: &resolver.BaseElement{Name: "highValueOrder", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},
				{BaseElement: &resolver.BaseElement{Name: "lowValueOrder", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},

				// UserSessionManager 内部类字段
				{BaseElement: &resolver.BaseElement{Name: "sessionDescription", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "activeSessions", Type: types.ElementTypeVariable}, VariableType: []string{"Session"}},
				{BaseElement: &resolver.BaseElement{Name: "managerName", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "managerVersion", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// processUserOperations 方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "userAge", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "errorMessage", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "maxRetries", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "retryCount1", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "recommendedProducts", Type: types.ElementTypeVariable}, VariableType: []string{"Product"}},
				{BaseElement: &resolver.BaseElement{Name: "grantedPermissions", Type: types.ElementTypeVariable}, VariableType: []string{"Permission"}},
				{BaseElement: &resolver.BaseElement{Name: "completedOrders", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},
				{BaseElement: &resolver.BaseElement{Name: "cancelledOrders", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},
				{BaseElement: &resolver.BaseElement{Name: "sessionId", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "complexOrderStructure", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},
				{BaseElement: &resolver.BaseElement{Name: "scores", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "coordinates", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "stringLength", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "orderProcessor", Type: types.ElementTypeVariable}, VariableType: []string{"Consumer", "Order"}},
				{BaseElement: &resolver.BaseElement{Name: "orderValidator", Type: types.ElementTypeVariable}, VariableType: []string{"Consumer", "Order"}},
				{BaseElement: &resolver.BaseElement{Name: "dataFile", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "dataReader", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "dataLine", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "multiplier", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "orderAmount", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "taxAmount", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "userHierarchy", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "optionalUserMaps", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},

				// authenticateUser 方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "isAuthenticated", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "securityContext", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "isValid", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "isAuthorized", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// UserServiceImpl 构造方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "normalizedName", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "validatedPort", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "serviceType", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "serviceCategory", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// processBatchOperations 方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "usernames", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "upper", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "ids", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "userKey", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "prices", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "discounted", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "tax", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// initializeService 静态方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "serviceCounter", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "defaultUsers", Type: types.ElementTypeVariable}, VariableType: []string{"User"}},
				{BaseElement: &resolver.BaseElement{Name: "minVersion", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "maxVersion", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "configKeys", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "keyValue", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// updateUserProfile 方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "validatedUserId", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "profileStatus", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "updateStatus", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "profileTags", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "userTags", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// UserSessionManager#manageSessions 方法内局部变量
				{BaseElement: &resolver.BaseElement{Name: "sessionCount", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "orderCache", Type: types.ElementTypeVariable}, VariableType: []string{"Order"}},
				{BaseElement: &resolver.BaseElement{Name: "i", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// 枚举enum constant
				{BaseElement: &resolver.BaseElement{Name: "RED", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "GREEN", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "BLUE", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "MERCURY", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "VENUS", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "EARTH", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "PLUS", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "MINUS", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "PENDING", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "COMPLETED", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "CIRCLE", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "SQUARE", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// enum里面的字段
				{BaseElement: &resolver.BaseElement{Name: "mass", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "radius", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "code", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "desc", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},

				// enum导致加入的局部变量
				{BaseElement: &resolver.BaseElement{Name: "color", Type: types.ElementTypeVariable}, VariableType: []string{"Color"}},
				{BaseElement: &resolver.BaseElement{Name: "earth", Type: types.ElementTypeVariable}, VariableType: []string{"Planet"}},
				{BaseElement: &resolver.BaseElement{Name: "result", Type: types.ElementTypeVariable}, VariableType: []string{types.PrimitiveType}},
				{BaseElement: &resolver.BaseElement{Name: "status", Type: types.ElementTypeVariable}, VariableType: []string{"Status"}},
			},
				description: "测试 TestVar.java 中所有变量的类型解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("--------------------------------\n")
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望变量数量: %d\n", len(tt.wantVariables))

				// 收集所有变量
				var actualVariables []*resolver.Variable
				for _, element := range res.Elements {
					if variable, ok := element.(*resolver.Variable); ok {
						actualVariables = append(actualVariables, variable)
						fmt.Printf("变量: %s, Type: %s, VariableType: %s\n", variable.GetName(), variable.GetType(), variable.VariableType)
					}
				}

				fmt.Printf("实际变量数量: %d\n", len(actualVariables))

				// 验证变量数量
				assert.Len(t, actualVariables, len(tt.wantVariables),
					"变量%s 数量不匹配，期望 %d，实际 %d", tt.name, len(tt.wantVariables), len(actualVariables))

				// 创建实际变量的映射
				actualVarMap := make(map[string]*resolver.Variable)
				for _, variable := range actualVariables {
					actualVarMap[variable.GetName()] = variable
				}

				// 逐个比较每个期望的变量
				for _, wantVariable := range tt.wantVariables {
					actualVariable, exists := actualVarMap[wantVariable.GetName()]
					assert.True(t, exists, "未找到变量: %s", wantVariable.GetName())

					if exists {
						// 验证变量名称
						assert.Equal(t, wantVariable.GetName(), actualVariable.GetName(),
							"变量 %s 名称不匹配，期望 %s，实际 %s", tt.name,
							wantVariable.GetName(), actualVariable.GetName())

						// 验证变量类型
						assert.Equal(t, wantVariable.GetType(), actualVariable.GetType(),
							"变量%s 类型不匹配，期望 %s，实际 %s", actualVariable.GetName(),
							wantVariable.GetType(), actualVariable.GetType())

						// 验证变量的 VariableType 字段
						assert.ElementsMatch(t, wantVariable.VariableType, actualVariable.VariableType,
							"变量 %s 的VariableType不匹配，期望 %v，实际 %v",
							wantVariable.GetName(), wantVariable.VariableType, actualVariable.VariableType)

					}
				}
			}
		})
	}
}

func TestJavaResolver_ResolveInterface(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantIfaceName string
		wantExtends   []string
		description   string
	}{
		{
			name: "简单接口声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/SimpleInterfaceTest.java",
				Content: []byte(`package com.example;
public interface SimpleInterface extends InterfaceA, InterfaceB, com.example.api.MyInterface{
    void doSomething();
    int getValue();
}`),
			},
			wantErr:       nil,
			wantIfaceName: "SimpleInterface",
			wantExtends:   []string{"InterfaceA", "InterfaceB", "MyInterface"},
			description:   "测试简单接口声明解析",
		},
		{
			name: "Printable接口声明",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestClass.java",
				Content: readFile("testdata/com/example/test/TestClass.java"),
			},
			wantErr:       nil,
			wantIfaceName: "Printable",
			wantExtends:   []string{},
			description:   "测试Printable接口声明解析",
		},
		{
			name: "Savable接口声明",
			sourceFile: &types.SourceFile{
				Path:    "testdata/com/example/test/TestClass.java",
				Content: readFile("testdata/com/example/test/TestClass.java"),
			},
			wantErr:       nil,
			wantIfaceName: "Savable",
			wantExtends:   []string{},
			description:   "测试Savable接口声明解析",
		},

	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				// 1. 收集所有接口
				ifaceMap := make(map[string]*resolver.Interface)
				for _, element := range res.Elements {
					if iface, ok := element.(*resolver.Interface); ok {
						ifaceMap[iface.GetName()] = iface
					}
				}

				// 2. 查找目标接口
				iface, exists := ifaceMap[tt.wantIfaceName]
				assert.True(t, exists, "未找到接口类型: %s", tt.wantIfaceName)
				if exists {
					assert.Equal(t, types.ElementTypeInterface, iface.GetType())

					// 验证方法数量
					assert.ElementsMatch(t, tt.wantExtends, iface.SuperInterfaces,
						"方法数量不匹配，期望 %v，实际 %v", tt.wantExtends, iface.SuperInterfaces)
				}
			}
		})
	}
}

func TestJavaResolver_ResolveLocalVariableValue(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)
	sourceFile := &types.SourceFile{
		Path: "testdata/com/example/test/TestClass.java",
		Content: []byte(`
			package com.example.test;
			public class TestClass {
				public void test() {
					int a = 1;
					String b = "hello";
					double c = 3.14;
					Person p = new Person("Alice", 30);
					Map<String, Integer> map = new HashMap<>();
					Set<Double> set = new HashSet<>();
					Runnable localRunnable = new Runnable() {
						@Override
						public void run() {
							System.out.println("Inner Runnable");
						}
					};
				}
			}
		`),
	}
	res, err := parser.Parse(context.Background(), sourceFile)
	assert.ErrorIs(t, err, nil)
	assert.NotNil(t, res)

	// 期望的变量名和类型
	expected := map[string]types.ElementType{
		"a":             types.ElementTypeVariable,
		"b":             types.ElementTypeVariable,
		"c":             types.ElementTypeVariable,
		"p":             types.ElementTypeVariable,
		"map":           types.ElementTypeVariable,
		"set":           types.ElementTypeVariable,
		"localRunnable": types.ElementTypeVariable,
	}

	found := map[string]bool{}
	cnt := 0
	for _, element := range res.Elements {
		if variable, ok := element.(*resolver.Variable); ok {
			cnt += 1
			name := variable.GetName()
			typ := variable.GetType()
			fmt.Println("name:", name, "typ:", typ)
			if wantType, ok := expected[name]; ok {
				assert.Equal(t, wantType, typ, "变量 %s 类型不匹配", name)
				found[name] = true
			}
		}
	}
	// 检查所有期望变量都被找到
	for name := range expected {
		assert.True(t, found[name], "未找到变量: %s", name)
	}
	if cnt != len(expected) {
		t.Errorf("变量数量不匹配，期望 %d，实际 %d", len(expected), cnt)
	}
}

func TestJavaResolver_ResolveCall(t *testing.T) {
}
func TestJavaResolver_AllResolveMethods(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	source := []byte(`
		package com.example.test;

		import java.util.List;
		import java.util.Map;
		import java.util.Set;
		import static java.lang.Math.PI;

		public class Base {
			protected int id;
		}

		public interface Named {
			String getName();
			
		}

		public interface Ageable {
			int getAge();
		}

		// 继承Base，实现Named接口
		public class Person extends Base implements Named {
			private String name;
			private int age;
			private List<String> tags;
			private Map<String, List<Integer>> scores;
			private Set<? extends Number> numbers;
			private static final double PI_VALUE = PI;
			public Person(String name, int age) {
				this.name = name;
				this.age = age;
			}
			public String getName() { return name; }
		}

		// 既继承Base又实现多个接口
		public class Student extends Base implements Named, Ageable {
			private String name;
			private int age;
			private List<String[]> matrix;
			public Student(String name, int age) {
				this.name = name;
				this.age = age;
			}
			public String getName() { return name; }
			public int getAge() { return age; }
		}

		// 只实现接口
		public class Teacher implements Named {
			private String name;
			public Teacher(String name) { this.name = name; }
			public String getName() { return name; }
		}

		public class TestClass {
			public void test() {
				int a = 1;
				Person p = new Person("Alice", 30);
				Student s = new Student("Bob", 20);
				Teacher t = new Teacher("Tom");
				double pi = PI;
				List<String> list = null;
				Map<String, Integer> map = null;
				Set<Double> set = null;
				List<String[]> matrix = null;
				sayHello();
			}
			public void sayHello() {}
		}
	`)

	sourceFile := &types.SourceFile{
		Path:    "testdata/com/example/test/AllTest.java",
		Content: source,
	}

	res, err := parser.Parse(context.Background(), sourceFile)
	assert.ErrorIs(t, err, nil)
	assert.NotNil(t, res)

	// 1. 包
	assert.NotNil(t, res.Package)
	fmt.Printf("【包】%s\n", res.Package.GetName())
	assert.Equal(t, "com.example.test", res.Package.GetName())

	// 2. 导入
	assert.NotNil(t, res.Imports)
	fmt.Printf("【导入】数量: %d\n", len(res.Imports))
	for _, ipt := range res.Imports {
		fmt.Printf("  导入: %s\n", ipt.GetName())
	}
	importNames := map[string]bool{}
	for _, ipt := range res.Imports {
		importNames[ipt.GetName()] = true
	}
	assert.True(t, importNames["java.util.List"])
	assert.True(t, importNames["java.util.Map"])
	assert.True(t, importNames["java.util.Set"])
	assert.True(t, importNames["java.lang.Math.PI"])

	// 3. 类
	for _, element := range res.Elements {
		if cls, ok := element.(*resolver.Class); ok {
			fmt.Printf("【类】%s, 字段: %d, 方法: %d, 继承: %v, 实现: %v\n",
				cls.GetName(), len(cls.Fields), len(cls.Methods), cls.SuperClasses, cls.SuperInterfaces)
			for _, field := range cls.Fields {
				fmt.Printf("  字段: %s %s %s\n", field.Modifier, field.Type, field.Name)
			}
			for _, method := range cls.Methods {
				fmt.Printf("  方法: %s %s %s(%v)\n", method.Declaration.Modifier, method.Declaration.ReturnType, method.Declaration.Name, method.Declaration.Parameters)
			}
		}
	}

	// 4. 接口
	for _, element := range res.Elements {
		if iface, ok := element.(*resolver.Interface); ok {
			fmt.Printf("【接口】%s, 方法: %d\n", iface.GetName(), len(iface.Methods))
			for _, method := range iface.Methods {
				fmt.Printf("  方法: %s %s %s(%v)\n", method.Modifier, method.ReturnType, method.Name, method.Parameters)
			}
		}
	}

	// 5. 变量
	for _, element := range res.Elements {
		if variable, ok := element.(*resolver.Variable); ok {
			fmt.Printf("【变量】%s, 类型: %s\n", variable.GetName(), variable.GetType())
		}
	}

	// 6. 方法调用
	for _, element := range res.Elements {
		if call, ok := element.(*resolver.Call); ok {
			fmt.Printf("【方法调用】%s, 所属: %s\n", call.GetName(), call.Owner)
			assert.Equal(t, types.ElementTypeMethodCall, call.GetType())
		}
	}
}
