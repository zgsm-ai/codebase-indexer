package parser

import (
	"codebase-indexer/pkg/codegraph/parser/resolver"
	"context"
	"fmt"
	treesitter "github.com/tree-sitter/go-tree-sitter"
	"strings"
)

type ParsedSource struct {
	Path     string
	Package  *resolver.Package
	Imports  []*resolver.Import
	Language Language
	Elements []resolver.Element
}

func (e *BaseElement) Update(ctx context.Context, captureName string, capture *treesitter.QueryCapture,
	source []byte, opts Options) error {
	node := &capture.Node

	if capture.Index == e.rootCaptureIndex { // root capture: @package @function @class etc
		// rootNode
		rootCaptureNode := node
		e.Range = []int32{
			int32(rootCaptureNode.StartPosition().Row),
			int32(rootCaptureNode.StartPosition().Column),
			int32(rootCaptureNode.StartPosition().Row),
			int32(rootCaptureNode.StartPosition().Column),
		}
		if opts.IncludeContent {
			content := source[node.StartByte():node.EndByte()]
			e.SetContent(content)
		}

	}

	if e.Name == EmptyString && isElementNameCapture(e.Type, captureName) {
		// 取root节点的name，比如definition.function.name
		// 获取名称 ,go import 带双引号
		name := strings.ReplaceAll(node.Utf8Text(source), DoubleQuote, EmptyString)
		if name == EmptyString {
			// TODO
			fmt.Printf("tree_sitter base_processor name_node %s %v name not found", captureName, e.Range)
		}
		e.Name = name
	}

	return nil
}

func (f *Function) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts Options) error {

	if err := f.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}
	node := &capture.Node

	if len(f.Parameters) == 0 && isParametersCapture(captureName) {
		f.Parameters = strings.Split(node.Utf8Text(source), Comma)
	}

	if isOwnerCapture(captureName) && f.Owner == EmptyString {
		f.Owner = node.Utf8Text(source)
	}

	return nil
}

func (m *Method) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts Options) error {

	if err := m.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}

	node := &capture.Node

	if len(m.Parameters) == 0 && isParametersCapture(captureName) {
		m.Parameters = strings.Split(node.Utf8Text(source), Comma)
	}

	if isOwnerCapture(captureName) && m.Owner == EmptyString {
		m.Owner = node.Utf8Text(source)
	}

	return nil
}

func (c *Call) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts Options) error {

	if err := c.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}
	node := &capture.Node

	if len(c.Arguments) == 0 && isArgumentsCapture(captureName) {
		c.Arguments = strings.Split(node.Utf8Text(source), Comma)
	}

	if c.Owner == EmptyString && isOwnerCapture(captureName) {
		c.Owner = node.Utf8Text(source)
	}

	return nil
}

func (v *Variable) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts Options) error {

	if err := v.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}

	node := &capture.Node

	// TODO 局部变量不是很容易区分，存在多层嵌套。找到它的名字不太容易。存在一行返回多个局部变量的情况,当前只取了第一个
	if v.Name == EmptyString {
		if nameNode := findIdentifierNode(node); nameNode != nil {
			v.Name = nameNode.Utf8Text(source)
		}
	}
	return nil
}

func (v *Import) Update(ctx context.Context, captureName string,
	capture *treesitter.QueryCapture, source []byte, opts Options) error {

	if err := v.BaseElement.Update(ctx, captureName, capture, source, opts); err != nil {
		return err
	}

	node := &capture.Node

	// TODO 各个scm 的source、alias full_name 解析。
	if v.Source == EmptyString && isSourceCapture(captureName) {
		v.Source = node.Utf8Text(source)
	}

	if v.Alias == EmptyString && isAliasCapture(captureName) {
		v.Alias = node.Utf8Text(source)
	}

	return nil
}

func initRootElement(elementTypeValue string) resolver.Element {
	elementType := toElementType(elementTypeValue)
	base := &resolver.BaseElement{}
	switch elementType {
	case resolver.ElementTypePackage:
		base.Type = resolver.ElementTypePackage
		return &resolver.Package{BaseElement: base}
	case resolver.ElementTypeImport:
		base.Type = resolver.ElementTypeImport
		return &resolver.Import{BaseElement: base}
	case resolver.ElementTypeFunction:
		base.Type = resolver.ElementTypeFunction
		return &resolver.Function{BaseElement: base}
	case resolver.ElementTypeClass:
		base.Type = resolver.ElementTypeClass
		return &resolver.Class{BaseElement: base}
	case resolver.ElementTypeMethod:
		base.Type = resolver.ElementTypeMethod
		return &resolver.Method{BaseElement: base}
	case resolver.ElementTypeFunctionCall:
		base.Type = resolver.ElementTypeFunctionCall
		return &resolver.Call{BaseElement: base}
	case resolver.ElementTypeMethodCall:
		base.Type = resolver.ElementTypeMethodCall
		return &resolver.Call{BaseElement: base}
	case resolver.ElementTypeInterface:
		base.Type = resolver.ElementTypeInterface
		return &resolver.Interface{BaseElement: base}
	default:
		base.Type = resolver.ElementTypeUndefined
		return base
	}
}
