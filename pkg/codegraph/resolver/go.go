package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	treesitter "github.com/tree-sitter/go-tree-sitter"
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

			switch types.ToElementType(nodeCaptureName) {
			case types.ElementTypeImportName:
				element.Name = content
			case types.ElementTypeImportAlias:
				element.Alias = content
			case types.ElementTypeImportPath:
				// Extract the import path (removing quotes)
				path := strings.Trim(content, `"'`)

				// If there's no explicit name set (from import.name), use the last part of path as name
				if element.Name == "" {
					// Extract the last component of the path as the default name
					pathParts := strings.Split(path, "/")
					if len(pathParts) > 0 {
						element.Name = pathParts[len(pathParts)-1]
					} else {
						element.Name = path
					}
				}

			}
		}
	}

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
			switch types.ToElementType(nodeCaptureName) {
			case types.ElementTypePackageName:
				element.Name = content
			}
		}
	}

	// 如果是标准库包，直接返回
	if yes, _ := r.isStandardLibrary(element.Name); yes {
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
		rootCaptureName := rc.CaptureNames[rootCapture.Index]
		if rootCaptureName == string(types.ElementTypeFunction) {
			// 获取函数声明节点
			funcNode := rootCapture.Node

			// 尝试获取返回类型节点
			resultNode := funcNode.ChildByFieldName("result")
			if resultNode != nil {
				// 使用analyzeReturnTypes函数提取并格式化返回类型
				element.ReturnType = analyzeReturnTypes(resultNode, rc.SourceFile.Content)
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
			switch types.ToElementType(nodeCaptureName) {
			case types.ElementTypeFunctionName:
				element.Declaration.Name = content
				element.BaseElement.Name = content // 使用BaseElement中的Name
				element.Scope = analyzeScope(content)
			case types.ElementTypeFunctionParameters:
				// 去掉括号
				parameters := strings.Trim(content, "()")
				// 解析参数
				if parameters != "" {
					// 创建参数列表
					element.Parameters = make([]Parameter, 0)

					// 分析整个参数字符串
					typeGroups := analyzeParameterGroups(parameters)

					// 处理每个类型组
					for _, group := range typeGroups {
						// 获取参数类型
						paramTypes := group.Type

						// 处理每个参数名
						for _, name := range group.Names {
							element.Parameters = append(element.Parameters, Parameter{
								Name: name,
								Type: paramTypes,
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
	Type  []string // 共享的类型
}

// analyzeParameterGroups 分析Go语言的参数列表，将其分组为类型组
func analyzeParameterGroups(parameters string) []ParamGroup {
	// 特殊处理函数类型参数的情况
	if strings.Contains(parameters, "func(") {
		// 首先需要正确分割多个参数，处理括号嵌套
		var parts []string
		var currentPart strings.Builder
		depth := 0

		for i := 0; i < len(parameters); i++ {
			char := parameters[i]
			switch char {
			case '(':
				depth++
				currentPart.WriteByte(char)
			case ')':
				depth--
				currentPart.WriteByte(char)
			case ',':
				if depth == 0 {
					// 外层逗号，分割参数
					parts = append(parts, currentPart.String())
					currentPart.Reset()
				} else {
					// 括号内的逗号，保留
					currentPart.WriteByte(char)
				}
			default:
				currentPart.WriteByte(char)
			}
		}

		// 添加最后一个部分
		if currentPart.Len() > 0 {
			parts = append(parts, currentPart.String())
		}

		// 处理每个部分
		var groups []ParamGroup
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			// 分离参数名和类型
			words := strings.SplitN(part, " ", 2)
			if len(words) == 2 {
				// 名称和类型
				paramName := words[0]
				paramType := words[1]

				// 处理函数类型参数中的括号平衡
				if strings.Contains(paramType, "func(") {
					leftCount := strings.Count(paramType, "(")
					rightCount := strings.Count(paramType, ")")

					// 补充缺失的右括号
					if rightCount < leftCount {
						for i := 0; i < leftCount-rightCount; i++ {
							paramType += ")"
						}
					}
				}

				groups = append(groups, ParamGroup{
					Names: []string{paramName},
					Type:  []string{paramType},
				})
			} else if len(words) == 1 {
				// 只有参数名或者类型
				paramValue := words[0]

				if strings.Contains(paramValue, "func(") {
					// 是函数类型
					leftCount := strings.Count(paramValue, "(")
					rightCount := strings.Count(paramValue, ")")

					// 补充缺失的右括号
					if rightCount < leftCount {
						for i := 0; i < leftCount-rightCount; i++ {
							paramValue += ")"
						}
					}

					groups = append(groups, ParamGroup{
						Names: []string{""},
						Type:  []string{paramValue},
					})
				} else {
					// 普通参数
					groups = append(groups, ParamGroup{
						Names: []string{paramValue},
						Type:  []string{""},
					})
				}
			}
		}

		return groups
	}

	// 下面是原始的参数解析逻辑
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
								Type:  []string{currentType},
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
								Type:  []string{""},
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
						Type:  []string{currentType},
					})
				} else {
					// 没有前面的参数名，这是单独的参数
					groups = append(groups, ParamGroup{
						Names: []string{words[0]},
						Type:  []string{""},
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
				Type:  []string{currentType},
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
			Type:  []string{""},
		})
	}

	return groups
}

func analyzeScope(content string) types.Scope {
	if len(content) > 0 && content[0] >= 'A' && content[0] <= 'Z' {
		return types.ScopeProject
	}
	return types.ScopePackage
}

func (r *GoResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {

	// 使用现有的BaseElement存储方法信息
	elements := []Element{element}

	// 如果有节点信息，从节点中提取更多信息
	if rc.Match != nil && rc.Match.Captures != nil && len(rc.Match.Captures) > 0 {
		// 获取根捕获，它应该是 definition.method节点
		rootCapture := rc.Match.Captures[0]
		rootCaptureName := rc.CaptureNames[rootCapture.Index]
		if rootCaptureName == string(types.ElementTypeMethod) {
			// 获取方法声明节点
			methodNode := rootCapture.Node

			// 尝试获取返回类型节点
			resultNode := methodNode.ChildByFieldName("result")
			if resultNode != nil {
				// 使用analyzeReturnTypes函数提取并格式化返回类型
				element.ReturnType = analyzeReturnTypes(resultNode, rc.SourceFile.Content)
			}

			// 尝试获取接收器节点
			receiverNode := methodNode.ChildByFieldName("receiver")
			if receiverNode != nil {
				receiverText := receiverNode.Utf8Text(rc.SourceFile.Content)
				// 去掉括号
				receiverText = strings.Trim(receiverText, "()")
				parts := strings.Fields(receiverText)
				if len(parts) >= 1 {
					// 最后一个部分是类型，可能带*前缀
					receiverType := parts[len(parts)-1]
					element.Owner = strings.TrimPrefix(receiverType, "*")
				}
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
			switch types.ToElementType(nodeCaptureName) {
			case types.ElementTypeMethodName:
				element.Declaration.Name = content
				element.BaseElement.Name = content // 使用BaseElement中的Name
				element.Scope = analyzeScope(content)
			case types.ElementTypeMethodParameters:
				// 去掉括号
				parameters := strings.Trim(content, "()")
				// 解析参数
				if parameters != "" {
					// 创建参数列表
					element.Parameters = make([]Parameter, 0)

					// 分析整个参数字符串
					typeGroups := analyzeParameterGroups(parameters)

					// 处理每个类型组
					for _, group := range typeGroups {
						// 获取参数类型
						paramTypes := group.Type

						// 处理每个参数名
						for _, name := range group.Names {
							element.Parameters = append(element.Parameters, Parameter{
								Name: name,
								Type: paramTypes,
							})
						}
					}
				}
			case types.ElementTypeMethodReceiver:
				// 提取接收器类型
				if element.Owner == "" {
					// 去除可能的括号和前缀
					receiverText := strings.Trim(content, "()")
					parts := strings.Fields(receiverText)
					if len(parts) >= 1 {
						// 最后一个部分是类型，可能带*前缀
						receiverType := parts[len(parts)-1]
						element.Owner = strings.TrimPrefix(receiverType, "*")
					} else {
						element.Owner = content
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

			switch types.ToElementType(nodeCaptureName) {
			case types.ElementTypeStructName:
				element.Name = content
				element.Scope = analyzeScope(content)

			case types.ElementTypeStructType:
				structTypeNode := capture.Node
				// 获取field_declaration_list - 遍历所有子节点找到字段列表
				var fieldListNode *sitter.Node

				// 在struct_type节点中查找field_declaration_list
				for i := uint(0); i < structTypeNode.ChildCount(); i++ {
					child := structTypeNode.Child(i)
					if child != nil && types.ToNodeKind(child.Kind()) == types.NodeKindFieldList {
						fieldListNode = child
						break
					}

				}
				// 遍历所有field_declaration子节点
				for j := uint(0); j < fieldListNode.ChildCount(); j++ {
					fieldNode := fieldListNode.Child(j)
					if fieldNode != nil && types.ToNodeKind(fieldNode.Kind()) == types.NodeKindField {
						// 获取字段名和类型
						nameNode := fieldNode.ChildByFieldName("name")
						typeNode := fieldNode.ChildByFieldName("type")

						if typeNode != nil {
							fieldType := typeNode.Utf8Text(rc.SourceFile.Content)
							var fieldName string

							if nameNode != nil {
								// 有名称的普通字段
								fieldName = nameNode.Utf8Text(rc.SourceFile.Content)
							} else {
								// 匿名嵌入字段，使用类型作为名称
								fieldName = fieldType
							}

							// 判断可见性（公有/私有）
							visibility := types.ScopeProject
							if len(fieldName) > 0 && fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
								visibility = types.ScopePackage
							}

							field := &Field{
								Modifier: string(visibility),
								Name:     fieldName,
								Type:     fieldType,
							}
							element.Fields = append(element.Fields, field)
						}
					}
				}
			}
		}
	}

	// 设置元素类型
	element.Type = types.ElementTypeStruct

	return elements, nil
}

func (r *GoResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	// 存储所有元素（变量和可能的引用）
	elements := []Element{element}

	// 存储变量引用
	var references []*Reference

	// 如果没有匹配信息，直接返回
	if rc.Match == nil || rc.Match.Captures == nil || len(rc.Match.Captures) == 0 {
		return elements, nil
	}

	// 获取根捕获，它应该是变量声明节点
	rootCapture := rc.Match.Captures[0]
	rootCaptureName := rc.CaptureNames[rootCapture.Index]
	updateRootElement(element, &rootCapture, rootCaptureName, rc.SourceFile.Content)

	// 设置变量名称 - 使用节点文本作为变量名
	element.BaseElement.Name = rootCapture.Node.Utf8Text(rc.SourceFile.Content)

	// 设置默认元素类型和作用域
	element.Type = types.ElementTypeVariable
	element.Scope = types.ScopeBlock

	// 根据捕获名称设置元素类型和作用域
	switch types.ToElementType(rootCaptureName) {
	case types.ElementTypeGlobalVariable:
		element.Type = types.ElementTypeGlobalVariable
		// 根据名称首字母判断作用域
		element.Scope = analyzeScope(element.BaseElement.Name)
	case types.ElementTypeVariable:
		element.Type = types.ElementTypeVariable
		element.Scope = types.ScopeFunction
	case types.ElementTypeLocalVariable:
		element.Type = types.ElementTypeLocalVariable
		element.Scope = types.ScopeFunction
		// 处理多变量声明
		elements = r.processMultipleVariableDeclaration(rootCapture, element, rc, elements, &references)

	case types.ElementTypeConstant:
		element.Type = types.ElementTypeConstant
		element.Scope = types.ScopeFunction
	}

	// 处理所有捕获节点
	for _, capture := range rc.Match.Captures {
		// 跳过无效节点或根节点
		if capture.Node.IsMissing() || capture.Node.IsError() || capture.Index == rootCapture.Index {
			continue
		}

		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)

		// 根据节点类型处理不同信息
		// 需要同时处理const，variable，local_variable
		switch {
		case strings.HasSuffix(nodeCaptureName, ".type"):

			// 检查是否为基本数据类型
			if isPrimitiveType(content) {
				// 设置为基本数据类型
				element.VariableType = []string{types.PrimitiveType}
			} else {
				// 保持原有类型
				element.VariableType = []string{content}
			}

		case strings.HasSuffix(nodeCaptureName, ".value"):
			// 变量值处理
			// 使用processVariableValue函数处理引用类型
			r.processVariableValue(&capture.Node, uint32(capture.Index), rc.SourceFile.Content, element.Path, &references)
		}
	}

	// 添加引用元素
	for _, ref := range references {
		elements = append(elements, ref)
	}

	return elements, nil
}

// processMultipleVariableDeclaration 处理Go中的多变量声明（如 a, b := 1, 2）
func (r *GoResolver) processMultipleVariableDeclaration(rootCapture treesitter.QueryCapture, element *Variable, rc *ResolveContext, elements []Element, references *[]*Reference) []Element {
	// 查找当前变量的父节点
	var parentNode *sitter.Node
	if rootCapture.Node.Parent() != nil {
		parentNode = rootCapture.Node.Parent().Parent() // 获取 short_var_declaration 节点
	}

	if parentNode != nil && parentNode.Kind() == string(types.NodeKindShortVarDeclaration) {
		// 获取左侧和右侧节点
		leftNode := parentNode.ChildByFieldName("left")
		rightNode := parentNode.ChildByFieldName("right")

		if leftNode != nil && rightNode != nil {
			// 收集所有变量名
			var varNames []*sitter.Node
			for i := uint(0); i < leftNode.ChildCount(); i++ {
				id := leftNode.Child(i)
				if id != nil && id.Kind() == string(types.NodeKindIdentifier) {
					varNames = append(varNames, id)
				}
			}

			// 收集所有右值表达式
			var varValues []*sitter.Node
			for i := uint(0); i < rightNode.ChildCount(); i++ {
				expr := rightNode.Child(i)
				// 过滤掉逗号等分隔符节点
				if expr != nil && expr.Kind() != "," {
					varValues = append(varValues, expr)
				}
			}

			// 找到当前变量在变量名列表中的位置
			currentVarIndex := -1
			for i, nameNode := range varNames {
				if nameNode.Utf8Text(rc.SourceFile.Content) == element.BaseElement.Name {
					currentVarIndex = i
					break
				}
			}

			// 如果找到了当前变量的位置，并且有对应的值，则更新内容
			if currentVarIndex >= 0 && currentVarIndex < len(varValues) {
				valueNode := varValues[currentVarIndex]

				// 使用新函数处理引用类型
				r.processVariableValue(valueNode, uint32(1000+currentVarIndex), rc.SourceFile.Content, element.Path, references)
			}

			// 为其他变量创建新的元素
			for i, nameNode := range varNames {
				// 跳过当前变量
				if i == currentVarIndex {
					continue
				}

				// 创建新的变量元素
				newBase := NewBaseElement(uint32(i + 1000)) // 使用一个不会与现有捕获冲突的ID
				newVariable := &Variable{
					BaseElement: newBase,
				}

				// 设置变量名称
				newVariable.BaseElement.Name = nameNode.Utf8Text(rc.SourceFile.Content)
				newVariable.BaseElement.Path = element.Path
				// 设置类型和作用域
				newVariable.Type = element.Type
				newVariable.Scope = element.Scope

				// 设置范围
				newVariable.SetRange([]int32{
					int32(nameNode.StartPosition().Row),
					int32(nameNode.StartPosition().Column),
					int32(nameNode.EndPosition().Row),
					int32(nameNode.EndPosition().Column),
				})

				// 设置变量值
				if i < len(varValues) {
					valueNode := varValues[i]

					// 使用新函数处理引用类型
					r.processVariableValue(valueNode, uint32(2000+i), rc.SourceFile.Content, newVariable.Path, references)
				}

				elements = append(elements, newVariable)
			}
		}
	}

	return elements
}

// processVariableValue 处理变量值，如果是引用类型，创建引用元素
func (r *GoResolver) processVariableValue(valueNode *sitter.Node, elementID uint32, content []byte, rootPath string, references *[]*Reference) {
	// 检查变量的值是否是引用类型
	refType := parseVariableValue(valueNode, content)
	if refType != "" {
		// 使用函数创建引用元素
		ref := createReferenceElement(refType, valueNode, elementID, content)
		// 设置引用元素的Path为根元素的Path
		ref.BaseElement.Path = rootPath
		// 添加到引用列表
		*references = append(*references, ref)
	}
}

// parseVariableValue 分析变量值节点，提取可能的引用类型
// 只在变量值是结构体实例化时返回类型名称
func parseVariableValue(node *sitter.Node, content []byte) string {
	// 检查节点类型
	switch types.ToNodeKind(node.Kind()) {
	case types.NodeKindIdentifier:
		// 普通标识符，不是结构体实例化，返回空
		return ""

	case types.NodeKindCompositeLiteral: // TODO: 添加到NodeKind常量
		// 结构体或数组/切片字面量
		typeNode := node.ChildByFieldName("type")
		if typeNode != nil {
			// 检查是否是结构体实例化
			if typeNode.Kind() == string(types.NodeKindTypeIdentifier) {
				// 是结构体实例化，返回类型名称
				return typeNode.Utf8Text(content)
			} else if typeNode.Kind() == string(types.NodeKindSelectorExpression) {
				// 带包名的结构体实例化，例如 pkg.Person{}
				field := typeNode.ChildByFieldName("field")
				operand := typeNode.ChildByFieldName("operand")
				if field != nil && operand != nil {
					// 返回完整的限定名，如 "pkg.Type"
					return operand.Utf8Text(content) + "." + field.Utf8Text(content)
				} else if field != nil {
					return field.Utf8Text(content)
				}
			} else {
				// 其他类型（如数组、切片等），不是结构体实例化，返回空
				return ""
			}
		}

	case types.NodeKindCallExpression:
		// 函数调用，不是结构体实例化，返回空
		return ""
	}

	return ""
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

			switch types.ToElementType(nodeCaptureName) {
			case types.ElementTypeInterfaceName:
				element.Name = content
				element.Scope = analyzeScope(content)
			case types.ElementTypeInterfaceType:
				// 处理接口类型节点
				interfaceTypeNode := capture.Node

				// 直接遍历接口类型节点的所有子节点
				for i := uint(0); i < interfaceTypeNode.ChildCount(); i++ {
					childNode := interfaceTypeNode.Child(i)
					if childNode == nil {
						continue
					}

					switch types.ToNodeKind(childNode.Kind()) {
					case types.NodeKindMethodElem:
						// 处理方法声明
						// 创建一个方法声明
						decl := &Declaration{
							Modifier:   "", // Go中接口方法没有显式修饰符
							Name:       "",
							Parameters: []Parameter{},
						}

						// 获取方法名
						nameNode := childNode.ChildByFieldName("name")
						if nameNode != nil {
							decl.Name = nameNode.Utf8Text(rc.SourceFile.Content)
						}

						// 获取参数列表
						parametersNode := childNode.ChildByFieldName("parameters")
						if parametersNode != nil {
							// 获取参数文本
							parametersText := parametersNode.Utf8Text(rc.SourceFile.Content)
							// 去掉括号
							parametersText = strings.Trim(parametersText, "()")

							// 解析参数
							if parametersText != "" {
								typeGroups := analyzeParameterGroups(parametersText)

								// 处理每个类型组
								for _, group := range typeGroups {
									paramTypes := group.Type

									// 处理每个参数名
									for _, name := range group.Names {
										decl.Parameters = append(decl.Parameters, Parameter{
											Name: name,
											Type: paramTypes,
										})
									}
								}
							}
						}

						// 获取返回类型
						resultNode := childNode.ChildByFieldName("result")
						if resultNode != nil {
							// 使用analyzeReturnTypes函数提取并格式化返回类型
							decl.ReturnType = analyzeReturnTypes(resultNode, rc.SourceFile.Content)
						}

						// 将方法添加到接口的Methods列表中
						element.Methods = append(element.Methods, decl)

					case types.NodeKindTypeElem:
						// 处理嵌入接口
						typeNode := childNode.Child(0) // 获取第一个子节点，即类型名称
						if typeNode != nil {
							interfaceName := typeNode.Utf8Text(rc.SourceFile.Content)

							// 检查是否是限定名称（包含点号）
							if strings.Contains(interfaceName, ".") {
								// 已经是完全限定名称，直接添加
								element.SuperInterfaces = append(element.SuperInterfaces, interfaceName)
							} else {
								// 否则，尝试查找是否有包前缀
								if rc.SourceFile != nil && rc.SourceFile.Path != "" {
									// 简单处理，直接添加无包名的接口名
									element.SuperInterfaces = append(element.SuperInterfaces, interfaceName)
								}
							}
						}
					}
				}
			}
		}
	}

	// 设置元素类型
	element.Type = types.ElementTypeInterface

	return elements, nil
}

func (r *GoResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储调用信息
	elements := []Element{element}

	// 如果没有匹配信息，直接返回
	if rc.Match == nil || rc.Match.Captures == nil || len(rc.Match.Captures) == 0 {
		return elements, nil
	}

	// 设置默认类型
	element.Type = types.ElementTypeFunctionCall

	// 处理所有捕获节点
	for _, capture := range rc.Match.Captures {
		// 跳过无效节点
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}

		nodeCaptureName := rc.CaptureNames[capture.Index]
		//content := capture.Node.Utf8Text(rc.SourceFile.Content)

		// 更新元素的Range和其他基本属性
		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeFunctionCall:
			// 处理整个函数调用表达式
			funcNode := capture.Node.ChildByFieldName("function")

			if funcNode != nil {
				// 检查是否为匿名函数立即调用模式（IIFE）
				switch types.ToNodeKind(funcNode.Kind()) {
				case types.NodeKindFuncLiteral:
					return nil, nil

				// 正常函数调用处理
				case types.NodeKindIdentifier:
					// 简单函数调用
					element.BaseElement.Name = funcNode.Utf8Text(rc.SourceFile.Content)
				case types.NodeKindSelectorExpression:
					// 带包名/接收者的函数调用，如pkg.Func()或obj.Method()
					field := funcNode.ChildByFieldName("field")
					operand := funcNode.ChildByFieldName("operand")

					if field != nil && field.Kind() == string(types.NodeKindFieldIdentifier) {
						element.BaseElement.Name = field.Utf8Text(rc.SourceFile.Content)
						if operand != nil {
							element.Owner = operand.Utf8Text(rc.SourceFile.Content)
						}
					}
				}
			}

		case types.ElementTypeFunctionArguments:
			// 专门处理参数列表，仅收集参数位置信息
			collectArgumentPositions(element, capture.Node, rc.SourceFile.Content)
		}
	}

	return elements, nil
}

// 只收集参数位置信息，不尝试推断类型
func collectArgumentPositions(element *Call, argsNode sitter.Node, content []byte) {
	// 如果参数已经处理过，不再重复处理
	if len(element.Parameters) > 0 {
		return
	}

	// 确认是参数列表
	if argsNode.Kind() != string(types.NodeKindArgumentList) {
		return
	}

	// 清空可能存在的参数
	element.Parameters = []*Parameter{}

	// 处理所有参数节点
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child == nil {
			continue
		}

		// 跳过括号和逗号等分隔符
		childKind := child.Kind()
		if childKind == "," || childKind == "(" || childKind == ")" {
			continue
		}

		// 获取参数值
		//value := child.Utf8Text(content)

		// 创建参数对象
		param := &Parameter{
			Name: "",
			Type: nil,
			//Type: []string{types.GetNodeTypeString(childKind, value)},
		}

		element.Parameters = append(element.Parameters, param)
	}
}

// analyzeReturnTypes 分析返回类型参数列表节点，提取类型信息
// 支持处理多返回值和带名称的返回值
func analyzeReturnTypes(resultNode *sitter.Node, content []byte) []string {
	if resultNode == nil {
		return []string{}
	}

	// 如果结果节点不是参数列表，直接返回文本作为单个元素的切片
	if resultNode.Kind() != string(types.NodeKindParameterList) {
		return []string{resultNode.Utf8Text(content)}
	}

	var returnTypes []string
	var lastType string
	var currentNames []string

	// 遍历所有参数声明
	for i := uint(0); i < resultNode.ChildCount(); i++ {
		child := resultNode.Child(i)
		if child == nil {
			continue
		}

		// 跳过非参数声明节点（如逗号、括号）
		if child.Kind() != string(types.NodeKindParameterDeclaration) {
			continue
		}

		// 获取名称和类型节点
		nameNode := child.ChildByFieldName("name")
		typeNode := child.ChildByFieldName("type")

		if nameNode != nil && typeNode != nil {
			// 这是一个命名返回值参数
			name := nameNode.Utf8Text(content)
			paramType := typeNode.Utf8Text(content)

			// 检查是否与上一个类型相同
			if paramType == lastType {
				// 如果类型相同，添加到当前名称组
				currentNames = append(currentNames, name)
			} else {
				// 如果有积累的同类型名称，先处理它们
				if len(currentNames) > 0 {
					// 为每个命名参数添加相同的类型
					for range currentNames {
						returnTypes = append(returnTypes, lastType)
					}
					currentNames = nil
				}

				// 开始新的类型组
				currentNames = append(currentNames, name)
				lastType = paramType
			}
		} else if typeNode != nil {
			// 这是一个无名返回值参数
			paramType := typeNode.Utf8Text(content)

			// 处理可能积累的同类型名称
			if len(currentNames) > 0 {
				for range currentNames {
					returnTypes = append(returnTypes, lastType)
				}
				currentNames = nil
				lastType = ""
			}

			// 添加当前类型
			returnTypes = append(returnTypes, paramType)
		}
	}

	// 处理最后一组命名参数
	if len(currentNames) > 0 {
		for range currentNames {
			returnTypes = append(returnTypes, lastType)
		}
	}

	return returnTypes
}

func (r *GoResolver) isStandardLibrary(pkgPath string) (bool, error) {
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

// createReferenceElement 创建并返回引用类型的元素
func createReferenceElement(refType string, node *sitter.Node, elementID uint32, content []byte) *Reference {
	// 创建引用元素
	baseRef := NewBaseElement(elementID)
	ref := &Reference{
		BaseElement: baseRef,
		Owner:       refType,
	}
	ref.BaseElement.Name = refType
	ref.BaseElement.Type = types.ElementTypeReference
	// 设置范围
	ref.SetRange([]int32{
		int32(node.StartPosition().Row),
		int32(node.StartPosition().Column),
		int32(node.EndPosition().Row),
		int32(node.EndPosition().Column),
	})

	// 确定引用类型
	if strings.Contains(refType, ".") {
		// 如果包含点，可能是包限定的函数调用或类型引用
		parts := strings.Split(refType, ".")
		if len(parts) == 2 {
			ref.Owner = parts[0]
			ref.BaseElement.Name = parts[1]
		}
	} else {
		// 如果是单独的标识符，可能是本地类型或函数
		ref.Owner = ""
	}

	return ref
}

// isPrimitiveType 检查类型名称是否为Go基本数据类型
func isPrimitiveType(typeName string) bool {
	// 去除可能的数组、切片或指针标记
	cleanType := strings.ToLower(typeName)

	// Go基本数据类型列表
	primitiveTypes := []string{
		"bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"byte", "rune", "string", "any", "interface{}",
	}

	for _, t := range primitiveTypes {
		if strings.Contains(cleanType, t) {
			return true
		}
	}
	return false
}
