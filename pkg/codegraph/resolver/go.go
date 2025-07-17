package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"path/filepath"
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
			case nodeCaptureName == string(types.ElementTypeImportName):
				element.Name = content
				element.Content = []byte(content)
			case nodeCaptureName == string(types.ElementTypeImportAlias):
				element.Alias = content
			case nodeCaptureName == string(types.ElementTypeImportPath):
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

				// Store the import path in Content if not already set
				if len(element.Content) == 0 {
					element.Content = []byte(path)
				}

				// Set filepaths as relative to project root
				if rc.ProjectInfo != nil && !rc.ProjectInfo.IsEmpty() {
					// Check if this is standard library import
					isStd, _ := r.isStandardLibrary(path)
					if !isStd {
						// For non-standard library imports, try to resolve the file path
						// relative to the project root
						sourceRoots := rc.ProjectInfo.GetDirs()
						if len(sourceRoots) > 0 {
							// Try to find the file path relative to any of the source roots
							var filePaths []string
							for _, root := range sourceRoots {
								// Construct potential file paths
								possiblePath := filepath.Join(root, path)
								filePaths = append(filePaths, possiblePath)
							}
							element.FilePaths = filePaths
						} else {
							// If we don't have source roots, just store the import path
							element.FilePaths = []string{path}
						}
					} else {
						// For standard library, just store the import path
						element.FilePaths = []string{path}
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
			if nodeCaptureName == string(types.ElementTypeFunctionName) {
				element.Declaration.Name = content
				element.BaseElement.Name = content // 使用BaseElement中的Name
			} else if nodeCaptureName == string(types.ElementTypeFunctionParameters) {
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
			if nodeCaptureName == string(types.ElementTypeMethodName) {
				element.Declaration.Name = content
				element.BaseElement.Name = content // 使用BaseElement中的Name
			} else if nodeCaptureName == string(types.ElementTypeMethodParameters) {
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
			} else if nodeCaptureName == string(types.ElementTypeMethodReceiver) {
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

			switch nodeCaptureName {
			case string(types.ElementTypeStructName):
				element.Name = content

				if len(content) > 0 {
					if content[0] >= 'A' && content[0] <= 'Z' {
						element.Scope = types.ScopeProject // 公开的，项目可见
					} else {
						element.Scope = types.ScopePackage // 私有的，仅包内可见
					}
				}

			case string(types.ElementTypeStructType):
				structTypeNode := capture.Node
				fmt.Println("structTypeNode", structTypeNode.Kind())
				// 获取field_declaration_list - 遍历所有子节点找到字段列表
				var fieldListNode *sitter.Node

				// 在struct_type节点中查找field_declaration_list
				for i := uint(0); i < structTypeNode.ChildCount(); i++ {
					child := structTypeNode.Child(i)
					if child != nil && child.Kind() == string(types.NodeKindFieldList) {
						fieldListNode = child
						break
					}

				}

				// 遍历所有field_declaration子节点
				for j := uint(0); j < fieldListNode.ChildCount(); j++ {
					fieldNode := fieldListNode.Child(j)
					if fieldNode != nil && fieldNode.Kind() == string(types.NodeKindField) {
						// 获取字段名和类型
						nameNode := fieldNode.ChildByFieldName("name")
						typeNode := fieldNode.ChildByFieldName("type")

						if nameNode != nil && typeNode != nil {
							fieldName := nameNode.Utf8Text(rc.SourceFile.Content)
							fieldType := typeNode.Utf8Text(rc.SourceFile.Content)

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
			//无用，暂时保留
			case string(types.ElementTypeClass) + types.Dot + "base",
				string(types.ElementTypeStruct) + types.Dot + "base":
				// 处理基类/嵌入结构体（在Go中是嵌入）
			//无用，暂时保留
			case string(types.ElementTypeClass) + types.Dot + "modifier",
				string(types.ElementTypeStruct) + types.Dot + "modifier":
				// 类修饰符处理（如果有）
			}
		}
	}

	// 设置元素类型
	element.Type = types.ElementTypeStruct

	return elements, nil
}

func (r *GoResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	// 使用现有的BaseElement存储变量信息
	elements := []Element{element}

	// 如果没有匹配信息，直接返回
	if rc.Match == nil || rc.Match.Captures == nil || len(rc.Match.Captures) == 0 {
		return elements, nil
	}

	// 获取根捕获，它应该是变量声明节点
	rootCapture := rc.Match.Captures[0]
	rootCaptureName := rc.CaptureNames[rootCapture.Index]
	nodeKind := rootCapture.Node.Kind()

	// 根据根捕获的类型设置变量类型和作用域
	setVariableTypeAndScope(element, rootCaptureName, nodeKind)

	// 尝试从根节点获取类型信息
	var variableType string
	if !rootCapture.Node.IsError() && !rootCapture.Node.IsMissing() {
		// 尝试获取类型节点
		typeNode := rootCapture.Node.ChildByFieldName("type")
		if typeNode != nil {
			variableType = typeNode.Utf8Text(rc.SourceFile.Content)
		}
	}

	// 处理所有捕获节点
	for _, capture := range rc.Match.Captures {
		// 跳过无效节点
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}

		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)

		// 更新元素的Range和其他基本属性
		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

		// 根据节点类型处理不同信息
		switch {
		case strings.HasSuffix(nodeCaptureName, ".name"):
			// 变量名，仅当尚未设置时更新
			if element.BaseElement.Name == "" {
				element.BaseElement.Name = content
			}
			// 设置变量名后更新作用域
			updateScopeBasedOnName(element, rootCaptureName)

		case strings.HasSuffix(nodeCaptureName, ".type"):
			// 变量类型，存储为纯文本
			variableType = content
			element.Content = []byte(content)

		case strings.HasSuffix(nodeCaptureName, ".value"):
			// 变量初始值，仅当没有类型信息时存储
			if variableType == "" && len(element.Content) == 0 {
				element.Content = []byte(content)
			}
		}
	}

	// 处理多变量声明（如 file, err := os.Open(...)）
	elements = appendMultiVariableDeclarations(elements, element, rc)

	return elements, nil
}

// 根据捕获名称和节点类型设置变量类型和作用域
func setVariableTypeAndScope(element *Variable, captureName, nodeKind string) {
	// 默认设置
	element.Type = types.ElementTypeVariable
	element.Scope = types.ScopeBlock

	// 根据捕获名称设置元素类型
	switch {
	case captureName == string(types.ElementTypeVariable) || strings.Contains(captureName, "variable"):
		// 通用变量
		element.Type = types.ElementTypeVariable

		// 根据节点类型判断作用域
		if nodeKind == "var_declaration" {
			// var x int 形式 - 可能是全局或局部变量
			// 暂时设为文件级别，等获取变量名后再确定
			element.Scope = types.ScopeFile
		} else {
			// 其他形式可能是块级别
			element.Scope = types.ScopeBlock
		}

	case captureName == string(types.ElementTypeLocalVariable):
		// 局部变量 - 函数内声明的变量
		element.Type = types.ElementTypeLocalVariable
		element.Scope = types.ScopeFunction

	case captureName == string(types.ElementTypeConstant):
		// 常量
		element.Type = types.ElementTypeConstant
		// 暂时设为文件级别，等获取变量名后再确定
		element.Scope = types.ScopeFile

	case captureName == string(types.ElementTypeField):
		// 结构体字段
		element.Type = types.ElementTypeField
		element.Scope = types.ScopeClass
	}
}

// 根据变量名更新作用域
func updateScopeBasedOnName(element *Variable, rootCaptureName string) {
	// 只有当变量不是局部变量且已有名称时才需要更新
	if element.Scope != types.ScopeFunction &&
		element.BaseElement != nil &&
		element.BaseElement.Name != "" {

		// 根据名称首字母大小写确定可见性
		if element.BaseElement.Name[0] >= 'A' && element.BaseElement.Name[0] <= 'Z' {
			// 大写开头 - 项目可见
			element.Scope = types.ScopeProject
		} else {
			// 小写开头 - 包内可见
			element.Scope = types.ScopePackage
		}

		// 局部变量强制为函数作用域
		if rootCaptureName == string(types.ElementTypeLocalVariable) {
			element.Scope = types.ScopeFunction
		}
	}
}

// 处理多变量声明
func appendMultiVariableDeclarations(elements []Element, element *Variable, rc *ResolveContext) []Element {
	if rc.Match == nil || rc.Match.Captures == nil {
		return elements
	}

	// 跟踪已处理的变量名
	processedNames := make(map[string]bool)
	if element.BaseElement.Name != "" {
		processedNames[element.BaseElement.Name] = true
	}

	for i, capture := range rc.Match.Captures {
		if i == 0 { // 跳过根捕获
			continue
		}

		nodeCaptureName := rc.CaptureNames[capture.Index]
		if strings.HasSuffix(nodeCaptureName, ".name") {
			variableName := capture.Node.Utf8Text(rc.SourceFile.Content)

			// 只处理尚未处理过的变量名
			if !processedNames[variableName] {
				processedNames[variableName] = true

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

	return elements
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
			case string(types.ElementTypeInterfaceName):
				element.Name = content
				if len(content) > 0 {
					if content[0] >= 'A' && content[0] <= 'Z' {
						element.Scope = types.ScopeProject // 公开的，项目可见
					} else {
						element.Scope = types.ScopePackage // 私有的，仅包内可见
					}
				}
			case string(types.ElementTypeInterfaceType):
				// 处理接口类型节点
				interfaceTypeNode := capture.Node

				// 直接遍历接口类型节点的所有子节点
				for i := uint(0); i < interfaceTypeNode.ChildCount(); i++ {
					methodNode := interfaceTypeNode.Child(i)
					if methodNode != nil && methodNode.Kind() == string(types.NodeKindMethodElem) {
						// 创建一个方法声明
						decl := &Declaration{
							Modifier:   "", // Go中接口方法没有显式修饰符
							Name:       "",
							Parameters: []Parameter{},
						}

						// 获取方法名
						nameNode := methodNode.ChildByFieldName("name")
						if nameNode != nil {
							decl.Name = nameNode.Utf8Text(rc.SourceFile.Content)
						}

						// 获取参数列表
						parametersNode := methodNode.ChildByFieldName("parameters")
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
									paramType := group.Type

									// 处理每个参数名
									for _, name := range group.Names {
										decl.Parameters = append(decl.Parameters, Parameter{
											Name: name,
											Type: paramType,
										})
									}
								}
							}
						}

						// 获取返回类型
						resultNode := methodNode.ChildByFieldName("result")
						if resultNode != nil {
							// 使用analyzeReturnTypes函数提取并格式化返回类型
							decl.ReturnType = analyzeReturnTypes(resultNode, rc.SourceFile.Content)
						}

						// 将方法添加到接口的Methods列表中
						element.Methods = append(element.Methods, decl)
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

		switch nodeCaptureName {
		case string(types.ElementTypeFunctionCall):
			// 处理整个函数调用表达式
			funcNode := capture.Node.ChildByFieldName("function")

			if funcNode != nil {
				// 检查是否为匿名函数立即调用模式（IIFE）
				if funcNode.Kind() == "func_literal" {
					// 这是一个匿名函数，设置特殊名称
					element.BaseElement.Name = "<anonymous>"
					element.Type = types.ElementTypeFunctionCall // 仍然使用函数调用类型

					// 如果是在go语句中，标记为goroutine
					parentNode := capture.Node.Parent()
					if parentNode != nil && parentNode.Kind() == "go_statement" {
						element.BaseElement.Name = "<goroutine>"
					}

					// 不需要进一步处理匿名函数参数
					continue
				}

				// 正常函数调用处理
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

		case string(types.ElementTypeFunctionArguments):
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
	if argsNode.Kind() != "argument_list" {
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
		value := child.Utf8Text(content)

		// 创建参数对象
		param := &Parameter{
			Name: value,
			Type: getNodeTypeString(childKind, value),
		}

		element.Parameters = append(element.Parameters, param)
	}
}

// getNodeTypeString 根据节点类型返回对应的类型字符串
func getNodeTypeString(nodeKind string, value string) string {
	switch nodeKind {
	case "identifier":
		return "unknown"
	case "int_literal":
		return "int"
	case "float_literal":
		return "float64"
	case "interpreted_string_literal", "raw_string_literal":
		return "string"
	case "true", "false":
		return "bool"
	case "nil":
		return "nil"
	case "selector_expression":
		return "selector"
	case "call_expression":
		return "function_result"
	case "binary_expression", "unary_expression":
		return "expression"
	case "array_literal", "slice_literal":
		return "array/slice"
	case "map_literal":
		return "map"
	case "composite_literal":
		return "struct"
	default:
		return nodeKind
	}
}

// analyzeReturnTypes 分析返回类型参数列表节点，提取类型信息
// 支持处理多返回值和带名称的返回值
func analyzeReturnTypes(resultNode *sitter.Node, content []byte) string {
	if resultNode == nil {
		return ""
	}

	// 如果结果节点不是参数列表，直接返回文本
	if resultNode.Kind() != "parameter_list" {
		return resultNode.Utf8Text(content)
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
		if child.Kind() != "parameter_declaration" {
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

	// 如果只有一个返回值，直接返回
	if len(returnTypes) == 1 {
		return returnTypes[0]
	}

	// 多个返回值用括号和逗号组合
	return "(" + strings.Join(returnTypes, ", ") + ")"
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
