package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
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
