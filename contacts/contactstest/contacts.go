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

package contactstest

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/phayes/freeport"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum/ethereumtest"
)

// Provider represents a cached list of contacts indexed by both alias and off-chain address.
// The methods defined over it are safe for concurrent access.
type Provider struct {
	mutex         sync.RWMutex
	walletBackend perun.WalletBackend

	nextRandomAlias uint

	peersByAlias map[string]perun.Peer // Stores a list of peers indexed by Alias.
	aliasByAddr  map[string]string     // Stores a list of alias, indexed by off-chain address string.
}

// NewProvider returns an in-memory contacts provider  with requested number of random contacts.
// The alias of the first random peer is "1", that of second is "2" and so on.
//
// It also provides a helper method NewRandomContact that adds a new peer to the contacts with
// off-chain address generated at random and commAddr pointing to a tcp port that is unused at the
// time of the time when function is invoked.
func NewProvider(numPeers uint, backend perun.WalletBackend) (*Provider, error) {
	c := &Provider{
		nextRandomAlias: 1,
		walletBackend:   backend,
		peersByAlias:    make(map[string]perun.Peer),
		aliasByAddr:     make(map[string]string),
	}
	for i := uint(0); i < numPeers; i++ {
		_, err := c.NewRandomContact()
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

// ReadByAlias returns the peer corresponding to given alias from the cache.
// The alias is a number, derived from an internal counter that is incremented on
// each new call to this function.
func (c *Provider) NewRandomContact() (alias string, _ error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	newPeer := perun.Peer{
		Alias:        fmt.Sprintf("%d", c.nextRandomAlias),
		OffChainAddr: ethereumtest.NewRandomAddress(rand.New(rand.NewSource(1729))),
		CommType:     "tcpip",
	}
	newPeer.OffChainAddrString = newPeer.OffChainAddr.String()
	port, err := freeport.GetFreePort()
	if err != nil {
		return "", errors.Wrap(err, "Getting a new free port")
	}
	newPeer.CommAddr = fmt.Sprintf("127.0.0.1:%d", port)

	c.nextRandomAlias++
	c.aliasByAddr[newPeer.OffChainAddrString] = newPeer.Alias
	c.peersByAlias[newPeer.Alias] = newPeer
	return newPeer.Alias, nil
}

// ReadByAlias returns the peer corresponding to given alias from the cache.
func (c *Provider) ReadByAlias(alias string) (_ perun.Peer, isPresent bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.readByAlias(alias)
}

func (c *Provider) readByAlias(alias string) (_ perun.Peer, isPresent bool) {
	var p perun.Peer
	p, isPresent = c.peersByAlias[alias]
	return p, isPresent
}

// ReadByOffChainAddr returns the peer corresponding to given off-chain address from the cache.
func (c *Provider) ReadByOffChainAddr(offChainAddr string) (_ perun.Peer, isPresent bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var alias string
	alias, isPresent = c.aliasByAddr[offChainAddr]
	if !isPresent {
		return perun.Peer{}, false
	}
	return c.readByAlias(alias)
}

// Write adds the peer to contacts cache. Returns an error if the alias is already used by same or different peer or,
// if the off-chain address string of the peer cannot be parsed using the wallet backend of this contacts provider.
func (c *Provider) Write(alias string, p perun.Peer) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if oldPeer, ok := c.peersByAlias[alias]; ok {
		if PeerEqual(oldPeer, p) {
			return errors.New("peer already present in contacts")
		}
		return errors.New("alias already used by another peer in contacts")
	}

	var err error
	p.OffChainAddr, err = c.walletBackend.ParseAddr(p.OffChainAddrString)
	if err != nil {
		return err
	}
	c.peersByAlias[alias] = p
	c.aliasByAddr[p.OffChainAddrString] = alias
	return nil
}

// Delete deletes the peer from contacts cache.
// Returns an error if peer corresponding to given alias is not found.
func (c *Provider) Delete(alias string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.peersByAlias[alias]; !ok {
		return errors.New("peer not found in contacts")
	}
	delete(c.peersByAlias, alias)
	return nil
}

// UpdateStorage is a Noop for in-memory contacts as there is no storage.
func (c *Provider) UpdateStorage() error {
	return nil
}
