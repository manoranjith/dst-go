package session

import (
	"github.com/hyperledger-labs/perun-node"
	"perun.network/go-perun/client"
)

// Remove the session defined in root level package ??
// Or replace it with an interface, that is accessible by the node ???
// Do after full implementation....
type Session struct {
	chClient perun.ChannelClient   // Perun Channel client.... Used for making calls.
	User     perun.User            // User of this session.... Move user inside session ?.. Wallet are attached to user.
	Contacts perun.Contacts        // Contact provider for this session.
	channels map[string][]*Channel // Map of channel IDs to channels in the Session.

	// send notification.
	// Mechanism to create subscription ID ?.... What is it unique of... ?
	// This subscription is for a session, but any number of subscriptions can be made and all are identical.
	// So use SessionID as the subscription ID. Later this can be changed.
	PayChProposalNotify PayChProposalNotify                  // Handler for sending notifications
	PayChResponders     map[string]*client.ProposalResponder // Map of proposalIDs to ProposalResponders.

	PayChProposalsCache []*client.ChannelProposal // Cached proposals due to missing subscription.
	PayChCloseCache     map[string]PayChCloseInfo // Cached channel close events due to missing subscription.
}

// To use type func | interface method, decide later... For now type func.
type PayChProposalNotify func(proposalID string, alias string, initBals BalInfo, ChDurSecs uint64)
type PayChCloseNotify func(finalBals BalInfo, _ error)

type PayChCloseInfo struct {
	finalBals BalInfo
	err       error
}

type SessionAPI interface {
	AddContact(contact perun.Peer) error
	GetContacts() ([]perun.Peer, error)
	OpenPayCh(alias string, initBals BalInfo, ChDurSecs uint64) (PayChState, error)
	GetPayChs() []PayChState
	// The gRPC adapter should provide the concrete function to send notifications.
	// It should take the given parameters and send it to the user.
	// Session adopts fire and forget model for calling this function and hence does not care about error.
	// Retries etc., should be handled by the correspoding implementation.
	// This function registers the call back and returns the subscription id which is constant for a session.
	// For now, session id itself is used as a subscription id.
	// For now, only one subscription per session (by the user of session) is allowed.
	SubPayChProposals(PayChProposalNotify) (subID string)
	// Clear the callback
	UnsubPayChProposals() error // Err if there is no subscription.
	RespondToPayChProposalNotif(proposalID string, accept bool) error
	SubPayChClose()
	// If persistOpenCh is
	// true - it will persist open channels, close the session and return the list of channels persisted.
	// false - it will close the session if no open channels, will err otherwise.
	CloseSession(persistOpenCh bool) (openPayChs []Channel, _ error)
}

func NewSession() {}
