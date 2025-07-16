package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"golang.org/x/tools/go/packages"
)

type GoResolver struct {
}

var _ ElementResolver = &GoResolver{}

func (r *GoResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, r, element, rc)
}

func (r *GoResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	// Use the existing BaseElement to store import information
	elements := []Element{element}

	// Set default import type
	element.Type = types.ElementTypeImport

	// If we have node information, extract more details from the nodes
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for _, capture := range rc.Match.Captures {
			// Skip missing or error nodes
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// Update root element's range and name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			switch {
			case strings.HasSuffix(nodeCaptureName, "import.name"):
				element.Name = content
				element.Content = []byte(content)
			case strings.HasSuffix(nodeCaptureName, "import.alias"):
				element.Alias = content
			case strings.HasSuffix(nodeCaptureName, "import.path"):
				// Extract the import path (removing quotes)
				path := strings.Trim(content, `"'`)
				element.Name = path
				element.Content = []byte(path)
				// Set filepaths as relative to project root
				if rc.ProjectInfo != nil && !rc.ProjectInfo.IsEmpty() {
					// Store as a slice even if it's just one path
					element.FilePaths = []string{path}
				}
			}
			element.Type = types.ElementTypeImport
		}
	}

	// Handle special case for standard library imports
	// if len(element.Name) > 0 {
	// 	if yes, _ := r.isStandardLibrary(element.Name); yes {
	// 		// Mark as standard library if detected
	// 		element.Source = "stdlib"
	// 	}
	// }

	return elements, nil
}

func (r *GoResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储包信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for _, capture := range rc.Match.Captures {
			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)
			switch nodeCaptureName {
			case string(types.ElementTypePackage) + types.Dot + "name":
				element.Name = content
				element.Content = []byte(content)
			case string(types.ElementTypePackage) + types.Dot + "path":
				// 包路径，如果有的话
			case string(types.ElementTypePackage) + types.Dot + "version":
				// 包版本，如果有的话
			}
		}
	}

	// 如果是标准库包，直接返回
	if yes, _ := r.isStandardLibrary(element.Name); yes {
		return elements, nil
	}

	// 处理项目特定的逻辑
	pj := rc.ProjectInfo

	// 如果项目信息为空，返回当前元素
	if pj.IsEmpty() {
		return elements, nil
	}

	// 尝试查找包对应的所有Go文件
	if rc.SourceFile != nil && rc.SourceFile.Path != "" {
		// 已经在处理源文件，不需要额外操作
		return elements, nil
	}

	return elements, nil
}

func (r *GoResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储函数信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		// 获取根捕获，它应该是 function_declaration 节点
		rootCapture := rc.Match.Captures[0]
		if rootCapture.Node.Kind() == "function_declaration" {
			// 获取函数声明节点
			funcNode := rootCapture.Node

			// 尝试获取返回类型节点
			resultNode := funcNode.ChildByFieldName("result")
			if resultNode != nil {
				// 提取返回类型文本
				returnType := resultNode.Utf8Text(rc.SourceFile.Content)
				element.ReturnType = returnType
			}
		}
		for _, capture := range rc.Match.Captures {

			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 使用updateRootElement更新Range和Name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			// 处理函数名
			if strings.HasSuffix(nodeCaptureName, "definition.function.name") {
				element.Declaration.Name = content
				element.BaseElement.Name = content // 使用BaseElement中的Name
			} else if strings.HasSuffix(nodeCaptureName, "definition.function.parameters") {
				// 去掉括号
				parameters := strings.Trim(content, "()")
				// 解析参数
				if parameters != "" {
					// 更智能的参数解析
					// 首先尝试查找所有类型标记位置
					// Go参数格式: name1, name2 type1, name3 type2, ...

					// 创建参数列表
					element.Parameters = make([]Parameter, 0)

					// 首先分析整个参数字符串
					typeGroups := analyzeParameterGroups(parameters)

					// 处理每个类型组
					for _, group := range typeGroups {
						// 获取参数类型
						paramType := group.Type

						// 处理每个参数名
						for _, name := range group.Names {
							element.Parameters = append(element.Parameters, Parameter{
								Name: name,
								Type: paramType,
							})
						}

					}
				}
			} else if strings.HasSuffix(nodeCaptureName, "function.return_type") {
				element.ReturnType = content
			}
		}
	}

	// 从原始函数声明中尝试提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		rootCapture := rc.Match.Captures[0]
		if rootCapture.Node.Kind() == "function_declaration" {
			funcNode := rootCapture.Node

			// 尝试解析返回类型
			if element.ReturnType == "" {
				resultNode := funcNode.ChildByFieldName("result")
				if resultNode != nil {
					element.ReturnType = resultNode.Utf8Text(rc.SourceFile.Content)
				}
			}

			// 如果参数解析失败，尝试直接从AST获取参数
			if len(element.Parameters) == 0 {
				paramsNode := funcNode.ChildByFieldName("parameters")
				if paramsNode != nil {
					// 遍历参数列表节点
					for i := uint(0); i < paramsNode.ChildCount(); i++ {
						paramNode := paramsNode.Child(i)
						if paramNode != nil && paramNode.Kind() == "parameter_declaration" {
							// 从参数声明节点提取名称和类型
							nameNode := paramNode.ChildByFieldName("name")
							typeNode := paramNode.ChildByFieldName("type")

							paramName := ""
							paramType := ""

							if nameNode != nil {
								paramName = nameNode.Utf8Text(rc.SourceFile.Content)
							}

							if typeNode != nil {
								paramType = typeNode.Utf8Text(rc.SourceFile.Content)
							}

							element.Parameters = append(element.Parameters, Parameter{
								Name: paramName,
								Type: paramType,
							})
						}
					}
				}
			}
		}
	}

	return elements, nil
}

// ParamGroup 表示一组共享同一类型的参数
type ParamGroup struct {
	Names []string // 参数名列表
	Type  string   // 共享的类型
}

// analyzeParameterGroups 分析Go语言的参数列表，将其分组为类型组
// 例如: "a, b int, c string" 将分为两组: {["a", "b"], "int"} 和 {["c"], "string"}
func analyzeParameterGroups(parameters string) []ParamGroup {
	var groups []ParamGroup

	// 分割多个参数组 (用逗号分隔)
	parts := strings.Split(parameters, ",")

	// 临时存储正在处理的参数名称
	var currentNames []string
	var currentType string
	var hasType bool

	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		// 检查这部分是否包含类型 (有空格的情况下)
		words := strings.Fields(part)

		if len(words) == 1 {
			// 只有一个词，可能是参数名或类型名

			// 查看后面是否还有部分
			if i < len(parts)-1 {
				// 查看下一部分是否包含类型信息
				nextPart := strings.TrimSpace(parts[i+1])
				nextWords := strings.Fields(nextPart)

				if len(nextWords) >= 2 {
					// 下一部分包含类型信息，所以这部分是单纯的参数名
					currentNames = append(currentNames, words[0])
					hasType = false
				} else {
					// 尝试查看是否是最后一个部分或者后面的部分构成类型
					isLastOrHasType := false
					for j := i + 1; j < len(parts); j++ {
						if len(strings.Fields(strings.TrimSpace(parts[j]))) >= 2 {
							isLastOrHasType = true
							break
						}
					}

					if isLastOrHasType {
						// 是参数名
						currentNames = append(currentNames, words[0])
						hasType = false
					} else {
						// 如果所有后续部分都只有一个词，则认为当前词是类型，前面积累的都是参数名
						// 这是类型信息
						if len(currentNames) > 0 {
							currentType = words[0]
							hasType = true

							// 保存这个组并重置
							groups = append(groups, ParamGroup{
								Names: append([]string{}, currentNames...),
								Type:  currentType,
							})

							currentNames = nil
							currentType = ""
							hasType = false
						} else {
							// 没有积累的参数名，这是单独的参数
							currentNames = append(currentNames, words[0])

							// 保存并重置
							groups = append(groups, ParamGroup{
								Names: append([]string{}, currentNames...),
								Type:  "",
							})

							currentNames = nil
							hasType = false
						}
					}
				}
			} else {
				// 最后一个部分，且只有一个词
				if len(currentNames) > 0 {
					// 如果前面有参数名，这是类型
					currentType = words[0]
					hasType = true

					// 保存这个组
					groups = append(groups, ParamGroup{
						Names: append([]string{}, currentNames...),
						Type:  currentType,
					})
				} else {
					// 没有前面的参数名，这是单独的参数
					groups = append(groups, ParamGroup{
						Names: []string{words[0]},
						Type:  "",
					})
				}

				// 重置
				currentNames = nil
				currentType = ""
				hasType = false
			}
		} else {
			// 有多个词，最后一个是类型，前面是参数名
			lastIdx := len(words) - 1
			paramName := strings.Join(words[:lastIdx], " ")
			paramType := words[lastIdx]

			// 如果已经有积累的参数名，先加上当前的参数名
			if len(currentNames) > 0 {
				currentNames = append(currentNames, paramName)
			} else {
				currentNames = []string{paramName}
			}

			currentType = paramType
			hasType = true

			// 保存这个组并重置
			groups = append(groups, ParamGroup{
				Names: append([]string{}, currentNames...),
				Type:  currentType,
			})

			currentNames = nil
			currentType = ""
			hasType = false
		}
	}

	// 处理可能没有保存的最后一组参数
	if len(currentNames) > 0 && !hasType {
		groups = append(groups, ParamGroup{
			Names: currentNames,
			Type:  "",
		})
	}

	return groups
}

func (r *GoResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储方法信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		// 获取根捕获，它应该是 function_declaration 节点
		rootCapture := rc.Match.Captures[0]
		if rootCapture.Node.Kind() == "function_declaration" {
			// 获取函数声明节点
			funcNode := rootCapture.Node

			// 尝试获取返回类型节点
			resultNode := funcNode.ChildByFieldName("result")
			if resultNode != nil {
				// 提取返回类型文本
				returnType := resultNode.Utf8Text(rc.SourceFile.Content)
				element.ReturnType = returnType
			}
		}
		for _, capture := range rc.Match.Captures {
			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 使用updateRootElement更新Range和Name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			// 处理方法名
			if strings.HasSuffix(nodeCaptureName, "method.name") {
				element.Declaration.Name = content
				element.BaseElement.Name = content // 使用BaseElement中的Name
			} else if strings.HasSuffix(nodeCaptureName, "method.parameters") {
				// 去掉括号
				parameters := strings.Trim(content, "()")
				// 解析参数
				if parameters != "" {
					params := strings.Split(parameters, ",")
					element.Parameters = make([]Parameter, len(params))
					for i, p := range params {
						// 分析参数名和类型
						parts := strings.Fields(strings.TrimSpace(p))
						if len(parts) >= 2 {
							element.Parameters[i] = Parameter{
								Name: parts[0],
								Type: strings.Join(parts[1:], " "),
							}
						} else if len(parts) == 1 && parts[0] != "" {
							element.Parameters[i] = Parameter{
								Type: parts[0],
							}
						}
					}
				}
			} else if strings.HasSuffix(nodeCaptureName, "method.return_type") {
				element.ReturnType = content
			} else if strings.HasSuffix(nodeCaptureName, "method.owner") {
				// 提取接收器类型
				element.Owner = content
			}
		}
	}

	// 从原始方法声明中尝试提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		rootCapture := rc.Match.Captures[0]
		if rootCapture.Node.Kind() == "method_declaration" {
			methodNode := rootCapture.Node

			// 尝试解析返回类型
			if element.ReturnType == "" {
				resultNode := methodNode.ChildByFieldName("result")
				if resultNode != nil {
					element.ReturnType = resultNode.Utf8Text(rc.SourceFile.Content)
				}
			}

			// 尝试解析接收器类型
			if element.Owner == "" {
				receiverNode := methodNode.ChildByFieldName("receiver")
				if receiverNode != nil {
					receiverText := receiverNode.Utf8Text(rc.SourceFile.Content)
					receiverText = strings.Trim(receiverText, "()")
					parts := strings.Fields(receiverText)
					if len(parts) >= 1 {
						// 最后一部分是类型，可能带*前缀
						receiverType := parts[len(parts)-1]
						element.Owner = strings.TrimPrefix(receiverType, "*")
					}
				}
			}
		}
	}

	return elements, nil
}

func (r *GoResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	// 将Go的struct视为类
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for _, capture := range rc.Match.Captures {
			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 使用updateRootElement更新Range和Name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			switch nodeCaptureName {
			case string(types.ElementTypeClass) + types.Dot + "name",
				string(types.ElementTypeStruct) + types.Dot + "name":
				element.Name = content
			case string(types.ElementTypeClass) + types.Dot + "field",
				string(types.ElementTypeStruct) + types.Dot + "field":
				// 解析字段
				fieldText := content
				parts := strings.Fields(fieldText)
				if len(parts) >= 2 {
					// 判断可见性（公有/私有）
					var visibility string
					fieldName := parts[0]
					if len(fieldName) > 0 && fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
						visibility = "public"
					} else {
						visibility = "private"
					}

					field := &Field{
						Modifier: visibility,
						Name:     fieldName,
						Type:     strings.Join(parts[1:], " "),
					}
					element.Fields = append(element.Fields, field)
				}
			case string(types.ElementTypeClass) + types.Dot + "base",
				string(types.ElementTypeStruct) + types.Dot + "base":
				// 处理基类/嵌入结构体（在Go中是嵌入）
			case string(types.ElementTypeClass) + types.Dot + "modifier",
				string(types.ElementTypeStruct) + types.Dot + "modifier":
				// 类修饰符处理（如果有）
			}
		}
	}

	return elements, nil
}

func (r *GoResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储变量信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for _, capture := range rc.Match.Captures {
			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 使用updateRootElement更新Range和Name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			// 处理变量名
			if strings.Contains(nodeCaptureName, ".name") {
				// 检查是否已经存在名称（可能处理多个变量声明时）
				if element.BaseElement.Name == "" {
					element.BaseElement.Name = content
				}
			} else if strings.Contains(nodeCaptureName, ".type") {
				// 变量类型
				element.Content = []byte(content)
			} else if strings.Contains(nodeCaptureName, ".value") {
				// 变量初始值，可以用Content存储
				if len(element.Content) == 0 { // 如果类型还没设置过
					element.Content = []byte(content)
				}
			}

			// 根据捕获名称确定作用域和类型
			if strings.Contains(nodeCaptureName, "global_variable") {
				// 全局变量
				element.Scope = types.ScopeFile
				element.Type = types.ElementTypeGlobalVariable
				// 判断变量名首字母是否大写，决定是包级别还是文件级别
				if len(element.BaseElement.Name) > 0 && element.BaseElement.Name[0] >= 'A' && element.BaseElement.Name[0] <= 'Z' {
					element.Scope = types.ScopePackage
				}
			} else if strings.Contains(nodeCaptureName, "local_variable") {
				// 局部变量
				element.Scope = types.ScopeFunction
				element.Type = types.ElementTypeLocalVariable
			} else if strings.Contains(nodeCaptureName, "constant") {
				// 常量
				element.Scope = types.ScopeFile
				element.Type = types.ElementTypeConstant
				// 判断常量名首字母是否大写，决定是包级别还是文件级别
				if len(element.BaseElement.Name) > 0 && element.BaseElement.Name[0] >= 'A' && element.BaseElement.Name[0] <= 'Z' {
					element.Scope = types.ScopePackage
				}
			} else if strings.Contains(nodeCaptureName, "field") {
				// 字段
				element.Scope = types.ScopeClass
			} else if strings.Contains(nodeCaptureName, "variable") {
				// 默认变量
				element.Scope = types.ScopeBlock
				element.Type = types.ElementTypeVariable
			}
		}
	}

	// 处理多变量声明（如 file, err := os.Open(...)）
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for i, capture := range rc.Match.Captures {
			if i == 0 { // 跳过根捕获
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			if strings.Contains(nodeCaptureName, ".name") && i > 1 { // 额外的变量名
				variableName := capture.Node.Utf8Text(rc.SourceFile.Content)

				// 只有当不是第一个变量时，才创建新元素
				if element.BaseElement.Name != variableName {
					newBase := NewBaseElement(uint32(capture.Index))
					newVariable := &Variable{
						BaseElement: newBase,
					}
					newVariable.BaseElement.Name = variableName
					newVariable.Type = element.Type
					newVariable.Scope = element.Scope
					newVariable.Content = element.Content

					elements = append(elements, newVariable)
				}
			}
		}
	}

	return elements, nil
}

func (r *GoResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储接口信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for _, capture := range rc.Match.Captures {
			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 使用updateRootElement更新Range和Name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			switch nodeCaptureName {
			case "definition.interface":
				// 处理整个接口定义
				node := capture.Node
				if node.Kind() == "type_declaration" {
					// 查找interface_type节点以获取方法定义
					var interfaceTypeNode *sitter.Node

					// 在type_spec中查找interface_type
					for i := uint(0); i < node.ChildCount(); i++ {
						child := node.Child(i)
						if child != nil && child.Kind() == "type_spec" {
							typeNode := child.ChildByFieldName("type")
							if typeNode != nil && typeNode.Kind() == "interface_type" {
								interfaceTypeNode = typeNode
								break
							}
						}
					}

					// 如果找到了interface_type节点，处理方法定义
					if interfaceTypeNode != nil {
						methodListNode := interfaceTypeNode.ChildByFieldName("method_list")
						if methodListNode != nil {
							// 提取方法信息
							for i := uint(0); i < methodListNode.ChildCount(); i++ {
								methodNode := methodListNode.Child(i)
								if methodNode == nil || methodNode.Kind() != "method_spec" {
									continue
								}

								// 创建一个方法声明
								decl := &Declaration{
									Name:       "",
									Parameters: []Parameter{},
								}

								// 获取方法名
								nameNode := methodNode.ChildByFieldName("name")
								if nameNode != nil {
									decl.Name = nameNode.Utf8Text(rc.SourceFile.Content)
								}

								// 处理参数列表
								parameterListNode := methodNode.ChildByFieldName("parameters")
								if parameterListNode != nil {
									for j := uint(0); j < parameterListNode.ChildCount(); j++ {
										paramNode := parameterListNode.Child(j)
										if paramNode == nil || paramNode.Kind() != "parameter_declaration" {
											continue
										}

										paramNameNode := paramNode.ChildByFieldName("name")
										paramTypeNode := paramNode.ChildByFieldName("type")

										paramName := fmt.Sprintf("arg%d", j)
										paramType := "unknown"

										if paramNameNode != nil {
											paramName = paramNameNode.Utf8Text(rc.SourceFile.Content)
										}

										if paramTypeNode != nil {
											paramType = paramTypeNode.Utf8Text(rc.SourceFile.Content)
										}

										param := Parameter{
											Name: paramName,
											Type: paramType,
										}
										decl.Parameters = append(decl.Parameters, param)
									}
								}

								// 处理返回类型
								resultNode := methodNode.ChildByFieldName("result")
								if resultNode != nil {
									decl.ReturnType = resultNode.Utf8Text(rc.SourceFile.Content)
								}

								element.Methods = append(element.Methods, decl)
							}
						}
					}
				}
			case "definition.interface.name":
				if element.Name == types.EmptyString {
					element.Name = content
				}
			case "interface.method":
				// 处理接口方法
				methodNode := capture.Node
				if methodNode.Kind() == "method_spec" {
					// 创建一个新的声明来表示方法
					decl := &Declaration{
						Name:       "",
						Parameters: []Parameter{},
					}

					// 遍历方法规范的子节点，寻找名称、参数和返回类型
					nameNode := methodNode.ChildByFieldName("name")
					if nameNode != nil {
						decl.Name = nameNode.Utf8Text(rc.SourceFile.Content)
					}

					// 处理参数列表
					parameterListNode := methodNode.ChildByFieldName("parameters")
					if parameterListNode != nil {
						for j := uint(0); j < parameterListNode.ChildCount(); j++ {
							paramNode := parameterListNode.Child(j)
							if paramNode == nil || paramNode.Kind() != "parameter_declaration" {
								continue
							}

							paramNameNode := paramNode.ChildByFieldName("name")
							paramTypeNode := paramNode.ChildByFieldName("type")

							paramName := fmt.Sprintf("arg%d", j)
							paramType := "unknown"

							if paramNameNode != nil {
								paramName = paramNameNode.Utf8Text(rc.SourceFile.Content)
							}

							if paramTypeNode != nil {
								paramType = paramTypeNode.Utf8Text(rc.SourceFile.Content)
							}

							param := Parameter{
								Name: paramName,
								Type: paramType,
							}
							decl.Parameters = append(decl.Parameters, param)
						}
					}

					// 处理返回类型
					resultNode := methodNode.ChildByFieldName("result")
					if resultNode != nil {
						decl.ReturnType = resultNode.Utf8Text(rc.SourceFile.Content)
					}

					element.Methods = append(element.Methods, decl)
				}
			case "interface.method.name":
				// 如果单独捕获了方法名，可以在这里处理
				// 通常会和method一起捕获处理
			case "interface.method.parameters":
				// 如果单独捕获了参数，可以在这里处理
				// 通常会和method一起捕获处理
			case "interface.method.return_type":
				// 如果单独捕获了返回类型，可以在这里处理
				// 通常会和method一起捕获处理
			}
		}
	}

	return elements, nil
}

func (r *GoResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储调用信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		for _, capture := range rc.Match.Captures {
			// 如果节点是missing或者error，则跳过
			if capture.Node.IsMissing() || capture.Node.IsError() {
				continue
			}

			nodeCaptureName := rc.CaptureNames[capture.Index]
			content := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 使用updateRootElement更新Range和Name
			updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

			switch nodeCaptureName {
			case "call.function":
				// 处理整个函数调用表达式
				callNode := capture.Node
				if callNode.Kind() == "call_expression" {
					// 尝试获取函数和参数节点
					funcNode := callNode.ChildByFieldName("function")
					argsNode := callNode.ChildByFieldName("arguments")

					if funcNode != nil {
						// 根据函数节点类型确定是函数调用还是方法调用
						if funcNode.Kind() == "identifier" {
							// 简单函数调用
							element.BaseElement.Name = funcNode.Utf8Text(rc.SourceFile.Content)
						} else if funcNode.Kind() == "selector_expression" {
							// 带包名/接收者的函数调用，如pkg.Func()或obj.Method()
							field := funcNode.ChildByFieldName("field")
							operand := funcNode.ChildByFieldName("operand")

							if field != nil && field.Kind() == "field_identifier" {
								element.BaseElement.Name = field.Utf8Text(rc.SourceFile.Content)
								if operand != nil {
									element.Owner = operand.Utf8Text(rc.SourceFile.Content)
								}
							}
						}
					}

					// 处理参数
					if argsNode != nil && argsNode.Kind() == "argument_list" && len(element.Parameters) == 0 {
						var paramIndex int = 0

						// 遍历参数列表，跳过括号和逗号
						for i := uint(0); i < argsNode.ChildCount(); i++ {
							argNode := argsNode.Child(i)
							if argNode == nil {
								continue
							}

							// 跳过括号和逗号等分隔符
							argKind := argNode.Kind()
							if argKind == "," || argKind == "(" || argKind == ")" {
								continue
							}

							// 处理实际参数
							param := &Parameter{
								Name: fmt.Sprintf("arg%d", paramIndex),
								Type: argKind,
							}
							element.Parameters = append(element.Parameters, param)
							paramIndex++
						}
					}
				}
			case "call.function.name":
				if element.BaseElement.Name == types.EmptyString {
					element.BaseElement.Name = content
				}
			case "call.function.owner":
				if element.Owner == types.EmptyString {
					element.Owner = content
				}
			case "call.function.arguments":
				// 如果参数已经从call.function处理过，则跳过
				if len(element.Parameters) > 0 {
					continue
				}

				// 处理参数列表
				argsText := content
				if strings.TrimSpace(argsText) != "" {
					// 去除首尾括号
					argsText = strings.TrimPrefix(argsText, "(")
					argsText = strings.TrimSuffix(argsText, ")")

					// 按逗号分割参数
					args := strings.Split(argsText, ",")
					for i, arg := range args {
						if strings.TrimSpace(arg) != "" {
							// 尝试确定参数类型
							argType := "unknown"
							trimmedArg := strings.TrimSpace(arg)
							if strings.HasPrefix(trimmedArg, "\"") && strings.HasSuffix(trimmedArg, "\"") {
								argType = "string_literal"
							} else if _, err := strconv.Atoi(trimmedArg); err == nil {
								argType = "int_literal"
							} else if strings.HasPrefix(trimmedArg, "true") || strings.HasPrefix(trimmedArg, "false") {
								argType = "boolean_literal"
							}

							param := &Parameter{
								Name: fmt.Sprintf("arg%d", i),
								Type: argType,
							}
							element.Parameters = append(element.Parameters, param)
						}
					}
				}
			}
		}
	}

	return elements, nil
}

func (g *GoResolver) isStandardLibrary(pkgPath string) (bool, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return false, fmt.Errorf("import_resolver load package: %v", err)
	}

	if len(pkgs) == 0 {
		return false, fmt.Errorf("import_resolver package not found: %s", pkgPath)
	}

	// 标准库包的PkgPath以"internal/"或非模块路径开头
	return !strings.Contains(pkgs[0].PkgPath, "."), nil
}
