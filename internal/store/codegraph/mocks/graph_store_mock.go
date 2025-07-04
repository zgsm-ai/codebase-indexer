// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/store/codegraph/store.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	codegraphpb "github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	types "github.com/zgsm-ai/codebase-indexer/internal/types"
)

// MockGraphStore is a mock of GraphStore interface.
type MockGraphStore struct {
	ctrl     *gomock.Controller
	recorder *MockGraphStoreMockRecorder
}

// MockGraphStoreMockRecorder is the mock recorder for MockGraphStore.
type MockGraphStoreMockRecorder struct {
	mock *MockGraphStore
}

// NewMockGraphStore creates a new mock instance.
func NewMockGraphStore(ctrl *gomock.Controller) *MockGraphStore {
	mock := &MockGraphStore{ctrl: ctrl}
	mock.recorder = &MockGraphStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGraphStore) EXPECT() *MockGraphStoreMockRecorder {
	return m.recorder
}

// BatchWrite mocks base method.
func (m *MockGraphStore) BatchWrite(ctx context.Context, docs []*codegraphpb.Document) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchWrite", ctx, docs)
	ret0, _ := ret[0].(error)
	return ret0
}

// BatchWrite indicates an expected call of BatchWrite.
func (mr *MockGraphStoreMockRecorder) BatchWrite(ctx, docs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchWrite", reflect.TypeOf((*MockGraphStore)(nil).BatchWrite), ctx, docs)
}

// BatchWriteCodeStructures mocks base method.
func (m *MockGraphStore) BatchWriteCodeStructures(ctx context.Context, docs []*codegraphpb.CodeDefinition) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchWriteCodeStructures", ctx, docs)
	ret0, _ := ret[0].(error)
	return ret0
}

// BatchWriteCodeStructures indicates an expected call of BatchWriteCodeStructures.
func (mr *MockGraphStoreMockRecorder) BatchWriteCodeStructures(ctx, docs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchWriteCodeStructures", reflect.TypeOf((*MockGraphStore)(nil).BatchWriteCodeStructures), ctx, docs)
}

// Close mocks base method.
func (m *MockGraphStore) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockGraphStoreMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockGraphStore)(nil).Close))
}

// Delete mocks base method.
func (m *MockGraphStore) Delete(ctx context.Context, files []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteByCodebase", ctx, files)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockGraphStoreMockRecorder) Delete(ctx, files interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteByCodebase", reflect.TypeOf((*MockGraphStore)(nil).Delete), ctx, files)
}

// DeleteAll mocks base method.
func (m *MockGraphStore) DeleteAll(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAll", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAll indicates an expected call of DeleteAll.
func (mr *MockGraphStoreMockRecorder) DeleteAll(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAll", reflect.TypeOf((*MockGraphStore)(nil).DeleteAll), ctx)
}

// QueryDefinition mocks base method.
func (m *MockGraphStore) QueryDefinition(ctx context.Context, opts *types.DefinitionRequest) ([]*types.DefinitionNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryDefinitions", ctx, opts)
	ret0, _ := ret[0].([]*types.DefinitionNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryDefinition indicates an expected call of QueryDefinition.
func (mr *MockGraphStoreMockRecorder) QueryDefinition(ctx, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryDefinitions", reflect.TypeOf((*MockGraphStore)(nil).QueryDefinition), ctx, opts)
}

// QueryRelation mocks base method.
func (m *MockGraphStore) QueryRelation(ctx context.Context, opts *types.RelationRequest) ([]*types.GraphNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryRelations", ctx, opts)
	ret0, _ := ret[0].([]*types.GraphNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryRelation indicates an expected call of QueryRelation.
func (mr *MockGraphStoreMockRecorder) QueryRelation(ctx, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryRelations", reflect.TypeOf((*MockGraphStore)(nil).QueryRelation), ctx, opts)
}
