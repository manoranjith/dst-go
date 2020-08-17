package node

import (
	"time"

	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/log"
	"github.com/hyperledger-labs/perun-node/session"
)

type Node struct {
	log.Logger

	cfg Config

	Adjudicator, AssetHolder wallet.Address
	Sessions                 map[string]*session.Session // Map of session ID to session instances.
}

func New(chainAddr, adjudicatorAddr, assetAddr, logLevel, logFile string) (*Node, error) {
	logger, err := log.NewLogger(logLevel, logFile)
	if err != nil {
		return nil, errors.WithMessage(err, "initializing logger for node")
	}

	// TODO: Currently, credentials are required to initialize a chain backend
	// for connecting to node and validating contracts. So store the config.
	adjudicator, err := ethereum.NewWalletBackend().ParseAddr(adjudicatorAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}
	asset, err := ethereum.NewWalletBackend().ParseAddr(assetAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}

	return &Node{
		Logger: logger,
		cfg: Config{
			ChainAddr:       chainAddr,
			AdjudicatorAddr: adjudicatorAddr,
			AssetAddr:       assetAddr,
			CommTypes:       []string{"tcp"},
			ContactTypes:    []string{"yaml"},
			Currencies:      []string{"ETH"},
		},
		Adjudicator: adjudicator,
		AssetHolder: asset,
		Sessions:    make(map[string]*session.Session),
	}, nil
}

func (n *Node) Time() int64 {
	n.Logger.Debug("Received request: node.Time")
	return time.Now().UTC().Unix()
}

func (n *Node) GetConfig() Config {
	n.Logger.Debug("Received request: node.GetConfig")
	return n.cfg
}

func (n *Node) Help() []string {
	// TODO: Collect and return list of supported APIs.
	return []string{}
}

func (n *Node) OpenSession(configFile string) (ID string, _ error) {
	n.Logger.Debug("Received request: node.OpenSession")
	// TODO: Parse and prepare the configuration.
	s, err := session.New(session.Config{})
	if err != nil {
		return "", err
	}
	s.Logger = n.Logger.WithField("session", s.ID)
	n.Sessions[s.ID] = s
	return s.ID, nil
}
