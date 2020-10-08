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

package ethereumtest

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	pethwallet "perun.network/go-perun/backend/ethereum/wallet"
	pwallet "perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
)

// Command to start the ganache-cli node:
//
// ganache-cli --account="0x1fedd636dbc7e8d41a0622a2040b86fea8842cef9d4aa4c582aad00465b7acff,10000000000000000000" \
//  --account="0xb0309c60b4622d3071fad3e16c2ce4d0b1e7758316c187754f4dd0cfb44ceb33,10000000000000000000"
//
// Ethereum address corresponding the above accounts: 0x8450c0055cB180C7C37A25866132A740b812937B and
// 0xc4bA4815c82727554e4c12A07a139b74c6742322.
//
// The account in the command corresponds to the on-chain account of first two users when seeding the rand source
// with 1729 and passing numParts as 0. If numParts is not zero, then the on-chain account is funded only for
// the first user. Hence DO NOT CHANGE THE RAND SEED for integration tests in this package.
//
// The contracts will be deployed only during the first run of tests and will be resused in subsequent runs. This
// saves ~0.3s of setup time in each run. Hence when running tests on development machine, START THE NODE ONLY ONCE.
const (
	OnChainTxTimeout = 10 * time.Second
	ChainURL         = "ws://127.0.0.1:8545"
	chainConnTimeout = 10 * time.Second
)

var adjudicatorAddr, assetAddr pwallet.Address

// SetupContractsT is the test friendly version of SetupContracts.
// It uses the passed testing.T to handle the errors and registers the cleanup functions on it.
func SetupContractsT(t *testing.T, onChainCred perun.Credential, chainURL string, onChainTxTimeout time.Duration) (
	adjudicator, asset pwallet.Address) {
	var err error
	adjudicator, asset, err = SetupContracts(onChainCred, chainURL, onChainTxTimeout)
	require.NoError(t, err)
	return adjudicator, asset
}

// SetupContracts checks if valid contracts are deployed in pre-computed addresses, if not it deployes them.
// Address generation mechanism in ethereum is used to pre-compute the contract address.
func SetupContracts(onChainCred perun.Credential, chainURL string, onChainTxTimeout time.Duration) (
	adjudicator, asset pwallet.Address, _ error) {
	if !isBlockchainRunning(chainURL) {
		return nil, nil, errors.New("cannot connect to ganache-cli node at " + chainURL)
	}

	if adjudicatorAddr == nil && assetAddr == nil {
		adjudicator = pethwallet.AsWalletAddr(crypto.CreateAddress(pethwallet.AsEthAddr(onChainCred.Addr), 0))
		asset = pethwallet.AsWalletAddr(crypto.CreateAddress(pethwallet.AsEthAddr(onChainCred.Addr), 1))
		adjudicatorAddr = adjudicator
		assetAddr = asset
	} else {
		adjudicator = adjudicatorAddr
		asset = assetAddr
	}

	chain, err := ethereum.NewChainBackend(chainURL, chainConnTimeout, onChainTxTimeout, onChainCred)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "initializaing chain backend")
	}

	err = chain.ValidateContracts(adjudicator, asset)
	if err == nil {
		// Contracts already deployed for  this ganache-cli instance. Using deployed contracts.
		return adjudicator, asset, nil
	}

	// Contracts not yet deployed for  this ganache-cli instance. Using deployed contracts.
	adjudicator, asset, err = deployContracts(chain)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "initializaing chain backend")
	}
	return adjudicator, asset, nil
}

func isBlockchainRunning(url string) bool {
	_, _, err := websocket.DefaultDialer.Dial(url, nil)
	return err == nil
}

func deployContracts(chain perun.ChainBackend) (adjudicator, asset pwallet.Address, _ error) {
	var err error
	adjudicator, err = chain.DeployAdjudicator()
	if err != nil {
		return nil, nil, err
	}
	asset, err = chain.DeployAsset(adjudicator)
	return adjudicator, asset, err
}
