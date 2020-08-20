package node_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/node"
)

var (
	testdataDir     = "../testdata/node"
	validConfigFile = "valid.yaml"
)

func Test_ParseConfig(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		_, err := node.ParseConfig(filepath.Join(testdataDir, validConfigFile))
		require.NoError(t, err)
	})
	t.Run("err_missingFile", func(t *testing.T) {
		_, err := node.ParseConfig("missing_file")
		require.Error(t, err)
	})
}

var validConfig = perun.NodeConfig{
	LogFile:         "",
	LogLevel:        "debug",
	ChainAddr:       "ws://127.0.0.1:8545",
	AdjudicatorAddr: "0x9daEdAcb21dce86Af8604Ba1A1D7F9BFE55ddd63",
	AssetAddr:       "0x5992089d61cE79B6CF90506F70DD42B8E42FB21d",
	CommTypes:       []string{"tcp"},
	ContactTypes:    []string{"yaml"},
	Currencies:      []string{"ETH"},

	ChainConnTimeout: 30 * time.Second,
	OnChainTxTimeout: 10 * time.Second,
	ResponseTimeout:  10 * time.Second,
}

func Test_New(t *testing.T) {
	t.Run("err_invalid_log_level", func(t *testing.T) {
		cfg := validConfig
		cfg.LogLevel = ""
		_, err := node.New(cfg)
		require.Error(t, err)
	})

	t.Run("err_invalid_adjudicator", func(t *testing.T) {
		cfg := validConfig
		cfg.AdjudicatorAddr = "invalid-addr"
		_, err := node.New(cfg)
		require.Error(t, err)
	})

	t.Run("err_invalid_asset", func(t *testing.T) {
		cfg := validConfig
		cfg.AssetAddr = "invalid-addr"
		_, err := node.New(cfg)
		require.Error(t, err)
	})

	var n perun.NodeAPI
	var err error
	t.Run("happy", func(t *testing.T) {
		n, err = node.New(validConfig)
		require.NoError(t, err)
		require.NotNil(t, n)
	})

	t.Run("happy_Time", func(t *testing.T) {
		assert.GreaterOrEqual(t, time.Now().UTC().Unix()+5, n.Time())
	})

	t.Run("happy_GetConfig", func(t *testing.T) {
		cfg := n.GetConfig()
		assert.Equal(t, validConfig, cfg)
	})

	t.Run("happy_Help", func(t *testing.T) {
		apis := n.Help()
		assert.Equal(t, []string{"payment"}, apis)
	})

	t.Run("happy_OpenSession_withChainData", func(t *testing.T) {
		sessionID, err := n.OpenSession("../testdata/session/session_with_chain.yaml")
		require.NoError(t, err)
		assert.NotZero(t, sessionID)
	})

	t.Run("happy_OpenSession_withoutChainData", func(t *testing.T) {
		sessionID, err := n.OpenSession("../testdata/session/session_without_chain.yaml")
		require.NoError(t, err)
		assert.NotZero(t, sessionID)
	})
	t.Run("Err_OpenSession_invalidFile", func(t *testing.T) {
		_, err := n.OpenSession("../testdata/session/invalid_format.yaml")
		require.Error(t, err)
	})
	t.Run("Err_OpenSession_unsupported_commtype", func(t *testing.T) {
		_, err := n.OpenSession("../testdata/session/session_unsupported_user_comm.yaml")
		require.Error(t, err)
	})
}
