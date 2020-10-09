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

package configgen

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/contacts/contactstest"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

var (
	aliceAlias, bobAlias = "alice", "bob"

	nodeCfg = perun.NodeConfig{
		LogFile:      "",
		LogLevel:     "debug",
		ChainURL:     "ws://127.0.0.1:8545",
		Adjudicator:  "0x9daEdAcb21dce86Af8604Ba1A1D7F9BFE55ddd63",
		Asset:        "0x5992089d61cE79B6CF90506F70DD42B8E42FB21d",
		CommTypes:    []string{"tcp"},
		ContactTypes: []string{"yaml"},
		Currencies:   []string{"ETH"},

		ChainConnTimeout: 30 * time.Second,
		OnChainTxTimeout: 10 * time.Second,
		ResponseTimeout:  10 * time.Second,
	}
)

// GenerateNodeConfig generates node configuration artifact (node.yaml) in the current directory.
func GenerateNodeConfig() error {
	if _, err := os.Stat("node.yaml"); !os.IsNotExist(err) {
		return errors.New("exists file - node")
	}
	nodeConfigFile, err := sessiontest.NewConfigFile(nodeCfg)
	if err != nil {
		return err
	}
	// Move session config file.
	filesToMove := map[string]string{nodeConfigFile: filepath.Join("node.yaml")}
	if err = moveFiles(filesToMove); err != nil {
		return err
	}
	return nil
}

// GenerateSessionConfig generates two sets of session configuration facts in directories alice and bob.
// Each directory would have: session.yaml, contacts.yaml and keystore (containing 2 key files - on-chain & off-chain).
// To use this configuration, start the node from same directory containing the session config artifacts directory and
// pass the path "alice/session.yaml" and "bob/session.yaml" for alice and bob respectively.
func GenerateSessionConfig() error {
	var err error
	if err = makeDirs(); err != nil {
		return err
	}

	// Create session config.
	prng := rand.New(rand.NewSource(1729)) // nolint: gosec		// math/rand is used to get deterministic random number.
	aliceCfg, err := sessiontest.NewConfig(prng)
	if err != nil {
		return err
	}
	aliceCfg.User.Alias = aliceAlias
	bobCfg, err := sessiontest.NewConfig(prng)
	if err != nil {
		return err
	}
	bobCfg.User.Alias = bobAlias

	// Create Contacts file.
	aliceContactsFile, err := contactstest.NewYAMLFile(peer(bobCfg.User))
	if err != nil {
		return err
	}
	bobContactsFile, err := contactstest.NewYAMLFile(peer(aliceCfg.User))
	if err != nil {
		return err
	}

	// Create session config file.
	aliceCfgFile, err := sessiontest.NewConfigFile(updatedConfig(aliceCfg))
	if err != nil {
		return err
	}
	bobCfgFile, err := sessiontest.NewConfigFile(updatedConfig(bobCfg))
	if err != nil {
		return err
	}

	// Move session config file.
	filesToMove := map[string]string{
		aliceCfgFile:                             filepath.Join(aliceAlias, "session.yaml"),
		aliceContactsFile:                        filepath.Join(aliceAlias, "contacts.yaml"),
		aliceCfg.DatabaseDir:                     filepath.Join(aliceAlias, "database"),
		aliceCfg.User.OnChainWallet.KeystorePath: filepath.Join(aliceAlias, "keystore"),

		bobCfgFile:                             filepath.Join(bobAlias, "session.yaml"),
		bobCfg.DatabaseDir:                     filepath.Join(bobAlias, "database"),
		bobContactsFile:                        filepath.Join(bobAlias, "contacts.yaml"),
		bobCfg.User.OnChainWallet.KeystorePath: filepath.Join(bobAlias, "keystore"),
	}
	if err = moveFiles(filesToMove); err != nil {
		return err
	}
	return nil
}

func makeDirs() error {
	var err error
	if _, err = os.Stat(aliceAlias); !os.IsNotExist(err) {
		return errors.New("exists dir - alice")
	}
	if _, err = os.Stat(bobAlias); !os.IsNotExist(err) {
		return errors.New("exists dir - bob")
	}

	if err = os.Mkdir(aliceAlias, 0750); err != nil {
		return errors.Wrap(err, aliceAlias)
	}
	if err = os.Mkdir(bobAlias, 0750); err != nil {
		return errors.Wrap(err, bobAlias)
	}
	return nil
}

func peer(userCfg session.UserConfig) perun.Peer {
	return perun.Peer{
		Alias:              userCfg.Alias,
		OffChainAddrString: userCfg.OffChainAddr,
		CommAddr:           userCfg.CommAddr,
		CommType:           userCfg.CommType,
	}
}

func updatedConfig(cfg session.Config) session.Config {
	copyCfg := cfg
	copyCfg.ContactsURL = filepath.Join(cfg.User.Alias, "contacts.yaml")
	copyCfg.DatabaseDir = filepath.Join(cfg.User.Alias, "database")
	copyCfg.User.OnChainWallet.KeystorePath = filepath.Join(cfg.User.Alias, "keystore")
	copyCfg.User.OffChainWallet.KeystorePath = filepath.Join(cfg.User.Alias, "keystore")
	return copyCfg
}

func moveFiles(srcDest map[string]string) error {
	errs := []string{}
	for src, dest := range srcDest {
		if err := os.Rename(src, dest); err != nil {
			errs = append(errs, fmt.Sprintf("%s to %s: %v", src, dest, err))
		}
	}
	if len(errs) != 0 {
		return fmt.Errorf("errors in moving files: %v", errs)
	}
	return nil
}
