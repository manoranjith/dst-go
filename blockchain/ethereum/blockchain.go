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

package ethereum

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"

	"github.com/direct-state-transfer/dst-go"
	implementation "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/implementation"
)

// NewOnChainTxBackend initializes a connection to blockchain node and sets up a wallet with given credentials
// for funding onchain transactions and channel balances.
//
func NewOnChainTxBackend(url string, timeout time.Duration, cred dst.Credential) (dst.OnChainTxBackend, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ethereumBackend, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to ethereum node")
	}

	ks := keystore.NewKeyStore(cred.Keystore, standardScryptN, standardScryptP)
	acc := accounts.Account{Address: ethwallet.AsEthAddr(cred.Addr)}
	err = ks.Unlock(acc, cred.Password)
	if err != nil {
		return nil, errors.Wrap(err, "unlocking on-chain keystore")
	}
	cb := ethchannel.NewContractBackend(ethereumBackend, ks, &acc)
	return &implementation.OnChainTxBackend{Cb: &cb}, nil
}
