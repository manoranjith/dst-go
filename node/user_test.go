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

package node_test

import (
	"math/rand"
	"testing"

	"perun.network/go-perun/wallet"

	"github.com/direct-state-transfer/dst-go/node"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
	ethereumtest "github.com/direct-state-transfer/dst-go/blockchain/ethereum/test"
)

func Test_New_Happy(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	cntParts := uint(4)
	setup, testUser := newTestUser(t, rng, cntParts)
	wb := setup.WalletBackend
	userCfg := node.UserConfig{
		Alias:       testUser.Alias,
		OnChainAddr: testUser.OnChain.Addr.String(),
		OnChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     "",
		},
		OffChainAddr: testUser.OffChain.Addr.String(),
		OffChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     "",
		},
	}

	userCfg.PartAddrs = make([]string, len(testUser.PartAddrs))
	for i, addr := range testUser.PartAddrs {
		userCfg.PartAddrs[i] = addr.String()
	}

	gotUser, err := node.NewUnlockedUser(wb, userCfg)
	require.NoError(t, err)
	require.NotZero(t, gotUser)
	require.Len(t, gotUser.PartAddrs, int(cntParts))
}

func Test_New_Unhappy_Parts(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	cntParts := uint(1)
	setup, testUser := newTestUser(t, rng, cntParts)
	wb := setup.WalletBackend
	userCfg := node.UserConfig{
		Alias:       testUser.Alias,
		OnChainAddr: testUser.OnChain.Addr.String(),
		OnChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     "",
		},
		OffChainAddr: testUser.OffChain.Addr.String(),
		OffChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     "",
		},
	}

	t.Run("invalid-parts-address", func(t *testing.T) {
		userCfg.PartAddrs = make([]string, cntParts)
		for i := range testUser.PartAddrs {
			userCfg.PartAddrs[i] = "invalid-addr"
		}

		gotUser, err := node.NewUnlockedUser(wb, userCfg)
		require.Error(t, err)
		require.Zero(t, gotUser)
	})
	t.Run("missing-parts-address", func(t *testing.T) {
		userCfg.PartAddrs = make([]string, cntParts)
		for i := range testUser.PartAddrs {
			userCfg.PartAddrs[i] = ethereumtest.NewRandomAddress(rng).String()
		}
		gotUser, err := node.NewUnlockedUser(wb, userCfg)
		require.Error(t, err)
		require.Zero(t, gotUser)
	})
}

func Test_New_Unhappy_Wallets(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	setup, testUser := newTestUser(t, rng, 0)

	type args struct {
		wb  dst.WalletBackend
		cfg node.UserConfig
	}
	tests := []struct {
		name string
		args args
		want dst.User
	}{
		{
			name: "invalid-onchain-address",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: "invalid-addr",
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
					OffChainAddr: testUser.OffChain.Addr.String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
				},
			},
		},
		{
			name: "invalid-offchain-address",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChain.Addr.String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
					OffChainAddr: "invalid-addr",
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
				},
			},
		},
		{
			name: "missing-onchain-account",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: ethereumtest.NewRandomAddress(rng).String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
					OffChainAddr: testUser.OffChain.Addr.String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
				},
			},
		},
		{
			name: "missing-offchain-account",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChain.Addr.String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
					OffChainAddr: ethereumtest.NewRandomAddress(rng).String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
				},
			},
		},
		{
			name: "invalid-onchain-password",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChain.Addr.String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "invalid-password",
					},
					OffChainAddr: testUser.OffChain.Addr.String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
				},
			},
		},
		{
			name: "valid-onchain-invalid-offchain-password",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChain.Addr.String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
					OffChainAddr: testUser.OffChain.Addr.String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "invalid-pwd",
					},
				},
			},
		},
		{
			name: "invalid-keystore-path",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChain.Addr.String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: "invalid-keystore-path",
						Password:     "",
					},
					OffChainAddr: testUser.OffChain.Addr.String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := node.NewUnlockedUser(tt.args.wb, tt.args.cfg)
			require.Error(t, err)
			assert.Zero(t, got)
		})
	}
}

func newTestUser(t *testing.T, rng *rand.Rand, cntParts uint) (*ethereumtest.WalletSetup, dst.User) {
	ws := ethereumtest.NewWalletSetup(t, rng, 2+cntParts)
	u := dst.User{}
	u.OnChain.Addr = ws.Accs[0].Address()
	u.OnChain.Wallet = ws.Wallet
	u.OffChain.Addr = ws.Accs[1].Address()
	u.OffChain.Wallet = ws.Wallet
	u.Alias = "test-user"
	u.OffChain.Addr = ws.Accs[1].Address()
	u.PartAddrs = make([]wallet.Address, cntParts)
	for i := range ws.Accs[2:] {
		u.PartAddrs[i] = ws.Accs[i].Address()
	}

	return ws, u
}
