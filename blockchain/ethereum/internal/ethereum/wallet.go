package ethereum

import (
	"errors"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/wallet"
)

// WalletBackend provides ethereum specific wallet backend functionality.
type WalletBackend struct {
	EncParams ScryptParams
}

// ScryptParams defines the parameters for scrypt algorithm. It determines the security level of algorithm
// used for encrypting the for storage on disk.
//
// Weak values should be used only for testing purposes (enables faster unlockcing). Use standard values otherwise.
type ScryptParams struct {
	N, P int
}

// NewWallet initializes an ethereum keystore at the given path and checks if all the keys in the keystore can
// be unlocked with the given password.
func (wb *WalletBackend) NewWallet(keystorePath, password string) (wallet.Wallet, error) {
	if _, err := os.Stat(keystorePath); os.IsNotExist(err) {
		return nil, errors.New("dir does not exists - " + keystorePath)
	}
	ks := keystore.NewKeyStore(keystorePath, wb.EncParams.N, wb.EncParams.P)
	return ethwallet.NewWallet(ks, password)
}

// NewAccount retreives the account correspoding to the given address, unlocks and returns it.
func (wb *WalletBackend) NewAccount(w wallet.Wallet, addr wallet.Address) (wallet.Account, error) {
	return w.Unlock(addr)
}
