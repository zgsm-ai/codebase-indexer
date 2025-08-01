package store

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"
)

// TODO 使用项目名+路径hash作为项目索引的目录

type GraphStorage interface {
	BatchSave(ctx context.Context, projectUuid string, values Entries) error
	Save(ctx context.Context, projectUuid string, entry *Entry) error
	Get(ctx context.Context, projectUuid string, key Key) ([]byte, error)
	Delete(ctx context.Context, projectUuid string, key Key) error
	DeleteAll(ctx context.Context, projectUuid string) error
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
	Value() []byte

	// Error 返回迭代过程中发生的错误
	Error() error

	// Close 关闭迭代器，释放相关资源
	Close() error
}

const (
	PathKeyPrefix = "path"
	SymKeyPrefix  = "sym"
	dataDir       = "data"
)

type Key interface {
	Get() (string, error)
}

type ElementPathKey struct {
	Language lang.Language
	Path     string
}

func (p ElementPathKey) Get() (string, error) {
	if p.Language == types.EmptyString {
		return types.EmptyString, fmt.Errorf("ElementPathKey field Language must not be empty")
	}
	if p.Path == types.EmptyString {
		return types.EmptyString, fmt.Errorf("ElementPathKey field Path must not be empty")
	}
	return fmt.Sprintf("%s:%s:%s", PathKeyPrefix, p.Language, utils.ToUnixPath(p.Path)), nil
}

type SymbolNameKey struct {
	Language lang.Language
	Name     string
}

func (s SymbolNameKey) Get() (string, error) {
	if s.Language == types.EmptyString {
		return types.EmptyString, fmt.Errorf("ElementPathKey field Language must not be empty")
	}
	if s.Name == types.EmptyString {
		return types.EmptyString, fmt.Errorf("ElementPathKey field Name must not be empty")
	}
	return fmt.Sprintf("%s:%s:%s", SymKeyPrefix, s.Language, s.Name), nil
}

type Entries interface {
	Len() int
	Value(i int) proto.Message
	Key(i int) Key
}
type Entry struct {
	Key   Key
	Value proto.Message
}

// checkDirWritable checks if directory is writable
func checkDirWritable(dir string) error {
	testFile := filepath.Join(dir, ".test-write")
	file, err := os.Create(testFile)
	if err != nil {
		return err
	}
	file.Close()
	return os.Remove(testFile)
}

// UnmarshalValue 将value解析成target
func UnmarshalValue(value []byte, target proto.Message) error {
	return proto.Unmarshal(value, target)
}
