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

package payment_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node/app/payment"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
)

var amountToSend = "0.5"

func Test_SendPayChUpdate(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("GetInfo").Return(chInfo)
		channelAPI.On("SendChUpdate", context.Background(), mock.Anything).Return(nil)

		gotErr := payment.SendPayChUpdate(context.Background(), channelAPI, peerAlias, amountToSend)
		require.NoError(t, gotErr)
	})

	t.Run("error", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("GetInfo").Return(chInfo)
		channelAPI.On("SendChUpdate", context.Background(), mock.Anything).Return(assert.AnError)

		gotErr := payment.SendPayChUpdate(context.Background(), channelAPI, peerAlias, amountToSend)
		require.Error(t, gotErr)
	})
}
