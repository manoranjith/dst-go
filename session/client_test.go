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

package session_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
)

func Test_ChannelClient_Interface(t *testing.T) {
	assert.Implements(t, (*perun.ChannelClient)(nil), new(session.Client))
}

func Test_Client_Close(t *testing.T) {
	// happy path test is covered in integration test, as internal components of
	// the client should be initialized.
	t.Run("err_channelClient_Err", func(t *testing.T) {
		chClient := &mocks.ChannelClient{}
		msgBus := &mocks.WireBus{}
		Client := session.Client{
			ChannelClient: chClient,
			WireBus:       msgBus,
		}

		chClient.On("Close").Return(errors.New("error for test"))
		msgBus.On("Close").Return(nil)
		assert.Error(t, Client.Close())
	})

	t.Run("err_wireBus_Err", func(t *testing.T) {
		chClient := &mocks.ChannelClient{}
		msgBus := &mocks.WireBus{}
		Client := session.Client{
			ChannelClient: chClient,
			WireBus:       msgBus,
		}

		chClient.On("Close").Return(nil)
		msgBus.On("Close").Return(errors.New("error for test"))
		assert.Error(t, Client.Close())
	})
}
