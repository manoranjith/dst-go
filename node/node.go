package node

import (
	"time"

	"github.com/pkg/errors"
	"perun.network/go-perun/apps/payment"
	"perun.network/go-perun/pkg/sync"
	"perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
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
	err := log.InitLogger(logLevel, logFile)
	if err != nil {
		return nil, errors.WithMessage(err, "initializing logger for node")
	}

	// TODO: (mano) Currently, credentials are required to initialize a chain backend
	// for connecting to node and validating contracts. So store the config.
	wb := ethereum.NewWalletBackend()
	adjudicator, err := wb.ParseAddr(adjudicatorAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}
	asset, err := wb.ParseAddr(assetAddr)
	if err != nil {
		return nil, errors.WithMessage(err, "default adjudicator addres")
	}

	emptyAddr, err := wb.ParseAddr("0x0")
	if err != nil {
		return nil, errors.WithMessage(err, "parsing empty address for app def")
	}
	payment.SetAppDef(emptyAddr) // dummy app def.

	return &Node{
		Logger: log.NewLoggerWithField("node", 1), // ID of the node is always 1.
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
		n.Logger.Error(err)
		return "", perun.ErrInvalidConfig
	}
	n.fillInSessionConfig(&sessionCfg)
	s, err := session.New(sessionCfg)
	if err != nil {
		return "", err
	}
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
