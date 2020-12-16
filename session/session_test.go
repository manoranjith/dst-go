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

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	pchannel "perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum/ethereumtest"
	"github.com/hyperledger-labs/perun-node/contacts/contactsyaml"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

func sessionWithDummyChClient(t *testing.T, isOpen bool, peers ...perun.Peer) perun.SessionAPI {
	prng := rand.New(rand.NewSource(1729))
	cfg := sessiontest.NewConfigT(t, prng, peers...)
	chClient := &mocks.ChClient{}
	session, err := session.NewSessionForTest(cfg, isOpen, chClient)
	require.NoError(t, err)
	require.NotNil(t, session)
	return session
}

func Test_AddContact(t *testing.T) {
	// == Setup ==
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))

	// In openSession, peer0 is already present, peer1 can be added.
	openSession := sessionWithDummyChClient(t, true, peers[0])
	closedSession := sessionWithDummyChClient(t, false, peers[0])

	t.Run("happy_add_contact", func(t *testing.T) {
		err := openSession.AddContact(peers[1])
		require.NoError(t, err)
	})

	t.Run("error_alias_used_for_diff_peer_id", func(t *testing.T) {
		// == Test ==
		peer1WithAlias0 := peers[1]
		peer1WithAlias0.Alias = "0"
		err := openSession.AddContact(peer1WithAlias0)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_peerID_already_registered", func(t *testing.T) {
		err := openSession.AddContact(peers[0])
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_session_closed", func(t *testing.T) {
		err := closedSession.AddContact(peers[0])
		require.Error(t, err)
		t.Log(err)
	})
}

func Test_GetContact(t *testing.T) {
	// == Setup ==
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(1))

	// In openSession, peer0 is present and peer1 is not present.
	openSession := sessionWithDummyChClient(t, true, peers[0])
	closedSession := sessionWithDummyChClient(t, false, peers[0])

	t.Run("happy_get_contact", func(t *testing.T) {
		peerID, err := openSession.GetContact(peers[0].Alias)
		require.NoError(t, err)
		assert.True(t, contactsyaml.PeerEqual(peerID, peers[0]))
	})

	t.Run("error_contact_not_found", func(t *testing.T) {
		_, err := openSession.GetContact("unknown-alias")
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_session_closed", func(t *testing.T) {
		_, err := closedSession.GetContact(peers[0].Alias)
		require.Error(t, err)
		t.Log(err)
	})
}

func Test_OpenCh(t *testing.T) {
	// == Setup ==
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2)) // Aliases of peers are their respective indices in the array.
	prng = rand.New(rand.NewSource(1729))
	cfg := sessiontest.NewConfigT(t, prng, peers...)
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, "1"},
		Bal:      []string{"1", "2"},
	}
	app := perun.App{
		Def:  pchannel.NoApp(),
		Data: pchannel.NoData(),
	}

	prepareChMock := func(openingBalInfo perun.BalInfo) perun.Channel {
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

	t.Run("happy_1_own_alias_first", func(t *testing.T) {
		// == Prepare mocks ==
		ch := prepareChMock(validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		chInfo, err := session.OpenCh(context.Background(), validOpeningBalInfo, app, 10)
		require.NoError(t, err)
		require.NotZero(t, chInfo)
	})

	t.Run("happy_2_own_alias_not_first", func(t *testing.T) {
		validOpeningBalInfo2 := perun.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{"1", perun.OwnAlias},
			Bal:      []string{"1", "2"},
		}

		// == Prepare mocks ==
		ch := prepareChMock(validOpeningBalInfo2)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		chInfo, err := session.OpenCh(context.Background(), validOpeningBalInfo2, app, 10)
		require.NoError(t, err)
		require.NotZero(t, chInfo)
	})

	t.Run("error_session_closed", func(t *testing.T) {
		// == Prepare mocks ==
		ch := prepareChMock(validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, false, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), validOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_missing_parts", func(t *testing.T) {
		invalidOpeningBalInfo := perun.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{perun.OwnAlias, "3"},
			Bal:      []string{"1", "2"},
		}

		// == Prepare mocks ==
		// Ignore ch and define only chClient mock with no method on it,
		// because the test will fail before ProposeChannel call.
		chClient := &mocks.ChClient{}
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), invalidOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_repeated_parts", func(t *testing.T) {
		invalidOpeningBalInfo := perun.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{"1", "1"},
			Bal:      []string{"1", "2"},
		}

		// == Prepare mocks ==
		// Ignore ch and define only chClient mock with no method on it,
		// because the test will fail before ProposeChannel call.
		chClient := &mocks.ChClient{}
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), invalidOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_missing_own_alias", func(t *testing.T) {
		invalidOpeningBalInfo := perun.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{"1", "2"},
			Bal:      []string{"1", "2"},
		}

		// == Prepare mocks ==
		// Ignore ch and define only chClient mock with no method on it,
		// because the test will fail before ProposeChannel call.
		chClient := &mocks.ChClient{}
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), invalidOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_unsupported_currency", func(t *testing.T) {
		invalidOpeningBalInfo := perun.BalInfo{
			Currency: "unsupported-currency",
			Parts:    []string{"1", perun.OwnAlias},
			Bal:      []string{"1", "2"},
		}

		// == Prepare mocks ==
		// Ignore ch and define only chClient mock with just Register method.
		// because the test will fail before ProposeChannel call.
		ch := prepareChMock(validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), invalidOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_invalid_amount", func(t *testing.T) {
		openingBalInfo := perun.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{"1", perun.OwnAlias},
			Bal:      []string{"abc", "gef"},
		}

		// == Prepare mocks ==
		// Ignore ch and define only chClient mock with just Register method.
		// because the test will fail before ProposeChannel call.
		ch := prepareChMock(validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, nil)
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), openingBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_ProposeChannel_AnError", func(t *testing.T) {
		// == Prepare testcase specific mocks ==
		ch := prepareChMock(validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, assert.AnError)
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), validOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_ProposeChannel_PeerRejected", func(t *testing.T) {
		// == Prepare testcase specific mocks ==
		ch := prepareChMock(validOpeningBalInfo)
		chClient := &mocks.ChClient{}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(ch, errors.New("channel proposal rejected"))
		chClient.On("Register", mock.Anything, mock.Anything).Return()
		session, err := session.NewSessionForTest(cfg, true, chClient)
		require.NoError(t, err)
		require.NotNil(t, session)

		// == Test ==
		_, err = session.OpenCh(context.Background(), validOpeningBalInfo, app, 10)
		require.Error(t, err)
		t.Log(err)
	})
}

func newPeers(t *testing.T, prng *rand.Rand, n uint) []perun.Peer {
	peers := make([]perun.Peer, n)
	for i := range peers {
		port, err := freeport.GetFreePort()
		require.NoError(t, err)
		peers[i].Alias = fmt.Sprintf("%d", i)
		peers[i].OffChainAddrString = ethereumtest.NewRandomAddress(prng).String()
		peers[i].CommType = "tcp"
		peers[i].CommAddr = fmt.Sprintf("127.0.0.1:%d", port)
	}
	return peers
}
