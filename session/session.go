package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/hyperledger-labs/perun-node"
	"github.com/pkg/errors"
	"perun.network/go-perun/client"
)

// Remove the session defined in root level package ??
// Or replace it with an interface, that is accessible by the node ???
// Do after full implementation....
type Session struct {
	mutex    sync.RWMutex
	ChClient perun.ChannelClient // Perun Channel client.... Used for making calls.
	User     perun.User          // User of this session.... Move user inside session ?.. Wallet are attached to user.
	Contacts perun.Contacts      // Contact provider for this session.
	Channels map[string]*Channel // Map of channel IDs to channels in the Session.

	Dialer perun.Registerer // instance for dialer for registering contacts with the sdk.

	// Only one subscription is allowed. Cache and deliver works only then
	PayChProposalNotify PayChProposalNotifier              // Map of subIDs to notifiers
	PayChProposalsCache []*client.ChannelProposal          // Cached proposals due to missing subscription.
	PayChResponders     map[string]perun.ProposalResponder // Map of proposalIDs (as hex string) to ProposalResponders.

	PayChCloseNotify PayChCloseNotify  // Map of subIDs to notifiers
	PayChCloseCache  []*PayChCloseInfo // Cached channel close events due to missing subscription.
}

// To use type func | interface method, decide later... For now type func.
type PayChProposalNotifier interface {
	PayChProposalNotify(proposalID string, alias string, initBals BalInfo, ChallengeDurSecs uint64)
}
type PayChCloseNotify interface {
	PayChCloseNotify(finalBals BalInfo, _ error)
}

type PayChCloseInfo struct {
	finalBals BalInfo
	err       error
}

type SessionAPI interface {
	AddContact(contact perun.Peer) error
	GetContact(alias string) (perun.Peer, error)
	OpenPayCh(alias string, initBals BalInfo, ChDurSecs uint64) error
	GetPayChs() []PayChState
	// The gRPC adapter should provide the concrete function to send notifications.
	// It should take the given parameters and send it to the user.
	// Session adopts fire and forget model for calling this function and hence does not care about error.
	// Retries etc., should be handled by the correspoding implementation.
	// This function registers the call back and returns the subscription id which is constant for a session.
	// For now, only one subscription per session (by the user of session) is allowed.
	// Errors when sub exists
	SubPayChProposals(PayChProposalNotifier) error
	// Clear the callback
	// Errors when no sub exists
	UnsubPayChProposals() error // Err if there is no subscription.
	RespondToPayChProposalNotif(proposalID string, accept bool) error
	// Subscribe to payment channel close events
	SubPayChClose(PayChCloseNotify) error
	UnsubPayChClose() error // Err if there is no subscription.
	// If persistOpenCh is
	// true - it will persist open channels, close the session and return the list of channels persisted.
	// false - it will close the session if no open channels, will err otherwise.
	CloseSession(persistOpenCh bool) (openPayChs []Channel, _ error)
}

func NewSession() {}

func (s *Session) ContainsPayCh(id string) bool {
	for _, ch := range s.Channels {
		if ch.ID == id {
			return true
		}
	}
	return false
}

func (s *Session) AddContact(peer perun.Peer) error {
	// Write returns only typed errors, to be reviewed.
	// It is more correct to say i/p to contacts does str -> addr (read) & addr -> str (write) ?
	// Makes a cleaner approach and easier error handling here ?
	return s.Contacts.Write(peer.Alias, peer)
}

func (s *Session) GetContact(alias string) (perun.Peer, error) {
	peer, isPresent := s.Contacts.ReadByAlias(alias)
	if !isPresent {
		return perun.Peer{}, perun.NewAPIError(perun.ErrUnknownAlias, nil)
	}
	return peer, nil
}

func (s *Session) OpenPayCh(alias string, initBals BalInfo, ChDurSecs uint64) error {
	peer, isPresent := s.Contacts.ReadByAlias(alias)
	if !isPresent {
		return perun.NewAPIError(perun.ErrUnknownAlias, nil)
	}
	s.Dialer.Register(peer.OffChainAddr, peer.CommAddr)
	// Use proposal maker hook from payment app.
	ch, err := s.ChClient.ProposeChannel(nil, &client.ChannelProposal{})
	if err != nil {
		return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Proposing Channel"))
	}
	// TODO: Use NewChannel function to prepare other factors
	s.Channels["testID"] = &Channel{Controller: ch}
	return nil
}

func (s *Session) GetPayChs() []PayChState {
	panic("not implemented") // TODO: Implement
}

func (s *Session) SubPayChProposals(notifier PayChProposalNotifier) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChProposalNotify != nil {
		return perun.NewAPIError(perun.ErrSubAlreadyExists, nil)
	}
	s.PayChProposalNotify = notifier
	return nil
}

// Errors for unknown subscription id.
func (s *Session) UnsubPayChProposals() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChProposalNotify == nil {
		return perun.NewAPIError(perun.ErrNoActiveSub, nil)
	}
	s.PayChProposalNotify = nil
	return nil
}

func (s *Session) RespondToPayChProposalNotif(proposalID string, accept bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	responder, ok := s.PayChResponders[proposalID]
	if !ok {
		return perun.NewAPIError(perun.ErrUnknownProposalID, nil)
	}
	switch accept {
	case true:
		sdkCh, err := responder.Accept(context.TODO(), client.ProposalAcc{Participant: s.User.OffChainAddr})
		if err != nil {
			return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Accepting channel proposal"))
		}

		chIDArr := sdkCh.ID()
		chID := BytesToHex(chIDArr[:])
		// TODO: Use new channel here
		ch := &Channel{
			ID:         chID,
			Controller: sdkCh,
			LockState:  ChannelOpen,
		}
		s.Channels[chID] = ch

	case false:
		err := responder.Reject(context.TODO(), "rejected by user")
		if err != nil {
			return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Rejecting channel proposal"))
		}
	}
	return nil
}

func (s *Session) SubPayChClose(notifier PayChCloseNotify) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChCloseNotify != nil {
		return perun.NewAPIError(perun.ErrSubAlreadyExists, nil)
	}
	s.PayChCloseNotify = notifier
	return nil
}

// Errors for unknown subscription id.
func (s *Session) UnsubPayChClose() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChCloseNotify == nil {
		return perun.NewAPIError(perun.ErrNoActiveSub, nil)
	}
	s.PayChCloseNotify = nil
	return nil
}

// If persistOpenCh is
// true - it will persist open channels, close the session and return the list of channels persisted.
// false - it will close the session if no open channels, will err otherwise.
func (s *Session) CloseSession(persistOpenCh bool) (openPayChs []Channel, _ error) {
	panic("not implemented") // TODO: Implement
}

func (s *Session) HandleProposal(_ *client.ChannelProposal, _ *client.ProposalResponder) {
}

func (s *Session) HandleUpdate(update client.ChannelUpdate, responder *client.UpdateResponder) {
	//if !s.ContainsPayCh(update.State.ID) {
	//	//Log the channel ID
	//	return
	//}
	//// As per v0.4.0 of go-perun SDK, a node can send only one update at a time.
	//// Since only two parties exists, there can be only one active responder at a time.
	//s.channels[update.State.ID].UpdateResponders = responder
	//if update.State.IsFinal {
	//	s.channels[update.State.ID].LockState = ChannelFinalized
	//	// TODO: Start settle timer.
	//}
	//if !s.channels[update.State.ID].HasActiveSub() {
	//	s.channels[update.State.ID].UpdateCache = update
	//}
	//// StateID during proposal is proposal id
	//alias := "as"  // retrieve peer index, address and get alias from contact
	//amount := "as" // retrieve  amount

	// s.channels[string(update.State.ID)].PayChUpdateNotify(alias string, amount string)
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("0x%x", b)
}
