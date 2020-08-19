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
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"perun.network/go-perun/apps/payment"

	"github.com/hyperledger-labs/perun-node"
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

func Test_Integ_OpenCh(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	sess1 := newTestSession(t, prng)
	sess2 := newTestSession(t, prng)
	fmt.Println("sess1", sess1.ID)
	fmt.Println("sess2", sess2.ID)
	fmt.Println("sess1 on", sess1.User.OnChain.Addr.String())
	fmt.Println("sess1 off", sess1.User.OffChain.Addr.String())
	fmt.Println("sess1 on", sess2.User.OnChain.Addr.String())
	fmt.Println("sess2 off", sess2.User.OffChain.Addr.String())

	own1, err := sess1.GetContact(perun.OwnAlias)
	require.NoError(t, err)
	fmt.Printf("\nsess1 own %+v\n", own1)
	own2, err := sess2.GetContact(perun.OwnAlias)
	require.NoError(t, err)
	fmt.Printf("\nsess2 own %+v\n", own2)

	// Add contact
	own1.Alias = "1"
	own2.Alias = "2"
	err = sess1.AddContact(own2)
	fmt.Println("err", err)
	require.NoError(t, err)
	err = sess2.AddContact(own1)
	fmt.Println("err", err)
	require.NoError(t, err)

	wb := ethereumtest.NewTestWalletBackend()
	emptyAddr, err := wb.ParseAddr("0x0")
	fmt.Println("err", err)
	require.NoError(t, err)
	payment.SetAppDef(emptyAddr) // dummy app def.
	paymentApp := session.App{
		Def:  payment.AppDef(),
		Data: &payment.NoData{},
	}

	// OpenCh: 1 proposes
	go func() {
		sess1Bals := make(map[string]string)
		sess1Bals["self"] = "1"
		sess1Bals["2"] = "2"
		chInfo, err := sess1.OpenCh("2", session.BalInfo{
			Currency: "ETH",
			Bals:     sess1Bals}, paymentApp, 15)
		fmt.Println("err", err)
		require.NoError(t, err)
		fmt.Printf("\nsess1 chInfo %+v\n", chInfo)
	}()

	// SubChProposals: 2 Subs
	var notifFrom1 session.ChProposalNotif
	chProposalNotifierAccept := func(notif session.ChProposalNotif) {
		fmt.Printf("\nNotification from 1: %+v\n", notif)
		notifFrom1 = notif
	}
	sess2.SubChProposals(chProposalNotifierAccept)
	time.Sleep(1 * time.Second)
	// Accept the notification
	err = sess2.RespondChProposal(notifFrom1.ProposalID, true)
	fmt.Println("err", err)
	require.NoError(t, err)

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

	// GetChannel ID from Session 1
	chInfos1 := sess1.GetChs()
	fmt.Printf("\nChannel Infos in s1%+v\n", chInfos1)

	chID1 := chInfos1[0].ChannelID

	// get channel instance
	ch1, err := sess1.GetCh(chID1)
	require.NoError(t, err)
	fmt.Println("channel id in s1", ch1.ID)

	// GetChannel ID from Session 1
	chInfos2 := sess2.GetChs()
	fmt.Printf("\nChannel Infos in s2 %+v\n", chInfos2)

	chID2 := chInfos2[0].ChannelID

	// get channel instance
	ch2, err := sess2.GetCh(chID2)
	require.NoError(t, err)
	fmt.Println("channel id in s2", ch2.ID) // err = paymentAppLib.SendPayChUpdate(ch, "2", "0.1")
	// require.NoError(t, err)
}

func newTestSession(t *testing.T, prng *rand.Rand) *session.Session {
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
	require.NotNil(t, sess)
	return sess
}
