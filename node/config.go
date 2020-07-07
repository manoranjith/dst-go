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

// WalletConfig defines the parameters required to configure a wallet.
type WalletConfig struct {
	KeystorePath string
	Password     string
}

// UserConfig defines the parameters required to configure a user.
// Addresss corresponding the respective accounts are stored as strings
// and should be parsed to concrete types using the wallet backend.
type UserConfig struct {
	Alias string

	OnChainAddr   string
	OnChainWallet WalletConfig

	PartAddrs      []string
	OffChainAddr   string
	OffChainWallet WalletConfig

	CommAddr string
	CommType string
}
