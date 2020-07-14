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

package internal_test

import (
	"math/rand"
	"testing"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/ethereumtest"
	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal"
)

func Test_WalletBackend_Interface(t *testing.T) {
	assert.Implements(t, (*dst.WalletBackend)(nil), new(internal.WalletBackend))
}

func Test_WalletBackend_NewWallet(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	wb := ethereumtest.NewTestWalletBackend()
	setup := ethereumtest.NewWalletSetup(t, rng, 1)

	t.Run("happy", func(t *testing.T) {
		w, err := wb.NewWallet(setup.KeystorePath, "")
		assert.NoError(t, err)
		assert.NotNil(t, w)
	})
	t.Run("invalid_pwd", func(t *testing.T) {
		w, err := wb.NewWallet(setup.KeystorePath, "invalid-pwd")
		assert.Error(t, err)
		assert.Nil(t, w)
	})
	t.Run("invalid_keystore_path", func(t *testing.T) {
		w, err := wb.NewWallet("invalid-ks-path", "")
		assert.Error(t, err)
		assert.Nil(t, w)
	})
}

func Test_WalletBackend_NewAccount(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	wb := ethereumtest.NewTestWalletBackend()
	setup := ethereumtest.NewWalletSetup(t, rng, 1)

	t.Run("happy", func(t *testing.T) {
		w, err := wb.NewAccount(setup.Wallet, setup.Accs[0].Address())
		assert.NoError(t, err)
		assert.NotNil(t, w)
	})
	t.Run("account_not_present", func(t *testing.T) {
		randomAddr := ethereumtest.NewRandomAddress(rng.Int63())
		w, err := wb.NewAccount(setup.Wallet, randomAddr)
		assert.Error(t, err)
		assert.Nil(t, w)
	})
}

func Test_WalletBackend_ParseAddr(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	wb := ethereumtest.NewTestWalletBackend()

	t.Run("happy_non_zero_value", func(t *testing.T) {
		validAddr := ethereumtest.NewRandomAddress(rng.Int63())
		gotAddr, err := wb.ParseAddr(validAddr.String())
		assert.NoError(t, err)
		require.NotNil(t, gotAddr)
		assert.Equal(t, validAddr.Bytes(), gotAddr.Bytes())
	})
	t.Run("happy_zero_value", func(t *testing.T) {
		validAddr := ethwallet.Address{}
		gotAddr, err := wb.ParseAddr(validAddr.String())
		assert.NoError(t, err)
		require.NotNil(t, gotAddr)
		assert.Equal(t, validAddr.Bytes(), gotAddr.Bytes())
	})
	t.Run("invalid-addr", func(t *testing.T) {
		gotAddr, err := wb.ParseAddr("invalid-addr")
		assert.Error(t, err)
		require.Nil(t, gotAddr)
	})
}
