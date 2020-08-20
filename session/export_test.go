package session

import (
	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/log"
)

func SetWalletBackend(wb perun.WalletBackend) {
	walletBackend = wb
}

func NewEmptySession() session {
	return session{
		Logger: log.NewLoggerWithField("for", "test"),
	}
}
