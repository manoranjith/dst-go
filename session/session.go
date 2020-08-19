package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
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

var walletBackend perun.WalletBackend

func init() {
	// This can be overridden (only) in tests by calling the SetWalletBackend function.
	walletBackend = ethereum.NewWalletBackend()
}

type (
	// Session ...
	Session struct {
		log.Logger

		ID       string
		User     perun.User
		ChAsset  channel.Asset
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
		Currency   string
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
		ChannelID string
		Currency  string
		ChState   *channel.State
		Parts     []string
		Error     string
	}
)

func New(cfg Config) (*Session, error) {
	wb := walletBackend

	user, err := NewUnlockedUser(wb, cfg.User)
	if err != nil {
		return nil, err
	}

	if cfg.User.CommType != "tcp" {
		return nil, perun.ErrUnsupportedCommType
	}
	commBackend := tcp.NewTCPBackend(30 * time.Second)

	chClientCfg := client.Config{
		Chain: client.ChainConfig{
			Adjudicator: cfg.Adjudicator,
			Asset:       cfg.Asset,
			URL:         cfg.ChainURL,
			ConnTimeout: cfg.ChainConnTimeout,
		},
		DatabaseDir: cfg.DatabaseDir,
	}
	chClient, err := client.NewEthereumPaymentClient(chClientCfg, user, commBackend)
	if err != nil {
		return nil, err
	}
	chAsset, err := wb.ParseAddr(cfg.Asset)
	if err != nil {
		return nil, err
	}

	contacts, err := initContacts(cfg.ContactsType, cfg.ContactsURL, wb, user.Peer)
	if err != nil {
		return nil, err
	}
	sessionID := calcSessionID(user.OffChainAddr.Bytes())
	sess := &Session{
		Logger:               log.NewLoggerWithField("session-id", sessionID),
		ID:                   sessionID,
		User:                 user,
		ChAsset:              chAsset,
		ChClient:             chClient,
		Contacts:             contacts,
		Channels:             make(map[string]*Channel),
		chProposalResponders: make(map[string]ChProposalResponderEntry),
	}
	chClient.Handle(sess, sess) // Init handlers
	return sess, nil
}

func initContacts(contactsType, contactsURL string, wb perun.WalletBackend, ownInfo perun.Peer) (perun.Contacts, error) {
	if contactsType != "yaml" {
		return nil, perun.ErrUnsupportedContactsType
	}
	contacts, err := contactsyaml.New(contactsURL, wb)
	if err != nil {
		return nil, err
	}

	ownInfo.Alias = perun.OwnAlias
	err = contacts.Write(perun.OwnAlias, ownInfo)
	if err != nil && !errors.Is(err, perun.ErrPeerExists) {
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
	if err != nil {
		s.Logger.Error(err)
	}
	return perun.GetAPIError(err)
}

func (s *Session) GetContact(alias string) (perun.Peer, error) {
	s.Logger.Debug("Received request: session.GetContact")
	s.RLock()
	defer s.RUnlock()

	peer, isPresent := s.Contacts.ReadByAlias(alias)
	if !isPresent {
		s.Logger.Error(perun.ErrUnknownAlias)
		return perun.Peer{}, perun.ErrUnknownAlias
	}
	return peer, nil
}

// OpenCh
// Panics if the random number generator doesn't return a valid nonce.
func (s *Session) OpenCh(peerAlias string, openingBals BalInfo, app App, challengeDurSecs uint64) (ChannelInfo, error) {
	s.Logger.Debug("Received request: session.OpenCh")
	s.Lock()
	defer s.Unlock()

	peer, isPresent := s.Contacts.ReadByAlias(peerAlias)
	if !isPresent {
		s.Logger.Error(perun.ErrUnknownAlias)
		return ChannelInfo{}, perun.ErrUnknownAlias
	}
	s.ChClient.Register(peer.OffChainAddr, peer.CommAddr)

	if !currency.IsSupported(openingBals.Currency) {
		s.Logger.Error(perun.ErrUnsupportedCurrency.Error)
		return ChannelInfo{}, perun.ErrUnsupportedCurrency
	}

	allocations, err := makeAllocation(openingBals, peerAlias, s.ChAsset) // Pass a proper asset.
	if err != nil {
		s.Logger.Error(err)
		return ChannelInfo{}, perun.GetAPIError(err)
	}
	partAddrs := []wallet.Address{s.User.OffChainAddr, peer.OffChainAddr}
	parts := []string{perun.OwnAlias, peer.Alias}
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
		s.Logger.Error(err)
		// TODO: (mano) Use errors.Is here once a sentinal error is defined in the sdk.
		if strings.Contains(err.Error(), "channel proposal rejected") {
			err = perun.ErrPeerRejected
		}
		return ChannelInfo{}, perun.GetAPIError(err)
	}

	ch := NewChannel(pch, openingBals.Currency, parts)
	s.Channels[ch.ID] = ch

	go func(s *Session, chID string) {
		err := pch.Watch()
		s.HandleClose(chID, err)
	}(s, ch.ID)

	return ch.GetInfo(), nil
}

func (s *Session) HandleClose(chID string, err error) {
	s.Logger.Debug("SDK Callback: Channel watcher returned.")

	// Might be a mutex messup... check later.
	ch := s.Channels[chID]
	ch.Lock()
	defer ch.Unlock()

	chInfo := ch.getChInfo()
	notif := ChCloseNotif{
		ChannelID: chInfo.ChannelID,
		Currency:  chInfo.Currency,
		ChState:   chInfo.State,
		Parts:     chInfo.Parts,
	}
	if err != nil {
		notif.Error = err.Error()
	}

	if ch.LockState != ChannelClosed {
		ch.LockState = ChannelClosed
		if s.chCloseNotifier == nil {
			s.chCloseNotifsCache = append(s.chCloseNotifsCache, notif)
			s.Logger.Debug("SDK Callback: Notification cached")
		} else {
			s.chCloseNotifier(notif)
			s.Logger.Debug("SDK Callback: Notification sent")
		}
	}
}

// makeAllocation makes an allocation or the given BalInfo and channel asset.
// It errors, if the amounts in the balInfo are invalid.
// It arranges balances in this order: own, peer.
// PeerAddrs in channel also should be in the same order.
func makeAllocation(bals BalInfo, peerAlias string, chAsset channel.Asset) (*channel.Allocation, error) {
	ownBalAmount, ok := bals.Bals[perun.OwnAlias]
	if !ok {
		return nil, errors.Wrap(perun.ErrMissingBalance, "for self")
	}
	peerBalAmount, ok := bals.Bals[peerAlias]
	if !ok {
		return nil, errors.Wrap(perun.ErrMissingBalance, "for peer")
	}

	ownBal, err := currency.NewParser(bals.Currency).Parse(ownBalAmount)
	if err != nil {
		return nil, errors.WithMessage(perun.ErrInvalidAmount, "for self"+err.Error())
	}
	peerBal, err := currency.NewParser(bals.Currency).Parse(peerBalAmount)
	if err != nil {
		return nil, errors.WithMessage(perun.ErrInvalidAmount, "for peer"+err.Error())
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
		panic(err)
	}
	return val
}

func (s *Session) GetCh(channelID string) (*Channel, error) {
	s.Logger.Debug("Internal call to get channel instance.", channelID, "--")
	s.Lock()
	defer s.Unlock()

	ch, ok := s.Channels[channelID]
	if !ok {
		return nil, perun.ErrUnknownChannelID
	}
	return ch, nil
}

func (s *Session) GetChs() []ChannelInfo {
	s.Logger.Debug("Received request: session.GetChannels")
	s.Lock()
	defer s.Unlock()

	chInfos := make([]ChannelInfo, len(s.Channels))
	i := 0
	for _, ch := range s.Channels {
		chInfos[i] = ch.GetInfo()
	}
	i++
	return chInfos
}

func (s *Session) HandleUpdate(chUpdate pclient.ChannelUpdate, responder *pclient.UpdateResponder) {
	s.Logger.Debug("SDK Callback: HandleUpdate")
	s.Lock()
	defer s.Unlock()
	expiry := time.Now().UTC().Add(30 * time.Minute).Unix()

	channelID := chUpdate.State.ID
	channelIDStr := BytesToHex(channelID[:])
	updateID := fmt.Sprintf("%s_%d", BytesToHex(channelID[:]), chUpdate.State.Version)

	ch, ok := s.Channels[channelIDStr]
	if !ok {
		s.Logger.Info("Received update for unknown channel", channelIDStr)
		return
	}

	s.Logger.Debug("Waiting for lock")
	ch.Lock()
	defer ch.Unlock()
	ch.Logger.Debug("SDK Callback: Start processing")

	ch.Logger.Debug(fmt.Sprintf("%+v", ch.CurrState))
	err := validateUpdate(ch.CurrState, chUpdate.State.Clone())
	if err != nil {
		ch.Logger.Info("Received invalid update")
		err := responder.Reject(context.TODO(), "invalid update")
		if err != nil {
			s.Logger.Error("Rejecting invalid update", err)
		}
	}

	if chUpdate.State.IsFinal {
		ch.Logger.Info("Received final update, channel is finalized.")
		ch.LockState = ChannelFinalized
	}

	entry := ChUpdateResponderEntry{
		chUpdateResponder: responder,
		Expiry:            expiry,
	}
	ch.chUpdateResponders[updateID] = entry

	notif := ChUpdateNotif{
		UpdateID:  updateID,
		Currency:  ch.Currency,
		CurrState: ch.CurrState,
		Update:    &chUpdate,
		Parts:     ch.Parts,
		Expiry:    expiry,
	}
	if ch.chUpdateNotifier == nil {
		ch.chUpdateNotifCache = append(ch.chUpdateNotifCache, notif)
		ch.Logger.Debug("SDK Callback: Notification cached")
	} else {
		ch.chUpdateNotifier(notif)
		ch.Logger.Debug("SDK Callback: Notification sent")
	}
}

// For now, treat all channels as payment channels.
// TODO: (mano) Fix it once support is added in the sdk.
func validateUpdate(current, proposed *channel.State) error {
	var oldSum, newSum = big.NewInt(0), big.NewInt(0)
	oldBals := current.Allocation.Balances[0]
	oldSum.Add(oldBals[0], oldBals[1])
	newBals := proposed.Allocation.Balances[0]
	newSum.Add(newBals[0], newBals[1])

	if newSum.Cmp(oldSum) != 0 {
		return errors.New("invalid update: sum of balances is not constant")
	}

	if newBals[0].Sign() == -1 || newBals[1].Sign() == -1 {
		return errors.New("this update results in negative balance, hence not allowed")
	}
	return nil
}

func (s *Session) HandleProposal(chProposal *pclient.ChannelProposal, responder *pclient.ProposalResponder) {
	s.Logger.Debug("SDK Callback: HandleProposal")
	s.Lock()
	defer s.Unlock()
	expiry := time.Now().UTC().Add(30 * time.Minute).Unix()

	parts := make([]string, len(chProposal.PeerAddrs))
	for i := range chProposal.PeerAddrs {
		p, ok := s.Contacts.ReadByOffChainAddr(chProposal.PeerAddrs[i])
		if !ok {
			s.Logger.Info("Received channel proposal from unknonwn peer", chProposal.PeerAddrs[i].String())
			err := responder.Reject(context.TODO(), "unknonwn peer")
			if err != nil {
				s.Logger.Error("Rejecting channel proposal from unknown peer", err)
			}
		}
		parts[i] = p.Alias
	}

	proposalID := chProposal.SessID()
	proposalIDStr := BytesToHex(proposalID[:])
	entry := ChProposalResponderEntry{
		chProposalResponder: responder,
		Parts:               parts,
		Expiry:              expiry,
	}
	s.chProposalResponders[proposalIDStr] = entry

	// TODO: (mano) Implement a mechanism to exchange currecy of transaction between the two parties.
	// Currently assume ETH as the currency for incoming channel.
	notif := ChProposalNotif{
		ProposalID: proposalIDStr,
		Currency:   currency.ETH,
		Proposal:   chProposal,
		Parts:      parts,
		Expiry:     expiry,
	}
	if s.chProposalNotifier == nil {
		s.chProposalNotifsCache = append(s.chProposalNotifsCache, notif)
		s.Logger.Debug("SDK Callback: Notification cached")
	} else {
		s.chProposalNotifier(notif)
		s.Logger.Debug("SDK Callback: Notification sent")
	}
}

func (s *Session) SubChProposals(notifier ChProposalNotifier) error {
	s.Logger.Debug("Received request: session.SubChProposals")
	s.Lock()
	defer s.Unlock()

	if s.chProposalNotifier != nil {
		return perun.ErrSubAlreadyExists
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
		return perun.ErrNoActiveSub
	}
	s.chProposalNotifier = nil
	return nil
}

func (s *Session) RespondChProposal(chProposalID string, accept bool) error {
	s.Logger.Debug("Received request: session.RespondChProposal")
	s.Lock()
	defer s.Unlock()

	entry, ok := s.chProposalResponders[chProposalID]
	if !ok {
		s.Logger.Info("Unknonw proposal ID")
		return perun.ErrUnknownProposalID
	}
	delete(s.chProposalResponders, chProposalID)
	currTime := time.Now().UTC().Unix()
	if entry.Expiry < currTime {
		s.Logger.Info("timeout:", entry.Expiry, "received response at:", currTime)
		return perun.ErrRespTimeoutExpired
	}

	switch accept {
	case true:
		pch, err := entry.chProposalResponder.Accept(context.TODO(), pclient.ProposalAcc{Participant: s.User.OffChainAddr})
		if err != nil {
			s.Logger.Error("Accepting channel proposal", err)
			return perun.GetAPIError(err)
		}

		// TODO: (mano) Implement a mechanism to exchange currecy of transaction between the two parties.
		// Currently assume ETH as the currency for incoming channel.
		ch := NewChannel(pch, currency.ETH, entry.Parts)
		s.Channels[ch.ID] = ch

	case false:
		err := entry.chProposalResponder.Reject(context.TODO(), "rejected by user")
		if err != nil {
			s.Logger.Error("Rejecting channel proposal", err)
			return perun.GetAPIError(err)
		}
	}
	return nil
}

func (s *Session) SubChCloses(notifier ChCloseNotifier) error {
	s.Logger.Debug("Received request: session.SubChCloses")
	s.Lock()
	defer s.Unlock()

	if s.chCloseNotifier != nil {
		return perun.ErrSubAlreadyExists
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
		return perun.ErrNoActiveSub
	}
	s.chCloseNotifier = nil
	return nil
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("0x%x", b)
}
