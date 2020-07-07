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

package node

import (
	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"

	"github.com/direct-state-transfer/dst-go"
)

// NewUser initializes a user account for the given config using the given wallet backend.
func NewUser(wb dst.WalletBackend, cfg UserConfig) (u dst.User, err error) {

	u.OnChainWallet, u.OnChainAcc, err = newWalletAcc(cfg.OnChainWallet, cfg.OnChainAddr, wb)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "onchain")
	}
	u.OffChainWallet, u.OffchainAcc, err = newWalletAcc(cfg.OffChainWallet, cfg.OffChainAddr, wb)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "offchain")
	}
	u.PartAccs = make(map[string]wallet.Account)
	for id, addrStr := range cfg.PartAddrs {
		partAcc, err := accFromWallet(addrStr, u.OffChainWallet, wb)
		if err != nil {
			return dst.User{}, errors.Wrapf(err, "participant account for channel id %s", id)
		}
		u.PartAccs[id] = partAcc
	}

	u.Peer = dst.Peer{
		Alias:      cfg.Alias,
		OffchainID: u.OffchainAcc.Address(),
	}
	return u, nil
}

// newWalletAcc initializes a wallet and retrieves the account corresponding to the given addr from it.
func newWalletAcc(cfg WalletConfig, addr string, wb dst.WalletBackend) (wallet.Wallet, wallet.Account, error) {
	w, err := wb.NewWallet(cfg.KeystorePath, cfg.Password)
	if err != nil {
		return nil, nil, errors.Wrap(err, "initializing wallet")
	}
	acc, err := accFromWallet(addr, w, wb)
	if err != nil {
		return nil, nil, errors.Wrap(err, "initializing account")
	}
	return w, acc, nil
}

// accFromWallet retrieves the account corresponding to the given address from the wallet.
func accFromWallet(addr string, w wallet.Wallet, wb dst.WalletBackend) (wallet.Account, error) {
	parsedAddr, err := wb.ParseAddr(addr)
	if err != nil {
		return nil, errors.Wrap(err, "parsing address")
	}
	return wb.NewAccount(w, parsedAddr)
}
