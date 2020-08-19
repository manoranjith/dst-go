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

package clienttest

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
)

var TestChainURL = "ws://127.0.0.1:8545"

// setup checks if valid contracts are deployed in pre-computed addresses, if not it deployes them.
// Address generation mechanism in ethereum is used to pre-compute the contract address.
func NewChainSetup(t *testing.T, onChainCred perun.Credential, chainURL string) (adjudicator, asset wallet.Address) {
	require.Truef(t, isBlockchainRunning(chainURL), "cannot connect to ganache-cli node at "+chainURL)

	adjudicator = ethwallet.AsWalletAddr(crypto.CreateAddress(ethwallet.AsEthAddr(onChainCred.Addr), 0))
	asset = ethwallet.AsWalletAddr(crypto.CreateAddress(ethwallet.AsEthAddr(onChainCred.Addr), 1))

	chain, err := ethereum.NewChainBackend(chainURL, 10*time.Second, onChainCred)
	require.NoError(t, err)

	if err = chain.ValidateContracts(adjudicator, asset); err != nil {
		t.Log("\nFirst run of test for this ganache-cli instance. Deploying contracts.\n")
		return deployContracts(t, chain)
	}
	t.Log("\nRepeated run of test for this ganache-cli instance. Using deployed contracts.\n")
	return adjudicator, asset
}

func isBlockchainRunning(url string) bool {
	_, _, err := websocket.DefaultDialer.Dial(url, nil)
	return err == nil
}

func deployContracts(t *testing.T, chain perun.ChainBackend) (adjudicator, asset wallet.Address) {
	adjudicator, err := chain.DeployAdjudicator()
	require.NoError(t, err)
	asset, err = chain.DeployAsset(adjudicator)
	require.NoError(t, err)
	return adjudicator, asset
}

func NewDatabaseDir(t *testing.T) (dir string) {
	databaseDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := os.RemoveAll(databaseDir); err != nil {
			t.Logf("Error in removing the file in test cleanup - %v", err)
		}
	})
	return databaseDir
}
