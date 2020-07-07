package protocol

import (
	"github.com/pkg/errors"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"

	"github.com/direct-state-transfer/dst-go"
)

// Protocol implements the dst.ProtocolService interface.
type Protocol struct {
	*client.Client

	adjudicatoraddr wallet.Address // Address of the adjudicator contract deployed on the blockchain.
	assetHolderAddr wallet.Address // Address of the asset holder contract deployed on the blockchain.
}

// New initializes a protocol service for the given user and backends.
func New(user dst.User, onChainTx dst.OnChainTxBackend, adjudicator, assetHolder wallet.Address) (*Protocol, error) {

	dialer, err := user.NewDialer(user.Transport.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "initializing a new dialer")
	}

	listener, err := user.NewListener(user.Transport.Addr)
	if err != nil {
		return nil, err
	}

	if err = onChainTx.ValidateContracts(adjudicator, assetHolder); err != nil {
		return nil, errors.Wrap(err, "validating contracts")
	}
	funderInst := onChainTx.NewFunder(assetHolder)
	adjudicatorInst := onChainTx.NewAdjudicator(adjudicator, user.OnChainAcc.Address())

	protocolClient := client.New(user.OffchainAcc, dialer, funderInst, adjudicatorInst, user.OffChainWallet)
	s := &Protocol{
		Client:          protocolClient,
		adjudicatoraddr: adjudicator,
		assetHolderAddr: assetHolder,
	}

	// TODO: Initialize Persister

	// TODO: Mechanism to interrupt the go routines (when the protocol needs to stop).
	go protocolClient.Handle(&proposalHandler{}, &updateHandler{})
	go protocolClient.Listen(listener)
	return s, nil
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
