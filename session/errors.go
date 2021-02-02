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

	"github.com/hyperledger-labs/perun-node"
)

// APIError represents the error that will returned by the API of perun node.
type APIError struct {
	category perun.ErrorCategory
	code     perun.ErrorCode
	message  string
	addInfo  interface{}
}

// Category returns the error category for this API Error.
func (e APIError) Category() perun.ErrorCategory {
	return e.category
}

// Code returns the error code for this API Error.
func (e APIError) Code() perun.ErrorCode {
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

// NewErrResourceNotFound returns an ErrResourceNotFound API Error with
// the given resource type, ID and error message.
func NewErrResourceNotFound(resourceType, resourceID, message string) APIError {
	return APIError{
		category: perun.ClientError,
		code:     perun.ErrV2ResourceNotFound,
		message:  message,
		addInfo: perun.ErrV2InfoResourceNotFound{
			Type: resourceType,
			ID:   resourceID,
		},
	}
}

// NewErrResourceExists returns an ErrResourceExists API Error with
// the given resource type, ID and error message.
func NewErrResourceExists(resourceType, resourceID, message string) APIError {
	return APIError{
		category: perun.ClientError,
		code:     perun.ErrV2ResourceExists,
		message:  message,
		addInfo: perun.ErrV2InfoResourceExists{
			Type: resourceType,
			ID:   resourceID,
		},
	}
}

// NewErrInvalidArgument returns an ErrInvalidArgument API Error with the given
// argument name, value, requirement for the argument and the error message.
func NewErrInvalidArgument(name, value, requirement, message string) APIError {
	return APIError{
		category: perun.ClientError,
		code:     perun.ErrV2InvalidArgument,
		message:  message,
		addInfo: perun.ErrV2InfoInvalidArgument{
			Name:        name,
			Value:       value,
			Requirement: requirement,
		},
	}
}
