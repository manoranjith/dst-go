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

func prepareChMockC2(t *testing.T, openingBalInfo perun.BalInfo) (*mocks.Channel, chan time.Time) {
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
	return ch, watcherSignal
}
func prepareChMockC3(t *testing.T, openingBalInfo perun.BalInfo) (*mocks.Channel, chan time.Time) {
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
	ch.On("Watch").WaitUntil(watcherSignal).Return(assert.AnError)
	return ch, watcherSignal
}

func Test_Getters(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}

	pch, _ := prepareChMockC2(t, validOpeningBalInfo)
	ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

	// == Test case specific prep ==
	assert.Equal(t, ch.ID(), fmt.Sprintf("%x", pch.ID()))
	assert.Equal(t, ch.Currency(), validOpeningBalInfo.Currency)
	assert.Equal(t, ch.Parts(), validOpeningBalInfo.Parts)
	assert.Equal(t, ch.ChallengeDurSecs(), uint64(10))
}

func Test_GetChInfo(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}

	t.Run("happy", func(t *testing.T) {
		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		chInfo := ch.GetChInfo()
		assert.Equal(t, chInfo.ChID, fmt.Sprintf("%x", pch.ID()))
		assert.Equal(t, chInfo.BalInfo.Parts, validOpeningBalInfo.Parts)
		assert.Equal(t, chInfo.BalInfo.Currency, validOpeningBalInfo.Currency)
	})

	t.Run("error_nil_state", func(t *testing.T) {
		pch := &mocks.Channel{}
		pch.On("ID").Return([32]byte{0, 1, 2})
		pch.On("State").Return(nil)
		watcherSignal := make(chan time.Time)
		pch.On("Watch").WaitUntil(watcherSignal).Return(nil)

		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		chInfo := ch.GetChInfo()
		assert.Zero(t, chInfo)
	})
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

		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		gotChInfo, err := ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.NoError(t, err)
		assert.NotZero(t, gotChInfo)
	})

	t.Run("error_UpdateBy_RejectedByPeer", func(t *testing.T) {
		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(errors.New("rejected by user"))
		_, err := ch.SendChUpdate(context.Background(), func(s *pchannel.State) {})
		require.Error(t, err)
	})

	t.Run("error_channel_closed", func(t *testing.T) {
		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
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
	pch, _ := prepareChMockC2(t, validOpeningBalInfo)

	allocation, err := session.MakeAllocation(updatedBalInfo, nil)
	require.NoError(t, err)
	nonFinalState := pchannel.State{
		ID:         [32]byte{0},
		Version:    0,
		App:        pchannel.NoApp(),
		Allocation: *allocation,
		Data:       pchannel.NoData(),
		IsFinal:    false,
	}
	finalState := nonFinalState
	finalState.IsFinal = true

	t.Run("happy_nonFinal", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		ch.HandleUpdate(*chUpdate, &mocks.ChUpdateResponder{})
	})

	t.Run("happy_final", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &finalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		ch.HandleUpdate(*chUpdate, &mocks.ChUpdateResponder{})
	})

	t.Run("happy_unexpected_chUpdate", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)
		ch.HandleUpdate(*chUpdate, &mocks.ChUpdateResponder{})
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
	pch, _ := prepareChMockC2(t, validOpeningBalInfo)
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

func Test_HandleUpdate_Sub(t *testing.T) {

	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}
	updatedBalInfo := validOpeningBalInfo
	updatedBalInfo.Bal = []string{"0.5", "2.5"}
	pch, _ := prepareChMockC2(t, validOpeningBalInfo)

	allocation, err := session.MakeAllocation(updatedBalInfo, nil)
	require.NoError(t, err)
	nonFinalState := pchannel.State{
		ID:         [32]byte{0},
		Version:    0,
		App:        pchannel.NoApp(),
		Allocation: *allocation,
		Data:       pchannel.NoData(),
		IsFinal:    false,
	}

	t.Run("happy_HandleSub", func(t *testing.T) {
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch.HandleUpdate(*chUpdate, &mocks.ChUpdateResponder{})

		notifs := make([]perun.ChUpdateNotif, 0, 2)
		notifier := func(notif perun.ChUpdateNotif) {
			notifs = append(notifs, notif)
		}
		err := ch.SubChUpdates(notifier)
		require.NoError(t, err)
		notifRecieved := func() bool {
			return len(notifs) == 1
		}
		assert.Eventually(t, notifRecieved, 2*time.Second, 100*time.Millisecond)
	})
	t.Run("happy_SubHandle", func(t *testing.T) {
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		notifs := make([]perun.ChUpdateNotif, 0, 2)
		notifier := func(notif perun.ChUpdateNotif) {
			notifs = append(notifs, notif)
		}
		err := ch.SubChUpdates(notifier)
		require.NoError(t, err)
		notifRecieved := func() bool {
			return len(notifs) == 1
		}

		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch.HandleUpdate(*chUpdate, &mocks.ChUpdateResponder{})
		assert.Eventually(t, notifRecieved, 2*time.Second, 100*time.Millisecond)
	})

}

func Test_HandleUpdate_Respond(t *testing.T) {

	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}
	updatedBalInfo := validOpeningBalInfo
	updatedBalInfo.Bal = []string{"0.5", "2.5"}
	pch, _ := prepareChMockC2(t, validOpeningBalInfo)

	allocation, err := session.MakeAllocation(updatedBalInfo, nil)
	require.NoError(t, err)
	nonFinalState := pchannel.State{
		ID:         [32]byte{0},
		Version:    1,
		App:        pchannel.NoApp(),
		Allocation: *allocation,
		Data:       pchannel.NoData(),
		IsFinal:    false,
	}
	finalState := nonFinalState
	finalState.IsFinal = true

	t.Run("happy_Handle_Respond_Accept", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Accept", mock.Anything).Return(nil)
		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		chInfo, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.NoError(t, err)
		assert.NotZero(t, chInfo)
	})

	t.Run("happy_Handle_Respond_Reject", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Reject", mock.Anything, mock.Anything).Return(nil)
		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		chInfo, err := ch.RespondChUpdate(context.Background(), updateID, false)
		require.NoError(t, err)
		assert.NotZero(t, chInfo)
	})

	t.Run("error_Handle_Respond_Accept_Error", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Accept", mock.Anything).Return(assert.AnError)
		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		_, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_Handle_Respond_Reject_Error", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Reject", mock.Anything, mock.Anything).Return(assert.AnError)
		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		_, err := ch.RespondChUpdate(context.Background(), updateID, false)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_Handle_Respond_Unknown_UpdateID", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Accept", mock.Anything).Return(nil)
		updateID := "random-update-id"
		ch.HandleUpdate(*chUpdate, responder)

		_, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("error_Handle_Respond_Expired", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &nonFinalState,
		}
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Accept", mock.Anything).Return(nil)
		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		time.Sleep(2 * time.Second)
		_, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.Error(t, err)
	})

	t.Run("error_Handle_Respond_ChannelClosed", func(t *testing.T) {
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)
		updateID := "any-update-id" // A closed channel returns error irrespective of update id.

		_, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("happy_Handle_Respond_Accept_Final", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &finalState,
		}
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Accept", mock.Anything).Return(nil)
		pch.On("SettleSecondary", mock.Anything).Return(nil).Run(func(_ mock.Arguments) {
			watcherSignal <- time.Now() // Return the watcher once channel is settled.
		})
		pch.On("Close").Return(nil)

		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		chInfo, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.NoError(t, err)
		assert.NotZero(t, chInfo)
	})

	t.Run("error_Handle_Respond_Accept_SettleSecondaryError", func(t *testing.T) {
		chUpdate := &pclient.ChannelUpdate{
			State: &finalState,
		}
		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		responder := &mocks.ChUpdateResponder{}
		responder.On("Accept", mock.Anything).Return(nil)
		pch.On("SettleSecondary", mock.Anything).Return(assert.AnError)

		updateID := fmt.Sprintf("%s_%d", ch.ID(), chUpdate.State.Version)
		ch.HandleUpdate(*chUpdate, responder)

		_, err := ch.RespondChUpdate(context.Background(), updateID, true)
		require.Error(t, err)
		t.Log(err)
	})
}

func Test_Close(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}

	t.Run("happy_finalizeNoError_settle", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		pch.On("Settle", mock.Anything).Return(nil).Run(func(_ mock.Arguments) {
			watcherSignal <- time.Now() // Return the watcher once channel is settled.
		})
		pch.On("Close").Return(nil)

		gotChInfo, err := ch.Close(context.Background())
		require.NoError(t, err)
		assert.NotZero(t, gotChInfo)
	})

	t.Run("happy_finalizeError_settle", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(assert.AnError)
		pch.On("Settle", mock.Anything).Return(nil).Run(func(_ mock.Arguments) {
			watcherSignal <- time.Now() // Return the watcher once channel is settled.
		})
		pch.On("Close").Return(nil)

		gotChInfo, err := ch.Close(context.Background())
		require.NoError(t, err)
		assert.NotZero(t, gotChInfo)
	})

	t.Run("happy_closeError", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		pch.On("Settle", mock.Anything).Return(nil).Run(func(_ mock.Arguments) {
			watcherSignal <- time.Now() // Return the watcher once channel is settled.
		})
		pch.On("Close").Return(assert.AnError)

		gotChInfo, err := ch.Close(context.Background())
		require.NoError(t, err)
		assert.NotZero(t, gotChInfo)
	})

	t.Run("error_settleError", func(t *testing.T) {
		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)

		// == Test case specific prep ==
		pch.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		pch.On("Settle", mock.Anything).Return(assert.AnError)

		_, err := ch.Close(context.Background())
		require.Error(t, err)
	})

	t.Run("error_channel_closed", func(t *testing.T) {

		pch, _ := prepareChMockC2(t, validOpeningBalInfo)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)

		// == Test case specific prep ==
		_, err := ch.Close(context.Background())
		require.Error(t, err)
		t.Log(err)
	})
}

func Test_HandleWatcherReturned(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	peers := newPeers(t, prng, uint(2))
	validOpeningBalInfo := perun.BalInfo{
		Currency: currency.ETH,
		Parts:    []string{perun.OwnAlias, peers[0].Alias},
		Bal:      []string{"1", "2"},
	}

	t.Run("happy_openCh_dropNotif", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		pch.On("Close").Return(nil)
		_ = session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		watcherSignal <- time.Now()
	})

	t.Run("happy_closedCh_dropNotif", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		pch.On("Close").Return(nil)
		_ = session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, false)
		watcherSignal <- time.Now()
	})

	t.Run("happy_openCh_hasSub_WatchNoError", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC2(t, validOpeningBalInfo)
		pch.On("Close").Return(nil)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		notifs := make([]perun.ChUpdateNotif, 0, 2)
		notifer := func(notif perun.ChUpdateNotif) {
			notifs = append(notifs, notif)
		}
		require.NoError(t, ch.SubChUpdates(notifer))
		watcherSignal <- time.Now()
	})

	t.Run("happy_openCh_hasSub_WatchError", func(t *testing.T) {
		pch, watcherSignal := prepareChMockC3(t, validOpeningBalInfo)
		pch.On("Close").Return(nil)
		ch := session.NewChForTest(pch, currency.ETH, validOpeningBalInfo.Parts, 10, true)
		notifs := make([]perun.ChUpdateNotif, 0, 2)
		notifer := func(notif perun.ChUpdateNotif) {
			notifs = append(notifs, notif)
		}
		require.NoError(t, ch.SubChUpdates(notifer))
		watcherSignal <- time.Now()
	})
}
