package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/direct-state-transfer/dst-go"
	internal "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/ethereum"
)

// Standard encryption parameters used for creating wallets. Using these parameters will
// cause the decryption to use 256MB of RAM and takes approx 1s on a modern processor.
const (
	standardScryptN = keystore.StandardScryptN
	standardScryptP = keystore.StandardScryptP
)

// NewWalletBackend initializes an ethereum specific wallet backend.
func NewWalletBackend() dst.WalletBackend {
	return &internal.WalletBackend{EncParams: internal.ScryptParams{
		N: standardScryptN,
		P: standardScryptP,
	}}
}
