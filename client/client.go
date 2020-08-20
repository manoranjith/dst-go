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

package client

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/go-perun/client"
	perunLog "perun.network/go-perun/log"
	"perun.network/go-perun/pkg/sortedkv/leveldb"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
)

// ChannelClient represents the methods on client.Client that are used.
type ChannelClient interface {
	ProposeChannel(context.Context, *client.ChannelProposal) (*client.Channel, error)
	Handle(client.ProposalHandler, client.UpdateHandler)
	Channel(channel.ID) (*client.Channel, error)
	Close() error

	EnablePersistence(persistence.PersistRestorer)
	OnNewChannel(handler func(*client.Channel))
	Restore(context.Context) error

	Log() perunLog.Logger
}

// Client is a wrapper type around the state channel client implementation from go-perun.
// It also manages the lifecycle of a message bus that is used for off-chain communication.
type Client struct {
	ChannelClient
	perun.WireBus

	// Registry that is used by the channel client for resolving off-chain address to comm address.
	msgBusRegistry perun.Registerer

	wg *sync.WaitGroup
}

// NewEthereumPaymentClient initializes a two party, ethereum payment channel client for the given user.
// It establishes a connection to the blockchain and verifies the integrity of contracts at the given address.
// It uses the comm backend to initialize adapters for off-chain communication network.
func NewEthereumPaymentClient(cfg Config, user perun.User, comm perun.CommBackend) (*Client, error) {
	funder, adjudicator, err := connectToChain(cfg.Chain, user.OnChain)
	if err != nil {
		return nil, err
	}
	offChainAcc, err := user.OffChain.Wallet.Unlock(user.OffChain.Addr)
	if err != nil {
		return nil, errors.WithMessage(err, "off-chain account")
	}
	dialer := comm.NewDialer()
	msgBus := net.NewBus(offChainAcc, dialer)

	c, err := client.New(offChainAcc.Address(), msgBus, funder, adjudicator, user.OffChain.Wallet)
	if err != nil {
		return nil, errors.Wrap(err, "initializing state channel client")
	}
	if err = loadPersister(c, cfg.DatabaseDir, cfg.PeerReconnTimeout); err != nil {
		return nil, err
	}

	client := &Client{
		ChannelClient:  c,
		WireBus:        msgBus,
		msgBusRegistry: dialer,
		wg:             &sync.WaitGroup{},
	}

	listener, err := comm.NewListener(user.CommAddr)
	if err != nil {
		return nil, err
	}
	client.runAsGoRoutine(func() { msgBus.Listen(listener) })

	return client, nil
}

func (c *Client) Register(offChainAddr wire.Address, commAddr string) {
	c.msgBusRegistry.Register(offChainAddr, commAddr)
}

func (c *Client) Handle(ph client.ProposalHandler, ch client.UpdateHandler) {
	c.runAsGoRoutine(func() { c.ChannelClient.Handle(ph, ch) })
}

// Close closes the client and waits until the listener and handler go routines return.
//
// Close depends on the following mechanisms implemented in client.Close and bus.Close to signal the go-routines:
// 1. When client.Close is invoked, it cancels the Update and Proposal handlers via a context.
// 2. When bus.Close in invoked, it invokes EndpointRegistry.Close that shuts down the listener via onCloseCallback.
func (c *Client) Close() error {
	if err := c.ChannelClient.Close(); err != nil {
		return errors.Wrap(err, "closing channel client")
	}
	if busErr := c.WireBus.Close(); busErr != nil {
		return errors.Wrap(busErr, "closing message bus")
	}
	c.wg.Wait()
	return nil
}

func connectToChain(cfg ChainConfig, cred perun.Credential) (channel.Funder, channel.Adjudicator, error) {
	walletBackend := ethereum.NewWalletBackend()
	assetAddr, err := walletBackend.ParseAddr(cfg.Asset)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "asset address")
	}
	adjudicatorAddr, err := walletBackend.ParseAddr(cfg.Adjudicator)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "adjudicator address")
	}

	chain, err := ethereum.NewChainBackend(cfg.URL, cfg.ConnTimeout, cred)
	if err != nil {
		return nil, nil, err
	}
	err = chain.ValidateContracts(adjudicatorAddr, assetAddr)
	return chain.NewFunder(assetAddr), chain.NewAdjudicator(adjudicatorAddr, cred.Addr), err
}

func loadPersister(c *client.Client, dbPath string, reconnTimeout time.Duration) error {
	db, err := leveldb.LoadDatabase(dbPath)
	if err != nil {
		return errors.Wrap(err, "initializing persistence database in dir - "+dbPath)
	}
	pr := keyvalue.NewPersistRestorer(db)
	c.EnablePersistence(pr)
	ctx, cancel := context.WithTimeout(context.Background(), reconnTimeout)
	defer cancel()
	return c.Restore(ctx)
}

func (c *Client) runAsGoRoutine(f func()) {
	c.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		f()
	}(c.wg)
}

// ProposalHandler implements the handler for incoming channel proposals.
type ProposalHandler struct{}

// HandleProposal implements the client.ProposalHandler interface defined in go-perun.
// This method is called on every incoming channel proposal.
// TODO: (mano) Implement an accept all handler until user api components are implemented.
// TODO: (mano) Replace with proper implementation after user api components are implemented.
func (ph *ProposalHandler) HandleProposal(_ *client.ChannelProposal, _ *client.ProposalResponder) {
	panic("proposalHandler.HandleProposal not implemented")
}

// UpdateHandler implements the handler for incoming state updates.
type UpdateHandler struct{}

// HandleUpdate implements the UpdateHandler interface.
// This method is called on every incoming state update for any channel managed by this client.
// TODO: (mano) Implement an accept all handler until user api components are implemented.
// TODO: (mano) Replace with proper implementation after user api components are implemented.
func (uh *UpdateHandler) HandleUpdate(_ client.ChannelUpdate, _ *client.UpdateResponder) {
	panic("updateHandler.HandleUpdate not implemented")
}
