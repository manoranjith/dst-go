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
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/direct-state-transfer/dst-go"
	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/implementation"
)

// Standard encryption parameters used for creating wallets. Using these parameters will
// cause the decryption to use 256MB of RAM and takes approx 1s on a modern processor.
const (
	standardScryptN = keystore.StandardScryptN
	standardScryptP = keystore.StandardScryptP
)

// NewWalletBackend initializes an ethereum specific wallet backend.
//
// It returns an interface to enable this function to be loaded as a symbol without the knowledge of any types
// defined in this package.
func NewWalletBackend() dst.WalletBackend {
	return &implementation.WalletBackend{EncParams: implementation.ScryptParams{
		N: standardScryptN,
		P: standardScryptP,
	}}
}
