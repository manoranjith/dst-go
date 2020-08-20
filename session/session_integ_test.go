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
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node"
	paymentAppLib "github.com/hyperledger-labs/perun-node/apps/payment"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum/ethereumtest"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

var (
	testdataDir = filepath.Join("..", "testdata", "contacts")

	aliceAlias       = "alice"
	alicePort        = 4341
	bobAlias         = "bob"
	bobPort          = 4342
	testContactsYAML = filepath.Join(testdataDir, "test.yaml")
)

func init() {
	session.SetWalletBackend(ethereumtest.NewTestWalletBackend())
}

func Test_Integ_New(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	_, testUser := sessiontest.NewTestUser(t, prng, 0)
	adjudicator, asset := ethereumtest.SetupContracts(t, testUser.OnChain, ethereumtest.TestChainURL, ethereumtest.OnChainTxTimeout)

	testUser.CommType = "tcp"
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	testUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", port)

	userCfg := session.UserConfig{
		Alias:       testUser.Alias,
		OnChainAddr: testUser.OnChain.Addr.String(),
		OnChainWallet: session.WalletConfig{
			KeystorePath: testUser.OnChain.Keystore,
			Password:     "",
		},
		OffChainAddr: testUser.OffChain.Addr.String(),
		OffChainWallet: session.WalletConfig{
			KeystorePath: testUser.OffChain.Keystore,
			Password:     "",
		},
		CommType: "tcp",
		CommAddr: testUser.CommAddr,
	}

	cfg := session.Config{
		User:             userCfg,
		ChainURL:         ethereumtest.TestChainURL,
		Adjudicator:      adjudicator.String(),
		Asset:            asset.String(),
		ChainConnTimeout: 30 * time.Second,
		ResponseTimeout:  10 * time.Second,
		OnChainTxTimeout: 5 * time.Second,
		DatabaseDir:      newDatabaseDir(t),

		ContactsType: "yaml",
		ContactsURL:  testContactsYAML,
	}

	sess, err := session.New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, sess)
}

func Test_Integ_Role_Bob(t *testing.T) {

	wg := sync.WaitGroup{}
	ctx := context.Background()

	alice, gotBobContact := newSession(t, aliceAlias)
	bob, gotAliceContact := newSession(t, bobAlias)
	var err error
	t.Log("alice session id:", alice.ID())
	t.Log("bob session id:", bob.ID())

	t.Log("add alice contact to bob")
	require.NoError(t, bob.AddContact(gotAliceContact))

	t.Log("add bob contact to alice")
	require.NoError(t, alice.AddContact(gotBobContact))

	t.Log("")
	t.Log("=====Starting channel proposal & accept sequence=====")
	t.Log("")
	t.Log("")
	t.Log("=====Starting channel proposal & accept sequence=====")
	t.Log("")
	var challengeDurSecs uint64 = 10
	var payChInfo paymentAppLib.PayChInfo

	wg.Add(1)
	go func() {
		defer wg.Done()

		aliceProposedBals := make(map[string]string)
		aliceProposedBals["self"] = "1"
		aliceProposedBals[bobAlias] = "2"
		aliceProposedBalInfo := perun.BalInfo{
			Currency: "ETH",
			Bals:     aliceProposedBals}
		payChInfo, err = paymentAppLib.OpenPayCh(ctx, alice, bobAlias, aliceProposedBalInfo, challengeDurSecs)
		require.NoError(t, err)
		t.Log("Alice opened payment channel", payChInfo)
	}()

	propNotif := make(chan paymentAppLib.PayChProposalNotif)
	proposalNotifier1 := func(notif paymentAppLib.PayChProposalNotif) {
		propNotif <- notif
	}
	err = paymentAppLib.SubPayChProposals(bob, proposalNotifier1)
	require.NoError(t, err)
	t.Log("Bob subscribed to payment proposal notifications")

	notif := <-propNotif
	t.Log("Bob received payment channel proposal notification", notif)

	err = paymentAppLib.RespondPayChProposal(ctx, bob, notif.ProposalID, true)
	require.NoError(t, err)
	t.Log("Bob accepted payment channel proposal")

	err = paymentAppLib.UnsubPayChProposals(bob)
	require.NoError(t, err)
	t.Log("Bob unsubscribed to payment proposal notifications")

	wg.Wait()

	t.Log("")
	t.Log("=====Completed channel proposal & accept sequence=====")
	t.Log("")

	// OpenCh: 2 proposes
	wg.Add(1)
	go func() {
		defer wg.Done()

		aliceProposedBals := make(map[string]string)
		aliceProposedBals["self"] = "1"
		aliceProposedBals[aliceAlias] = "2"
		aliceProposedBalInfo := perun.BalInfo{
			Currency: "ETH",
			Bals:     aliceProposedBals}
		_, err = paymentAppLib.OpenPayCh(ctx, bob, aliceAlias, aliceProposedBalInfo, challengeDurSecs)
		require.True(t, errors.Is(err, perun.ErrPeerRejected))
		t.Log(" payment channel rejected by peer")

	}()
	propNotif2 := make(chan paymentAppLib.PayChProposalNotif)
	proposalNotifier2 := func(notif paymentAppLib.PayChProposalNotif) {
		propNotif2 <- notif
	}
	err = paymentAppLib.SubPayChProposals(alice, proposalNotifier2)
	require.NoError(t, err)
	t.Log("Alice subscribed to payment proposal notifications")

	notif3 := <-propNotif2
	t.Log("Alice received payment channel proposal notification", notif3)

	err = paymentAppLib.RespondPayChProposal(ctx, alice, notif3.ProposalID, false)
	require.NoError(t, err)
	t.Log("Alice accepted payment channel proposal")

	err = paymentAppLib.UnsubPayChProposals(alice)
	require.NoError(t, err)
	t.Log("Alice unsubscribed to payment proposal notifications")

	wg.Wait()

	t.Log("Alice: Getting channel object from session")
	ch1, err := alice.GetCh(payChInfo.ChannelID)
	require.NoError(t, err)

	t.Log("Bob: Getting channel object from session")
	ch2, err := bob.GetCh(payChInfo.ChannelID)
	require.NoError(t, err)

	t.Log("Alice: Getting balance")
	balInfo := paymentAppLib.GetBalance(ch1)
	t.Log("Alice: Got Balance -", balInfo)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = paymentAppLib.SendPayChUpdate(ctx, ch1, "bob", "0.5")
		require.NoError(t, err)
	}()

	var updateNotifFrom1 = make(chan paymentAppLib.PayChUpdateNotif)
	PayChUpdateNotifAccept := func(notif paymentAppLib.PayChUpdateNotif) {
		fmt.Printf("\n Update Notification from 1: %+v\n", notif)
		updateNotifFrom1 <- notif
	}
	err = paymentAppLib.SubPayChUpdates(ch2, PayChUpdateNotifAccept)
	time.Sleep(1 * time.Second)

	notif2 := <-updateNotifFrom1
	err = paymentAppLib.RespondPayChUpdate(ctx, ch2, notif2.UpdateID, true)
	require.NoError(t, err)
	fmt.Println("Update was accepted")

	wg.Wait()

	balInfo = paymentAppLib.GetBalance(ch1)
	fmt.Printf("\n%+v", balInfo)

	// 2 closes the channel
	wg.Add(1)
	go func() {
		defer wg.Done()
		closingBal, err := paymentAppLib.ClosePayCh(ctx, ch1)
		require.NoError(t, err)
		fmt.Printf("\n%+v\n", closingBal)
		fmt.Println("channel was closed")
	}()

	// accept final update
	notif2 = <-updateNotifFrom1
	err = paymentAppLib.RespondPayChUpdate(ctx, ch2, notif2.UpdateID, true)
	require.NoError(t, err)
	fmt.Println("Update was accepted")

	// 1 subs from chClose
	var closeNotifFrom2 = make(chan paymentAppLib.PayChCloseNotif)
	PayChCloseNotifier := func(notif paymentAppLib.PayChCloseNotif) {
		fmt.Printf("\n Close Notification in session 1: %+v\n", notif)
		closeNotifFrom2 <- notif
	}
	err = paymentAppLib.SubPayChCloses(alice, PayChCloseNotifier)
	require.NoError(t, err)

	fmt.Printf("\n%+v\n", <-closeNotifFrom2)
	fmt.Println("channel notification was received")
	wg.Wait()
}

func newSession(t *testing.T, role string) (perun.SessionAPI, perun.Peer) {
	prng := rand.New(rand.NewSource(1729))

	_, aliceUser := sessiontest.NewTestUser(t, prng, uint(0))
	aliceUser.Alias = aliceAlias
	aliceUser.CommType = "tcp"
	aliceUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", 4341)
	aliceUser.OffChainAddrString = aliceUser.OffChainAddr.String()

	_, bobUser := sessiontest.NewTestUser(t, prng, uint(0))
	bobUser.Alias = bobAlias
	bobUser.CommType = "tcp"
	bobUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", 4342)
	bobUser.OffChainAddrString = bobUser.OffChainAddr.String()

	switch role {
	case aliceAlias:
		alice := newTestSession(t, aliceUser)
		return alice, bobUser.Peer
	case bobAlias:
		bob := newTestSession(t, bobUser)
		return bob, aliceUser.Peer
	}
	return nil, perun.Peer{}
}

func newTestSession(t *testing.T, testUser perun.User) perun.SessionAPI {
	adjudicator, asset := ethereumtest.SetupContracts(t, testUser.OnChain, ethereumtest.TestChainURL, ethereumtest.OnChainTxTimeout)

	emptyContacts, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	require.NoError(t, emptyContacts.Close())
	t.Cleanup(func() {
		if err = os.Remove(emptyContacts.Name()); err != nil {
			t.Log("Error in test cleanup: removing file - " + emptyContacts.Name())
		}
	})

	userCfg := session.UserConfig{
		Alias:       testUser.Alias,
		OnChainAddr: testUser.OnChain.Addr.String(),
		OnChainWallet: session.WalletConfig{
			KeystorePath: testUser.OnChain.Keystore,
			Password:     "",
		},
		OffChainAddr: testUser.OffChain.Addr.String(),
		OffChainWallet: session.WalletConfig{
			KeystorePath: testUser.OffChain.Keystore,
			Password:     "",
		},
		CommType: "tcp",
		CommAddr: testUser.CommAddr,
	}

	cfg := session.Config{
		User:             userCfg,
		ChainURL:         ethereumtest.TestChainURL,
		Adjudicator:      adjudicator.String(),
		Asset:            asset.String(),
		ChainConnTimeout: 30 * time.Second,
		ResponseTimeout:  10 * time.Second,
		OnChainTxTimeout: 5 * time.Second,
		DatabaseDir:      newDatabaseDir(t),

		ContactsType: "yaml",
		ContactsURL:  emptyContacts.Name(),
	}

	sess, err := session.New(cfg)
	require.NoError(t, err)
	require.NotNil(t, sess)
	return sess
}

func newDatabaseDir(t *testing.T) (dir string) {
	databaseDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.RemoveAll(databaseDir); err != nil {
			t.Logf("Error in removing the file in test cleanup - %v", err)
		}
	})
	return databaseDir
}
