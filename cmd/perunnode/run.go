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

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/hyperledger-labs/perun-node/api/grpc"
	"github.com/hyperledger-labs/perun-node/node"
)

const (
	// flag names for run command.
	configfileF       = "configfile"
	loglevelF         = "loglevel"
	logfileF          = "logfile"
	chainurlF         = "chainurl"
	adjudicatorF      = "adjudicator"
	assetF            = "asset"
	chainconntimeoutF = "chainconntimeout"
	onchaintxtimeoutF = "onchaintxtimeout"
	responsetimeoutF  = "responsetimeout"
	grpcPortF         = "grpcport"

	// default values for flags in run command.
	defaultConfigFile = "node.yaml"
	defaultGrpcPort   = 50001
)

var (
	// node level viper instance for parsing configuration from flags and configuration files.
	nodeViper *viper.Viper

	// flags in the run command is binded with the viper instance to override values from config file.
	flagsToBind = []string{
		logfileF,
		loglevelF,
		chainurlF,
		adjudicatorF,
		assetF,
		chainconntimeoutF,
		onchaintxtimeoutF,
		responsetimeoutF,
	}
)

func init() {
	nodeViper = viper.New()
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().String(configfileF, defaultConfigFile, "node config file")

	runCmd.Flags().String(loglevelF, "", "Log level. Supported levels: debug, info, error")
	runCmd.Flags().String(logfileF, "", "Log file path. Use empty string for stdout")
	runCmd.Flags().String(chainurlF, "", "URL of the blockchain node")
	runCmd.Flags().String(adjudicatorF, "", "Address as of the adjudicator contract as hex string with 0x prefix")
	runCmd.Flags().String(assetF, "", "Address as of the asset contract as hex string with 0x prefix")
	runCmd.Flags().Duration(chainconntimeoutF, time.Duration(0),
		"Connection timeout for connecting to the blockchain node")
	runCmd.Flags().Duration(onchaintxtimeoutF, time.Duration(0),
		"Max duration to wait for an on-chain transaction to be mined.")
	runCmd.Flags().Duration(responsetimeoutF, time.Duration(0),
		"Max duration to wait for a response in off-chain communication.")

	runCmd.Flags().Uint64(grpcPortF, defaultGrpcPort, "port at which grpc payment channel API server should listen")

	// Bind the configuration flags to viper instance used for to override the values defined in config file.
	for i := range flagsToBind {
		nodeViper.BindPFlag(flagsToBind[i], runCmd.Flags().Lookup(flagsToBind[i]))
	}
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the perunnode",
	Long: `
Start the perun node. Currently, the node serves the payment API via grpc
interface. Configuration can be specified in the config file or via flags.
If both config file and flags are given, values in flags are used.

Specify configFile with some config flags or all of the config flags.`,
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	nodeCfgFile, err := cmd.Flags().GetString(configfileF)
	if err != nil {
		panic("unknown flag configFile\n")
	}
	fmt.Printf("Using node config file - %s\n", nodeCfgFile)

	nodeCfg, err := node.ParseConfig(nodeViper, nodeCfgFile)
	if err != nil {
		fmt.Printf("Error reading node config file: %v\n", err)
		return
	}

	nodeAPI, err := node.New(nodeCfg)
	if err != nil {
		fmt.Printf("Error initializing nodeAPI: %v\n", err)
		return
	}

	grpcPort, err := cmd.Flags().GetUint64(grpcPortF)
	if err != nil {
		panic("unknown flag port\n")
	}
	grpcAddr := fmt.Sprintf(":%d", grpcPort)
	fmt.Printf("%s\n\n", prettify(nodeCfg))
	fmt.Printf("Started perun payment channel API server with the above config at %s\n", grpcAddr)
	if err := grpc.ListenAndServePayChAPI(nodeAPI, grpcAddr); err != nil {
		log.Printf("server returned with error: %v", err)
	}
}
