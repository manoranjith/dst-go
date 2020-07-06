package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	"perun.network/go-perun/wallet"

	"github.com/direct-state-transfer/dst-go"
	internal "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/ethereum"
)

// NewOnChainTxBackend initializes a connection to blockchain node and
// sets up a wallet for funding onchain transactions.
func NewOnChainTxBackend(url, userKeystorePath string, userAddr wallet.Address) (dst.OnChainTxBackend, error) {
	ethereumBackend, err := ethclient.Dial(url)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to ethereum node")
	}
	ks := keystore.NewKeyStore(userKeystorePath, standardScryptN, standardScryptP)
	acc := &accounts.Account{Address: common.BytesToAddress(userAddr.Bytes())}
	cb := ethchannel.NewContractBackend(ethereumBackend, ks, acc)
	return &internal.OnChainTxBackend{Cb: &cb}, nil
}
