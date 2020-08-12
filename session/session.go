package session

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hyperledger-labs/perun-node"
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
	PayChProposalNotify PayChProposalNotify                // Map of subIDs to notifiers
	PayChProposalsCache []*client.ChannelProposal          // Cached proposals due to missing subscription.
	PayChResponders     map[string]perun.ProposalResponder // Map of proposalIDs (as hex string) to ProposalResponders.

	PayChCloseNotify PayChCloseNotify  // Map of subIDs to notifiers
	PayChCloseCache  []*PayChCloseInfo // Cached channel close events due to missing subscription.
}

// To use type func | interface method, decide later... For now type func.
type PayChProposalNotify interface {
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
	GetContacts() ([]perun.Peer, error)
	OpenPayCh(alias string, initBals BalInfo, ChDurSecs uint64) (PayChState, error)
	GetPayChs() []PayChState
	// The gRPC adapter should provide the concrete function to send notifications.
	// It should take the given parameters and send it to the user.
	// Session adopts fire and forget model for calling this function and hence does not care about error.
	// Retries etc., should be handled by the correspoding implementation.
	// This function registers the call back and returns the subscription id which is constant for a session.
	// For now, only one subscription per session (by the user of session) is allowed.
	// Errors when sub exists
	SubPayChProposals(PayChProposalNotify) error
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

func (s *Session) AddContact(contact perun.Peer) error {
	panic("not implemented") // TODO: Implement
}

func (s *Session) GetContacts() ([]perun.Peer, error) {
	panic("not implemented") // TODO: Implement
}

func (s *Session) OpenPayCh(alias string, initBals BalInfo, ChDurSecs uint64) (PayChState, error) {
	ch, err := s.ChClient.ProposeChannel(nil, &client.ChannelProposal{})
	_ = ch
	return PayChState{}, err
}

func (s *Session) GetPayChs() []PayChState {
	panic("not implemented") // TODO: Implement
}

func (s *Session) SubPayChProposals(notifier PayChProposalNotify) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChProposalNotify != nil {
		return errors.New("already subscribed")
	}
	s.PayChProposalNotify = notifier
	return nil
}

// Errors for unknown subscription id.
func (s *Session) UnsubPayChProposals() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChProposalNotify == nil {
		return errors.New("not subscribed")
	}
	s.PayChProposalNotify = nil
	return nil
}

func (s *Session) RespondToPayChProposalNotif(proposalID string, accept bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	responder, ok := s.PayChResponders[proposalID]
	if !ok {
		return errors.New("unknown proposal id")
	}
	if !accept {
		return responder.Reject(context.TODO(), "rejected by user")
	}
	sdkCh, err := responder.Accept(context.TODO(), client.ProposalAcc{})
	if err != nil {
		return err
	}

	chIDArr := sdkCh.ID()
	chID := BytesToHex(chIDArr[:])
	ch := &Channel{
		ID:         chID,
		Controller: sdkCh,
		LockState:  ChannelOpen,
	}
	s.Channels[chID] = ch
	return nil
}

func (s *Session) SubPayChClose(notifier PayChCloseNotify) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChCloseNotify != nil {
		return errors.New("already subscribed")
	}
	s.PayChCloseNotify = notifier
	return nil
}

// Errors for unknown subscription id.
func (s *Session) UnsubPayChClose() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.PayChCloseNotify == nil {
		return errors.New("not subscribed")
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
