// Code generated by MockGen. DO NOT EDIT.
// Source: use_case.go
//
// Generated by this command:
//
//	mockgen -source=use_case.go -destination=mocks.go -package=gomock
//

// Package gomock is a generated GoMock package.
package gomock

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// Mockrepository is a mock of repository interface.
type Mockrepository struct {
	ctrl     *gomock.Controller
	recorder *MockrepositoryMockRecorder
	isgomock struct{}
}

// MockrepositoryMockRecorder is the mock recorder for Mockrepository.
type MockrepositoryMockRecorder struct {
	mock *Mockrepository
}

// NewMockrepository creates a new mock instance.
func NewMockrepository(ctrl *gomock.Controller) *Mockrepository {
	mock := &Mockrepository{ctrl: ctrl}
	mock.recorder = &MockrepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mockrepository) EXPECT() *MockrepositoryMockRecorder {
	return m.recorder
}

// Insert mocks base method.
func (m *Mockrepository) Insert(ctx context.Context, val string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Insert", ctx, val)
	ret0, _ := ret[0].(error)
	return ret0
}

// Insert indicates an expected call of Insert.
func (mr *MockrepositoryMockRecorder) Insert(ctx, val any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*Mockrepository)(nil).Insert), ctx, val)
}

// Mocktransactor is a mock of transactor interface.
type Mocktransactor struct {
	ctrl     *gomock.Controller
	recorder *MocktransactorMockRecorder
	isgomock struct{}
}

// MocktransactorMockRecorder is the mock recorder for Mocktransactor.
type MocktransactorMockRecorder struct {
	mock *Mocktransactor
}

// NewMocktransactor creates a new mock instance.
func NewMocktransactor(ctrl *gomock.Controller) *Mocktransactor {
	mock := &Mocktransactor{ctrl: ctrl}
	mock.recorder = &MocktransactorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *Mocktransactor) EXPECT() *MocktransactorMockRecorder {
	return m.recorder
}

// WithinTx mocks base method.
func (m *Mocktransactor) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WithinTx", ctx, fn)
	ret0, _ := ret[0].(error)
	return ret0
}

// WithinTx indicates an expected call of WithinTx.
func (mr *MocktransactorMockRecorder) WithinTx(ctx, fn any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WithinTx", reflect.TypeOf((*Mocktransactor)(nil).WithinTx), ctx, fn)
}
