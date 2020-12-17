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
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	pchannel "perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
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
	prng = rand.New(rand.NewSource(1729))
	cfg := sessiontest.NewConfigT(t, prng, peers...)
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}
	app := perun.App{
		Def:  pchannel.NoApp(),
		Data: pchannel.NoData(),
	}

	t.Run("happy", func(t *testing.T) {

		pch := prepareChMockC2(t, validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(pch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()

		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		chInfo, err := session.OpenCh(context.Background(), validOpeningBalInfo, app, 10)
		require.NoError(t, err)
		require.NotZero(t, chInfo)

		chID := fmt.Sprintf("%x", pch.ID())
		ch, err := session.GetCh(chID)
		require.NoError(t, err)
		assert.Equal(t, ch.ID(), chID)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		gotChInfo, err := ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.NoError(t, err)
		assert.NotZero(t, gotChInfo)
	})

	t.Run("error_UpdateBy_error", func(t *testing.T) {
		pch := prepareChMockC2(t, validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(pch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()

		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		chInfo, err := session.OpenCh(context.Background(), validOpeningBalInfo, app, 10)
		require.NoError(t, err)
		require.NotZero(t, chInfo)

		chID := fmt.Sprintf("%x", pch.ID())
		ch, err := session.GetCh(chID)
		require.NoError(t, err)
		assert.Equal(t, ch.ID(), chID)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(assert.AnError)
		_, err = ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.Error(t, err)
	})
}
