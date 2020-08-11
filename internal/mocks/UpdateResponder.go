// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// UpdateResponder is an autogenerated mock type for the UpdateResponder type
type UpdateResponder struct {
	mock.Mock
}

// Accept provides a mock function with given fields: _a0
func (_m *UpdateResponder) Accept(_a0 context.Context) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Reject provides a mock function with given fields: ctx, reason
func (_m *UpdateResponder) Reject(ctx context.Context, reason string) error {
	ret := _m.Called(ctx, reason)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, reason)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
