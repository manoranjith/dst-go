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
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	"perun.network/go-perun/wallet"

	"github.com/direct-state-transfer/dst-go"
	implementation "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/implementation"
)

// NewOnChainTxBackend initializes a connection to blockchain node and
// sets up a wallet for funding onchain transactions.
//
// It returns an interface to enable this function to be loaded as a symbol using types defined in dst-go alone.
func NewOnChainTxBackend(url, userKeystorePath string, userAddr wallet.Address) (dst.OnChainTxBackend, error) {
	ethereumBackend, err := ethclient.Dial(url)
	if err != nil {
		return nil, errors.Wrap(err, "connecting to ethereum node")
	}
	ks := keystore.NewKeyStore(userKeystorePath, standardScryptN, standardScryptP)
	acc := &accounts.Account{Address: common.BytesToAddress(userAddr.Bytes())}
	cb := ethchannel.NewContractBackend(ethereumBackend, ks, acc)
	return &implementation.OnChainTxBackend{Cb: &cb}, nil
}
