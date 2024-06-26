// Code generated by mockery v2.42.1. DO NOT EDIT.

package mockery

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// mockRepository is an autogenerated mock type for the repository type
type mockRepository struct {
	mock.Mock
}

// Insert provides a mock function with given fields: ctx, val
func (_m *mockRepository) Insert(ctx context.Context, val string) error {
	ret := _m.Called(ctx, val)

	if len(ret) == 0 {
		panic("no return value specified for Insert")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, val)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// newMockRepository creates a new instance of mockRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockRepository {
	mock := &mockRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// mockTransactor is an autogenerated mock type for the transactor type
type mockTransactor struct {
	mock.Mock
}

// WithinTx provides a mock function with given fields: ctx, fn
func (_m *mockTransactor) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	ret := _m.Called(ctx, fn)

	if len(ret) == 0 {
		panic("no return value specified for WithinTx")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, func(context.Context) error) error); ok {
		r0 = rf(ctx, fn)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// newMockTransactor creates a new instance of mockTransactor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockTransactor(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockTransactor {
	mock := &mockTransactor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
