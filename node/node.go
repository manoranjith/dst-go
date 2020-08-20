package node

import (
	"time"

	"github.com/pkg/errors"
	psync "perun.network/go-perun/pkg/sync"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/log"
	"github.com/hyperledger-labs/perun-node/session"
)

type node struct {
	log.Logger
	cfg      perun.NodeConfig
	sessions map[string]perun.SessionAPI
	psync.Mutex
}

// New returns a perun NodeAPI instance initialized using the given config.
// This should be called only once, subsequent calls after the first non error
// response will return an error.
func New(cfg perun.NodeConfig) (*node, error) {
	// To validate the contracts, credentials are required for connecting to the
	// blockchain, which only a session has.
	// For now, just check if the addresses are valid.
	wb := ethereum.NewWalletBackend()
	_, err := wb.ParseAddr(cfg.AdjudicatorAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator address")
	}
	_, err = wb.ParseAddr(cfg.AssetAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator address")
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

// OpenSession opens a new session for the given configuration.
// The following parameters are optional. If missing the default values of the node will be used.
// chainURL, asset & adjudicator addresses.
//
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
	n.Logger.Debugf("Starting node with this configuration - %+v", sessionCfg)
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
