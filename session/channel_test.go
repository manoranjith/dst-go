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
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	pchannel "perun.network/go-perun/channel"
	pclient "perun.network/go-perun/client"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
)

func Test_ChAPI_Interface(t *testing.T) {
	assert.Implements(t, (*perun.ChAPI)(nil), new(session.Channel))
}

func prepareChMockC2(t *testing.T, openingBalInfo perun.BalInfo) *mocks.Channel {
	ch := &mocks.Channel{}
	ch.On("ID").Return([32]byte{0, 1, 2})
	allocation, err := session.MakeAllocation(openingBalInfo, nil)
	require.NoError(t, err)
	state := &pchannel.State{
		ID:         [32]byte{0},
		Version:    0,
		App:        pchannel.NoApp(),
		Allocation: *allocation,
		Data:       pchannel.NoData(),
		IsFinal:    false,
	}
	ch.On("State").Return(state)
	watcherSignal := make(chan time.Time)
	ch.On("Watch").WaitUntil(watcherSignal).Return(nil)
	return ch
}

func Test_SendChUpdate(t *testing.T) {

	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}

	t.Run("happy", func(t *testing.T) {

		pch := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		gotChInfo, err := ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.NoError(t, err)
		assert.NotZero(t, gotChInfo)
	})

	t.Run("error_UpdateBy_RejectedByPeer", func(t *testing.T) {
		pch := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(errors.New("rejected by user"))
		_, err := ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.Error(t, err)
	})

	t.Run("error_channel_closed", func(t *testing.T) {
		pch := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)

		// == Test case specific prep ==
		_, err := ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.Error(t, err)
	})
}

func Test_HandleUpdate(t *testing.T) {

	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}
	updatedBalInfo := validOpeningBalInfo
	updatedBalInfo.Bal = []string{"0.5", "2.5"}
	pch := prepareChMockC2(t, validOpeningBalInfo)

	allocation, err := session.MakeAllocation(updatedBalInfo, nil)
	require.NoError(t, err)
	state := &pchannel.State{
		ID:         [32]byte{0},
		Version:    0,
		App:        pchannel.NoApp(),
		Allocation: *allocation,
		Data:       pchannel.NoData(),
		IsFinal:    false,
	}

	t.Run("happy", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: state,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		ch.HandleUpdateWInterface(*chUpdate, &mocks.ChUpdateResponder{})
	})

	t.Run("happy", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: state,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)
		ch.HandleUpdateWInterface(*chUpdate, &mocks.ChUpdateResponder{})
	})

}

func Test_SubUnsubChUpdate(t *testing.T) {

	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}

	dummyNotifier := func(notif perun.ChUpdateNotif) {}
	pch := prepareChMockC2(t, validOpeningBalInfo)
	ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

	// SubTest 1: Sub succesfully ==
	err := ch.SubChUpdates(dummyNotifier)
	require.NoError(t, err)

	// SubTest 2: Sub again, should error ==
	err = ch.SubChUpdates(dummyNotifier)
	require.Error(t, err)

	// SubTest 3: UnSub succesfully ==
	err = ch.UnsubChUpdates()
	require.NoError(t, err)

	// SubTest 4: UnSub again, should error ==
	err = ch.UnsubChUpdates()
	require.Error(t, err)

	t.Run("error_Sub_channelClosed", func(t *testing.T) {
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)
		err = ch.SubChUpdates(dummyNotifier)
		require.Error(t, err)
	})
	t.Run("error_Unsub_channelClosed", func(t *testing.T) {
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)
		err = ch.UnsubChUpdates()
		require.Error(t, err)
	})
}
