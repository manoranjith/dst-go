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

package contactsyaml_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/contacts/contactstest"
	"github.com/hyperledger-labs/perun-node/contacts/contactsyaml"
)

var (
	testdataDir = filepath.Join("..", "..", "testdata", "contacts")

	validYAMLFile               = filepath.Join(testdataDir, "test.yaml")
	zeroEntriesYAMLFile         = filepath.Join(testdataDir, "test_zero_entries.yaml")
	updatedYAMLFile             = filepath.Join(testdataDir, "test_added_entries.yaml")
	invalidOffChainAddrYAMLFile = filepath.Join(testdataDir, "invalid_addr.yaml")
	corruptedYAMLFile           = filepath.Join(testdataDir, "corrupted.yaml")
	nonExistentYAMLFile         = "./con.yml"
)

func Test_Provider_ContactsReader_Interface(t *testing.T) {
	assert.Implements(t, (*perun.ContactsReader)(nil), new(contactsyaml.Provider))
}

func Test_Provider_Contacts_Interface(t *testing.T) {
	assert.Implements(t, (*perun.Contacts)(nil), new(contactsyaml.Provider))
}

func Test_Provider_GenericReadWriteDelete(t *testing.T) {
	contactstest.GenericReadWriteDelete(t, contactsCloner(validYAMLFile))
}

func Test_New(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		gotContacts, err := contactsyaml.New(tempCopyOfFile(t, validYAMLFile), contactstest.WalletBackend)
		assert.NoError(t, err)

		gotPeer1, isPresent := gotContacts.ReadByAlias(contactstest.Peer1.Alias)
		assert.Equal(t, contactstest.Peer1, gotPeer1)
		assert.True(t, isPresent)

		gotPeer2, isPresent := gotContacts.ReadByAlias(contactstest.Peer2.Alias)
		assert.Equal(t, contactstest.Peer2, gotPeer2)
		assert.True(t, isPresent)

		_, isPresent = gotContacts.ReadByAlias(contactstest.MissingPeer.Alias)
		assert.False(t, isPresent)
	})

	t.Run("corrupted_yaml", func(t *testing.T) {
		_, err := contactsyaml.New(tempCopyOfFile(t, corruptedYAMLFile), contactstest.WalletBackend)
		assert.Error(t, err)
		t.Log(err)
	})

	t.Run("invalid_offchain_addr", func(t *testing.T) {
		_, err := contactsyaml.New(tempCopyOfFile(t, invalidOffChainAddrYAMLFile), contactstest.WalletBackend)
		assert.Error(t, err)
		t.Log(err)
	})

	t.Run("missing_file", func(t *testing.T) {
		_, err := contactsyaml.New(nonExistentYAMLFile, contactstest.WalletBackend)
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Provider_UpdateStorage(t *testing.T) {
	t.Run("happy_add_entries_to_empty_file", func(t *testing.T) {
		c, testYAMLFile := contactsClonerWithPath(zeroEntriesYAMLFile)(t, contactstest.WalletBackend)
		c, err := contactsyaml.New(testYAMLFile, contactstest.WalletBackend)
		assert.NoError(t, err)
		require.NoError(t, c.Write(contactstest.Peer1.Alias, contactstest.Peer1))
		require.NoError(t, c.Write(contactstest.Peer2.Alias, contactstest.Peer2))

		assert.NoError(t, c.UpdateStorage())
		assert.True(t, compareFileContent(t, testYAMLFile, validYAMLFile))
	})

	t.Run("happy_add_entries_to_non_empty_file", func(t *testing.T) {
		c, testYAMLFile := contactsClonerWithPath(validYAMLFile)(t, contactstest.WalletBackend)
		assert.NoError(t, c.Write(contactstest.MissingPeer.Alias, contactstest.MissingPeer))

		assert.NoError(t, c.UpdateStorage())
		assert.True(t, compareFileContent(t, testYAMLFile, updatedYAMLFile))
	})

	t.Run("file_permission_error", func(t *testing.T) {
		c, testYAMLFile := contactsClonerWithPath(validYAMLFile)(t, contactstest.WalletBackend)

		// Change file permission and test
		err := os.Chmod(testYAMLFile, 0o444)
		require.NoError(t, err)
		err = c.UpdateStorage()
		assert.Error(t, err)
		t.Log(err)
	})
}

func contactsCloner(testDataFile string) contactstest.ContactsCloner {
	return func(t *testing.T, walletBackend perun.WalletBackend) perun.Contacts {
		c, _ := cloneContactsWithPath(t, testDataFile, walletBackend)
		return c
	}
}

func contactsClonerWithPath(testDataFile string) func(*testing.T, perun.WalletBackend) (perun.Contacts, string) {
	return func(t *testing.T, walletBackend perun.WalletBackend) (perun.Contacts, string) {
		return cloneContactsWithPath(t, testDataFile, walletBackend)
	}
}

func cloneContactsWithPath(t *testing.T, testDataFile string, walletBackend perun.WalletBackend) (perun.Contacts, string) {
	tempFilePath := tempCopyOfFile(t, testDataFile)
	c, err := contactsyaml.New(tempFilePath, walletBackend)
	require.NoError(t, err)
	return c, tempFilePath
}

func tempCopyOfFile(t *testing.T, srcFilePath string) (tempFilePath string) {
	tempFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	sourceFile, err := os.Open(srcFilePath)
	require.NoError(t, err)

	_, err = io.Copy(tempFile, sourceFile)
	require.NoError(t, err)
	require.NoError(t, tempFile.Close())
	require.NoError(t, sourceFile.Close())

	t.Cleanup(func() {
		if err = os.Remove(tempFile.Name()); err != nil {
			t.Log("Error in test cleanup: removing file - " + tempFile.Name())
		}
	})
	return tempFile.Name()
}

func compareFileContent(t *testing.T, file1, file2 string) bool {
	f1, err := ioutil.ReadFile(file1)
	require.NoError(t, err)
	f2, err := ioutil.ReadFile(file2)
	require.NoError(t, err)

	return bytes.Equal(f1, f2)
}
