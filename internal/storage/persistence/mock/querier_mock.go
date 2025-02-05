// Code generated by MockGen. DO NOT EDIT.
// Source: ./.. (interfaces: Querier)
//
// Generated by this command:
//
//	mockgen ./.. Querier
//

// Package mock_persistence is a generated GoMock package.
package mock_persistence

import (
	context "context"
	reflect "reflect"

	persistence "github.com/GyroZepelix/mithril-cms/internal/storage/persistence"
	uuid "github.com/google/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockQuerier is a mock of Querier interface.
type MockQuerier struct {
	ctrl     *gomock.Controller
	recorder *MockQuerierMockRecorder
	isgomock struct{}
}

// MockQuerierMockRecorder is the mock recorder for MockQuerier.
type MockQuerierMockRecorder struct {
	mock *MockQuerier
}

// NewMockQuerier creates a new mock instance.
func NewMockQuerier(ctrl *gomock.Controller) *MockQuerier {
	mock := &MockQuerier{ctrl: ctrl}
	mock.recorder = &MockQuerierMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQuerier) EXPECT() *MockQuerierMockRecorder {
	return m.recorder
}

// CreateContent mocks base method.
func (m *MockQuerier) CreateContent(ctx context.Context, arg persistence.CreateContentParams) (persistence.Post, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateContent", ctx, arg)
	ret0, _ := ret[0].(persistence.Post)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateContent indicates an expected call of CreateContent.
func (mr *MockQuerierMockRecorder) CreateContent(ctx, arg any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateContent", reflect.TypeOf((*MockQuerier)(nil).CreateContent), ctx, arg)
}

// CreateUser mocks base method.
func (m *MockQuerier) CreateUser(ctx context.Context, arg persistence.CreateUserParams) (persistence.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUser", ctx, arg)
	ret0, _ := ret[0].(persistence.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateUser indicates an expected call of CreateUser.
func (mr *MockQuerierMockRecorder) CreateUser(ctx, arg any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUser", reflect.TypeOf((*MockQuerier)(nil).CreateUser), ctx, arg)
}

// GetContent mocks base method.
func (m *MockQuerier) GetContent(ctx context.Context, id uuid.UUID) (persistence.Post, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContent", ctx, id)
	ret0, _ := ret[0].(persistence.Post)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContent indicates an expected call of GetContent.
func (mr *MockQuerierMockRecorder) GetContent(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContent", reflect.TypeOf((*MockQuerier)(nil).GetContent), ctx, id)
}

// GetUser mocks base method.
func (m *MockQuerier) GetUser(ctx context.Context, id uuid.UUID) (persistence.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, id)
	ret0, _ := ret[0].(persistence.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUser indicates an expected call of GetUser.
func (mr *MockQuerierMockRecorder) GetUser(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockQuerier)(nil).GetUser), ctx, id)
}

// GetUserByUsername mocks base method.
func (m *MockQuerier) GetUserByUsername(ctx context.Context, username string) (persistence.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByUsername", ctx, username)
	ret0, _ := ret[0].(persistence.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserByUsername indicates an expected call of GetUserByUsername.
func (mr *MockQuerierMockRecorder) GetUserByUsername(ctx, username any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByUsername", reflect.TypeOf((*MockQuerier)(nil).GetUserByUsername), ctx, username)
}

// ListContents mocks base method.
func (m *MockQuerier) ListContents(ctx context.Context) ([]persistence.Post, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListContents", ctx)
	ret0, _ := ret[0].([]persistence.Post)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListContents indicates an expected call of ListContents.
func (mr *MockQuerierMockRecorder) ListContents(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListContents", reflect.TypeOf((*MockQuerier)(nil).ListContents), ctx)
}

// ListUsers mocks base method.
func (m *MockQuerier) ListUsers(ctx context.Context) ([]persistence.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListUsers", ctx)
	ret0, _ := ret[0].([]persistence.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListUsers indicates an expected call of ListUsers.
func (mr *MockQuerierMockRecorder) ListUsers(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListUsers", reflect.TypeOf((*MockQuerier)(nil).ListUsers), ctx)
}
