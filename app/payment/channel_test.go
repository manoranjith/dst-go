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

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/app/payment"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
)

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

func Test_GetBalInfo(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("GetInfo").Return(chInfo)

		gotBalInfo := payment.GetBalInfo(channelAPI)
		assert.Equal(t, wantBalInfo, gotBalInfo)
	})
}

func Test_SubPayChUpdates(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var notifier perun.ChUpdateNotifier
		var notif payment.PayChUpdateNotif
		dummyNotifier := func(gotNotif payment.PayChUpdateNotif) {
			notif = gotNotif
		}
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("SubChUpdates", mock.MatchedBy(func(gotNotifier perun.ChUpdateNotifier) bool {
			notifier = gotNotifier
			return true
		})).Return(nil)

		gotErr := payment.SubPayChUpdates(channelAPI, dummyNotifier)
		assert.NoError(t, gotErr)
		require.NotNil(t, notifier)

		notifier(chUpdateNotif)
		require.NotZero(t, notif)
		require.Equal(t, chUpdateNotif.UpdateID, notif.UpdateID)
		require.Equal(t, wantUpdatedBalInfo, notif.ProposedBals)
		require.Equal(t, versionString, notif.Version)
		require.Equal(t, chUpdateNotif.Update.State.IsFinal, notif.Final)
		require.Equal(t, chUpdateNotif.Expiry, notif.Timeout)
	})
	t.Run("error", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("SubChUpdates", mock.Anything).Return(assert.AnError)

		dummyNotifier := func(notif payment.PayChUpdateNotif) {}
		gotErr := payment.SubPayChUpdates(channelAPI, dummyNotifier)
		assert.Error(t, gotErr)
	})
}

func Test_UnsubPayChUpdates(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("UnsubChUpdates").Return(nil)

		gotErr := payment.UnsubPayChUpdates(channelAPI)
		assert.NoError(t, gotErr)
	})
	t.Run("error", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("UnsubChUpdates").Return(assert.AnError)

		gotErr := payment.UnsubPayChUpdates(channelAPI)
		assert.Error(t, gotErr)
	})
}

// nolint: dupl	// not duplicate of Test_RespondPayChProposal.
func Test_RespondPayChUpdate(t *testing.T) {
	updateID := "update-id-1"
	t.Run("happy_accept", func(t *testing.T) {
		accept := true
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("RespondChUpdate", context.Background(), updateID, accept).Return(nil)

		gotErr := payment.RespondPayChUpdate(context.Background(), channelAPI, updateID, accept)
		assert.NoError(t, gotErr)
	})
	t.Run("happy_reject", func(t *testing.T) {
		accept := false
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("RespondChUpdate", context.Background(), updateID, accept).Return(nil)

		gotErr := payment.RespondPayChUpdate(context.Background(), channelAPI, updateID, accept)
		assert.NoError(t, gotErr)
	})
	t.Run("error", func(t *testing.T) {
		accept := true
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("RespondChUpdate", context.Background(), updateID, accept).Return(assert.AnError)

		gotErr := payment.RespondPayChUpdate(context.Background(), channelAPI, updateID, accept)
		assert.Error(t, gotErr)
	})
}

func Test_ClosePayCh(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("Close", context.Background()).Return(chInfo, nil)

		gotPayChInfo, err := payment.ClosePayCh(context.Background(), channelAPI)
		require.NoError(t, err)
		assert.Equal(t, wantBalInfo, gotPayChInfo.BalInfo)
		assert.Equal(t, "0", gotPayChInfo.Version)
		assert.NotZero(t, gotPayChInfo.ChannelID)
	})
	t.Run("error", func(t *testing.T) {
		channelAPI := &mocks.ChannelAPI{}
		channelAPI.On("Close", context.Background()).Return(chInfo, assert.AnError)

		_, err := payment.ClosePayCh(context.Background(), channelAPI)
		require.Error(t, err)
	})
}
