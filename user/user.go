package user

import (
	"github.com/pkg/errors"

	"github.com/direct-state-transfer/dst-go"
)

// New initializes a user account for the given config using the given wallet backend.
func New(wb dst.WalletBackend, cfg Config) (u dst.User, err error) {

	u.OnChainWallet, err = wb.NewWallet(cfg.OnChainWallet.KeystorePath, cfg.OnChainWallet.Password)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "initializing onchain wallet")
	}
	u.OnChainAcc, err = wb.NewAccount(u.OnChainWallet, cfg.OnChainWallet.Addr)
	if err != nil {
		//Code shouldn't reach here. As wallet unlocks with given password, account should init without error.
		return dst.User{}, errors.Wrap(err, "initializing onchain account")
	}
	u.OffChainWallet, err = wb.NewWallet(cfg.OffChainWallet.KeystorePath, cfg.OffChainWallet.Password)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "initializing offchain wallet")
	}
	u.OffchainAcc, err = wb.NewAccount(u.OffChainWallet, cfg.OffChainWallet.Addr)
	if err != nil {
		//Code shouldn't reach here. As wallet unlocks with given password, account should init without error.
		return dst.User{}, errors.Wrap(err, "initializing offchain account")
	}

	//TODO: Initialize network adapter. This should be injected as a parameter to New()

	u.Peer = dst.Peer{
		Alias:      cfg.Alias,
		OffchainID: u.OffchainAcc.Address(),
	}
	return u, nil
}
