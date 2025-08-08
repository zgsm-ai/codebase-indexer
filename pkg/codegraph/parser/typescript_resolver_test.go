package parser

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeScriptResolver_ResolveImport(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantImports []resolver.Import
		description string
	}{
		{
			name: "ES6导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_imports.ts",
				Content: []byte(`import { Component, OnInit } from '@angular/core';
import * as _ from 'lodash';
import moment from 'moment';
import type { User, UserSettings } from './types';
import React, { useState, useEffect } from 'react';`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "Component", Type: types.ElementTypeImport}, Source: "@angular/core"},
				{BaseElement: &resolver.BaseElement{Name: "OnInit", Type: types.ElementTypeImport}, Source: "@angular/core"},
				{BaseElement: &resolver.BaseElement{Name: "lodash", Type: types.ElementTypeImport}, Source: "lodash", Alias: "_"},
				{BaseElement: &resolver.BaseElement{Name: "moment", Type: types.ElementTypeImport}, Source: "moment"},
				{BaseElement: &resolver.BaseElement{Name: "User", Type: types.ElementTypeImport}, Source: "./types"},
				{BaseElement: &resolver.BaseElement{Name: "UserSettings", Type: types.ElementTypeImport}, Source: "./types"},
				{BaseElement: &resolver.BaseElement{Name: "React", Type: types.ElementTypeImport}, Source: "react"},
				{BaseElement: &resolver.BaseElement{Name: "useState", Type: types.ElementTypeImport}, Source: "react"},
				{BaseElement: &resolver.BaseElement{Name: "useEffect", Type: types.ElementTypeImport}, Source: "react"},
			},
			description: "测试TypeScript ES6导入语法",
		},
		{
			name: "动态导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_dynamic_imports.ts",
				Content: []byte(`async function loadModule() {
  const module = await import('./dynamic-module');
  return module;
}

const lazyLoad = () => import('./lazy-component');`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "module", Type: types.ElementTypeImport}, Source: "./dynamic-module"},
				{BaseElement: &resolver.BaseElement{Name: "lazyLoad", Type: types.ElementTypeImport}, Source: "./lazy-component"},
			},
			description: "测试TypeScript动态导入语法",
		},
		{
			name: "重命名导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_renamed_imports.ts",
				Content: []byte(`import { Component as AngularComponent } from '@angular/core';
import { useState as useStateHook } from 'react';
import * as lodash from 'lodash';`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "Component", Type: types.ElementTypeImport}, Source: "@angular/core", Alias: "AngularComponent"},
				{BaseElement: &resolver.BaseElement{Name: "useState", Type: types.ElementTypeImport}, Source: "react", Alias: "useStateHook"},
				{BaseElement: &resolver.BaseElement{Name: "lodash", Type: types.ElementTypeImport}, Source: "lodash", Alias: "lodash"},
			},
			description: "测试TypeScript重命名导入语法",
		},
		{
			name: "类型导入",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_type_imports.ts",
				Content: []byte(`import type { User, Admin } from './models';
import type { Response } from './api';
import type { ThemeConfig } from './theme';`),
			},
			wantErr: nil,
			wantImports: []resolver.Import{
				{BaseElement: &resolver.BaseElement{Name: "User", Type: types.ElementTypeImport}, Source: "./models"},
				{BaseElement: &resolver.BaseElement{Name: "Admin", Type: types.ElementTypeImport}, Source: "./models"},
				{BaseElement: &resolver.BaseElement{Name: "Response", Type: types.ElementTypeImport}, Source: "./api"},
				{BaseElement: &resolver.BaseElement{Name: "ThemeConfig", Type: types.ElementTypeImport}, Source: "./theme"},
			},
			description: "测试TypeScript类型导入语法",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望导入数量: %d, 实际导入数量: %d\n", len(tt.wantImports), len(res.Imports))

				// 验证导入数量
				assert.Len(t, res.Imports, len(tt.wantImports), "导入数量不匹配")

				// 创建实际导入的映射
				actualImports := make(map[string]*resolver.Import)
				for i, importItem := range res.Imports {
					// 添加基础字段断言
					assert.NotEmpty(t, importItem.GetPath(), "Import[%d] Path 不能为空", i)
					assert.NotEmpty(t, importItem.GetRange(), "Import[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(importItem.GetRange()), "Import[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, importItem.GetType(), "Import[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(importItem.Scope), "Import[%d] Scope 不能为空", i)

					key := fmt.Sprintf("%s_%s", importItem.GetName(), importItem.Source)
					actualImports[key] = importItem
				}

				// 逐个验证期望的导入
				for _, wantImport := range tt.wantImports {
					key := fmt.Sprintf("%s_%s", wantImport.GetName(), wantImport.Source)
					actualImport, exists := actualImports[key]
					assert.True(t, exists, "未找到导入: %s from %s", wantImport.GetName(), wantImport.Source)

					if exists {
						assert.Equal(t, wantImport.GetName(), actualImport.GetName())
						assert.Equal(t, wantImport.Source, actualImport.Source)
						assert.Equal(t, types.ElementTypeImport, actualImport.GetType())
						if wantImport.Alias != "" {
							assert.Equal(t, wantImport.Alias, actualImport.Alias)
						}
					}
				}
			}
		})
	}
}

func TestTypeScriptResolver_ResolveVariable(t *testing.T) {
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
			name: "TypeScript变量声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_variables.ts",
				Content: []byte(`
	// 基本类型变量
	let version: number = 4.5;
	var isStable: boolean = true;
	
	// 复杂类型变量
	const user: { name: string; age: number } = { name: 'Alice', age: 30 };
	const numbers: number[] = [1, 2, 3, 4, 5];
	const tuple: [string, number] = ['hello', 42];
	
	// 联合类型和类型别名
	const id: ID = 'abc123';
	let itemId: string | number = 101;
	
	// 泛型类型
	const items: Array<string> = ['a', 'b', 'c'];
	const dictionary: Map<string, number> = new Map();
	
	// 对象解构和类型
	interface Person {
		name: string;
		age: number;
		address?: string;
	}
	
	const { name: personName, age }: Person = { name: 'Bob', age: 25 };
	
	// 数组解构
	const [first, second, ...rest]: number[] = [1, 2, 3, 4, 5];
	
	// 类型断言
	const someValue: any = 'this is a string';
	const strLength: number = (someValue as string).length;
	
	// 字面量类型
	const direction: 'north' | 'south' | 'east' | 'west' = 'north';
	
	// 使用类引用作为类型
	const button: HTMLButtonElement = document.createElement('button');
	const listener: EventListener = (event) => console.log(event);
	`),
			},
			wantErr: nil,
			wantVariables: []resolver.Variable{
				{BaseElement: &resolver.BaseElement{Name: "version", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "isStable", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "user", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "numbers", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "tuple", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "id", Type: types.ElementTypeVariable}, VariableType: []string{"ID"}},
				{BaseElement: &resolver.BaseElement{Name: "itemId", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "items", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "dictionary", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "personName", Type: types.ElementTypeVariable}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "age", Type: types.ElementTypeVariable}, VariableType: []string{"Person"}},
				{BaseElement: &resolver.BaseElement{Name: "first", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "second", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "rest", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "someValue", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "strLength", Type: types.ElementTypeVariable}, VariableType: []string{"primitive_type"}},
				{BaseElement: &resolver.BaseElement{Name: "direction", Type: types.ElementTypeVariable}, VariableType: []string{"'north' | 'south' | 'east' | 'west'"}},
				{BaseElement: &resolver.BaseElement{Name: "button", Type: types.ElementTypeVariable}, VariableType: []string{"HTMLButtonElement"}},
			},
			description: "测试TypeScript各种变量声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望变量数量: %d\n", len(tt.wantVariables))

				// 收集所有变量
				var actualVariables []*resolver.Variable
				for _, element := range res.Elements {
					if variable, ok := element.(*resolver.Variable); ok {
						actualVariables = append(actualVariables, variable)
					}
				}

				fmt.Printf("实际变量数量: %d\n", len(actualVariables))

				// 验证变量数量精确匹配
				assert.Equal(t, len(tt.wantVariables), len(actualVariables),
					"变量数量不匹配，期望 %d，实际 %d", len(tt.wantVariables), len(actualVariables))

				// 创建实际变量的映射
				actualVarMap := make(map[string]*resolver.Variable)
				for i, variable := range actualVariables {
					// 添加基础字段断言
					assert.NotEmpty(t, variable.GetName(), "Variable[%d] Name 不能为空", i)
					assert.NotEmpty(t, variable.GetPath(), "Variable[%d] Path 不能为空", i)
					assert.NotEmpty(t, variable.GetRange(), "Variable[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(variable.GetRange()), "Variable[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, variable.GetType(), "Variable[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(variable.Scope), "Variable[%d] Scope 不能为空", i)

					actualVarMap[variable.GetName()] = variable
					fmt.Printf("变量: %s, Type: %s, VariableType: %v\n",
						variable.GetName(), variable.GetType(), variable.VariableType)
				}

				// 验证每个期望的变量
				foundCount := 0
				for _, wantVariable := range tt.wantVariables {
					actualVariable, exists := actualVarMap[wantVariable.GetName()]
					assert.True(t, exists, "未找到变量: %s", wantVariable.GetName())

					if exists {
						foundCount++
						// 验证变量名称和类型
						assert.Equal(t, wantVariable.GetName(), actualVariable.GetName(),
							"变量名称不匹配")
						assert.Equal(t, types.ElementTypeVariable, actualVariable.GetType(),
							"变量类型不匹配")

						// 验证变量的具体类型
						assert.Equal(t, wantVariable.VariableType, actualVariable.VariableType,
							"变量 %s 的VariableType不匹配，期望 %v，实际 %v",
							wantVariable.GetName(), wantVariable.VariableType, actualVariable.VariableType)
					}
				}

				// 验证找到了所有期望的变量
				assert.Equal(t, len(tt.wantVariables), foundCount,
					"找到的变量数量不匹配，期望 %d，实际 %d", len(tt.wantVariables), foundCount)
			}
		})
	}
}

func TestTypeScriptResolver_ResolveFunction(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name          string
		sourceFile    *types.SourceFile
		wantErr       error
		wantFunctions []resolver.Function
		description   string
	}{
		{
			name: "TypeScript函数声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_functions.ts",
				Content: []byte(`
	// 命名函数和参数类型
	function add(a: number, b: number): number {
		return a + b;
	}
	
	// 可选参数和默认参数
	function greet(name: string, greeting: string = 'Hello', suffix?: string): string {
		return "ss";
	}
	
	// 剩余参数
	function sum(...numbers: number[]): number {
		return numbers.reduce((total, num) => total + num, 0);
	}
	
	// 函数重载
	function process(value: string): string;
	function process(value: number): number;
	function process(value: string | number): string | number {
		if (typeof value === 'string') {
			return value.toUpperCase();
		} else {
			return value * 2;
		}
	}
	
	// 箭头函数
	const multiply = (a: number, b: number): number => a * b;
	const square = (x: number) => x * x;
	
	// 泛型函数
	function identity<T>(value: T): T {
		return value;
	}
	
	// 多泛型参数
	function pair<T, U>(first: T, second: U): [T, U] {
		return [first, second];
	}
	
	// 异步函数
	async function fetchData(url: string): Promise<any> {
		const response = await fetch(url);
		return response.json();
	}
	
	// 生成器函数
	function* idGenerator(): Generator<number> {
		let id = 1;
		while (true) {
			yield id++;
		}
	}
	`),
			},
			wantErr: nil,
			wantFunctions: []resolver.Function{
				{
					BaseElement: &resolver.BaseElement{Name: "add", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "add",
						Parameters: []resolver.Parameter{
							{Name: "a", Type: []string{"primitive_type"}},
							{Name: "b", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "greet", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "greet",
						Parameters: []resolver.Parameter{
							{Name: "name", Type: []string{"primitive_type"}},
							{Name: "greeting", Type: []string{"primitive_type"}},
							{Name: "suffix", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "sum", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "sum",
						Parameters: []resolver.Parameter{
							{Name: "numbers", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "process", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "process",
						Parameters: []resolver.Parameter{
							{Name: "value", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "process", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "process",
						Parameters: []resolver.Parameter{
							{Name: "value", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "process", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "process",
						Parameters: []resolver.Parameter{
							{Name: "value", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "multiply", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "multiply",
						Parameters: []resolver.Parameter{
							{Name: "a", Type: []string{"primitive_type"}},
							{Name: "b", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "square", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "square",
						Parameters: []resolver.Parameter{
							{Name: "x", Type: []string{"primitive_type"}},
						},
						ReturnType: nil,
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "identity", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "identity",
						Parameters: []resolver.Parameter{
							{Name: "value", Type: []string{"T"}},
						},
						ReturnType: []string{"T"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "pair", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "pair",
						Parameters: []resolver.Parameter{
							{Name: "first", Type: []string{"T"}},
							{Name: "second", Type: []string{"U"}},
						},
						ReturnType: []string{"[T, U]"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "fetchData", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name: "fetchData",
						Parameters: []resolver.Parameter{
							{Name: "url", Type: []string{"primitive_type"}},
						},
						ReturnType: []string{"primitive_type"},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "idGenerator", Type: types.ElementTypeFunction},
					Declaration: resolver.Declaration{
						Name:       "idGenerator",
						Parameters: []resolver.Parameter{},
						ReturnType: []string{"primitive_type"},
					},
				},
			},
			description: "测试TypeScript各种函数声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望函数数量: %d\n", len(tt.wantFunctions))

				// 收集所有函数
				var actualFunctions []*resolver.Function
				for _, element := range res.Elements {
					if function, ok := element.(*resolver.Function); ok {
						actualFunctions = append(actualFunctions, function)
					}
				}

				fmt.Printf("实际函数数量: %d\n", len(actualFunctions))

				// 验证函数数量精确匹配
				assert.Equal(t, len(tt.wantFunctions), len(actualFunctions),
					"函数数量不匹配，期望 %d，实际 %d", len(tt.wantFunctions), len(actualFunctions))

				// 创建实际函数的映射
				actualFuncMap := make(map[string]*resolver.Function)
				for i, function := range actualFunctions {
					// 添加基础字段断言
					assert.NotEmpty(t, function.GetName(), "Function[%d] Name 不能为空", i)
					assert.NotEmpty(t, function.GetPath(), "Function[%d] Path 不能为空", i)
					assert.NotEmpty(t, function.GetRange(), "Function[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(function.GetRange()), "Function[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, function.GetType(), "Function[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(function.Scope), "Function[%d] Scope 不能为空", i)

					actualFuncMap[function.GetName()] = function
					fmt.Printf("函数: %s, 参数数量: %d, 返回类型: %v\n",
						function.GetName(), len(function.Parameters), function.Declaration.ReturnType)
				}

				// 验证每个期望的函数
				foundCount := 0
				for _, wantFunction := range tt.wantFunctions {
					actualFunction, exists := actualFuncMap[wantFunction.GetName()]
					assert.True(t, exists, "未找到函数: %s", wantFunction.GetName())

					if exists {
						foundCount++
						// 验证函数名称和类型
						assert.Equal(t, wantFunction.GetName(), actualFunction.GetName(),
							"函数名称不匹配")
						assert.Equal(t, types.ElementTypeFunction, actualFunction.GetType(),
							"函数类型不匹配")

						// 验证参数数量精确匹配
						assert.Equal(t, len(wantFunction.Parameters), len(actualFunction.Parameters),
							"函数 %s 的参数数量不匹配，期望 %d，实际 %d",
							wantFunction.GetName(), len(wantFunction.Parameters), len(actualFunction.Parameters))

						// 验证每个参数
						for i, wantParam := range wantFunction.Parameters {
							if i < len(actualFunction.Parameters) {
								actualParam := actualFunction.Parameters[i]
								assert.Equal(t, wantParam.Name, actualParam.Name,
									"函数 %s 的第 %d 个参数名称不匹配", wantFunction.GetName(), i+1)
								assert.Equal(t, wantParam.Type, actualParam.Type,
									"函数 %s 的第 %d 个参数类型不匹配", wantFunction.GetName(), i+1)
							}
						}

						// 验证返回类型
						assert.Equal(t, wantFunction.Declaration.ReturnType, actualFunction.Declaration.ReturnType,
							"函数 %s 的返回类型不匹配", wantFunction.GetName())
					}
				}

				// 验证找到了所有期望的函数
				assert.Equal(t, len(tt.wantFunctions), foundCount,
					"找到的函数数量不匹配，期望 %d，实际 %d", len(tt.wantFunctions), foundCount)
			}
		})
	}
}

func TestTypeScriptResolver_ResolveInterface(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name           string
		sourceFile     *types.SourceFile
		wantErr        error
		wantInterfaces []resolver.Interface
		description    string
	}{
		{
			name: "TypeScript接口声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_interfaces.ts",
				Content: []byte(`
	// 基础接口
	interface User {
		id: number;
		name: string;
		email: string;
	}
	
	// 可选属性
	interface Config {
		endpoint: string;
		timeout?: number;
		retries?: number;
	}
	
	// 函数属性
	interface Validator {
		validate(value: string): boolean;
		format(value: string): string;
	}
	
	// 索引签名
	interface Dictionary {
		[key: string]: any;
		count: number; // 特定必需属性
	}
	
	// 扩展接口
	interface Employee extends User {
		role: string;
		department: string;
		salary: number;
	}
	
	// 多重扩展
	interface Admin extends Employee, Validator {
		permissions: string[];
	}
	
	// 泛型接口
	interface Repository<T> {
		getAll(): T[];
		getById(id: string): T;
		save(item: T): void;
	}
	`),
			},
			wantErr: nil,
			wantInterfaces: []resolver.Interface{
				{
					BaseElement: &resolver.BaseElement{Name: "User", Type: types.ElementTypeInterface},
					// User 接口只有属性，没有方法
				},
				{
					BaseElement: &resolver.BaseElement{Name: "Config", Type: types.ElementTypeInterface},
					// Config 接口只有属性，没有方法
				},
				{
					BaseElement: &resolver.BaseElement{Name: "Validator", Type: types.ElementTypeInterface},
					Methods: []*resolver.Declaration{
						{Name: "validate", Parameters: []resolver.Parameter{{Name: "value", Type: []string{"primitive_type"}}}, ReturnType: []string{"primitive_type"}},
						{Name: "format", Parameters: []resolver.Parameter{{Name: "value", Type: []string{"primitive_type"}}}, ReturnType: []string{"primitive_type"}},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "Dictionary", Type: types.ElementTypeInterface},
					// Dictionary 接口只有索引签名和属性，没有方法
				},
				{
					BaseElement:     &resolver.BaseElement{Name: "Employee", Type: types.ElementTypeInterface},
					SuperInterfaces: []string{"User"},
					// Employee 接口只有属性，没有方法
				},
				{
					BaseElement:     &resolver.BaseElement{Name: "Admin", Type: types.ElementTypeInterface},
					SuperInterfaces: []string{"Employee", "Validator"},
					// Admin 接口只有属性，没有方法
				},
				{
					BaseElement: &resolver.BaseElement{Name: "Repository", Type: types.ElementTypeInterface},
					Methods: []*resolver.Declaration{
						{Name: "getAll", ReturnType: []string{"primitive_type"}},
						{Name: "getById", Parameters: []resolver.Parameter{{Name: "id", Type: []string{"primitive_type"}}}, ReturnType: []string{"T"}},
						{Name: "save", Parameters: []resolver.Parameter{{Name: "item", Type: []string{"T"}}}, ReturnType: []string{"primitive_type"}},
					},
				},
			},
			description: "测试TypeScript各种接口声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望接口数量: %d\n", len(tt.wantInterfaces))

				// 收集所有接口
				var actualInterfaces []*resolver.Interface
				for _, element := range res.Elements {
					if iface, ok := element.(*resolver.Interface); ok {
						actualInterfaces = append(actualInterfaces, iface)
					}
				}

				fmt.Printf("实际接口数量: %d\n", len(actualInterfaces))

				// 验证接口数量
				assert.GreaterOrEqual(t, len(actualInterfaces), len(tt.wantInterfaces)-2,
					"接口数量过少，期望至少 %d，实际 %d", len(tt.wantInterfaces)-2, len(actualInterfaces))

				// 创建实际接口的映射
				actualIfaceMap := make(map[string]*resolver.Interface)
				for i, iface := range actualInterfaces {
					// 添加基础字段断言
					assert.NotEmpty(t, iface.GetName(), "Interface[%d] Name 不能为空", i)
					assert.NotEmpty(t, iface.GetPath(), "Interface[%d] Path 不能为空", i)
					assert.NotEmpty(t, iface.GetRange(), "Interface[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(iface.GetRange()), "Interface[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, iface.GetType(), "Interface[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(iface.Scope), "Interface[%d] Scope 不能为空", i)

					actualIfaceMap[iface.GetName()] = iface
					fmt.Printf("接口: %s, 方法数量: %d, 继承: %v\n",
						iface.GetName(), len(iface.Methods), iface.SuperInterfaces)
				}

				// 验证每个期望的接口
				foundCount := 0
				for _, wantInterface := range tt.wantInterfaces {
					actualInterface, exists := actualIfaceMap[wantInterface.GetName()]
					assert.True(t, exists, "未找到接口: %s", wantInterface.GetName())

					if exists {
						foundCount++
						// 验证接口名称和类型
						assert.Equal(t, wantInterface.GetName(), actualInterface.GetName(),
							"接口名称不匹配")
						assert.Equal(t, types.ElementTypeInterface, actualInterface.GetType(),
							"接口类型不匹配")

						// 验证继承关系
						if len(wantInterface.SuperInterfaces) > 0 {
							assert.Equal(t, len(wantInterface.SuperInterfaces), len(actualInterface.SuperInterfaces),
								"接口 %s 的继承数量不匹配，期望 %d，实际 %d",
								wantInterface.GetName(), len(wantInterface.SuperInterfaces), len(actualInterface.SuperInterfaces))

							// 验证每个继承的接口
							for i, expectedSuper := range wantInterface.SuperInterfaces {
								if i < len(actualInterface.SuperInterfaces) {
									assert.Equal(t, expectedSuper, actualInterface.SuperInterfaces[i],
										"接口 %s 的第 %d 个继承接口不匹配", wantInterface.GetName(), i+1)
								}
							}
						}

						// 验证方法数量（如果定义了期望的方法）
						if len(wantInterface.Methods) > 0 {
							assert.Equal(t, len(wantInterface.Methods), len(actualInterface.Methods),
								"接口 %s 的方法数量不匹配，期望 %d，实际 %d",
								wantInterface.GetName(), len(wantInterface.Methods), len(actualInterface.Methods))

							// 创建实际方法的映射
							actualMethodMap := make(map[string]*resolver.Declaration)
							for _, method := range actualInterface.Methods {
								actualMethodMap[method.Name] = method
							}

							// 验证每个期望的方法
							for _, wantMethod := range wantInterface.Methods {
								actualMethod, methodExists := actualMethodMap[wantMethod.Name]
								assert.True(t, methodExists, "接口 %s 中未找到方法: %s",
									wantInterface.GetName(), wantMethod.Name)

								if methodExists {
									// 验证方法参数
									assert.Equal(t, len(wantMethod.Parameters), len(actualMethod.Parameters),
										"接口 %s 的方法 %s 参数数量不匹配",
										wantInterface.GetName(), wantMethod.Name)

									// 验证返回类型
									assert.Equal(t, wantMethod.ReturnType, actualMethod.ReturnType,
										"接口 %s 的方法 %s 返回类型不匹配",
										wantInterface.GetName(), wantMethod.Name)

									// 验证每个参数
									for i, wantParam := range wantMethod.Parameters {
										if i < len(actualMethod.Parameters) {
											actualParam := actualMethod.Parameters[i]
											assert.Equal(t, wantParam.Name, actualParam.Name,
												"接口 %s 的方法 %s 第 %d 个参数名称不匹配",
												wantInterface.GetName(), wantMethod.Name, i+1)
											assert.Equal(t, wantParam.Type, actualParam.Type,
												"接口 %s 的方法 %s 第 %d 个参数类型不匹配",
												wantInterface.GetName(), wantMethod.Name, i+1)
										}
									}
								}
							}
						}
					}
				}

				// 验证找到了所有期望的接口
				assert.Equal(t, len(tt.wantInterfaces), foundCount,
					"找到的接口数量不匹配，期望 %d，实际 %d", len(tt.wantInterfaces), foundCount)
			}
		})
	}
}

func TestTypeScriptResolver_ResolveClass(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantClasses []resolver.Class
		description string
	}{
		{
			name: "TypeScript类声明",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_classes.ts",
				Content: []byte(`
	// 基本类
	class Person {
		// 字段
		name: string;
		age: number;
		
		// 构造函数
		constructor(name: string, age: number) {
			this.name = name;
			this.age = age;
		}
		
		// 方法
		greet(): string {
			return "Hello";
		}
	}
	
	// 继承
	class Employee extends Person {
		// 带有访问修饰符的字段
		private employeeId: number;
		protected department: string;
		public role: string;
		readonly startDate: Date;
		
		constructor(name: string, age: number, employeeId: number, department: string, role: string) {
			super(name, age);
			this.employeeId = employeeId;
			this.department = department;
			this.role = role;
			this.startDate = new Date();
		}
		
		// 覆盖方法
		greet(): string {
			return "Hello from Employee";
		}
		
		// 静态方法
		static createManager(name: string, age: number): Employee {
			return new Employee(name, age, 1000, 'Management', 'Manager');
		}
	}
	
	// 实现接口
	interface Printable {
		print(): void;
	}
	
	class Document implements Printable {
		content: string;
		
		constructor(content: string) {
			this.content = content;
		}
		
		print(): void {
			console.log(this.content);
		}
	}
	`),
			},
			wantErr: nil,
			wantClasses: []resolver.Class{
				{
					BaseElement: &resolver.BaseElement{Name: "Person", Type: types.ElementTypeClass},
					Fields: []*resolver.Field{
						{Name: "name", Type: "primitive_type"},
						{Name: "age", Type: "primitive_type"},
					},
					Methods: []*resolver.Method{
						{Declaration: resolver.Declaration{Name: "constructor", Modifier: "public"}},
						{Declaration: resolver.Declaration{Name: "greet", Modifier: "public", ReturnType: []string{"primitive_type"}}},
					},
				},
				{
					BaseElement:  &resolver.BaseElement{Name: "Employee", Type: types.ElementTypeClass},
					SuperClasses: []string{"Person"},
					Fields: []*resolver.Field{
						{Name: "employeeId", Type: "primitive_type", Modifier: "private"},
						{Name: "department", Type: "primitive_type", Modifier: "protected"},
						{Name: "role", Type: "primitive_type", Modifier: "public"},
						{Name: "startDate", Type: "primitive_type", Modifier: ""},
					},
					Methods: []*resolver.Method{
						{Declaration: resolver.Declaration{Name: "constructor", Modifier: "public"}},
						{Declaration: resolver.Declaration{Name: "greet", Modifier: "public", ReturnType: []string{"primitive_type"}}},
						{Declaration: resolver.Declaration{Name: "createManager", Modifier: "public", ReturnType: []string{"Employee"}}},
					},
				},
				{
					BaseElement: &resolver.BaseElement{Name: "Document", Type: types.ElementTypeClass},
					Fields: []*resolver.Field{
						{Name: "content", Type: "primitive_type"},
					},
					Methods: []*resolver.Method{
						{Declaration: resolver.Declaration{Name: "constructor", Modifier: "public"}},
						{Declaration: resolver.Declaration{Name: "print", Modifier: "public", ReturnType: []string{"primitive_type"}}},
					},
				},
			},
			description: "测试TypeScript各种类声明的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望类数量: %d\n", len(tt.wantClasses))

				// 收集所有类
				var actualClasses []*resolver.Class
				for _, element := range res.Elements {
					if class, ok := element.(*resolver.Class); ok {
						actualClasses = append(actualClasses, class)
					}
				}

				fmt.Printf("实际类数量: %d\n", len(actualClasses))

				// 验证类数量精确匹配
				assert.Equal(t, len(tt.wantClasses), len(actualClasses),
					"类数量不匹配，期望 %d，实际 %d", len(tt.wantClasses), len(actualClasses))

				// 创建实际类的映射
				actualClassMap := make(map[string]*resolver.Class)
				for i, class := range actualClasses {
					// 添加基础字段断言
					assert.NotEmpty(t, class.GetName(), "Class[%d] Name 不能为空", i)
					assert.NotEmpty(t, class.GetPath(), "Class[%d] Path 不能为空", i)
					assert.NotEmpty(t, class.GetRange(), "Class[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(class.GetRange()), "Class[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, class.GetType(), "Class[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(class.Scope), "Class[%d] Scope 不能为空", i)

					actualClassMap[class.GetName()] = class
					fmt.Printf("类: %s, 字段数量: %d, 方法数量: %d, 继承: %v, 实现: %v\n",
						class.GetName(), len(class.Fields), len(class.Methods),
						class.SuperClasses, class.SuperInterfaces)
				}

				// 验证每个期望的类
				foundCount := 0
				for _, wantClass := range tt.wantClasses {
					actualClass, exists := actualClassMap[wantClass.GetName()]
					assert.True(t, exists, "未找到类: %s", wantClass.GetName())

					if exists {
						foundCount++
						// 验证类名称和类型
						assert.Equal(t, wantClass.GetName(), actualClass.GetName(),
							"类名称不匹配")
						assert.Equal(t, types.ElementTypeClass, actualClass.GetType(),
							"类类型不匹配")

						// 验证继承关系
						assert.Equal(t, len(wantClass.SuperClasses), len(actualClass.SuperClasses),
							"类 %s 的继承数量不匹配，期望 %d，实际 %d",
							wantClass.GetName(), len(wantClass.SuperClasses), len(actualClass.SuperClasses))
						for i, expectedSuper := range wantClass.SuperClasses {
							if i < len(actualClass.SuperClasses) {
								assert.Equal(t, expectedSuper, actualClass.SuperClasses[i],
									"类 %s 的第 %d 个继承类不匹配", wantClass.GetName(), i+1)
							}
						}

						// 验证接口实现
						assert.Equal(t, len(wantClass.SuperInterfaces), len(actualClass.SuperInterfaces),
							"类 %s 的实现接口数量不匹配，期望 %d，实际 %d",
							wantClass.GetName(), len(wantClass.SuperInterfaces), len(actualClass.SuperInterfaces))
						for i, expectedInterface := range wantClass.SuperInterfaces {
							if i < len(actualClass.SuperInterfaces) {
								assert.Equal(t, expectedInterface, actualClass.SuperInterfaces[i],
									"类 %s 的第 %d 个实现接口不匹配", wantClass.GetName(), i+1)
							}
						}

						// 验证字段数量和详情
						assert.Equal(t, len(wantClass.Fields), len(actualClass.Fields),
							"类 %s 的字段数量不匹配，期望 %d，实际 %d",
							wantClass.GetName(), len(wantClass.Fields), len(actualClass.Fields))

						// 创建实际字段的映射
						actualFieldMap := make(map[string]*resolver.Field)
						for _, field := range actualClass.Fields {
							actualFieldMap[field.Name] = field
						}

						// 验证每个期望的字段
						for _, wantField := range wantClass.Fields {
							actualField, fieldExists := actualFieldMap[wantField.Name]
							assert.True(t, fieldExists, "类 %s 中未找到字段: %s",
								wantClass.GetName(), wantField.Name)

							if fieldExists {
								assert.Equal(t, wantField.Name, actualField.Name,
									"类 %s 的字段名称不匹配", wantClass.GetName())
								assert.Equal(t, wantField.Type, actualField.Type,
									"类 %s 的字段 %s 类型不匹配", wantClass.GetName(), wantField.Name)
								assert.Equal(t, wantField.Modifier, actualField.Modifier,
									"类 %s 的字段 %s 修饰符不匹配", wantClass.GetName(), wantField.Name)
							}
						}

						// 验证方法数量和详情
						assert.Equal(t, len(wantClass.Methods), len(actualClass.Methods),
							"类 %s 的方法数量不匹配，期望 %d，实际 %d",
							wantClass.GetName(), len(wantClass.Methods), len(actualClass.Methods))

						// 创建实际方法的映射
						actualMethodMap := make(map[string]*resolver.Method)
						for _, method := range actualClass.Methods {
							actualMethodMap[method.Declaration.Name] = method
						}

						// 验证每个期望的方法
						for _, wantMethod := range wantClass.Methods {
							actualMethod, methodExists := actualMethodMap[wantMethod.Declaration.Name]
							assert.True(t, methodExists, "类 %s 中未找到方法: %s",
								wantClass.GetName(), wantMethod.Declaration.Name)

							if methodExists {
								assert.Equal(t, wantMethod.Declaration.Name, actualMethod.Declaration.Name,
									"类 %s 的方法名称不匹配", wantClass.GetName())
								assert.Equal(t, wantMethod.Declaration.Modifier, actualMethod.Declaration.Modifier,
									"类 %s 的方法 %s 修饰符不匹配", wantClass.GetName(), wantMethod.Declaration.Name)
								assert.Equal(t, wantMethod.Declaration.ReturnType, actualMethod.Declaration.ReturnType,
									"类 %s 的方法 %s 返回类型不匹配", wantClass.GetName(), wantMethod.Declaration.Name)
							}
						}
					}
				}

				// 验证找到了所有期望的类
				assert.Equal(t, len(tt.wantClasses), foundCount,
					"找到的类数量不匹配，期望 %d，实际 %d", len(tt.wantClasses), foundCount)
			}
		})
	}
}

func TestTypeScriptResolver_ResolveMethodCalls(t *testing.T) {
	logger := initLogger()
	parser := NewSourceFileParser(logger)

	testCases := []struct {
		name        string
		sourceFile  *types.SourceFile
		wantErr     error
		wantCalls   []resolver.Call
		description string
	}{
		{
			name: "TypeScript方法调用",
			sourceFile: &types.SourceFile{
				Path: "testdata/ts_method_calls.ts",
				Content: []byte(`
	// 链式方法调用
	const result = calc.add(10).multiply(2);
	
	// 通过对象方法调用
	const math = {
		sum: (a: number, b: number) => a + b,
		diff: (a: number, b: number) => a - b
	};
	
	const sum = math.sum(10, 20);
	const diff = math.diff(30, 15);
	
	// DOM API 方法调用
	document.getElementById('app').addEventListener('click', () => {
		console.log('Clicked!');
	});
	
	// 嵌套方法调用
	console.log(math.sum(calc.add(5), calc.multiply(3)));
	
	// 方法调用与解构
	const { log, warn, error } = console;
	log('Info message');
	warn('Warning message');
	error('Error message');
	`),
			},
			wantErr: nil,
			wantCalls: []resolver.Call{
				{BaseElement: &resolver.BaseElement{Name: "add", Type: types.ElementTypeMethodCall}, Owner: "calc"},
				{BaseElement: &resolver.BaseElement{Name: "multiply", Type: types.ElementTypeMethodCall}, Owner: "calc.add(10)"},
				{BaseElement: &resolver.BaseElement{Name: "sum", Type: types.ElementTypeMethodCall}, Owner: "math"},
				{BaseElement: &resolver.BaseElement{Name: "diff", Type: types.ElementTypeMethodCall}, Owner: "math"},
				{BaseElement: &resolver.BaseElement{Name: "getElementById", Type: types.ElementTypeMethodCall}, Owner: "document"},
				{BaseElement: &resolver.BaseElement{Name: "addEventListener", Type: types.ElementTypeMethodCall}, Owner: "document.getElementById('app')"},
				{BaseElement: &resolver.BaseElement{Name: "log", Type: types.ElementTypeMethodCall}, Owner: "console"},
				{BaseElement: &resolver.BaseElement{Name: "log", Type: types.ElementTypeMethodCall}, Owner: "console"},
				{BaseElement: &resolver.BaseElement{Name: "sum", Type: types.ElementTypeMethodCall}, Owner: "math"},
				{BaseElement: &resolver.BaseElement{Name: "add", Type: types.ElementTypeMethodCall}, Owner: "calc"},
				{BaseElement: &resolver.BaseElement{Name: "multiply", Type: types.ElementTypeMethodCall}, Owner: "calc"},
				{BaseElement: &resolver.BaseElement{Name: "log", Type: types.ElementTypeFunctionCall}},
				{BaseElement: &resolver.BaseElement{Name: "warn", Type: types.ElementTypeFunctionCall}},
				{BaseElement: &resolver.BaseElement{Name: "error", Type: types.ElementTypeFunctionCall}},
			},
			description: "测试TypeScript各种方法调用的解析",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := parser.Parse(context.Background(), tt.sourceFile)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotNil(t, res)

			if err == nil {
				fmt.Printf("测试用例: %s\n", tt.name)
				fmt.Printf("期望调用数量: %d\n", len(tt.wantCalls))

				// 收集所有方法调用
				var actualCalls []*resolver.Call
				for _, element := range res.Elements {
					if element.GetType() == types.ElementTypeMethodCall || element.GetType() == types.ElementTypeFunctionCall {
						if call, ok := element.(*resolver.Call); ok {
							actualCalls = append(actualCalls, call)
						}
					}
				}

				fmt.Printf("实际调用数量: %d\n", len(actualCalls))

				// 打印所有找到的调用
				for i, call := range actualCalls {
					fmt.Printf("[%d] 调用: %s, 类型: %s, 所有者: %s\n",
						i+1, call.GetName(), call.GetType(), call.Owner)
				}

				// 验证调用数量精确匹配
				assert.Equal(t, len(tt.wantCalls), len(actualCalls),
					"调用数量不匹配，期望 %d，实际 %d", len(tt.wantCalls), len(actualCalls))

				// 创建实际调用的映射
				actualCallMap := make(map[string]*resolver.Call)
				for i, call := range actualCalls {
					// 添加基础字段断言
					assert.NotEmpty(t, call.GetName(), "Call[%d] Name 不能为空", i)
					assert.NotEmpty(t, call.GetPath(), "Call[%d] Path 不能为空", i)
					assert.NotEmpty(t, call.GetRange(), "Call[%d] Range 不能为空", i)
					assert.Equal(t, 4, len(call.GetRange()), "Call[%d] Range 应该包含4个元素", i)
					assert.NotEqual(t, types.ElementTypeUndefined, call.GetType(), "Call[%d] Type 不能为 undefined", i)
					assert.NotEmpty(t, string(call.Scope), "Call[%d] Scope 不能为空", i)

					key := fmt.Sprintf("%s_%s_%s", call.GetName(), call.Owner, call.GetType())
					actualCallMap[key] = call
				}

				// 验证每个期望的调用
				foundCount := 0
				for _, wantCall := range tt.wantCalls {
					key := fmt.Sprintf("%s_%s_%s", wantCall.GetName(), wantCall.Owner, wantCall.GetType())
					actualCall, exists := actualCallMap[key]
					assert.True(t, exists, "未找到调用: %s (所有者: %s, 类型: %s)",
						wantCall.GetName(), wantCall.Owner, wantCall.GetType())

					if exists {
						foundCount++
						// 验证调用名称和类型
						assert.Equal(t, wantCall.GetName(), actualCall.GetName(),
							"调用名称不匹配")
						assert.Equal(t, wantCall.GetType(), actualCall.GetType(),
							"调用类型不匹配")
						assert.Equal(t, wantCall.Owner, actualCall.Owner,
							"调用所有者不匹配")
					}
				}

				// 验证找到了所有期望的调用
				assert.Equal(t, len(tt.wantCalls), foundCount,
					"找到的调用数量不匹配，期望 %d，实际 %d", len(tt.wantCalls), foundCount)
			}
		})
	}
}
