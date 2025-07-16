package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type JavaResolver struct {
}

var _ ElementResolver = &JavaResolver{}

func (j *JavaResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	//复用resolve函数
	return resolve(ctx, j, element, rc)
}

func (j *JavaResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {

	element.FilePaths = []string{}
	var importName string
	// 获取import name
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			return nil, fmt.Errorf("import is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeImportName:
			importName = content
		}
	}
	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/") + ".java"
	pj := rc.ProjectInfo
	fullPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), classPath))

	elements := []Element{element}
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}
	element.BaseElement.Scope = types.ScopePackage
	element.BaseElement.Type = types.ElementTypeImport
	element.BaseElement.Name = importName

	if pj.IsEmpty() {
		// TODO logger
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{fullPath}
		return elements, nil
	}

	// 处理静态导入，有则会移除，没有就不会动
	importName = strings.TrimPrefix(importName, "static ")
	fmt.Println("import name:", importName)
	// 处理包导入
	if strings.HasSuffix(importName, ".*") {
		pkgPath := strings.ReplaceAll(strings.TrimSuffix(importName, ".*"), ".", "/")
		fullPkgPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), pkgPath))
		files := pj.FindFilesInDirIndex(fullPkgPath, ".java")
		element.FilePaths = files
		if len(element.FilePaths) == 0 {
			return nil, fmt.Errorf("cannot find file which package belongs to: %s", importName)
		}
		return elements, nil
	}

	element.FilePaths = pj.FindMatchingFiles(fullPath)
	if len(element.FilePaths) == 0 {
		return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
	}

	return elements, nil
}

func (j *JavaResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			return nil, fmt.Errorf("package is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypePackageName:
			element.BaseElement.Name = content
		}
	}
	element.BaseElement.Scope = types.ScopePackage
	element.BaseElement.Type = types.ElementTypePackage
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}
	// package不需要额外处理，直接返回
	return []Element{element}, nil
}

func (j *JavaResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	// TODO java中不存在单独的函数，暂时不实现
	return nil, fmt.Errorf("not implemented")
}

func (j *JavaResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	// TODO 方法解析在class中做
	return nil, fmt.Errorf("not implemented")
}

func (j *JavaResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			// TODO 报错还是继续？
			return nil, fmt.Errorf("class is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ElementType(captureName) {
		case types.ElementTypeClassName:
			element.BaseElement.Name = content
		case types.ElementTypeClassExtends:
			element.SuperClasses = append(element.SuperClasses, content)
		case types.ElementTypeClassImplements:
			element.SuperInterfaces = append(element.SuperInterfaces, content)
		}
	}
	element.BaseElement.Scope = types.ScopeClass
	element.BaseElement.Type = types.ElementTypeClass
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}

	cls := parseClassNode(&rc.Match.Captures[0].Node, rc.SourceFile.Content, element.BaseElement.Name)
	element.Fields = cls.Fields
	element.Methods = cls.Methods
	return []Element{element}, nil
}

func (j *JavaResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			// TODO 报错还是继续？
			return nil, fmt.Errorf("variable is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeLocalVariableName:
			element.BaseElement.Name = content
		}
	}
	// TODO 作用是什么？
	// element.BaseElement.Scope = types.ScopeLocal
	element.BaseElement.Type = types.ElementTypeLocalVariable
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}
	return []Element{element}, nil
}

func (j *JavaResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	// panic("implement me")
	return nil, fmt.Errorf("not implemented")
}

func parseClassNode(node *sitter.Node, content []byte, className string) *Class {
	class := &Class{}
	node = node.Child(node.ChildCount() - 1)
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := types.ToNodeKind(child.Kind())
		switch kind {
		case types.NodeKindField:
			field := parseFieldNode(child, content)
			class.Fields = append(class.Fields, field)
		case types.NodeKindMethod:
			method := parseMethodNode(child, content)
			method.Owner = className
			class.Methods = append(class.Methods, method)
			// TODO 可能存在类嵌套的情况，需要递归解析
		}
	}
	return class
}

func parseFieldNode(node *sitter.Node, content []byte) *Field {
	field := &Field{}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := types.ToNodeKind(child.Kind())
		content := child.Utf8Text(content)
		switch kind {
		case types.NodeKindVariableDeclarator:
			field.Name = content
		case types.NodeKindModifier:
			field.Modifier = content
		case types.NodeKindIdentifier:
			field.Type = content
		default:
			if types.IsTypeNode(kind) {
				field.Type = content
			}
		}
	}
	return field
}

func parseMethodNode(node *sitter.Node, content []byte) *Method {
	var method = &Method{}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		content := child.Utf8Text(content)
		kind := types.ToNodeKind(child.Kind())
		switch kind {
		case types.NodeKindIdentifier:
			method.Declaration.Name = content
		case types.NodeKindModifier:
			method.Declaration.Modifier = content
		case types.NodeKindFormalParameters:
			method.Declaration.Parameters = parseParameters(content)
		default:
			if types.IsTypeNode(kind) {
				method.Declaration.ReturnType = content
			}
		}
	}
	return method
}
func parseParameters(content string) []Parameter {
	// 参数格式 (int a, Function<String, Integer> func, Runnable r, int... nums, List<String[]> arrs)
	var params []Parameter
	level := 0
	// 去掉第一个(
	start := 1
	for i, c := range content {
		switch c {
		case '<':
			level++
		case '>':
			level--
		case ',':
			if level == 0 {
				paramStr := strings.TrimSpace(content[start:i])
				if paramStr != "" {
					params = append(params, parseSingleParameter(paramStr))
				}
				start = i + 1
			}
		}
	}
	// 最后一个参数
	if start < len(content) {
		// 去掉最后一个)
		paramStr := strings.TrimSpace(content[start : len(content)-1])
		if paramStr != "" {
			params = append(params, parseSingleParameter(paramStr))
		}
	}
	return params
}

func parseSingleParameter(paramStr string) Parameter {
	// 去掉注解
	paramStr = regexp.MustCompile(`@\w+\s+`).ReplaceAllString(paramStr, "")
	// 拆分类型和名称
	parts := strings.Fields(paramStr)
	// if len(parts) < 2 {
	// 	// 可能是省略了参数名，可特殊处理 或 报错 日志？
	// 	return &Parameter{Name: paramStr}
	// }
	name := parts[len(parts)-1]
	typ := strings.Join(parts[:len(parts)-1], " ")

	return Parameter{
		Name: strings.TrimSpace(name),
		Type: strings.TrimSpace(typ),
	}
}
