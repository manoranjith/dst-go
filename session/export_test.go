package session

import (
	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/log"
)

func SetWalletBackend(wb perun.WalletBackend) {
	walletBackend = wb
}

// NewEmptySession returns a session struct with an initialized logger.
func NewEmptySession() session {
	return session{
		Logger: log.NewLoggerWithField("test", ""),
	}
}
