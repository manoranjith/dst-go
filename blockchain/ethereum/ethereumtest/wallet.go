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
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/direct-state-transfer/dst-go"
	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/wallet"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

// NewTestWalletBackend initializes an ethereum specific wallet backend with weak encryption parameters.
func NewTestWalletBackend() dst.WalletBackend {
	return &internal.WalletBackend{EncParams: internal.ScryptParams{
		N: internal.WeakScryptN,
		P: internal.WeakScryptP,
	}}
}

// WalletSetup can generate any number of keys for testing. To enable faster unlocking of keys, it uses
// weak encryption parameters for storage encryption of keys .
type WalletSetup struct {
	WalletBackend dst.WalletBackend
	KeystorePath  string
	Keystore      *keystore.KeyStore
	Wallet        wallet.Wallet
	Accs          []wallet.Account
}

// NewWalletSetup initializes a wallet with n accounts. Empty password string and weak encrytion parameters are used.
func NewWalletSetup(t *testing.T, rng *rand.Rand, n uint) *WalletSetup {
	wb := NewTestWalletBackend()

	ksPath, err := ioutil.TempDir("", "dst-go-test-keystore-*")
	if err != nil {
		t.Fatalf("Could not create temporary directory for keystore: %v", err)
	}
	ks := keystore.NewKeyStore(ksPath, internal.WeakScryptN, internal.WeakScryptP)
	w, err := ethwallet.NewWallet(ks, "")
	if err != nil {
		t.Fatalf("Could not create wallet: %v", err)
	}
	accs := make([]wallet.Account, n)
	for idx := uint(0); idx < n; idx++ {
		accs[idx] = w.NewRandomAccount(rng)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(ksPath)
		if err != nil {
			t.Log("error in cleanup - ", err)
		}
	})
	return &WalletSetup{
		WalletBackend: wb,
		KeystorePath:  ksPath,
		Keystore:      ks,
		Wallet:        w,
		Accs:          accs,
	}
}

// NewRandomAddress generates a random wallet address. It generates the address only as a byte array.
// Hence it does not generate any public or private keys corresponding to the address.
// If you need an address with keys, use Wallet.NewAccount method.
//
// Take randSeed as arg instead of rand.Rand because tests are run concurrently by `go test` command and
// function Read is safe for concurrent use, while method Read is not.
func NewRandomAddress(seed int64) wallet.Address {
	var a common.Address
	rand.Seed(seed)
	rand.Read(a[:])
	return ethwallet.AsWalletAddr(a)
}
