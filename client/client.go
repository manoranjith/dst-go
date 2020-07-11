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
	"github.com/pkg/errors"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"

	"github.com/direct-state-transfer/dst-go/blockchain/ethereum"

	"github.com/direct-state-transfer/dst-go"
)

// Backend implements the dst.ProtocolService interface.
type Backend struct {
	*client.Client

	adjudicatoraddr wallet.Address // Address of the adjudicator contract deployed on the blockchain.
	assetAddr       wallet.Address // Address of the asset holder contract deployed on the blockchain.
}

// New initializes a protocol service for the given user and backends.
func New(url string, user dst.User, comm dst.CommBackend, adjudicator, asset wallet.Address) (*Backend, error) {

	dialer := comm.NewDialer()
	listener, err := comm.NewListener(user.CommAddr)
	if err != nil {
		return nil, err
	}

	onChainTx, err := ethereum.NewOnChainTxBackend(url, user.OnChainKeystore, user.OnChainAcc.Address())
	if err != nil {
		return nil, err
	}
	if err = onChainTx.ValidateContracts(adjudicator, asset); err != nil {
		return nil, errors.Wrap(err, "validating contracts")
	}
	funderInst := onChainTx.NewFunder(asset)
	adjudicatorInst := onChainTx.NewAdjudicator(adjudicator, user.OnChainAcc.Address())

	client := client.New(user.OffchainAcc, dialer, funderInst, adjudicatorInst, user.OffChainWallet)
	backend := &Backend{
		Client:          client,
		adjudicatoraddr: adjudicator,
		assetAddr:       asset,
	}

	// TODO: Initialize Persister

	// TODO: Mechanism to interrupt the go routines (when the protocol needs to stop).
	go client.Handle(&proposalHandler{}, &updateHandler{})
	go client.Listen(listener)

	return backend, nil
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
