package node

import (
	"path/filepath"

	"github.com/hyperledger-labs/perun-node"
	"github.com/spf13/viper"
)

// Config represents the configuratio parameters for the node.

func ParseConfig(configFile string) (perun.NodeConfig, error) {
	v := viper.New()
	v.SetConfigFile(filepath.Clean(configFile))

	var cfg perun.NodeConfig
	err := v.ReadInConfig()
	if err != nil {
		return perun.NodeConfig{}, err
	}
	return cfg, v.Unmarshal(&cfg)
}
