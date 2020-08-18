package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	pclient "perun.network/go-perun/client"
	"perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/client"
	"github.com/hyperledger-labs/perun-node/comm/tcp"
	"github.com/hyperledger-labs/perun-node/contacts/contactsyaml"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/log"
)

type (
	// Session ...
	Session struct {
		log.Logger

		ID       string
		User     perun.User
		ChClient perun.ChannelClient
		Contacts perun.Contacts

		Channels map[string]*Channel

		chProposalNotifier    ChProposalNotifier
		chProposalNotifsCache []ChProposalNotif
		chProposalResponders  map[string]ChProposalResponderEntry

		chCloseNotifier    ChCloseNotifier
		chCloseNotifsCache []ChCloseNotif

		sync.RWMutex
	}

	ChProposalNotifier func(ChProposalNotif)

	ChProposalNotif struct {
		ProposalID string
		Proposal   *pclient.ChannelProposal
		Parts      []string
		Expiry     int64
	}

	ChProposalResponderEntry struct {
		chProposalResponder ChProposalResponder
		Parts               []string
		Expiry              int64
	}

	//go:generate mockery -name ProposalResponder -output ../internal/mocks

	// Proposal Responder defines the methods on proposal responder that will be used by the perun node.
	ChProposalResponder interface {
		Accept(context.Context, pclient.ProposalAcc) (*pclient.Channel, error)
		Reject(ctx context.Context, reason string) error
	}

	ChCloseNotifier func(ChCloseNotif)

	ChCloseNotif struct {
		ChState *channel.State
		Expiry  int64
	}
)

func New(cfg Config) (*Session, error) {
	wb := ethereum.NewWalletBackend()

	user, err := NewUnlockedUser(wb, cfg.User)
	if err != nil {
		return nil, err
	}

	if cfg.User.CommType != "tcp" {
		return nil, errors.New("unsupported comm type, use only tcp")
	}
	commBackend := tcp.NewTCPBackend(30 * time.Second)

	chClientCfg := client.Config{
		Chain: client.ChainConfig{
			Adjudicator: cfg.Adjudicator,
			Asset:       cfg.Asset,
			URL:         cfg.ChainURL,
		},
		DatabaseDir: cfg.DatabaseDir,
	}
	chClient, err := client.NewEthereumPaymentClient(chClientCfg, user, commBackend)
	if err != nil {
		return nil, err
	}

	contacts, err := initContacts(cfg.ContactsType, cfg.ContactsURL, wb, user.Peer)
	if err != nil {
		return nil, err
	}
	sessionID := calcSessionID(user.OffChainAddr.Bytes())
	return &Session{
		Logger:   log.NewLoggerWithField("session-id", sessionID),
		ID:       sessionID,
		ChClient: chClient,
		Contacts: contacts,
		Channels: make(map[string]*Channel),
	}, nil
}

func initContacts(contactsType, contactsURL string, wb perun.WalletBackend, self perun.Peer) (perun.Contacts, error) {
	if contactsType != "yaml" {
		return nil, errors.New("unsupported contacts provider type, use only yaml")
	}
	contacts, err := contactsyaml.New(contactsURL, wb)
	if err != nil {
		return nil, err
	}

	// user.Peer.Alias = contactsyaml.OwnAlias
	err = contacts.Write(contactsyaml.OwnAlias, self)
	if err != nil && !errors.Is(err, contactsyaml.ErrPeerExists) {
		return nil, errors.Wrap(err, "registering own user in contacts")
	}
	return contacts, nil
}

// calcSessionID calculates the sessionID as sha256 hash over the off-chain address of the user and
// the current UTC time.
//
// A time dependant parameter is required to ensure the same user is able to open multiple sessions
// with the same node and have unique session id for each.
func calcSessionID(userOffChainAddr []byte) string {
	h := sha256.New()
	h.Write(userOffChainAddr)
	h.Write([]byte(time.Now().UTC().String()))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *Session) AddContact(peer perun.Peer) error {
	s.Logger.Debug("Received request: session.AddContact")
	s.Lock()
	defer s.Unlock()

	err := s.Contacts.Write(peer.Alias, peer)
	// TODO, check and name errors.
	return err
}

func (s *Session) GetContact(alias string) (perun.Peer, error) {
	s.Logger.Debug("Received request: session.GetContact")
	s.RLock()
	defer s.RUnlock()

	peer, isPresent := s.Contacts.ReadByAlias(alias)
	if !isPresent {
		return perun.Peer{}, errors.New("")
	}
	return peer, nil
}

func (s *Session) OpenCh(peerAlias string, openingBals BalInfo, app App, challengeDurSecs uint64) (*Channel, error) {
	s.Logger.Debug("Received request: session.OpenCh")
	s.Lock()
	defer s.Unlock()

	peer, isPresent := s.Contacts.ReadByAlias(peerAlias)
	if !isPresent {
		return nil, errors.New("") // return error.... to add known errors.
	}
	s.ChClient.Register(peer.OffChainAddr, peer.CommAddr)

	if !currency.IsSupported(openingBals.Currency) {
		return nil, errors.New("") // return error.... to add known errors.
	}

	allocations, err := makeAllocation(openingBals, peerAlias, nil) // Pass a proper asset.
	if err != nil {
		return nil, err
	}
	partAddrs := []wallet.Address{s.User.OffChainAddr, peer.OffChainAddr}
	parts := []string{"self", peerAlias}
	proposal := &pclient.ChannelProposal{
		ChallengeDuration: challengeDurSecs,
		Nonce:             nonce(),
		ParticipantAddr:   s.User.OffChainAddr,
		AppDef:            app.Def,
		InitData:          app.Data,
		InitBals:          allocations,
		PeerAddrs:         partAddrs,
	}
	pch, err := s.ChClient.ProposeChannel(context.TODO(), proposal)
	if err != nil {
		return nil, err
	}

	ch := NewChannel(pch, openingBals.Currency, parts)
	s.Channels[ch.ID] = ch

	return ch, nil
}

// func (s *Session) GetChannels() []*channel.State

// makeAllocation makes an allocation or the given BalInfo and channel asset.
// It errors, if the amounts in the balInfo are invalid.
// It arranges balances in this order: own, peer.
// PeerAddrs in channel also should be in the same order.
func makeAllocation(bals BalInfo, peerAlias string, chAsset channel.Asset) (*channel.Allocation, error) {
	ownBal, err := currency.NewParser(bals.Currency).Parse("self")
	if err != nil {
		return nil, errors.WithMessage(err, "own balance")
	}
	peerBal, err := currency.NewParser(bals.Currency).Parse(peerAlias)
	if err != nil {
		return nil, errors.WithMessage(err, "peer balance")
	}
	return &channel.Allocation{
		Assets:   []channel.Asset{chAsset},
		Balances: [][]*big.Int{{ownBal, peerBal}},
	}, nil
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

func (s *Session) HandleUpdate(update pclient.ChannelUpdate, resp *pclient.UpdateResponder) {
	s.Logger.Debug("SDK Callback: HandleUpdate")
	s.Lock()
	defer s.Unlock()
	expiry := time.Now().UTC().Add(30 * time.Minute).Unix()

	channelID := update.State.ID
	channelIDStr := fmt.Sprintf("%s_%d", BytesToHex(channelID[:]), update.State.Version)
	ch, ok := s.Channels[channelIDStr]
	if !ok {
		// reject as unknown channel
	}
	if update.State.IsFinal {
		ch.LockState = ChannelFinalized
	}

	entry := ChUpdateResponderEntry{
		chUpdateResponder: resp,
		Expiry:            expiry,
	}
	ch.chUpdateResponders[channelIDStr] = entry

	notif := ChUpdateNotif{channelIDStr, &update, expiry}
	if ch.chUpdateNotifier == nil {
		ch.chUpdateNotifCache = append(ch.chUpdateNotifCache, notif)
	} else {
		ch.chUpdateNotifier(notif)
	}
}

func (s *Session) HandleProposal(req *pclient.ChannelProposal, res *pclient.ProposalResponder) {
	s.Logger.Debug("SDK Callback: HandleProposal")
	s.Lock()
	defer s.Unlock()
	expiry := time.Now().UTC().Add(30 * time.Minute).Unix()

	parts := make([]string, len(req.PeerAddrs))
	for i := range req.PeerAddrs {
		p, ok := s.Contacts.ReadByOffChainAddr(req.PeerAddrs[i])
		if !ok {
			// reject proposal
		}
		parts[i] = p.Alias
	}

	proposalID := req.SessID()
	proposalIDStr := BytesToHex(proposalID[:])
	entry := ChProposalResponderEntry{
		chProposalResponder: res,
		Parts:               parts,
		Expiry:              expiry,
	}
	s.chProposalResponders[proposalIDStr] = entry

	notif := ChProposalNotif{proposalIDStr, req, parts, expiry}
	if s.chProposalNotifier == nil {
		s.chProposalNotifsCache = append(s.chProposalNotifsCache, notif)
	} else {
		s.chProposalNotifier(notif)
	}
}

func (s *Session) SubChProposals(notifier ChProposalNotifier) error {
	s.Logger.Debug("Received request: session.SubChProposals")
	s.Lock()
	defer s.Unlock()

	if s.chProposalNotifier != nil {
		return errors.New("")
	}
	s.chProposalNotifier = notifier

	// Send all cached notifications
	// TODO: (mano) This works for gRPC, but change to send in background.
	for i := len(s.chProposalNotifsCache) - 1; i >= 0; i-- {
		s.chProposalNotifier(s.chProposalNotifsCache[0])
		s.chProposalNotifsCache = s.chProposalNotifsCache[1 : i+1]
	}

	return nil
}

func (s *Session) UnsubChProposals() error {
	s.Logger.Debug("Received request: session.UnsubChProposals")
	s.Lock()
	defer s.Unlock()

	if s.chProposalNotifier == nil {
		return errors.New("")
	}
	s.chProposalNotifier = nil
	return nil
}

func (s *Session) RespondChProposal(chProposalID string, accept bool) error {
	s.Logger.Debug("Received request: session.RespondChProposal")
	s.Lock()
	defer s.Unlock()

	entry, ok := s.chProposalResponders[chProposalID]
	delete(s.chProposalResponders, chProposalID)
	if !ok {
		return errors.New("")
	}
	if entry.Expiry > time.Now().UTC().Unix() {
		return errors.New("")
	}

	switch accept {
	case true:
		pch, err := entry.chProposalResponder.Accept(context.TODO(), pclient.ProposalAcc{Participant: s.User.OffChainAddr})
		if err != nil {
			return errors.New("")
		}

		// TODO: (mano) Implement a mechanism to exchange currecy of transaction between the two parties.
		// Currently assume ETH as the currency for incoming channel.
		ch := NewChannel(pch, currency.ETH, entry.Parts)
		s.Channels[ch.ID] = ch

	case false:
		err := entry.chProposalResponder.Reject(context.TODO(), "rejected by user")
		if err != nil {
			return errors.New("")
		}
	}
	return nil
}

func (s *Session) SubChCloses(notifier ChCloseNotifier) error {
	s.Logger.Debug("Received request: session.SubChCloses")
	s.Lock()
	defer s.Unlock()

	if s.chCloseNotifier != nil {
		return errors.New("")
	}
	s.chCloseNotifier = notifier

	// TODO: (mano) This works for gRPC, but change to send in background.
	// Send all cached notifications
	for i := len(s.chCloseNotifsCache); i > 0; i-- {
		s.chCloseNotifier(s.chCloseNotifsCache[0])
		s.chCloseNotifsCache = s.chCloseNotifsCache[1:i]
	}

	return nil
}

func (s *Session) UnsubChCloses() error {
	s.Logger.Debug("Received request: session.UnsubChCloses")
	s.Lock()
	defer s.Unlock()

	if s.chCloseNotifier == nil {
		return errors.New("")
	}
	s.chCloseNotifier = nil
	return nil
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("0x%x", b)
}
