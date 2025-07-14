package resolver

import (
	"context"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type ElementResolver interface {
	Resolve(ctx context.Context, element Element, rc *ResolveContext) error
	resolveImport(ctx context.Context, element Import, rc *ResolveContext) error
	resolvePackage(ctx context.Context, element Package, rc *ResolveContext) error
	resolveFunction(ctx context.Context, element Function, rc *ResolveContext) error
	resolveMethod(ctx context.Context, element Method, rc *ResolveContext) error
	resolveClass(ctx context.Context, element Class, rc *ResolveContext) error
	resolveVariable(ctx context.Context, element Variable, rc *ResolveContext) error
	resolveInterface(ctx context.Context, element Interface, rc *ResolveContext) error
}

func resolve(ctx context.Context, b ElementResolver, element Element, rc *ResolveContext) error {
	switch element.(type) {
	case Import:
		return b.resolveImport(ctx, element.(Import), rc)
	case Package:
		return b.resolvePackage(ctx, element.(Package), rc)
	case Function:
		return b.resolveFunction(ctx, element.(Function), rc)
	case Method:
		return b.resolveMethod(ctx, element.(Method), rc)
	case Class:
		return b.resolveClass(ctx, element.(Class), rc)
	case Variable:
		return b.resolveVariable(ctx, element.(Variable), rc)
	default:
		return fmt.Errorf("element_resover not supported element %v", element)
	}
}

// findIdentifierNode 递归遍历语法树节点，查找类型为"identifier"的节点
func findIdentifierNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	// 检查当前节点是否为identifier类型
	if node.Kind() == identifier {
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
