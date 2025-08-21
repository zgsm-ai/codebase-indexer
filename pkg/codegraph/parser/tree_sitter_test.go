package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	sitter "github.com/tree-sitter/go-tree-sitter"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func TestQuery(t *testing.T) {
	parser := sitter.NewParser()
	defer parser.Close()
	lang := sitter.NewLanguage(sittergo.Language())
	err := parser.SetLanguage(lang)
	assert.NoError(t, err)

	// 示例Go代码
	sourceCode := []byte(`
		package main

		import "fmt"

		func add(a, b int) int {
			return a + b
		}

// 方法声明（带接收者的函数）
func (p *Person) SayHello() string { return "Hello" }

		func main() {
			sum := add(3, 5)
			fmt.Println("Sum:", sum)
		}
	`)

	tree := parser.Parse(sourceCode, nil)
	assert.NotNil(t, tree)
	defer tree.Close()

	// 创建查询：查找所有函数调用
	queryStr := `
		(function_declaration
			name: (identifier) @name
			parameters: (parameter_list) @arguments
		) @function
	`
	query, _ := sitter.NewQuery(lang, queryStr)

	// 创建查询光标
	cursor := sitter.NewQueryCursor()
	defer cursor.Close()

	// 执行查询
	matches := cursor.Matches(query, tree.RootNode(), sourceCode)

	// 处理匹配结果
	for {
		match := matches.Next()
		if match == nil {
			break
		}

		// 处理每个匹配项
		t.Logf("Match #%d:\n", match.Id())

		// 遍历捕获的节点
		for _, capture := range match.Captures {
			node := capture.Node

			captureName := query.CaptureNames()[capture.Index]

			// 获取节点的位置信息
			startLine := node.StartPosition().Row
			startColumn := node.StartPosition().Column
			endLine := node.EndPosition().Row
			endColumn := node.EndPosition().Column
			// 获取节点的文本内容
			content := node.Utf8Text(sourceCode)

			t.Logf("  Capture: %-15s | Range: [%d:%d-%d:%d] | Content: %s\n",
				captureName, startLine, startColumn, endLine, endColumn, content)
		}

	}
}

func TestWalk(t *testing.T) {
	parser := sitter.NewParser()
	defer parser.Close()
	lang := sitter.NewLanguage(sittergo.Language())
	err := parser.SetLanguage(lang)
	assert.NoError(t, err)
	// 示例Go代码
	sourceCode := []byte(`
		package main

		import "fmt"

		// 计算两数之和
		func add(a, b int) int {
			return a + b
		}

		func main() {
			fmt.Println(add(3, 5))
		}
	`)
	// 解析代码生成语法树
	tree := parser.Parse(sourceCode, nil)
	defer tree.Close()

	// 获取根节点并创建遍历器
	rootNode := tree.RootNode()
	cursor := rootNode.Walk()
	defer cursor.Close()

	// 前序遍历语法树
	for {
		// 获取当前节点
		currentNode := cursor.Node()

		// 处理节点（示例：打印节点类型和位置）
		startLine := currentNode.StartPosition().Row
		endLine := currentNode.EndPosition().Row
		fmt.Printf("CaptureNode: %-15s | Lines: %d-%d | content: %s\n",
			currentNode.Kind(), startLine, endLine, currentNode.Utf8Text(sourceCode))

		// 优先访问子节点（深度优先）
		if cursor.GotoFirstChild() {
			continue
		}

		// 没有子节点时，尝试访问下一个兄弟节点
		for {
			if cursor.GotoNextSibling() {
				break // 找到下一个兄弟节点，继续遍历
			}

			// 没有更多兄弟节点，返回父节点
			if !cursor.GotoParent() {
				return // 已经回到根节点，遍历结束
			}
		}
	}
}

func TestWalkRecur(t *testing.T) {
	parser := sitter.NewParser()
	defer parser.Close()
	lang := sitter.NewLanguage(sittergo.Language())
	err := parser.SetLanguage(lang)
	assert.NoError(t, err)
	// 示例Go代码
	sourceCode := []byte(`
		package main

		import "fmt"

		// 计算两数之和
		func add(a, b int) int {
			return a + b
		}

		func main() {
			fmt.Println(add(3, 5))
		}
	`)
	// 解析代码生成语法树
	tree := parser.Parse(sourceCode, nil)
	defer tree.Close()
	// 获取根节点并创建遍历器
	rootNode := tree.RootNode()

	var dfs func(node *sitter.Node, depth int)
	dfs = func(node *sitter.Node, depth int) {
		if node == nil {
			return
		}

		// 处理节点（示例：打印节点类型和位置）
		startLine := node.StartPosition().Row
		endLine := node.EndPosition().Row
		indent := strings.Repeat("  ", depth) // 缩进表示层级
		fmt.Printf("%sNode: %-15s | Lines: %d-%d | content: %s\n",
			indent, node.Kind(), startLine, endLine, node.Utf8Text(sourceCode))
		childCount := node.ChildCount()
		for i := uint(0); i < childCount; i++ {
			dfs(node.Child(i), depth+1)
		}
	}
	dfs(rootNode, 0)
}
