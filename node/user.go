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

// NewUnlockedUser initializes a user and unlocks all the accounts.
// (those corresponding to on-chain address, off-chain address and all participant addresses.
func NewUnlockedUser(wb dst.WalletBackend, cfg UserConfig) (dst.User, error) {
	if wb == nil {
		return dst.User{}, errors.New("wallet backend should not be nil")
	}
	var err error
	u := dst.User{}
	u.OnChain.Wallet, err = newWallet(wb, cfg.OnChainWallet, cfg.OnChainAddr)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "on-chain")
	}
	u.OffChain.Wallet, err = newWallet(wb, cfg.OffChainWallet, cfg.OffChainAddr)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "off-chain")
	}
	u.PartAddrs, err = parseUnlock(wb, u.OffChain.Wallet, cfg.PartAddrs...)
	if err != nil {
		return dst.User{}, errors.Wrap(err, "participant addresses")
	}
	u.OffChainAddr = u.OffChain.Addr
	u.CommAddr = cfg.CommAddr
	u.CommType = cfg.CommType

	return u, nil
}

// newWalletAcc initializes the wallet using the wallet backend and unlocks accounts corresponding
// to each of the given addresses.
func newWallet(wb dst.WalletBackend, cfg WalletConfig, addr string) (wallet.Wallet, error) {
	w, err := wb.NewWallet(cfg.KeystorePath, cfg.Password)
	if err != nil {
		return nil, errors.Wrap(err, "initializing wallet")
	}
	_, err = parseUnlock(wb, w, addr)
	if err != nil {
		return nil, errors.Wrap(err, "participant addresses")
	}
	return w, nil
}

// parseUnlock parses the given addresses string using the wallet backend and unlocks accounts
// corresponding to each of the given addresses.
func parseUnlock(wb dst.WalletBackend, w wallet.Wallet, addrs ...string) ([]wallet.Address, error) {
	var err error
	parsedAddrs := make([]wallet.Address, len(addrs))
	for i, addr := range addrs {
		parsedAddrs[i], err = wb.ParseAddr(addr)
		if err != nil {
			return nil, errors.Wrap(err, "addr - "+addr)
		}
		_, err = w.Unlock(parsedAddrs[i])
		if err != nil {
			return nil, errors.Wrap(err, "acc - "+addr)
		}
	}
	return parsedAddrs, nil
}
