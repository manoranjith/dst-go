package session

import (
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

	if cfg.ContactsType != "yaml" {
		return nil, errors.New("unsupported contacts provider type, use only yaml")
	}
	contacts, err := contactsyaml.New(cfg.ContactsURL, wb)
	if err != nil {
		return nil, err
	}
	user.Peer.Alias = contactsyaml.OwnAlias
	err = contacts.Write(contactsyaml.OwnAlias, user.Peer)
	if err != nil && !errors.Is(err, contactsyaml.ErrPeerExists) {
		return nil, errors.Wrap(err, "registering own user in contacts")
	}

	return &Session{
		// A mechanism for session id. user offchain address + time.
		ID:       user.OffChainAddrString,
		ChClient: chClient,
		Contacts: contacts,
	}, nil
}
