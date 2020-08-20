package node

import (
	"time"

	"github.com/pkg/errors"
	psync "perun.network/go-perun/pkg/sync"
	pwallet "perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"
	"github.com/hyperledger-labs/perun-node/log"
	"github.com/hyperledger-labs/perun-node/session"
)

type Node struct {
	log.Logger

	Cfg Config

	Adjudicator, AssetHolder pwallet.Address
	Sessions                 map[string]perun.SessionAPI // Map of session ID to session instances.

	psync.Mutex
}

func New(cfg Config) (*Node, error) {
	err := log.InitLogger(cfg.LogLevel, cfg.LogFile)
	if err != nil {
		return nil, errors.WithMessage(err, "initializing logger for node")
	}

	// TODO: (mano) Currently, credentials are required to initialize a chain backend
	// for connecting to node and validating contracts. So store the config.
	wb := ethereum.NewWalletBackend()
	adjudicator, err := wb.ParseAddr(cfg.AdjudicatorAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}
	asset, err := wb.ParseAddr(cfg.AssetAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}

	return &Node{
		Logger: log.NewLoggerWithField("node", 1), // ID of the node is always 1.
		Cfg: Config{
			LogLevel: cfg.LogLevel,
			LogFile:  cfg.LogFile,

			ChainAddr:       cfg.ChainAddr,
			AdjudicatorAddr: cfg.AdjudicatorAddr,
			AssetAddr:       cfg.AssetAddr,
			CommTypes:       []string{"tcp"},
			ContactTypes:    []string{"yaml"},
			Currencies:      []string{"ETH"},
		},
		Adjudicator: adjudicator,
		AssetHolder: asset,
		Sessions:    make(map[string]perun.SessionAPI),
	}, nil
}

func (n *Node) Time() int64 {
	n.Logger.Debug("Received request: node.Time")
	return time.Now().UTC().Unix()
}

// TODO: Change return type to map. Or is it easier to store a map and directly return it everytime
// ?
func (n *Node) GetConfig() Config {
	n.Logger.Debug("Received request: node.GetConfig")
	return n.Cfg
}

func (n *Node) Help() []string {
	return []string{"payment"}
}

// OpenSession opens a new session based on the given configuration.
// Parameters to connect to the chain (chainURL, asset & adjudicator addresses) are optional.
// If missing default values from the node will be used.
//
// The node also initializes a logger for the generated session that logs along with its session id.
func (n *Node) OpenSession(configFile string) (ID string, _ error) {
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
	n.Sessions[s.ID()] = s
	return s.ID(), nil
}

// fillInSessionConfig fills in the missing values in session configuration
// for those fields that have a default value in the node config.
func (n *Node) fillInSessionConfig(cfg *session.Config) {
	if cfg.ChainURL == "" {
		cfg.ChainURL = n.Cfg.ChainAddr
	}
	if cfg.Asset == "" {
		cfg.Asset = n.Cfg.AssetAddr
	}
	if cfg.Adjudicator == "" {
		cfg.Adjudicator = n.Cfg.AdjudicatorAddr
	}

	cfg.ChainConnTimeout = n.Cfg.ChainConnTimeout
	cfg.OnChainTxTimeout = n.Cfg.OnChainTxTimeout
	cfg.ResponseTimeout = n.Cfg.ResponseTimeout
}
