package store

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"
)

// 使用项目名_路径hash(projectUuid)作为项目索引的目录

type GraphStorage interface {
	BatchSave(ctx context.Context, projectUuid string, values Entries) error
	Put(ctx context.Context, projectUuid string, entry *Entry) error
	Get(ctx context.Context, projectUuid string, key Key) ([]byte, error)
	Exists(ctx context.Context, projectUuid string, key Key) (bool, error)
	Delete(ctx context.Context, projectUuid string, key Key) error
	DeleteAll(ctx context.Context, projectUuid string) error
	Iter(ctx context.Context, projectUuid string) Iterator
	Size(ctx context.Context, projectUuid string, keyPrefix string) int
	Close() error
	ExistsProject(projectUuid string) (bool, error)
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
	PathKeySystemPrefix = "@path"
	SymKeySystemPrefix  = "@sym"
	dataDir             = "data"
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
	return fmt.Sprintf("%s:%s:%s", PathKeySystemPrefix, p.Language, utils.ToUnixPath(p.Path)), nil
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
	return fmt.Sprintf("%s:%s:%s", SymKeySystemPrefix, s.Language, s.Name), nil
}

func IsSymbolNameKey(key string) bool {
	return strings.HasPrefix(key, SymKeySystemPrefix)
}

func IsElementPathKey(key string) bool {
	return strings.HasPrefix(key, PathKeySystemPrefix)
}

func ToSymbolNameKey(key string) (SymbolNameKey, error) {
	keySplit := strings.Split(key, types.Colon)
	if len(key) != 3 {
		return SymbolNameKey{}, fmt.Errorf("invalid symbol_name key: %s", key)
	}
	if keySplit[0] != SymKeySystemPrefix {
		return SymbolNameKey{}, fmt.Errorf("invalid symbol_name key: %s", key)
	}
	language, err := lang.ToLanguage(keySplit[1])
	if err != nil {
		return SymbolNameKey{}, lang.ErrUnSupportedLanguage
	}
	if keySplit[2] == types.EmptyString {
		return SymbolNameKey{}, fmt.Errorf("invalid symbol_name key: %s", key)
	}
	return SymbolNameKey{
		Language: language,
		Name:     keySplit[2],
	}, nil
}

func ToElementPathKey(key string) (ElementPathKey, error) {
	keySplit := strings.Split(key, types.Colon)
	if len(key) != 3 {
		return ElementPathKey{}, fmt.Errorf("invalid element_path key: %s", key)
	}
	if keySplit[0] != SymKeySystemPrefix {
		return ElementPathKey{}, fmt.Errorf("invalid element_path key: %s", key)
	}
	language, err := lang.ToLanguage(keySplit[1])
	if err != nil {
		return ElementPathKey{}, lang.ErrUnSupportedLanguage
	}
	if keySplit[2] == types.EmptyString {
		return ElementPathKey{}, fmt.Errorf("invalid element_path key: %s", key)
	}
	return ElementPathKey{
		Language: language,
		Path:     keySplit[2],
	}, nil
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
