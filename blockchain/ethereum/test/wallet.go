package ethereum

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/direct-state-transfer/dst-go"
	internal "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/ethereum"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/wallet"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

// Weak encryption parameters used for creating test wallets that can be decrypted and unlocked faster.
const (
	weakScryptN = 2
	weakScryptP = 1
)

// UserSetup is a test setup that initailizes a user with random addresses and keys, generated using a WalletSetup.
// WalletSetup uses weak encryption parameters is used for storage encryption of keys for faster unlocking.
type UserSetup struct {
	WalletBackend dst.WalletBackend
	Keystore      *keystore.KeyStore
	KeystorePath  string
	User          dst.User
}

// NewTestWalletBackend initializes an ethereum specific wallet backend with weak encryption parameters.
func NewTestWalletBackend() dst.WalletBackend {
	return &internal.WalletBackend{EncParams: internal.ScryptParams{N: weakScryptN, P: weakScryptP}}
}

// NewUserSetup initializes and returns an initialized user with test accounts.
func NewUserSetup(t *testing.T, rng *rand.Rand) *UserSetup {
	ws := NewWalletSetup(t, rng, 3)
	user := dst.User{
		Peer: dst.Peer{
			Alias:      "test-user",
			OffchainID: ws.Accounts[0].Address(),
		},
		OnChainAcc:     ws.Accounts[1],
		OnChainWallet:  ws.Wallet,
		OffchainAcc:    ws.Accounts[2],
		OffChainWallet: ws.Wallet,
	}

	//No cleanup required.
	return &UserSetup{
		WalletBackend: ws.WalletBackend,
		Keystore:      ws.Keystore,
		KeystorePath:  ws.KeystorePath,
		User:          user,
	}
}

// WalletSetup can generate any number of keys for testing. To enable faster unlocking of keys, it uses
// weak encryption parameters for storage encryption of keys .
type WalletSetup struct {
	WalletBackend dst.WalletBackend
	KeystorePath  string
	Keystore      *keystore.KeyStore
	Wallet        wallet.Wallet
	Accounts      []wallet.Account
}

// NewWalletSetup initializes a wallet with n accounts. Empty password string and weak encrytion parameters are used.
func NewWalletSetup(t *testing.T, rng *rand.Rand, n int) *WalletSetup {
	wb := NewTestWalletBackend()

	ksPath, err := ioutil.TempDir("", "dst-go-test-keystore-*")
	if err != nil {
		t.Fatalf("Could not create temporary directory for keystore: %v", err)
	}
	ks := keystore.NewKeyStore(ksPath, weakScryptN, weakScryptP)
	w, err := ethwallet.NewWallet(ks, "")
	if err != nil {
		t.Fatalf("Could not create wallet: %v", err)
	}
	accs := make([]wallet.Account, n)
	for idx := 0; idx < n; idx++ {
		accs[idx] = w.NewRandomAccount(rng)
	}

	t.Cleanup(func() { os.RemoveAll(ksPath) })
	return &WalletSetup{
		WalletBackend: wb,
		KeystorePath:  ksPath,
		Keystore:      ks,
		Wallet:        w,
		Accounts:      accs,
	}
}

// NewRandomAddress generates a random wallet address. It generates the address only as a byte array.
// Hence it does not generate any public or private keys corresponding to the address.
// If you need an address with keys, use Wallet.NewAccount method.
func NewRandomAddress(rnd *rand.Rand) wallet.Address {
	var a common.Address
	rnd.Read(a[:])
	return ethwallet.AsWalletAddr(a)
}
