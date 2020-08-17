package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/hyperledger-labs/perun-node"
	"github.com/pkg/errors"
	_ "perun.network/go-perun/backend/ethereum" // backend init
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"
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
	ProposalNotifier ProposalNotifier                   // Map of subIDs to notifiers
	ProposalsCache   []*ProposalNotification            // Cached proposals due to missing subscription.
	ProposalResponders  map[string]perun.ProposalResponder // Map of proposalIDs (as hex string) to ProposalResponders.

	ChCloseNotifier ChCloseNotifier // Map of subIDs to notifiers
	ChClosesCache   []*ChCloseInfo  // Cached channel close events due to missing subscription.
}

type ProposalNotification struct {
	proposal *client.ChannelProposal
	expiry   int64
}

// To use type func | interface method, decide later... For now type func.
type PayChProposalNotifier interface {
	PayChProposalNotify(proposalID string, peerAlias string, initBals BalInfo, ChallengeDurSecs uint64, expiry int64)
}
type ChCloseNotifier interface {
	PayChCloseNotify(finalBals BalInfo, _ error)
}

type ChCloseInfo struct {
	finalBals BalInfo
	err       error
}

type SessionAPI interface {
	AddContact(contact perun.Peer) error
	GetContact(alias string) (perun.Peer, error)
	OpenCh(alias string, initBals BalInfo, app App, ChDurSecs uint64) (*Channel, error)
	GetChs() []Channel
	// The gRPC adapter should provide the concrete function to send notifications.
	// It should take the given parameters and send it to the user.
	// Session adopts fire and forget model for calling this function and hence does not care about error.
	// Retries etc., should be handled by the correspoding implementation.
	// This function registers the call back and returns the subscription id which is constant for a session.
	// For now, only one subscription per session (by the user of session) is allowed.
	// Errors when sub exists
	SubChProposals(ProposalNotifier) error
	// Clear the callback
	// Errors when no sub exists
	UnsubChProposals() error // Err if there is no subscription.
	RespondToChProposalNotif(proposalID string, accept bool) error
	// Subscribe to payment channel close events
	SubPayChClose(ChCloseNotifier) error
	UnsubPayChClose() error // Err if there is no subscription.
	// If persistOpenCh is
	// true - it will persist open channels, close the session and return the list of channels persisted.
	// false - it will close the session if no open channels, will err otherwise.
	CloseSession(persistOpenCh bool) (openPayChs []Channel, _ error)
}

func NewSession() {}

func (s *Session) ContainsCh(id string) bool {
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

type App struct {
	Def  wallet.Address
	Data channel.Data
}

func (s *Session) OpenCh(peerAlias string, initBals BalInfo, app App, ChDurSecs uint64) (*Channel, error) {
	peer, isPresent := s.Contacts.ReadByAlias(peerAlias)
	if !isPresent {
		return nil, perun.NewAPIError(perun.ErrUnknownAlias, nil)
	}
	s.Dialer.Register(peer.OffChainAddr, peer.CommAddr)

	if !Exists(initBals.Currency) {
		return nil, perun.NewAPIError(perun.ErrUnknownCurrency, errors.New(initBals.Currency))
	}
	currencyParser := NewParser(initBals.Currency)
	selfBal, err := currencyParser.Parse(initBals.Bals["self"])
	if err != nil {
		return nil, perun.NewAPIError(perun.ErrInvalidAmount, errors.New("for self"))
	}
	peerBal, err := currencyParser.Parse(initBals.Bals[peerAlias])
	if err != nil {
		return nil, perun.NewAPIError(perun.ErrInvalidAmount, errors.New("for peer"))
	}

	proposal := &client.ChannelProposal{
		ChallengeDuration: ChDurSecs,
		Nonce:             nonce(),
		ParticipantAddr:   s.User.OffChainAddr,
		AppDef:            app.Def,
		InitData:          app.Data,
		InitBals: &channel.Allocation{
			Assets:   []channel.Asset{}, // TODO: Set this asset properly
			Balances: [][]*big.Int{{selfBal, peerBal}},
		},
		PeerAddrs: []wallet.Address{s.User.OffChainAddr, peer.OffChainAddr},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ch, err := s.ChClient.ProposeChannel(ctx, proposal)
	if err != nil {
		return nil, perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Proposing Channel"))
	}
	// TODO: Use NewChannel function to prepare other factors
	// chID := ch.ID()
	// s.Channels[BytesToHex(chID[:])] = &Channel{Controller: ch}
	// s.Channels[BytesToHex(chID[:])].AppParams[currency] = initBals.Currency
	// return s.Channels[BytesToHex(chID[:])], nil
	_ = ch
	return &Channel{}, nil
}

func nonce() *big.Int {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(max, big.NewInt(1))

	val, err := rand.Int(rand.Reader, max)
	if err != nil {
		_ = err
		// log.Panic("Could not create nonce")
	}
	return val
}

func (s *Session) GetChs() []Channel {
	chs := make([]Channel, len(s.Channels))
	for _, val := range s.Channels {
		chs = append(chs, *val)
	}
	return chs
}

func (s *Session) SubChProposals(notifier ProposalNotifier) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ProposalNotifier != nil {
		return perun.NewAPIError(perun.ErrSubAlreadyExists, nil)
	}
	s.ProposalNotifier = notifier
	return nil
}

// Errors for unknown subscription id.
func (s *Session) UnsubChProposals() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ProposalNotifier == nil {
		return perun.NewAPIError(perun.ErrNoActiveSub, nil)
	}
	s.ProposalNotifier = nil
	return nil
}

func (s *Session) RespondToChProposalNotif(proposalID string, accept bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	responder, ok := s.ProposalResponders[proposalID]
	if !ok {
		return perun.NewAPIError(perun.ErrUnknownProposalID, nil)
	}
	switch accept {
	case true:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sdkCh, err := responder.Accept(ctx, client.ProposalAcc{Participant: s.User.OffChainAddr})
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := responder.Reject(ctx, "rejected by user")
		if err != nil {
			return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Rejecting channel proposal"))
		}
	}
	return nil
}

func (s *Session) SubPayChClose(notifier ChCloseNotifier) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ChCloseNotifier != nil {
		return perun.NewAPIError(perun.ErrSubAlreadyExists, nil)
	}
	s.ChCloseNotifier = notifier
	return nil
}

// Errors for unknown subscription id.
func (s *Session) UnsubPayChClose() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.ChCloseNotifier == nil {
		return perun.NewAPIError(perun.ErrNoActiveSub, nil)
	}
	s.ChCloseNotifier = nil
	return nil
}

// If persistOpenCh is
// true - it will persist open channels, close the session and return the list of channels persisted.
// false - it will close the session if no open channels, will err otherwise.
func (s *Session) CloseSession(persistOpenCh bool) (openPayChs []Channel, _ error) {
	panic("not implemented") // TODO: Implement
}

func (s *Session) HandleProposal(prop *client.ChannelProposal, resp *client.ProposalResponder) {
	expiry := time.Now().Add(5 * time.Minute).UTC().Unix() // add proper calculation

	proposalID := prop.ProposalID()
	proposalIDStr := BytesToHex(proposalID[:])
	_ = proposalIDStr
	// s.ProposalResponders[proposalIDStr] = resp

	// TODO: check if proposer in contacts, else reject it and log .

	if s.ProposalNotifier == nil {
		s.ProposalsCache = append(s.ProposalsCache, &ProposalNotification{prop, expiry})
	} else {
		s.ProposalNotifier(prop, expiry)
	}
}

func (s *Session) HandleUpdate(update client.ChannelUpdate, responder *client.UpdateResponder) {
	expiry := time.Now().Add(5 * time.Minute).UTC().Unix() // add proper calculation

	channelID := BytesToHex(update.State.ID[:])
	if !s.ContainsCh(channelID) {
		// Log the channel and reject it. Unknown channel.
	}

	// As per v0.4.0 of go-perun SDK, a node can send only one update at a time.
	// Since only two parties exists, there can be only one active responder at a time.
	s.Channels[channelID].UpdateResponders = responder
	if update.State.IsFinal {
		s.Channels[channelID].LockState = ChannelFinalized
		// Call settle if accepted, waiting for two blockcs is implemented in go-perun now.
	}
	if !s.Channels[channelID].HasActiveSub() {
		// s.Channels[channelID].UpdateCache = update
	}
	s.Channels[channelID].UpdateNotify(update.State, expiry)
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("0x%x", b)
}
