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

// +build integration

package session_test

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	ppayment "perun.network/go-perun/apps/payment"
	pchannel "perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

// init() initializes the payment app in go-perun.
func init() {
	wb := ethereum.NewWalletBackend()
	emptyAddr, err := wb.ParseAddr("0x0")
	if err != nil {
		panic("Error parsing zero address for app payment def: " + err.Error())
	}
	ppayment.SetAppDef(emptyAddr) // dummy app def.
}

// This test includes all methods on SessionAPI and ChannelAPI.
func Test_Integ_Role(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))

	aliceAlias, bobAlias := "alice", "bob"

	// Start with empty contacts.
	aliceCfg := sessiontest.NewConfig(t, prng)
	bobCfg := sessiontest.NewConfig(t, prng)

	alice, err := session.New(aliceCfg)
	require.NoErrorf(t, err, "initializing alice session")
	t.Logf("alice session id: %s\n", alice.ID())

	bob, err := session.New(bobCfg)
	require.NoErrorf(t, err, "initializing bob session")
	t.Logf("bob session id: %s\n", bob.ID())

	var aliceContact, bobContact perun.Peer
	t.Run("GetContact", func(t *testing.T) {
		t.Run("happy", func(t *testing.T) {
			aliceContact, err = alice.GetContact(perun.OwnAlias)
			require.NoErrorf(t, err, "Alice: GetContact")
			aliceContact.Alias = aliceAlias

			bobContact, err = bob.GetContact(perun.OwnAlias)
			require.NoErrorf(t, err, "Bob: GetContact")
			bobContact.Alias = bobAlias
		})
		t.Run("missing", func(t *testing.T) {
			_, err = alice.GetContact("random alias")
			assert.Errorf(t, err, "Alice: GetContact")
			t.Log(err)
		})
	})

	t.Run("AddContact", func(t *testing.T) {
		t.Run("happy", func(t *testing.T) {
			err = alice.AddContact(bobContact)
			require.NoErrorf(t, err, "Alice: AddContact")

			err = bob.AddContact(aliceContact)
			require.NoErrorf(t, err, "Bob: GetContact")
		})
		t.Run("already_exists", func(t *testing.T) {
			// Try to add bob contact again
			err = alice.AddContact(bobContact)
			assert.Errorf(t, err, "Alice: AddContact")
			t.Log(err)
		})
	})

	const challengeDurSecs uint64 = 10
	wg := &sync.WaitGroup{}
	ctx := context.Background()

	t.Run("OpenCh_Sub_Unsub_ChProposal_Respond_Accept", func(t *testing.T) {
		// Propose Channel by alice.
		wg.Add(1)
		go func() {
			defer wg.Done()
			bals := make(map[string]string)
			bals[perun.OwnAlias] = "1"
			bals[bobAlias] = "2"
			balInfo := perun.BalInfo{
				Currency: currency.ETH,
				Bals:     bals,
			}
			app := perun.App{
				Def:  ppayment.AppDef(),
				Data: &ppayment.NoData{},
			}
			// nolint: govet	// err does not shadow, using a new var to prevent data race.
			_, err := alice.OpenCh(ctx, bobAlias, balInfo, app, challengeDurSecs)
			require.NoErrorf(t, err, "alice opening channel with bob")
		}()

		// Accept channel by bob.
		bobChProposalNotif := make(chan perun.ChProposalNotif)
		bobChProposalNotifier := func(notif perun.ChProposalNotif) {
			bobChProposalNotif <- notif
		}
		err = bob.SubChProposals(bobChProposalNotifier)
		require.NoError(t, err, "bob subscribing channel proposals")

		notif := <-bobChProposalNotif
		err = bob.RespondChProposal(ctx, notif.ProposalID, true)
		require.NoError(t, err, "bob accepting channel proposal")

		err = bob.UnsubChProposals()
		require.NoError(t, err, "bob unsubscribing channel proposals")
		wg.Wait()
	})

	t.Run("OpenCh_Sub_Unsub_ChProposal_Respond_Reject", func(t *testing.T) {
		// Propose Channel by bob.
		wg.Add(1)
		go func() {
			defer wg.Done()
			bals := make(map[string]string)
			bals[aliceAlias] = "1"
			bals[perun.OwnAlias] = "2"
			balInfo := perun.BalInfo{
				Currency: currency.ETH,
				Bals:     bals,
			}
			app := perun.App{
				Def:  ppayment.AppDef(),
				Data: &ppayment.NoData{},
			}
			// nolint: govet	// err does not shadow, using a new var to prevent data race.
			_, err := bob.OpenCh(ctx, aliceAlias, balInfo, app, challengeDurSecs)
			require.Error(t, err, "bob channel rejected by alice")
			t.Log(err)
		}()

		// Reject channel by alice.
		aliceChProposalNotif := make(chan perun.ChProposalNotif)
		aliceChProposalNotifier := func(notif perun.ChProposalNotif) {
			aliceChProposalNotif <- notif
		}
		err = alice.SubChProposals(aliceChProposalNotifier)
		require.NoError(t, err, "alice subscribing channel proposals")

		notif := <-aliceChProposalNotif
		err = alice.RespondChProposal(ctx, notif.ProposalID, false)
		require.NoError(t, err, "alice rejecting channel proposal")

		err = alice.UnsubChProposals()
		require.NoError(t, err, "alice unsubscribing channel proposals")
		wg.Wait()
	})

	var aliceCh, bobCh perun.ChannelAPI
	t.Run("GetChInfos_GetCh", func(t *testing.T) {
		aliceChInfos := alice.GetChInfos()
		require.Lenf(t, aliceChInfos, 1, "alice session should have exactly one channel")
		bobChInfos := bob.GetChInfos()
		require.Lenf(t, bobChInfos, 1, "bob session should have exactly one channel")

		aliceCh, err = alice.GetCh(aliceChInfos[0].ChannelID)
		require.NoError(t, err, "getting alice ChannelAPI instance")

		bobCh, err = bob.GetCh(aliceChInfos[0].ChannelID)
		require.NoError(t, err, "getting bob ChannelAPI instance")
	})

	t.Run("SendUpdate_Sub_Unsub_ChUpdate_Respond_Accept", func(t *testing.T) {
		// Send update by bob.
		wg.Add(1)
		go func() {
			defer wg.Done()
			bobChInfo := bobCh.GetInfo()
			var ownIdx, peerIdx int
			if bobChInfo.Parts[0] == perun.OwnAlias {
				ownIdx = 0
			} else {
				ownIdx = 1
			}
			peerIdx = ownIdx ^ 1
			amountToSend := decimal.NewFromFloat(0.5e18).BigInt()

			updater := func(state *pchannel.State) {
				bals := state.Allocation.Clone().Balances[0]
				bals[ownIdx].Sub(bals[ownIdx], amountToSend)
				bals[peerIdx].Add(bals[peerIdx], amountToSend)
				state.Allocation.Balances[0] = bals
			}

			err := bobCh.SendChUpdate(ctx, updater)
			require.NoError(t, err)
		}()

		// Accept channel by alice.
		aliceChUpdateNotif := make(chan perun.ChUpdateNotif)
		aliceChUpdateNotifier := func(notif perun.ChUpdateNotif) {
			aliceChUpdateNotif <- notif
		}
		err := aliceCh.SubChUpdates(aliceChUpdateNotifier)
		require.NoError(t, err, "alice subscribing channel proposals")

		notif := <-aliceChUpdateNotif
		err = aliceCh.RespondChUpdate(ctx, notif.UpdateID, true)
		require.NoError(t, err, "alice accepting channel update")

		err = aliceCh.UnsubChUpdates()
		require.NoError(t, err, "alice unsubscribing channel updates")
		wg.Wait()
	})

	t.Run("SendUpdate_Sub_Unsub_ChUpdate_Respond_Reject", func(t *testing.T) {
		// Send update by alice.
		wg.Add(1)
		go func() {
			defer wg.Done()
			aliceChInfo := aliceCh.GetInfo()
			var ownIdx, peerIdx int
			if aliceChInfo.Parts[0] == perun.OwnAlias {
				ownIdx = 0
			} else {
				ownIdx = 1
			}
			peerIdx = ownIdx ^ 1
			amountToSend := decimal.NewFromFloat(0.5e18).BigInt()

			updater := func(state *pchannel.State) {
				bals := state.Allocation.Clone().Balances[0]
				bals[ownIdx].Sub(bals[ownIdx], amountToSend)
				bals[peerIdx].Add(bals[peerIdx], amountToSend)
				state.Allocation.Balances[0] = bals
			}

			err := aliceCh.SendChUpdate(ctx, updater)
			require.Error(t, err, "alice update rejected by bob")
			t.Log(err)
		}()

		// Reject channel by bob.
		bobChUpdateNotif := make(chan perun.ChUpdateNotif)
		bobChUpdateNotifier := func(notif perun.ChUpdateNotif) {
			bobChUpdateNotif <- notif
		}
		err := bobCh.SubChUpdates(bobChUpdateNotifier)
		require.NoError(t, err, "bob subscribing channel proposals")

		notif := <-bobChUpdateNotif
		err = bobCh.RespondChUpdate(ctx, notif.UpdateID, false)
		require.NoError(t, err, "bob accepting channel update")

		err = bobCh.UnsubChUpdates()
		require.NoError(t, err, "bob unsubscribing channel updates")
		wg.Wait()
	})
}
