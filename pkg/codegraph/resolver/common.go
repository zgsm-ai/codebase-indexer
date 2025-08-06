package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"fmt"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// findIdentifierNode 递归遍历语法树节点，查找类型为"identifier"的节点
func findIdentifierNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	// 检查当前节点是否为identifier类型
	if node.Kind() == types.Identifier {
		return node
	}

	// 遍历所有子节点
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}

		// 递归查找子节点中的identifier
		result := findIdentifierNode(child)
		if result != nil {
			return result // 找到则立即返回
		}
	}

	// 未找到identifier节点
	return nil
}

func updateRootElement(
	rootElement Element,
	capture *sitter.QueryCapture,
	captureName string,
	content []byte) {
	node := capture.Node
	// 设置range
	if capture.Index == rootElement.GetRootIndex() { // root capture: @package @function @class etc
		// rootNode
		rootElement.SetRange([]int32{
			int32(node.StartPosition().Row),
			int32(node.StartPosition().Column),
			int32(node.EndPosition().Row),
			int32(node.EndPosition().Column),
		})
	}

	// 设置name TODO 这里这里去掉，在 resolve中处理名字
	if rootElement.GetName() == types.EmptyString && IsElementNameCapture(rootElement.GetType(), captureName) {
		// 取root节点的name，比如definition.function.name
		// 获取名称 ,go import 带双引号
		name := strings.ReplaceAll(node.Utf8Text(content), types.DoubleQuote, types.EmptyString)
		if name == types.EmptyString {
			// TODO 日志
			fmt.Printf("tree_sitter base_processor name_node %s %v name not found", captureName, rootElement.GetRange())
		}
		rootElement.SetName(name)
	}
}

// 清理参数字符串，支持cpp和java
// (int const a,const int b,[[maybe_unused]] const std::string& name)
// -> (int a, int b, std::string name)
func CleanParam(param string) string {

	// 1. 清理注释
	reComment := regexp.MustCompile(`/\*.*?\*/`)
	param = reComment.ReplaceAllString(param, "")

	// 2. 去除属性标记 [[...]]
	reAttr := regexp.MustCompile(`\[\[.*?\]\]`)
	param = reAttr.ReplaceAllString(param, "")

	// 3. 去除关键字修饰符
	reKeywords := regexp.MustCompile(`\b(const|volatile|mutable|__?restrict)\b`)
	param = reKeywords.ReplaceAllString(param, "")

	// 4. 去除指针和引用符号
	rePtrRef := regexp.MustCompile(`[*&]+`)
	param = rePtrRef.ReplaceAllString(param, "")

	// 5. 清理 @注解
	reAt := regexp.MustCompile(`@\w+\s+`)
	param = reAt.ReplaceAllString(param, "")

	// 6. super/extends
	reSuper := regexp.MustCompile(`super|extends`)
	param = reSuper.ReplaceAllString(param, "")

	// 7. 清理问号
	reQuestion := regexp.MustCompile(`\?`)
	param = reQuestion.ReplaceAllString(param, "")

	// 8. 去除这种情况 struct TempPoint {
	// int tx, ty;
	// } temp_pt = {10, 20};
	// 直接提取结构体名字
	reStruct := regexp.MustCompile(`struct\s+(\w+)\s*\{`)
	matches := reStruct.FindAllStringSubmatch(param, -1)

	if len(matches) > 0 {
		param = matches[0][1]
	}

	// 9. 过滤类型里的 struct、enum、union 关键字
	reTypePrefix := regexp.MustCompile(`\b(struct|enum|union)\b\s*`)
	param = reTypePrefix.ReplaceAllString(param, "")

	// 10. 清理多余空白
	param = strings.TrimSpace(param)
	reSpaces := regexp.MustCompile(`\s+`)
	param = reSpaces.ReplaceAllString(param, " ")

	return param
}
func updateElementRange(element Element, capture *sitter.QueryCapture) {
	element.SetRange([]int32{
		int32(capture.Node.StartPosition().Row),
		int32(capture.Node.StartPosition().Column),
		int32(capture.Node.EndPosition().Row),
		int32(capture.Node.EndPosition().Column),
	})
}
func NewReference(rootElement Element, curNode *sitter.Node, name string, owner string) *Reference {
	return &Reference{
		BaseElement: &BaseElement{
			Name: name,
			Path: rootElement.GetPath(),
			Type: types.ElementTypeReference,
			Range: []int32{
				int32(curNode.StartPosition().Row),
				int32(curNode.StartPosition().Column),
				int32(curNode.EndPosition().Row),
				int32(curNode.EndPosition().Column),
			},
			Scope: types.ScopeFunction,
		},
		Owner: owner,
	}
}

// 解析java语法中的type_list类型，返回类型列表
func parseTypeList(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	if types.ToNodeKind(node.Kind()) != types.NodeKindTypeList {
		return []string{node.Utf8Text(content)}
	}
	typs := []string{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.Kind() == types.Comma {
			continue
		}
		switch types.ToNodeKind(child.Kind()) {
		case types.NodeKindScopedTypeIdentifier:
			typs = append(typs, child.Utf8Text(content))
		case types.NodeKindTypeIdentifier:
			typs = append(typs, child.Utf8Text(content))
		case types.NodeKindGenericType:
			typs = append(typs, parseGenericType(child, content)...)
		}

	}
	return typs
}

// 处理cpp语法中的base_class_clause类型，返回类型列表
func parseBaseClassClause(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}

	// 如果不是base_class_clause节点，直接返回节点内容
	if types.ToNodeKind(node.Kind()) != types.NodeKindBaseClassClause {
		return []string{node.Utf8Text(content)}
	}

	typs := []string{}

	// 从后往前遍历所有子节点
	for i := int(node.NamedChildCount()) - 1; i >= 0; i-- {
		child := node.NamedChild(uint(i))
		if child == nil || child.Kind() == types.Comma || child.Kind() == types.Colon {
			continue
		}

		// 处理类型节点
		var baseClasses []string

		if types.ToNodeKind(child.Kind()) == types.NodeKindTypeIdentifier {
			// 直接是type_identifier
			baseClasses = []string{child.Utf8Text(content)}
		} else {
			// 不是type_identifier，递归查找所有的type_identifier
			baseClasses = findAllTypeIdentifiers(child, content)
		}

		// 如果找到了类型标识符，添加到结果中
		if len(baseClasses) > 0 {
			typs = append(typs, baseClasses...)
		}
	}

	return typs
}
func removeDuplicates(slice []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// 递归查找当前节点及其下面所有的identifier
func findAllIdentifiers(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	var identifiers []string
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if types.ToNodeKind(n.Kind()) == types.NodeKindIdentifier {
			identifiers = append(identifiers, n.Utf8Text(content))
			return
		}
		for i := uint(0); i < n.NamedChildCount(); i++ {
			child := n.NamedChild(i)
			if child.IsMissing() || child.IsError() {
				continue
			}
			switch types.ToNodeKind(child.Kind()) {
			case types.NodeKindIdentifier:
				identifiers = append(identifiers, child.Utf8Text(content))
			default:
				// 递归查找类型中的标识符
				walk(child)
			}
		}
	}
	walk(node)
	return identifiers
}

func findAllTypeIdentifiers(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	var identifiers []string
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if types.ToNodeKind(n.Kind()) == types.NodeKindTypeIdentifier {
			identifiers = append(identifiers, n.Utf8Text(content))
			return
		}
		for i := uint(0); i < n.NamedChildCount(); i++ {
			child := n.NamedChild(i)
			if child.IsMissing() || child.IsError() {
				continue
			}
			switch types.ToNodeKind(child.Kind()) {
			case types.NodeKindTypeIdentifier:
				identifiers = append(identifiers, child.Utf8Text(content))
			default:
				// 递归查找类型中的标识符
				walk(child)
			}
		}
	}
	walk(node)
	return identifiers
}
