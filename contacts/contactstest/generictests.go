package contactstest

import (
	"testing"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/stretchr/testify/assert"
)

var (
	Peer1 = perun.Peer{
		Alias:              "Alice",
		OffChainAddrString: "0x928268172392079898338058137658695146658578982175",
		CommType:           "tcpip",
		CommAddr:           "127.0.0.1:5751",
	}
	Peer2 = perun.Peer{
		Alias:              "Bob",
		OffChainAddrString: "0x33697833370718072480937308896027275057015318468",
		CommType:           "tcpip",
		CommAddr:           "127.0.0.1:5750",
	}
	MissingPeer = perun.Peer{
		Alias:              "Tom",
		OffChainAddrString: "0x71873088960230724809336978333707275057015318468",
		CommType:           "tcpip",
		CommAddr:           "127.0.0.1:5753",
	}

	WalletBackend = ethereum.NewWalletBackend()
)

func init() {
	Peer1.OffChainAddr, _ = WalletBackend.ParseAddr(Peer1.OffChainAddrString)             // nolint:errcheck
	Peer2.OffChainAddr, _ = WalletBackend.ParseAddr(Peer2.OffChainAddrString)             // nolint:errcheck
	MissingPeer.OffChainAddr, _ = WalletBackend.ParseAddr(MissingPeer.OffChainAddrString) // nolint:errcheck
}

type ContactsCloner func(t *testing.T, b perun.WalletBackend) perun.Contacts

func GenericReadWriteDelete(t *testing.T, newClone ContactsCloner) {
	t.Run("ReadByAlias", func(t *testing.T) {
		c := newClone(t, WalletBackend)
		t.Run("happy", func(t *testing.T) {
			gotPeer, isPresent := c.ReadByAlias(Peer1.Alias)
			assert.True(t, isPresent)
			assert.Equal(t, gotPeer, Peer1)
		})

		t.Run("missing_peer", func(t *testing.T) {
			_, isPresent := c.ReadByAlias(MissingPeer.Alias)
			assert.False(t, isPresent)
		})
	})

	t.Run("ReadByOffChainAddr", func(t *testing.T) {
		c := newClone(t, WalletBackend)
		t.Run("happy", func(t *testing.T) {
			gotPeer, isPresent := c.ReadByOffChainAddr(Peer1.OffChainAddrString)
			assert.True(t, isPresent)
			assert.Equal(t, gotPeer, Peer1)
		})

		t.Run("missing_peer", func(t *testing.T) {
			_, isPresent := c.ReadByOffChainAddr(MissingPeer.OffChainAddrString)
			assert.False(t, isPresent)
		})
	})

	t.Run("Write_Read", func(t *testing.T) {
		c := newClone(t, WalletBackend)
		t.Run("happy", func(t *testing.T) {
			assert.NoError(t, c.Write(MissingPeer.Alias, MissingPeer))
			gotPeer, isPresent := c.ReadByAlias(MissingPeer.Alias)
			assert.True(t, isPresent)
			assert.Equal(t, gotPeer, MissingPeer)
		})

		t.Run("peer_already_present", func(t *testing.T) {
			err := c.Write(Peer1.Alias, Peer1)
			assert.Error(t, err)
			t.Log(err)
		})

		t.Run("alias_used_by_diff_peer", func(t *testing.T) {
			err := c.Write(Peer1.Alias, Peer2)
			assert.Error(t, err)
			t.Log(err)
		})

		t.Run("invalid_offchain_addr", func(t *testing.T) {
			c := newClone(t, WalletBackend)

			missingPeerCopy := MissingPeer
			missingPeerCopy.OffChainAddrString = "invalid-addr"
			err := c.Write(missingPeerCopy.Alias, missingPeerCopy)
			assert.Error(t, err)
			t.Log(err)
		})
	})

	t.Run("Delete_Read", func(t *testing.T) {
		c := newClone(t, WalletBackend)
		t.Run("happy", func(t *testing.T) {
			assert.NoError(t, c.Delete(Peer1.Alias))
			_, isPresent := c.ReadByAlias(Peer1.Alias)
			assert.False(t, isPresent)
		})

		t.Run("missing_peer", func(t *testing.T) {
			err := c.Delete(MissingPeer.Alias)
			assert.Error(t, err)
			t.Log(err)
		})
	})
}
