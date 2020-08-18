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

		Channels            map[string]*Channel
		proposalNotifier    ProposalNotifier
		proposalNotifsCache []ProposalNotification

		sync.RWMutex
	}

	ProposalNotification struct {
		proposal *pclient.ChannelProposal
		expiry   int64
	}

	ProposalNotifier func(ProposalNotification)
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

	proposal := &pclient.ChannelProposal{
		ChallengeDuration: challengeDurSecs,
		Nonce:             nonce(),
		ParticipantAddr:   s.User.OffChainAddr,
		AppDef:            app.Def,
		InitData:          app.Data,
		InitBals:          allocations,
		PeerAddrs:         []wallet.Address{s.User.OffChainAddr, peer.OffChainAddr},
	}
	pch, err := s.ChClient.ProposeChannel(context.TODO(), proposal)
	if err != nil {
		return nil, err
	}

	ch := NewChannel(pch)
	s.Channels[ch.ID] = ch

	return ch, nil
}

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

func (s *Session) SubChProposals(notifier ProposalNotifier) error {
	s.Logger.Debug("Received request: session.SubChProposals")
	s.Lock()
	defer s.Unlock()

	if s.proposalNotifier != nil {
		return errors.New("")
	}
	s.proposalNotifier = notifier

	// Send all cached notifications
	for i := len(s.proposalNotifsCache) - 1; i >= 0; i-- {
		s.proposalNotifier(s.proposalNotifsCache[0])
		s.proposalNotifsCache = s.proposalNotifsCache[1 : i+1]
	}

	return nil
}

func (s *Session) UnsubChProposals(notifier ProposalNotifier) error {
	s.Logger.Debug("Received request: session.UnsubChProposals")
	s.Lock()
	defer s.Unlock()

	if s.proposalNotifier == nil {
		return errors.New("")
	}
	s.proposalNotifier = nil
	return nil
}

func BytesToHex(b []byte) string {
	return fmt.Sprintf("0x%x", b)
}
