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
	"context"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	pethchannel "perun.network/go-perun/backend/ethereum/channel"
	pethchanneltest "perun.network/go-perun/backend/ethereum/channel/test"
	pethwallet "perun.network/go-perun/backend/ethereum/wallet"
	pkeystore "perun.network/go-perun/backend/ethereum/wallet/keystore"
	pwallet "perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum/internal"
)

// Chain related parameters for connecting to ganache-cli node in integration test environment.
const (
	RandSeedForTestAccs = 1729 // Seed required for generating accounts used in integration tests.
	OnChainTxTimeout    = 1 * time.Minute
	ChainURL            = "ws://127.0.0.1:8545"
	ChainConnTimeout    = 10 * time.Second
)

// ChainBackendSetup is a test setup that uses a simulated blockchain backend (for details on this backend,
// see go-ethereum) with required contracts deployed on it and a UserSetup.
type ChainBackendSetup struct {
	*WalletSetup
	ChainBackend       perun.ChainBackend
	AdjAddr, AssetAddr pwallet.Address
}

// NewChainBackendSetup returns a simulated contract backend with assetHolder and adjudicator contracts deployed.
// It also generates the given number of accounts and funds them each with 10 ether.
// and returns a test ChainBackend using the given randomness.
func NewChainBackendSetup(t *testing.T, rng *rand.Rand, numAccs uint) *ChainBackendSetup {
	walletSetup := NewWalletSetupT(t, rng, numAccs)

	cbEth := newSimContractBackend(t, walletSetup.Accs, walletSetup.Keystore)
	cb := &internal.ChainBackend{Cb: &cbEth, TxTimeout: OnChainTxTimeout}

	onChainAddr := walletSetup.Accs[0].Address()
	adjudicator, err := cb.DeployAdjudicator(onChainAddr)
	require.NoError(t, err)
	asset, err := cb.DeployAsset(adjudicator, onChainAddr)
	require.NoError(t, err)

	// No cleanup required.
	return &ChainBackendSetup{
		WalletSetup:  walletSetup,
		ChainBackend: cb,
		AdjAddr:      adjudicator,
		AssetAddr:    asset,
	}
}

// newSimContractBackend sets up a simulated contract backend with the first entry (index 0) in accs
// as the user account. All accounts are funded with 10 ethers.
func newSimContractBackend(t *testing.T, accs []pwallet.Account, ks *keystore.KeyStore) pethchannel.ContractBackend {
	simBackend := pethchanneltest.NewSimulatedBackend()
	ctx, cancel := context.WithTimeout(context.Background(), OnChainTxTimeout)
	defer cancel()
	for _, acc := range accs {
		simBackend.FundAddress(ctx, pethwallet.AsEthAddr(acc.Address()))
	}

	ksWallet, err := pkeystore.NewWallet(ks, "") // Password for test accounts is always empty string.
	require.NoError(t, err)

	tr := pkeystore.NewTransactor(*ksWallet, types.NewEIP155Signer(big.NewInt(1337)))
	return pethchannel.NewContractBackend(simBackend, tr)
}
