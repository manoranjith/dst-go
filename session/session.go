package session

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/client"
	"github.com/hyperledger-labs/perun-node/comm/tcp"
	"github.com/hyperledger-labs/perun-node/contacts/contactsyaml"
	"github.com/hyperledger-labs/perun-node/log"
)

// Session ...
type Session struct {
	log.Logger

	ID       string
	ChClient perun.ChannelClient
	Contacts perun.Contacts

	sync.RWMutex
}

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
