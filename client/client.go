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

package client

import (
	"context"
	"sync"
	"time"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/peer"

	"github.com/pkg/errors"
	"perun.network/go-perun/channel/persistence/keyvalue"
	"perun.network/go-perun/client"
	"perun.network/go-perun/pkg/sortedkv/leveldb"

	"github.com/direct-state-transfer/dst-go/blockchain/ethereum"

	"github.com/direct-state-transfer/dst-go"
)

// Client is a wrapper type around the state channel client implementation from go-perun.
// It also manages the lifecycle of listener and handler go-routines.
type Client struct {
	*client.Client

	wg *sync.WaitGroup
}

// NewEthereumPaymentClient initializes a two party, ethereum payment channel client for the given user.
// It establishes a connection to the blockchain and verifies the integrity of contracts at the given address.
// It uses the comm backend to initialize adapters for off-chain communication network.
func NewEthereumPaymentClient(cfg Config, user dst.User, comm dst.CommBackend) (*Client, error) {
	dialer, listener, err := initComm(comm, user.CommAddr)
	if err != nil {
		return nil, err
	}
	funder, adjudicator, err := connectToChain(cfg.Chain, user.OnChain)
	if err != nil {
		return nil, err
	}
	// Only off-chain account is unlocked. Accounts for the participant addresses
	// will be unlocked by the client when required.
	offChainAcc, err := user.OffChain.Wallet.Unlock(user.OffChain.Addr)
	if err != nil {
		return nil, errors.WithMessage(err, "off-chain address")
	}

	c := client.New(offChainAcc, dialer, funder, adjudicator, user.OffChain.Wallet)
	if err := loadPersister(c, cfg.DatabaseDir, cfg.PeerReconnTimeout); err != nil {
		return nil, err
	}
	client := &Client{
		Client: c,
		wg:     &sync.WaitGroup{},
	}

	client.runAsGoRoutine(func() { client.Handle(&proposalHandler{}, &updateHandler{}) })
	client.runAsGoRoutine(func() { client.Listen(listener) })

	return client, nil
}

// Close closes the client and waits until the listener and handler go routines return.
func (c *Client) Close() error {
	err := c.Client.Close()
	c.wg.Wait()
	return errors.Wrap(err, "closing channel client")
}

func initComm(comm dst.CommBackend, listenAddr string) (peer.Dialer, peer.Listener, error) {
	if comm == nil {
		return nil, nil, errors.New("comm backend should not be nil")
	}
	dialer := comm.NewDialer()
	listener, err := comm.NewListener(listenAddr)
	if err != nil {
		return nil, nil, err
	}
	return dialer, listener, nil
}

func connectToChain(cfg ChainConfig, cred dst.Credential) (channel.Funder, channel.Adjudicator, error) {
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
	if err != nil {
		return nil, nil, errors.Wrap(err, "validating contracts")
	}
	funder := chain.NewFunder(assetAddr)
	adjudicator := chain.NewAdjudicator(adjudicatorAddr, cred.Addr)
	return funder, adjudicator, nil
}

func loadPersister(c *client.Client, dbPath string, reconnTimeout time.Duration) error {
	db, err := leveldb.LoadDatabase(dbPath)
	if err != nil {
		return errors.Wrap(err, "initializing persistence db in dir - "+dbPath)
	}
	pr, err := keyvalue.NewPersistRestorer(db)
	if err != nil {
		return errors.Wrap(err, "enabling persistence")
	}
	c.EnablePersistence(pr)
	ctx, cancel := context.WithTimeout(context.Background(), reconnTimeout)
	defer cancel()
	return c.Reconnect(ctx)
}

func (c *Client) runAsGoRoutine(f func()) {
	c.wg.Add(1)
	go func(wg *sync.WaitGroup) {
		f()
		wg.Done()
	}(c.wg)
}

type proposalHandler struct{}

// HandleProposal implements the client.ProposalHandler interface defined in go-perun.
// This method is called on every incoming channel proposal.
// TODO: Implement an accept all handler until user api components are implemented.
// TODO: Replace with proper implementation after user api components are implemented.
func (ph *proposalHandler) HandleProposal(_ *client.ChannelProposal, _ *client.ProposalResponder) {}

type updateHandler struct{}

// HandleUpdate implements the UpdateHandler interface.
// This method is called on every incoming state update for any channel managed by this client.
// TODO: Implement an accept all handler until user api components are implemented.
// TODO: Replace with proper implementation after user api components are implemented.
func (uh *updateHandler) HandleUpdate(_ client.ChannelUpdate, _ *client.UpdateResponder) {}
