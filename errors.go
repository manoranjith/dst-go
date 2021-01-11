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

package perun

import (
	"errors"
)

type ErrorCategory int

const (
	ClientError ErrorCategory = iota
	ParticipantError
	ProtocolFatalError
	InternalError
)

type ErrorCode int

const (
	ResourceNotFound      ErrorCode = 201
	ResourceAlreadyExists ErrorCode = 202
	InvalidArguments      ErrorCode = 203 // TODO: Remove reason, it will go into message.
	FailedPreCondition    ErrorCode = 204 // TODO: Remove reason and any info, all will go into message.
)

type ResourceAlreadyExistsInfo struct {
	ResourceType string
	ResourceID   string
}

type InvalidArgumentsInfo struct {
	ArgName  string
	ArgValue string
}

type APIError2 struct {
	category ErrorCategory
	code     ErrorCode
	message  string
	addInfo  interface{}
}

func (e APIError2) Category() ErrorCategory {
	return e.category
}

func (e APIError2) Code() ErrorCode {
	return e.code
}

func (e APIError2) Message() string {
	return e.message
}

func (e APIError2) AddInfo() interface{} {
	return e.addInfo
}

func NewErrorResourceAlreadyExists(resourceType, resourceID, message string) APIError2Inf {
	return APIError2{
		category: ClientError,
		code:     ResourceAlreadyExists,
		message:  message,
		addInfo: ResourceAlreadyExistsInfo{
			ResourceType: resourceType,
			ResourceID:   resourceID,
		},
	}
}

func NewErrorInvalidArguments(argName, argValue, message string) APIError2Inf {
	return APIError2{
		category: ClientError,
		code:     InvalidArguments,
		message:  message,
		addInfo: InvalidArgumentsInfo{
			ArgName:  argName,
			ArgValue: argValue,
		},
	}
}

func NewErrorFailedPreCondition(message string) APIError2Inf {
	return APIError2{
		category: ClientError,
		code:     FailedPreCondition,
		message:  message,
	}
}

type APIError2Inf interface {
	Category() ErrorCategory
	Code() ErrorCode
	Message() string
	AddInfo() interface{}
	Error() string
}

func (e APIError2) Error() string {
	return ""
}

// APIError represents the errors that will be communicated via the user API.
type APIError string

func (e APIError) Error() string {
	return string(e)
}

// GetAPIError returns the APIError contained in err if err is an APIError.
// If not, it returns ErrInternalServer API error.
func GetAPIError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		return ErrInternalServer
	}
	return apiErr
}

// Sentinal Error values that are relevant for the end user of the node.
var (
	ErrUnknownSessionID  = APIError("No session corresponding to the specified ID")
	ErrUnknownProposalID = APIError("No channel proposal corresponding to the specified ID")
	ErrUnknownChID       = APIError("No channel corresponding to the specified ID")
	ErrUnknownAlias      = APIError("No peer corresponding to the specified ID was found in ID Provider")
	ErrUnknownUpdateID   = APIError("No response was expected for the given channel update ID")

	ErrUnsupportedCurrency       = APIError("Currency not supported by this node instance")
	ErrUnsupportedIDProviderType = APIError("ID Provider type not supported by this node instance")
	ErrUnsupportedCommType       = APIError("Communication protocol not supported by this node instance")

	ErrInsufficientBal     = APIError("Insufficient balance in sender account")
	ErrInvalidAmount       = APIError("Invalid amount string")
	ErrMissingBalance      = APIError("Missing balance")
	ErrInvalidConfig       = APIError("Invalid configuration detected")
	ErrInvalidOffChainAddr = APIError("Invalid off-chain address string")
	ErrInvalidPayee        = APIError("Invalid payee, no such participant in the channel")

	ErrNoActiveSub      = APIError("No active subscription was found")
	ErrSubAlreadyExists = APIError("A subscription for this context already exists")

	ErrSessionClosed      = APIError("Session is closed")
	ErrChFinalized        = APIError("Channel is finalized")
	ErrChClosed           = APIError("Channel is closed")
	ErrPeerAliasInUse     = APIError("Alias already used by another peer in the ID Provider")
	ErrPeerExists         = APIError("Peer ID already available in the ID provider")
	ErrRespTimeoutExpired = APIError("Response to the notification was sent after the timeout has expired")
	ErrPeerRejected       = APIError("The request was rejected by peer")

	ErrOpenCh         = APIError("Session cannot be closed (without force option) as there are open channels")
	ErrInternalServer = APIError("Internal Server Error")
)
