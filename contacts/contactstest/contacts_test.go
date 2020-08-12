package contactstest_test

import (
	"fmt"
	"testing"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/contacts/contactstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validContacts map[string]perun.Peer

func init() {
	validContacts = make(map[string]perun.Peer)
	validContacts[contactstest.Peer1.Alias] = contactstest.Peer1
	validContacts[contactstest.Peer2.Alias] = contactstest.Peer2
}

func Test_NewProvider(t *testing.T) {
	provider, err := contactstest.NewProvider(3, contactstest.WalletBackend)
	require.NoError(t, err)
	for i := 1; i <= 3; i++ {
		peer, isPresent := provider.ReadByAlias(fmt.Sprintf("%d", i))
		assert.True(t, isPresent)
		assert.NotZero(t, peer.CommAddr)
		assert.NotZero(t, peer.CommType)
		assert.NotZero(t, peer.OffChainAddrString)
		assert.NotNil(t, peer.OffChainAddr)
	}
}

func Test_Provider_GenericReadWriteDelete(t *testing.T) {
	contactstest.GenericReadWriteDelete(t, contactsCloner(validContacts))
}

func Test_Provider_UpdateStorage(t *testing.T) {
	// This method is a Noop. Returns nil always.
	provider := &contactstest.Provider{}
	assert.Nil(t, provider.UpdateStorage())
}

func contactsCloner(testData map[string]perun.Peer) contactstest.ContactsCloner {
	return func(t *testing.T, walletBackend perun.WalletBackend) perun.Contacts {
		clone, err := contactstest.NewProvider(0, contactstest.WalletBackend)
		require.NoError(t, err)
		for alias, peer := range testData {
			require.NoError(t, clone.Write(alias, peer))
		}
		return clone
	}
}
