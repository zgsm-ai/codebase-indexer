package store

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
)

const (
	TestProjectID = "test-project"
)

var (
	ErrTestError = errors.New("test error")
)

// GenerateTestProjectUUID generates a test project UUID using the same method as workspace.Project.Uuid()
func GenerateTestProjectUUID(name, path string) string {
	hash := sha256.Sum256([]byte(path))
	return name + "_" + hex.EncodeToString(hash[:])
}

// TestKey 用于测试的键类型
type TestKey struct {
	key string
}

func (k TestKey) Get() (string, error) {
	return k.key, nil
}

// MockLogger 用于测试的 mock logger
type MockLogger struct{}

func (m *MockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {}
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (m *MockLogger) Fatal(msg string, keysAndValues ...interface{}) {}

// TestValues 用于测试的 Entries 实现
type TestValues struct {
	values []proto.Message
	keys   []Key
}

func (tv *TestValues) Len() int {
	return len(tv.values)
}

func (tv *TestValues) Key(i int) Key {
	if i < len(tv.keys) {
		return tv.keys[i]
	}
	return ElementPathKey{Language: lang.Go, Path: fmt.Sprintf("key-%d", i)}
}

func (tv *TestValues) Value(i int) proto.Message {
	if i < len(tv.values) {
		return tv.values[i]
	}
	return &codegraphpb.TestMessage{Value: "default"}
}

// CreateTestValues 创建测试用的Values实现
func CreateTestValues(values []proto.Message, keys []Key) *TestValues {
	if keys == nil {
		keys = make([]Key, len(values))
		for i := range values {
			keys[i] = ElementPathKey{Language: lang.Go, Path: fmt.Sprintf("key-%d", i)}
		}
	}
	return &TestValues{
		values: values,
		keys:   keys,
	}
}

// CreateTestMessages 创建测试用的Protobuf消息
func CreateTestMessages(count int, valuePrefix string) []proto.Message {
	values := make([]proto.Message, count)
	for i := 0; i < count; i++ {
		values[i] = &codegraphpb.TestMessage{Value: fmt.Sprintf("%s-%d", valuePrefix, i)}
	}
	return values
}

// CreateTestKeys 创建测试用的键
func CreateTestKeys(count int, keyPrefix string) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		keys[i] = fmt.Sprintf("%s-%d", keyPrefix, i)
	}
	return keys
}
