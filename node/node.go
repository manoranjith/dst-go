package node

import (
	"time"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/log"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/pkg/errors"
	psync "perun.network/go-perun/pkg/sync"
)

type node struct {
	log.Logger

	cfg perun.NodeConfig

	sessions map[string]perun.SessionAPI // Map of session ID to session instances.

	psync.Mutex
}

func New(cfg perun.NodeConfig) (*node, error) {
	// To validate the contracts, credentials are required for connecting to the
	// blockchain, which only a session has.
	// For now, just check if the addresses are valid.
	wb := ethereum.NewWalletBackend()
	_, err := wb.ParseAddr(cfg.AdjudicatorAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}
	_, err = wb.ParseAddr(cfg.AssetAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}

	err = log.InitLogger(cfg.LogLevel, cfg.LogFile)
	if err != nil {
		return nil, errors.WithMessage(err, "initializing logger for node")
	}
	return &node{
		Logger: log.NewLoggerWithField("node", 1), // ID of the node is always 1.
		cfg: perun.NodeConfig{
			LogLevel: cfg.LogLevel,
			LogFile:  cfg.LogFile,

			ChainAddr:       cfg.ChainAddr,
			AdjudicatorAddr: cfg.AdjudicatorAddr,
			AssetAddr:       cfg.AssetAddr,
			CommTypes:       []string{"tcp"},
			ContactTypes:    []string{"yaml"},
			Currencies:      []string{"ETH"},
		},
		sessions: make(map[string]perun.SessionAPI),
	}, nil
}

func (n *node) Time() int64 {
	n.Logger.Debug("Received request: node.Time")
	return time.Now().UTC().Unix()
}

func (n *node) GetConfig() perun.NodeConfig {
	n.Logger.Debug("Received request: node.GetConfig")
	return n.cfg
}

func (n *node) Help() []string {
	return []string{"payment"}
}

// OpenSession opens a new session based on the given configuration.
// Parameters to connect to the chain (chainURL, asset & adjudicator addresses) are optional.
// If missing default values from the node will be used.
//
// The node also initializes a logger for the generated session that logs along with its session id.
func (n *node) OpenSession(configFile string) (ID string, _ error) {
	n.Logger.Debug("Received request: node.OpenSession")
	n.Logger.Debug(configFile)
	n.Lock()
	defer n.Unlock()

	sessionCfg, err := session.ParseConfig(configFile)
	if err != nil {
		n.Logger.Error(err)
		return "", perun.ErrInvalidConfig
	}
	n.fillInSessionConfig(&sessionCfg)
	n.Logger.Debugf("%+v", sessionCfg)
	s, err := session.New(sessionCfg)
	if err != nil {
		return "", err
	}
	n.sessions[s.ID()] = s
	return s.ID(), nil
}

// fillInSessionConfig fills in the missing values in session configuration
// for those fields that have a default value in the node config.
func (n *node) fillInSessionConfig(cfg *session.Config) {
	if cfg.ChainURL == "" {
		cfg.ChainURL = n.cfg.ChainAddr
	}
	if cfg.Asset == "" {
		cfg.Asset = n.cfg.AssetAddr
	}
	if cfg.Adjudicator == "" {
		cfg.Adjudicator = n.cfg.AdjudicatorAddr
	}

	cfg.ChainConnTimeout = n.cfg.ChainConnTimeout
	cfg.OnChainTxTimeout = n.cfg.OnChainTxTimeout
	cfg.ResponseTimeout = n.cfg.ResponseTimeout
}
