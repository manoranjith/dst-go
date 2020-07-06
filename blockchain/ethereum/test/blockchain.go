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

package ethereumtest

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/implementation"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethchanneltest "perun.network/go-perun/backend/ethereum/channel/test"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/wallet"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
)

// DefaultTxTimeout is the default transaction timeout for simulated backend.
const DefaultTxTimeout = 5 * time.Second

// OnChainTxBackendSetup is a test setup that uses a simulated blockchain backend (for details on this backend,
// see go-ethereum) with required contracts deployed on it and a UserSetup.
type OnChainTxBackendSetup struct {
	*WalletSetup
	OnChainTxBackend   dst.OnChainTxBackend
	AdjAddr, AssetAddr wallet.Address
}

// NewOnChainTxBackendSetup returns a simulated contract backend  with assetHolder, adjudicator contracts deployed.
// It also generate the given number of accounts and funds them each with 10 ether.
// and returns a test OnChainTxBackend using the given randomness.
func NewOnChainTxBackendSetup(t *testing.T, rng *rand.Rand, cntFundedAccs uint) (_ *OnChainTxBackendSetup) {
	// userSetup := NewUserSetup(t, rng)
	walletSetup := NewWalletSetup(t, rng, cntFundedAccs)

	cbEth := newSimContractBackend(walletSetup.Accs, walletSetup.Keystore)
	cb := &implementation.OnChainTxBackend{Cb: &cbEth}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTxTimeout)
	defer cancel()
	adjudicator, err := ethchannel.DeployAdjudicator(ctx, cbEth)
	require.NoError(t, err)
	asset, err := ethchannel.DeployETHAssetholder(ctx, cbEth, adjudicator)
	require.NoError(t, err)

	// No cleanup required.
	return &OnChainTxBackendSetup{
		WalletSetup:      walletSetup,
		OnChainTxBackend: cb,
		AdjAddr:          ethwallet.AsWalletAddr(adjudicator),
		AssetAddr:        ethwallet.AsWalletAddr(asset),
	}
}

// newSimContractBackend sets up a simulated contract backend with the first entry (index 0) in accs
// as the user accounts. All accounts are funded with 10 ethers.
func newSimContractBackend(accs []wallet.Account, ks *keystore.KeyStore) ethchannel.ContractBackend {
	simBackend := ethchanneltest.NewSimulatedBackend()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTxTimeout)
	defer cancel()
	for i := range accs {
		simBackend.FundAddress(ctx, ethwallet.AsEthAddr(accs[i].Address()))
	}

	onchainAcc := &accs[0].(*ethwallet.Account).Account
	contractBackend := ethchannel.NewContractBackend(simBackend, ks, onchainAcc)
	return contractBackend
}
