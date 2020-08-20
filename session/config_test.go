// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/hyperledger-labs/perun-node
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

package session_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/session"
)

var (
	testConfigFile = "../testdata/session/valid.yaml"

	// test cofiguration as in the testdata file at
	// ${PROJECT_ROOT}/testdata/session/valid.yaml
	testCfg = session.Config{
		User: session.UserConfig{
			Alias:       perun.OwnAlias,
			OnChainAddr: "0x9282681723920798983380581376586951466585",
			OnChainWallet: session.WalletConfig{
				KeystorePath: "./test-keystore-on-chain",
				Password:     "test-password-on-chain",
			},
			OffChainAddr: "0x3369783337071807248093730889602727505701",
			OffChainWallet: session.WalletConfig{
				KeystorePath: "./test-keystore-off-chain",
				Password:     "test-password-off-chain",
			},
			CommAddr: "127.0.0.1:5751",
			CommType: "tcp",
		},
		ContactsType: "yaml",
		ContactsURL:  "./test-contacts.yaml",
		ChainURL:     "ws://127.0.0.1:8545",
		Asset:        "0x2681807986951466585898338058137657239292",
		Adjudicator:  "0x9833928268137658696658581723920798514805",
		DatabaseDir:  "./test-db",
	}
)

func Test_ParseConfig(t *testing.T) {
	cfg, err := session.ParseConfig(testConfigFile)
	require.NoError(t, err)
	assert.Equal(t, cfg, testCfg)
}
