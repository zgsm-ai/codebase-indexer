package store

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"google.golang.org/protobuf/proto"
)

// TODO 使用项目名+路径hash作为项目索引的目录

type GraphStorage interface {
	BatchSave(ctx context.Context, projectUuid string, values Values) error
	Save(ctx context.Context, projectUuid string, value proto.Message) error
	Get(ctx context.Context, projectUuid string, key Key) (proto.Message, error)
	Delete(ctx context.Context, projectUuid string, key Key) error
	Iter(ctx context.Context, projectUuid string) Iterator
	Size(ctx context.Context, projectUuid string) int
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
	pathPrefix = "path"
	symPrefix  = "sym"
)

//// PathKey 键生成函数  unix path
//func PathKey(path string) string {
//	return fmt.Sprintf("%s:%s", pathPrefix, utils.ToUnixPath(path))
//}
//
//func SymbolKey(symbol string) string {
//	return fmt.Sprintf("%s:%s", symPrefix, symbol)
//}

type Key interface {
	Get() string
}

type PathKey string

func (p PathKey) Get() string {
	return fmt.Sprintf("%s:%s", pathPrefix, utils.ToUnixPath(string(p)))
}

type SymbolKey string

func (s SymbolKey) Get() string {
	return fmt.Sprintf("%s:%s", symPrefix, string(s))
}

type Values interface {
	Len() int
	Value(i int) proto.Message
	Key(i int) string
}
