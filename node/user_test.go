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
	"strconv"
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
	cntPartAccs := uint(4)
	setup, testUser := newTestUser(t, rng, cntPartAccs)
	wb := setup.WalletBackend
	userCfg := node.UserConfig{
		Alias:       testUser.Alias,
		OnChainAddr: testUser.OnChainAcc.Address().String(),
		OnChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     ""},
		OffChainAddr: testUser.OffchainAcc.Address().String(),
		OffChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     ""}}

	userCfg.PartAddrs = make(map[string]string)
	for k, v := range testUser.PartAccs {
		userCfg.PartAddrs[k] = v.Address().String()
	}

	gotUser, err := node.NewUser(wb, userCfg)
	require.NoError(t, err)
	require.NotZero(t, gotUser)
	require.Len(t, gotUser.PartAccs, int(cntPartAccs))

}
func Test_New_Unhappy_Parts(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	cntPartAccs := uint(1)
	setup, testUser := newTestUser(t, rng, cntPartAccs)
	wb := setup.WalletBackend
	userCfg := node.UserConfig{
		Alias:       testUser.Alias,
		OnChainAddr: testUser.OnChainAcc.Address().String(),
		OnChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     ""},
		OffChainAddr: testUser.OffchainAcc.Address().String(),
		OffChainWallet: node.WalletConfig{
			KeystorePath: setup.KeystorePath,
			Password:     ""}}

	t.Run("invalid-parts-address", func(t *testing.T) {
		userCfg.PartAddrs = make(map[string]string)
		for k := range testUser.PartAccs {
			userCfg.PartAddrs[k] = "invalid-addr"
		}

		gotUser, err := node.NewUser(wb, userCfg)
		require.Error(t, err)
		require.Zero(t, gotUser)
	})
	t.Run("missing-parts-address", func(t *testing.T) {
		userCfg.PartAddrs = make(map[string]string)
		for k := range testUser.PartAccs {
			userCfg.PartAddrs[k] = ethereumtest.NewRandomAddress(rng).String()
		}

		gotUser, err := node.NewUser(wb, userCfg)
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
						Password:     ""},
					OffChainAddr: testUser.OffchainAcc.Address().String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
		},
		{
			name: "invalid-offchain-address",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChainAcc.Address().String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""},
					OffChainAddr: "invalid-addr",
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
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
						Password:     ""},
					OffChainAddr: testUser.OffchainAcc.Address().String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
		},
		{
			name: "missing-offchain-account",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChainAcc.Address().String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""},
					OffChainAddr: ethereumtest.NewRandomAddress(rng).String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
		},
		{
			name: "invalid-onchain-password",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChainAcc.Address().String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "invalid-password"},
					OffChainAddr: testUser.OffchainAcc.Address().String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
		},
		{
			name: "valid-onchain-invalid-offchain-password",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChainAcc.Address().String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""},
					OffChainAddr: testUser.OffchainAcc.Address().String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     "invalid-pwd"}}},
		},
		{
			name: "invalid-keystore-path",
			args: args{
				wb: setup.WalletBackend,
				cfg: node.UserConfig{
					Alias:       testUser.Alias,
					OnChainAddr: testUser.OnChainAcc.Address().String(),
					OnChainWallet: node.WalletConfig{
						KeystorePath: "invalid-keystore-path",
						Password:     ""},
					OffChainAddr: testUser.OffchainAcc.Address().String(),
					OffChainWallet: node.WalletConfig{
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := node.NewUser(tt.args.wb, tt.args.cfg)
			require.Error(t, err)
			assert.Zero(t, got)
		})
	}
}

func newTestUser(t *testing.T, rng *rand.Rand, cntPartAccs uint) (*ethereumtest.WalletSetup, dst.User) {
	ws := ethereumtest.NewWalletSetup(t, rng, 2+cntPartAccs)
	user := dst.User{
		OnChainAcc:     ws.Accs[0],
		OnChainWallet:  ws.Wallet,
		OffchainAcc:    ws.Accs[1],
		OffChainWallet: ws.Wallet,
		Peer: dst.Peer{
			Alias:      "test-user",
			OffchainID: ws.Accs[1].Address(),
		},
	}
	user.PartAccs = make(map[string]wallet.Account)
	for i := range ws.Accs[2:] {
		user.PartAccs[strconv.Itoa(i-1)] = ws.Accs[i]
	}

	return ws, user
}
