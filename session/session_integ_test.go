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

// func Test_Integ_OpenCh(t *testing.T) {
// 	sess1 := newTestSession(t *testing.T)
// 	sess2 := newTestSession(t *testing.T)
// }

func newTestSession(t *testing.T) *session.Session {
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
	return sess
}
