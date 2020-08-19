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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"perun.network/go-perun/apps/payment"

	"github.com/hyperledger-labs/perun-node"
	paymentAppLib "github.com/hyperledger-labs/perun-node/apps/payment"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum/ethereumtest"
	"github.com/hyperledger-labs/perun-node/client/clienttest"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

var (
	testdataDir = filepath.Join("..", "testdata", "contacts")

	testContactsYAML = filepath.Join(testdataDir, "test.yaml")
)

func init() {
	session.SetWalletBackend(ethereumtest.NewTestWalletBackend())
}

func Test_Integ_New(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	_, testUser := sessiontest.NewTestUser(t, prng, 0)
	adjudicator, asset := clienttest.NewChainSetup(t, testUser.OnChain, clienttest.TestChainURL)

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
		ChainURL:         clienttest.TestChainURL,
		Adjudicator:      adjudicator.String(),
		Asset:            asset.String(),
		ChainConnTimeout: 30 * time.Second,
		DatabaseDir:      clienttest.NewDatabaseDir(t),

		ContactsType: "yaml",
		ContactsURL:  testContactsYAML,
	}

	sess, err := session.New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, sess)
}

func newPaymentAppDef(t *testing.T) {
	wb := ethereumtest.NewTestWalletBackend()
	emptyAddr, err := wb.ParseAddr("0x0")
	require.NoError(t, err)
	payment.SetAppDef(emptyAddr) // dummy app def.
}

var ()

func Test_Integ_Role_Bob(t *testing.T) {

	wg := sync.WaitGroup{}

	alice, gotBobContact := newSession(t, aliceAlias)
	bob, gotAliceContact := newSession(t, bobAlias)
	var err error
	t.Log("alice session id:", alice.ID)
	t.Log("bob session id:", bob.ID)

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
		aliceProposedBalInfo := session.BalInfo{
			Currency: "ETH",
			Bals:     aliceProposedBals}
		payChInfo, err = paymentAppLib.OpenPayCh(alice, bobAlias, aliceProposedBalInfo, challengeDurSecs)
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

	err = paymentAppLib.RespondPayChProposal(bob, notif.ProposalID, true)
	require.NoError(t, err)
	t.Log("Bob accepted payment channel proposal")

	err = paymentAppLib.UnsubPayChProposals(bob)
	require.NoError(t, err)
	t.Log("Bob unsubscribed to payment proposal notifications")

	wg.Wait()

	t.Log("")
	t.Log("=====Completed channel proposal & accept sequence=====")
	t.Log("")

	// // OpenCh: 2 proposes
	// go func() {
	// 	sess2Bals := make(map[string]string)
	// 	sess2Bals["self"] = "2"
	// 	sess2Bals["1"] = "1"
	// 	chInfo, err := sess2.OpenCh("1", session.BalInfo{
	// 		Currency: "ETH",
	// 		Bals:     sess2Bals}, paymentApp, 15)
	// 	fmt.Println("err", err)
	// 	require.NoError(t, err)
	// 	fmt.Printf("\nsess2 chInfo %+v\n", chInfo)
	// }()

	// // SubChProposals: 1 Subs
	// var notifFrom2 session.ChProposalNotif
	// chProposalNotifierReject := func(notif session.ChProposalNotif) {
	// 	fmt.Printf("\nNotification from 2: %+v\n", notif)
	// 	notifFrom2 = notif
	// }
	// sess1.SubChProposals(chProposalNotifierReject)
	// time.Sleep(1 * time.Second)
	// // Accept the notification
	// err = sess1.RespondChProposal(notifFrom2.ProposalID, false)
	// fmt.Println("err", err)
	// require.NoError(t, err)

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
		err = paymentAppLib.SendPayChUpdate(ch1, "bob", "0.5")
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
	err = paymentAppLib.RespondPayChUpdate(ch2, notif2.UpdateID, true)
	require.NoError(t, err)
	fmt.Println("Update was accepted")

	wg.Wait()

	// balInfo = paymentAppLib.GetBalance(ch1)
	// fmt.Printf("\n%+v", balInfo)

	// // 2 closes the channel
	// closingBal, err := paymentAppLib.ClosePayCh(ch2)
	// require.NoError(t, err)
	// fmt.Printf("\n%+v\n", closingBal)
	// fmt.Println("channel was closed")

	// // 1 subs from chClose
	// var closeNotifFrom2 paymentAppLib.PayChCloseNotif
	// PayChCloseNotifier := func(notif paymentAppLib.PayChCloseNotif) {
	// 	fmt.Printf("\n Close Notification in session 1: %+v\n", notif)
	// 	closeNotifFrom2 = notif
	// }
	// err = paymentAppLib.SubPayChCloses(alice, PayChCloseNotifier)
	// require.NoError(t, err)

	// time.Sleep(3 * time.Second)
	// fmt.Printf("\n%+v\n", closeNotifFrom2)
	// fmt.Println("channel notification was received")
}

func newTestSession(t *testing.T, testUser perun.User) *session.Session {
	adjudicator, asset := clienttest.NewChainSetup(t, testUser.OnChain, clienttest.TestChainURL)

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
		ChainURL:         clienttest.TestChainURL,
		Adjudicator:      adjudicator.String(),
		Asset:            asset.String(),
		ChainConnTimeout: 30 * time.Second,
		DatabaseDir:      clienttest.NewDatabaseDir(t),

		ContactsType: "yaml",
		ContactsURL:  emptyContacts.Name(),
	}

	sess, err := session.New(cfg)
	require.NoError(t, err)
	require.NotNil(t, sess)
	return sess
}
