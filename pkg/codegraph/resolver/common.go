package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"strings"
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
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
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
