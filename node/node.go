package node

import (
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"perun.network/go-perun/pkg/sync"
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

	sync.Mutex
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
			LogLevel: logLevel,
			LogFile:  logFile,

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

// TODO: Change return type to map. Or is it easier to store a map and directly return it everytime
// ?
func (n *Node) GetConfig() Config {
	n.Logger.Debug("Received request: node.GetConfig")
	return n.cfg
}

func (n *Node) Help() []string {
	// TODO: Collect and return list of supported APIs.
	return []string{}
}

// OpenSession opens a new session based on the given configuration.
// Parameters to connect to the chain (chainURL, asset & adjudicator addresses) are optional.
// If missing default values from the node will be used.
//
// The node also initializes a logger for the generated session that logs along with its session id.
func (n *Node) OpenSession(configFile string) (ID string, _ error) {
	n.Logger.Debug("Received request: node.OpenSession")
	n.Lock()
	defer n.Unlock()

	sessionCfg, err := session.ParseConfig(configFile)
	if err != nil {
		return "", err
	}
	n.fillInSessionConfig(&sessionCfg)
	s, err := session.New(sessionCfg)
	if err != nil {
		return "", err
	}

	// TODO: (mano) Add func in log module to preserve log level when deriving logger with field.
	// Ignore error from ParseLevel as the log level was already used to init node logger without errors.
	level, _ := logrus.ParseLevel(n.cfg.LogFile)
	sessionLogger := n.Logger.WithField("session", s.ID)
	sessionLogger.Level = level
	s.Logger = sessionLogger

	n.Sessions[s.ID] = s
	return s.ID, nil
}

// fillInSessionConfig fills in the missing values in session configuration
// for those fields that have a default value in the node config.
func (n *Node) fillInSessionConfig(cfg *session.Config) {
	if cfg.ChainURL == "" {
		cfg.ChainURL = n.cfg.ChainAddr
	}
	if cfg.Asset == "" {
		cfg.Asset = n.cfg.AssetAddr
	}
	if cfg.Adjudicator == "" {
		cfg.Adjudicator = n.cfg.AdjudicatorAddr
	}
}
