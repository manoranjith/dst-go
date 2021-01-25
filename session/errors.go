// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/hyperledger-labs/perun-node
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package session

import (
	"fmt"
	"time"
)

// ErrorCategory represents the category of the error, which describes how the
// error should be handled by the client.
type ErrorCategory int

const (
	// ParticipantError is caused by one of the channel paritipants not acting
	// as per the perun protocol.
	//
	// To resolve this, the client should negotiate with the peer outside of
	// this system to act in accordance with the perun protocol.
	ParticipantError ErrorCategory = iota

	// ClientError is caused by the errors in the request from the client. it
	// could be errors in arguments or errors in configuration provided by the
	// client to access the external systems or errors in the state of external
	// systems not managed by the node.
	//
	// To resolve this, the client should provide valid arguments, provide
	// correct configuration to access the external systems or fix the external
	// systems; and retry.
	ClientError

	// ProtocolFatalError is caused when the protocol aborts due to unexpected
	// failure in external system during execution. It could also result in loss
	// of funds.
	//
	// To resolve this, user should maually inspect the error message and
	// handle it.
	ProtocolFatalError
	// InternalError is caused due to unintended behavior in the node software.
	//
	// To resolve this, user should maually inspect the error message and
	// handle it.
	InternalError
)

// String implements the stringer interface for ErrorCategory.
func (c ErrorCategory) String() string {
	return [...]string{
		"Client",
		"Participant",
		"Protocol Fatal",
		"Internal",
	}[c]
}

// ErrorCode is a numeric code assigned to identify the specific type of error.
// The keys in the additional field is fixed for each error code.
type ErrorCode int

// Error code definitions.
const (
	ErrPeerResponseTimedout      ErrorCode = 101
	ErrRejectedByPeer            ErrorCode = 102
	ErrPeerNotFunded             ErrorCode = 103
	ErrUserResponseTimedOut      ErrorCode = 104
	ErrResourceNotFound          ErrorCode = 201
	ErrResourceExists            ErrorCode = 202
	ErrInvalidArgument           ErrorCode = 203
	ErrFailedPreCondition        ErrorCode = 204
	ErrInvalidConfig             ErrorCode = 205
	ErrChainNodeNotReachable     ErrorCode = 206
	ErrInvalidContracts          ErrorCode = 207
	ErrTxTimedOut                ErrorCode = 301
	ErrInsufficientBalForTx      ErrorCode = 302
	ErrChainNodeDisconnected     ErrorCode = 303
	ErrInsufficientBalForDeposit ErrorCode = 304
	ErrUnknownInternal           ErrorCode = 401
	ErrOffChainComm              ErrorCode = 402
)

type (
	// PeerResponseTimedoutInfo represents the fields in the additional info for
	// ErrPeerResponseTimedout.
	PeerResponseTimedoutInfo struct {
		PeerAlias       string
		ResponseTimeout string
	}
)

// APIError represents the error that will returned by the API of perun node.
type APIError struct {
	category ErrorCategory
	code     ErrorCode
	message  string
	addInfo  interface{}
}

// Category returns the error category for this API Error.
func (e APIError) Category() ErrorCategory {
	return e.category
}

// Code returns the error code for this API Error.
func (e APIError) Code() ErrorCode {
	return e.code
}

// Message returns the error message for this API Error.
func (e APIError) Message() string {
	return e.message
}

// AddInfo returns the additional info for this API Error.
func (e APIError) AddInfo() interface{} {
	return e.addInfo
}

// Error implement the error interface for API error.
func (e APIError) Error() string {
	return fmt.Sprintf("Category: %s, Code: %d, Message: %s, AddInfo: %+v",
		e.Category(), e.Code(), e.Message(), e.AddInfo())
}

// NewErrPeerResponseTimedout returns an ErrPeerResponseTimedout API Error with
// the given peer alias and response timeout.
func NewErrPeerResponseTimedout(peerAlias string, timeout time.Duration) APIError {
	return APIError{
		category: ParticipantError,
		code:     ErrPeerResponseTimedout,
		message:  "Peer Response Timed Out",
		addInfo: PeerResponseTimedoutInfo{
			PeerAlias:       peerAlias,
			ResponseTimeout: timeout.String(),
		},
	}
}
