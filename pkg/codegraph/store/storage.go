package store

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	DeleteAllWithPrefix(ctx context.Context, projectUuid string, prefix string) error
	Iter(ctx context.Context, projectUuid string) Iterator
	// IterPrefix 创建一个只遍历指定前缀的迭代器
	IterPrefix(ctx context.Context, projectUuid string, prefix string) Iterator
	Size(ctx context.Context, projectUuid string, keyPrefix string) int
	Close() error
	ProjectIndexExists(projectUuid string) (bool, error)
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
	PathKeySystemPrefix      = "@path"
	SymKeySystemPrefix       = "@sym"
	CalleeMapKeySystemPrefix = "@callee"
	ProjectMetaKeyPrefix     = "@meta"
	dataDir                  = "data"
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
	return fmt.Sprintf("%s:%s:%s", PathKeySystemPrefix, p.Language, p.Path), nil
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

type CalleeMapKey struct {
	SymbolName string
	Timestamp  int64 // 版本时间戳（纳秒），0表示查询所有版本（第二位）
	ValueCount int   // 该版本包含的value数量，0表示查询所有版本（第三位）
}

func (c CalleeMapKey) Get() (string, error) {
	// 如果 Timestamp 和 ValueCount 都为 0，返回基础key（用于前缀扫描）
	if c.Timestamp == 0 && c.ValueCount == 0 {
		return fmt.Sprintf("%s:%s", CalleeMapKeySystemPrefix, c.SymbolName), nil
	}
	// 返回完整的版本化key: @callee:symbolName:timestamp:valueCount
	// 时间戳在前，便于 LevelDB 按时间排序，查找最新版本更高效
	return fmt.Sprintf("%s:%s:%d:%d", CalleeMapKeySystemPrefix, c.SymbolName, c.Timestamp, c.ValueCount), nil
}

// ProjectMetaKey 项目元数据的key
type ProjectMetaKey struct {
	MetaType string // 元数据类型，如 "callgraph_built"
}

func (p ProjectMetaKey) Get() (string, error) {
	if p.MetaType == types.EmptyString {
		return types.EmptyString, fmt.Errorf("ProjectMetaKey field MetaType must not be empty")
	}
	return fmt.Sprintf("%s:%s", ProjectMetaKeyPrefix, p.MetaType), nil
}

func IsSymbolNameKey(key string) bool {
	return strings.HasPrefix(key, SymKeySystemPrefix)
}
func IsCalleeMapKey(key string) bool {
	return strings.HasPrefix(key, CalleeMapKeySystemPrefix)
}
func IsElementPathKey(key string) bool {
	return strings.HasPrefix(key, PathKeySystemPrefix)
}
func IsProjectMetaKey(key string) bool {
	return strings.HasPrefix(key, ProjectMetaKeyPrefix)
}

// ParseCalleeMapKey 解析版本化的CalleeMapKey
// 格式：@callee:symbolName:timestamp:valueCount
func ParseCalleeMapKey(key string) (CalleeMapKey, error) {
	if !strings.HasPrefix(key, CalleeMapKeySystemPrefix) {
		return CalleeMapKey{}, fmt.Errorf("invalid callee map key: %s", key)
	}

	parts := strings.Split(key, types.Colon)
	if len(parts) != 4 {
		return CalleeMapKey{}, fmt.Errorf("invalid callee map key format, expected 4 parts: %s", key)
	}

	// 解析版本信息（格式：@callee:symbolName:timestamp:valueCount）
	timestamp, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return CalleeMapKey{}, fmt.Errorf("invalid timestamp in key %s: %w", key, err)
	}

	valueCount, err := strconv.Atoi(parts[3])
	if err != nil {
		return CalleeMapKey{}, fmt.Errorf("invalid value_count in key %s: %w", key, err)
	}

	return CalleeMapKey{
		SymbolName: parts[1],
		Timestamp:  timestamp,
		ValueCount: valueCount,
	}, nil
}

// GetCalleeMapKeyPrefix 获取用于前缀扫描的基础key
func GetCalleeMapKeyPrefix(symbolName string) string {
	return fmt.Sprintf("%s:%s:", CalleeMapKeySystemPrefix, symbolName)
}

func ToSymbolNameKey(key string) (SymbolNameKey, error) {
	// 查找第一个冒号位置
	first := strings.Index(key, types.Colon)
	if first == -1 {
		return SymbolNameKey{}, fmt.Errorf("invalid symbol_name key: %s", key)
	}
	// 查找第二个冒号位置
	second := strings.Index(key[first+1:], types.Colon)
	if second == -1 {
		return SymbolNameKey{}, fmt.Errorf("invalid symbol_name key: %s", key)
	}

	second += first + 1

	if key[:first] != SymKeySystemPrefix {
		return SymbolNameKey{}, fmt.Errorf("invalid symbol_name key: %s", key)
	}

	return SymbolNameKey{
		Language: lang.Language(key[first+1 : second]),
		Name:     key[second+1:],
	}, nil
}

func ToElementPathKey(key string) (ElementPathKey, error) {
	// 查找第一个冒号位置
	first := strings.Index(key, types.Colon)
	if first == -1 {
		return ElementPathKey{}, fmt.Errorf("invalid element_path key: %s", key)
	}
	// 查找第二个冒号位置
	second := strings.Index(key[first+1:], types.Colon)
	if second == -1 {
		return ElementPathKey{}, fmt.Errorf("invalid element_path key: %s", key)
	}

	second += first + 1

	if key[:first] != PathKeySystemPrefix {
		return ElementPathKey{}, fmt.Errorf("invalid element_path key: %s", key)
	}

	return ElementPathKey{
		Language: lang.Language(key[first+1 : second]),
		Path:     key[second+1:],
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
