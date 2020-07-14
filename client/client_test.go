// +build integration

// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/direct-state-transfer/dst-go
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

package client_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/direct-state-transfer/dst-go/client"
	"github.com/direct-state-transfer/dst-go/comm/tcp"
	"github.com/direct-state-transfer/dst-go/node/nodetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/ethereumtest"
)

func Test_Integ_NewEthereumPaymentClient(t *testing.T) {
	// NOTE : This integration setup requires manaual setup of a ganache-cli node.
	// Run the below command in a terminal before running this test.
	//
	// ganache-cli \
	// --account="0x7d51a817ee07c3f28581c47a5072142193337fdca4d7911e58c5af2d03895d1a,100000000000000000000000" \
	// --account="0x6aeeb7f09e757baa9d3935a042c3d0d46a2eda19e9b676283dce4eaf32e29dc9,100000000000000000000000"
	//
	// Deploy asset holder and adjudicator contracts.A quick hacky solution is to clone the project at
	// github.com/perun-network/perun-eth-demo, build it and run the below cmd:
	// ./perun-eth-demo demo --config bob.yaml.
	//
	// tested with the following version - Ganache CLI v6.9.1 (ganache-core: 2.10.2)

	prng := rand.New(rand.NewSource(1729))
	_, user := nodetest.NewTestUser(t, prng, 0)
	cfg := client.Config{
		Chain: client.ChainConfig{
			Adjudicator: "0xDc4A7e107aD6dBDA1870df34d70B51796BBd1335",
			Asset:       "0xb051EAD0C6CC2f568166F8fEC4f07511B88678bA",
			URL:         "ws://127.0.0.1:8545",
			ConnTimeout: 10 * time.Second,
		},
		PeerReconnTimeout: 0,
	}
	// TODO: Test if handle and lister are running as expected.

	t.Run("happy", func(t *testing.T) {
		cfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		_, err := client.NewEthereumPaymentClient(cfg, user, tcp.NewTCPAdapter(5*time.Second))
		assert.NoError(t, err)
	})

	t.Run("err_comm_nil", func(t *testing.T) {
		cfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		_, err := client.NewEthereumPaymentClient(cfg, user, nil)
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_invalid_chain_url", func(t *testing.T) {
		invalidCfg := cfg
		invalidCfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		invalidCfg.Chain.URL = "invalid-url"

		_, err := client.NewEthereumPaymentClient(invalidCfg, user, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_malformed_asset_addr", func(t *testing.T) {
		invalidCfg := cfg
		invalidCfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		invalidCfg.Chain.Asset = "invalid-addr"

		_, err := client.NewEthereumPaymentClient(invalidCfg, user, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_malformed_adj_addr", func(t *testing.T) {
		invalidCfg := cfg
		invalidCfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		invalidCfg.Chain.Adjudicator = "invalid-addr"

		_, err := client.NewEthereumPaymentClient(invalidCfg, user, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_invalid_adjudicator", func(t *testing.T) {
		invalidCfg := cfg
		invalidCfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		randomAddr := ethereumtest.NewRandomAddress(prng)
		invalidCfg.Chain.Adjudicator = randomAddr.String()

		_, err := client.NewEthereumPaymentClient(invalidCfg, user, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_invalid_asset", func(t *testing.T) {
		invalidCfg := cfg
		invalidCfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		randomAddr := ethereumtest.NewRandomAddress(prng)
		invalidCfg.Chain.Asset = randomAddr.String()

		_, err := client.NewEthereumPaymentClient(invalidCfg, user, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_invalid_on_chain_password", func(t *testing.T) {
		cfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		invalidUser := user
		ws := ethereumtest.NewWalletSetup(t, prng, 0)
		invalidUser.OnChain.Wallet = ws.Wallet
		invalidUser.OnChain.Keystore = ws.KeystorePath
		invalidUser.OnChain.Password = "invalid-password"
		_, err := client.NewEthereumPaymentClient(cfg, invalidUser, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_invalid_off_chain_password", func(t *testing.T) {
		cfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		invalidUser := user
		ws := ethereumtest.NewWalletSetup(t, prng, 0)
		invalidUser.OffChain.Wallet = ws.Wallet
		invalidUser.OffChain.Keystore = ws.KeystorePath
		invalidUser.OffChain.Password = "invalid-password"
		_, err := client.NewEthereumPaymentClient(cfg, invalidUser, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	t.Run("err_invalid_comm_addr", func(t *testing.T) {
		cfg.DatabaseDir = newDatabaseDir(t) // start with empty persistence dir each time.
		invalidUser := user
		invalidUser.CommAddr = "invalid-addr"
		_, err := client.NewEthereumPaymentClient(cfg, invalidUser, tcp.NewTCPAdapter(5*time.Second))
		assert.Error(t, err)
	})

	t.Run("err_persistence_path_is_file", func(t *testing.T) {
		emptyFile, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		err = emptyFile.Close()
		require.NoError(t, err)
		t.Cleanup(func() {
			// nolint:govet	// not shadowing, err is inside anonymous function.
			if err := os.Remove(emptyFile.Name()); err != nil {
				t.Logf("error in test cleanup, deleting file %s", emptyFile.Name())
			}
		})

		cfgInvalidPeristence := cfg
		cfgInvalidPeristence.DatabaseDir = emptyFile.Name()
		_, err = client.NewEthereumPaymentClient(cfgInvalidPeristence, user, tcp.NewTCPAdapter(5*time.Second))
		t.Log(err)
		assert.Error(t, err)
	})

	// TODO: (mano) Faulty Persistence data and Reconnection errors.
}

func newDatabaseDir(t *testing.T) (dir string) {
	databaseDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.Remove(databaseDir); err != nil {
			t.Logf("error in test cleanup, deleting dir %s", databaseDir)
		}
	})
	return databaseDir
}
