package session

import "github.com/hyperledger-labs/perun-node"

func SetWalletBackend(wb perun.WalletBackend) {
	walletBackend = wb
}
