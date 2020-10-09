// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/hyperledger-labs/perun-node
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package node

import (
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/hyperledger-labs/perun-node"
)

// ParseConfig parses the node configuration from a file using the given viper instance.
// Any overrides set in the viper instance (such as binding flags from a command) will be
// applied as per the precedence order defined in viper.
func ParseConfig(v *viper.Viper, configFile string) (perun.NodeConfig, error) {
	v.SetConfigFile(filepath.Clean(configFile))

	var cfg perun.NodeConfig
	err := v.ReadInConfig()
	if err != nil {
		return perun.NodeConfig{}, err
	}
	return cfg, v.Unmarshal(&cfg)
}
