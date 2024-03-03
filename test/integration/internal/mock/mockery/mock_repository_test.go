// Code generated by mockery v2.40.1. DO NOT EDIT.

package mockery

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// repositoryMock is an autogenerated mock type for the repository type
type repositoryMock struct {
	mock.Mock
}

// Insert provides a mock function with given fields: ctx, val
func (_m *repositoryMock) Insert(ctx context.Context, val string) error {
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

// newRepositoryMock creates a new instance of repositoryMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newRepositoryMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *repositoryMock {
	mock := &repositoryMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
