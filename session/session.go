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
	pchannel "perun.network/go-perun/channel"
	pclient "perun.network/go-perun/client"
	pwallet "perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/client"
	"github.com/hyperledger-labs/perun-node/comm/tcp"
	"github.com/hyperledger-labs/perun-node/contacts/contactsyaml"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/log"
)

// walletBackend is used for parsing off-chain addresses in incoming contacts.
var walletBackend perun.WalletBackend

func init() {
	// This can be overridden (only) in tests by calling the SetWalletBackend function.
	walletBackend = ethereum.NewWalletBackend()
}

type (
	session struct {
		log.Logger

		id         string
		timeoutCfg timeoutConfig
		user       perun.User
		chAsset    pchannel.Asset
		chClient   perun.ChannelClient
		contacts   perun.Contacts

		channels map[string]*channel

		chProposalNotifier    perun.ChProposalNotifier
		chProposalNotifsCache []perun.ChProposalNotif
		chProposalResponders  map[string]chProposalResponderEntry

		chCloseNotifier    perun.ChCloseNotifier
		chCloseNotifsCache []perun.ChCloseNotif

		sync.RWMutex
	}

	chProposalResponderEntry struct {
		responder        chProposalResponder
		challengeDurSecs uint64
		parts            []string
		expiry           int64
	}

	//go:generate mockery -name chProposalResponder -output ../internal/mocks

	// Proposal Responder defines the methods on proposal responder that will be used by the perun node.
	chProposalResponder interface {
		Accept(context.Context, pclient.ProposalAcc) (*pclient.Channel, error)
		Reject(ctx context.Context, reason string) error
	}
)

func New(cfg Config) (*session, error) {
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
			Adjudicator:      cfg.Adjudicator,
			Asset:            cfg.Asset,
			URL:              cfg.ChainURL,
			ConnTimeout:      cfg.ChainConnTimeout,
			OnChainTxTimeout: cfg.OnChainTxTimeout,
		},
		DatabaseDir:       cfg.DatabaseDir,
		PeerReconnTimeout: cfg.PeerReconnTimeout,
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
	timeoutCfg := timeoutConfig{
		onChainTx: cfg.OnChainTxTimeout,
		response:  cfg.ResponseTimeout,
	}
	sess := &session{
		Logger:               log.NewLoggerWithField("session-id", sessionID),
		id:                   sessionID,
		timeoutCfg:           timeoutCfg,
		user:                 user,
		chAsset:              chAsset,
		chClient:             chClient,
		contacts:             contacts,
		channels:             make(map[string]*channel),
		chProposalResponders: make(map[string]chProposalResponderEntry),
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

func (s *session) ID() string {
	return s.id
}

func (s *session) AddContact(peer perun.Peer) error {
	s.Logger.Debug("Received request: session.AddContact")
	s.Lock()
	defer s.Unlock()

	err := s.contacts.Write(peer.Alias, peer)
	if err != nil {
		s.Logger.Error(err)
	}
	return perun.GetAPIError(err)
}

func (s *session) GetContact(alias string) (perun.Peer, error) {
	s.Logger.Debug("Received request: session.GetContact")
	s.RLock()
	defer s.RUnlock()

	peer, isPresent := s.contacts.ReadByAlias(alias)
	if !isPresent {
		s.Logger.Error(perun.ErrUnknownAlias)
		return perun.Peer{}, perun.ErrUnknownAlias
	}
	return peer, nil
}

// OpenCh
// Panics if the random number generator doesn't return a valid nonce.
func (s *session) OpenCh(peerAlias string, openingBals perun.BalInfo, app perun.App, challengeDurSecs uint64) (perun.ChannelInfo, error) {
	s.Logger.Debug("Received request: session.OpenCh")
	s.Lock()
	defer s.Unlock()

	peer, isPresent := s.contacts.ReadByAlias(peerAlias)
	if !isPresent {
		s.Logger.Error(perun.ErrUnknownAlias)
		return perun.ChannelInfo{}, perun.ErrUnknownAlias
	}
	s.chClient.Register(peer.OffChainAddr, peer.CommAddr)

	if !currency.IsSupported(openingBals.Currency) {
		s.Logger.Error(perun.ErrUnsupportedCurrency.Error)
		return perun.ChannelInfo{}, perun.ErrUnsupportedCurrency
	}

	allocations, err := makeAllocation(openingBals, peerAlias, s.chAsset) // Pass a proper asset.
	if err != nil {
		s.Logger.Error(err)
		return perun.ChannelInfo{}, perun.GetAPIError(err)
	}
	partAddrs := []pwallet.Address{s.user.OffChainAddr, peer.OffChainAddr}
	parts := []string{perun.OwnAlias, peer.Alias}
	proposal := &pclient.ChannelProposal{
		ChallengeDuration: challengeDurSecs,
		Nonce:             nonce(),
		ParticipantAddr:   s.user.OffChainAddr,
		AppDef:            app.Def,
		InitData:          app.Data,
		InitBals:          allocations,
		PeerAddrs:         partAddrs,
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.timeoutCfg.proposeCh(challengeDurSecs))
	defer cancel()
	pch, err := s.chClient.ProposeChannel(ctx, proposal)
	if err != nil {
		s.Logger.Error(err)
		// TODO: (mano) Use errors.Is here once a sentinal error value is defined in the sdk.
		if strings.Contains(err.Error(), "channel proposal rejected") {
			err = perun.ErrPeerRejected
		}
		return perun.ChannelInfo{}, perun.GetAPIError(err)
	}

	ch := NewChannel(pch, openingBals.Currency, parts, s.timeoutCfg)
	s.channels[ch.id] = ch

	go func(s *session, chID string) {
		err := pch.Watch()
		s.HandleClose(chID, err)
	}(s, ch.id)

	return ch.GetInfo(), nil
}

func (s *session) HandleClose(chID string, err error) {
	s.Logger.Debug("SDK Callback: Channel watcher returned.")

	ch := s.channels[chID]
	ch.Lock()
	defer ch.Unlock()

	chInfo := ch.getChInfo()
	notif := perun.ChCloseNotif{
		ChannelID: chInfo.ChannelID,
		Currency:  chInfo.Currency,
		ChState:   chInfo.State,
		Parts:     chInfo.Parts,
	}
	if err != nil {
		notif.Error = err.Error()
	}

	if ch.lockState != closed {
		ch.lockState = closed
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
func makeAllocation(bals perun.BalInfo, peerAlias string, chAsset pchannel.Asset) (*pchannel.Allocation, error) {
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
	return &pchannel.Allocation{
		Assets:   []pchannel.Asset{chAsset},
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

func (s *session) GetCh(channelID string) (perun.ChannelAPI, error) {
	s.Logger.Debug("Internal call to get channel instance.")
	s.Lock()
	defer s.Unlock()

	ch, ok := s.channels[channelID]
	if !ok {
		return nil, perun.ErrUnknownChannelID
	}
	return ch, nil
}

func (s *session) GetChInfos() []perun.ChannelInfo {
	s.Logger.Debug("Received request: session.GetChannels")
	s.Lock()
	defer s.Unlock()

	chInfos := make([]perun.ChannelInfo, len(s.channels))
	i := 0
	for _, ch := range s.channels {
		chInfos[i] = ch.GetInfo()
	}
	i++
	return chInfos
}

func (s *session) HandleUpdate(chUpdate pclient.ChannelUpdate, responder *pclient.UpdateResponder) {
	s.Logger.Debug("SDK Callback: HandleUpdate")
	s.Lock()
	defer s.Unlock()
	expiry := time.Now().UTC().Add(s.timeoutCfg.response).Unix()

	channelID := chUpdate.State.ID
	channelIDStr := BytesToHex(channelID[:])
	updateID := fmt.Sprintf("%s_%d", BytesToHex(channelID[:]), chUpdate.State.Version)

	ch, ok := s.channels[channelIDStr]
	if !ok {
		s.Logger.Info("Received update for unknown channel", channelIDStr)
		return
	}

	s.Logger.Debug("Waiting for lock")
	ch.Lock()
	defer ch.Unlock()
	ch.Logger.Debug("SDK Callback: Start processing")

	ch.Logger.Debug(fmt.Sprintf("%+v", ch.currState))

	if chUpdate.State.IsFinal {
		ch.Logger.Info("Received final update, channel is finalized.")
		ch.lockState = finalized
	}

	entry := chUpdateResponderEntry{
		responder: responder,
		expiry:    expiry,
	}
	ch.chUpdateResponders[updateID] = entry

	notif := perun.ChUpdateNotif{
		UpdateID:  updateID,
		Currency:  ch.currency,
		CurrState: ch.currState,
		Update:    &chUpdate,
		Parts:     ch.parts,
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

func (s *session) HandleProposal(chProposal *pclient.ChannelProposal, responder *pclient.ProposalResponder) {
	s.Logger.Debug("SDK Callback: HandleProposal")
	s.Lock()
	defer s.Unlock()
	expiry := time.Now().UTC().Add(s.timeoutCfg.response).Unix()

	parts := make([]string, len(chProposal.PeerAddrs))
	for i := range chProposal.PeerAddrs {
		p, ok := s.contacts.ReadByOffChainAddr(chProposal.PeerAddrs[i])
		if !ok {
			s.Logger.Info("Received channel proposal from unknonwn peer", chProposal.PeerAddrs[i].String())
			ctx, cancel := context.WithTimeout(context.Background(), s.timeoutCfg.respChProposalReject())
			defer cancel()
			err := responder.Reject(ctx, "peer not found in session contacts")
			if err != nil {
				s.Logger.Error("Rejecting channel proposal from unknown peer", err)
			}
			expiry = 0
		}
		parts[i] = p.Alias
	}

	proposalID := chProposal.SessID()
	proposalIDStr := BytesToHex(proposalID[:])
	entry := chProposalResponderEntry{
		responder:        responder,
		challengeDurSecs: chProposal.ChallengeDuration,
		parts:            parts,
		expiry:           expiry,
	}
	s.chProposalResponders[proposalIDStr] = entry

	// Set ETH as the currency interpretter for incoming channel.
	// TODO: (mano) Provide an option for user to configure when more currency interpretters are supported.
	notif := perun.ChProposalNotif{
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

// Start listening for sub before notif
func (s *session) SubChProposals(notifier perun.ChProposalNotifier) error {
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

func (s *session) UnsubChProposals() error {
	s.Logger.Debug("Received request: session.UnsubChProposals")
	s.Lock()
	defer s.Unlock()

	if s.chProposalNotifier == nil {
		return perun.ErrNoActiveSub
	}
	s.chProposalNotifier = nil
	return nil
}

func (s *session) RespondChProposal(chProposalID string, accept bool) error {
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
	if entry.expiry < currTime {
		s.Logger.Info("timeout:", entry.expiry, "received response at:", currTime)
		return perun.ErrRespTimeoutExpired
	}

	switch accept {
	case true:
		ctx, cancel := context.WithTimeout(context.Background(), s.timeoutCfg.respChProposalAccept(entry.challengeDurSecs))
		defer cancel()
		pch, err := entry.responder.Accept(ctx, pclient.ProposalAcc{Participant: s.user.OffChainAddr})
		if err != nil {
			s.Logger.Error("Accepting channel proposal", err)
			return perun.GetAPIError(err)
		}

		// Set ETH as the currency interpretter for incoming channel.
		// TODO: (mano) Provide an option for user to configure when more currency interpretters are supported.
		ch := NewChannel(pch, currency.ETH, entry.parts, s.timeoutCfg)
		s.channels[ch.id] = ch

	case false:
		ctx, cancel := context.WithTimeout(context.Background(), s.timeoutCfg.respChProposalReject())
		defer cancel()
		err := entry.responder.Reject(ctx, "rejected by user")
		if err != nil {
			s.Logger.Error("Rejecting channel proposal", err)
			return perun.GetAPIError(err)
		}
	}
	return nil
}

func (s *session) SubChCloses(notifier perun.ChCloseNotifier) error {
	s.Logger.Debug("Received request: session.SubChCloses")
	s.Lock()
	defer s.Unlock()

	if s.chCloseNotifier != nil {
		return perun.ErrSubAlreadyExists
	}
	s.chCloseNotifier = notifier

	// Send all cached notifications
	// TODO: (mano) This works for gRPC, but change to send in background.
	for i := len(s.chCloseNotifsCache); i > 0; i-- {
		s.chCloseNotifier(s.chCloseNotifsCache[0])
		s.chCloseNotifsCache = s.chCloseNotifsCache[1:i]

	}
	return nil
}

func (s *session) UnsubChCloses() error {
	s.Logger.Debug("Received request: session.UnsubChCloses")
	s.Lock()
	defer s.Unlock()

	if s.chCloseNotifier == nil {
		return perun.ErrNoActiveSub
	}
	s.chCloseNotifier = nil
	return nil
}

func (s *session) Close(force bool) error {
	s.Logger.Debug("Received request: session.Close")
	s.chProposalNotifier = nil
	s.chCloseNotifier = nil

	// Currently do force close always.
	err := s.chClient.Close()
	if err != nil {
		s.Logger.Error("Channel client closed with error: ", err)
	}
	return perun.GetAPIError(err)
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("0x%x", b)
}
