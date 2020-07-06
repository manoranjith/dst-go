package ethereum_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/direct-state-transfer/dst-go"
	ethereum "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/ethereum"
	ethereumtest "github.com/direct-state-transfer/dst-go/blockchain/ethereum/test"
)

func Test_WalletBackend_Interface(t *testing.T) {
	assert.Implements(t, (*dst.WalletBackend)(nil), new(ethereum.WalletBackend))
}

func Test_WalletBackend_NewWallet(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	wb := ethereumtest.NewTestWalletBackend()
	setup := ethereumtest.NewWalletSetup(t, rng, 1)

	t.Run("valid", func(t *testing.T) {
		w, err := wb.NewWallet(setup.KeystorePath, "")
		assert.NoError(t, err)
		assert.NotNil(t, w)
	})
	t.Run("invalid-pwd", func(t *testing.T) {
		w, err := wb.NewWallet(setup.KeystorePath, "invalid-pwd")
		assert.Error(t, err)
		assert.Nil(t, w)
	})
	t.Run("invalid-keystore-path", func(t *testing.T) {
		w, err := wb.NewWallet("invalid-ks-path", "")
		assert.Error(t, err)
		assert.Nil(t, w)
	})
}

func Test_WalletBackend_NewAccount(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	wb := ethereumtest.NewTestWalletBackend()
	setup := ethereumtest.NewWalletSetup(t, rng, 1)

	t.Run("valid", func(t *testing.T) {
		w, err := wb.NewAccount(setup.Wallet, setup.Accounts[0].Address())
		assert.NoError(t, err)
		assert.NotNil(t, w)
	})
	t.Run("account-not-present", func(t *testing.T) {
		randomAddr := ethereumtest.NewRandomAddress(rng)
		w, err := wb.NewAccount(setup.Wallet, randomAddr)
		assert.Error(t, err)
		assert.Nil(t, w)
	})
}
