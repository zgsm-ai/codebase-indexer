package store

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
)

type GraphStorage interface {
	Put(ctx context.Context, key string, value proto.Message) error
	Get(ctx context.Context, key string) (proto.Message, error)
	Delete(ctx context.Context, key string) error
	Iter(ctx context.Context) Iterator
	Close() error
}

// Iterator 定义了遍历存储中元素的接口
type Iterator interface {
	// Next 移动到下一个元素。如果没有更多元素，返回 false
	Next() bool

	// Key 返回当前元素的键
	Key() string

	// Value 返回当前元素的值
	Value() proto.Message

	// Error 返回迭代过程中发生的错误
	Error() error

	// Close 关闭迭代器，释放相关资源
	Close() error
}

const (
	pathPrefix = "path:"
	symPrefix  = "sym:"
)

// PathKey 键生成函数  unix path
func PathKey(path string) []byte {
	return []byte(fmt.Sprintf("%s%s", pathPrefix, utils.ToUnixPath(path)))
}

func SymbolKey(symbol string) []byte {
	return []byte(fmt.Sprintf("%s%s", symPrefix, symbol))
}
