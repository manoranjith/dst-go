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

	"github.com/spf13/cobra"

	"github.com/hyperledger-labs/perun-node/api/grpc"
	"github.com/hyperledger-labs/perun-node/node"
)

func init() {
	runCmd.Flags().String("configFile", "node.yaml", "node config file")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the perunnode",
	Long: `
Start the perun node. Currently, the node serves the payment API via grpc
interface. Configuration can be specified in the config file or via flags.
If both config file and flags are given, values in flags are used.`,
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	if !cmd.Flags().Changed("configFile") {
		fmt.Printf("Error required flags(s) not set: %v", "configFile")
		cmd.Usage() // nolint: errcheck, gosec	// This will not error.
		return
	}

	nodeCfgFile, err := cmd.Flags().GetString("configFile")
	if err != nil {
		fmt.Printf("App internal error: unknonw flag configFile\n")
		return
	}
	fmt.Printf("Using node config file - %s\n", nodeCfgFile)

	nodeCfg, err := node.ParseConfig(nodeCfgFile)
	if err != nil {
		fmt.Printf("Error reading node config file: %v\n", err)
		return
	}

	nodeAPI, err := node.New(nodeCfg)
	if err != nil {
		fmt.Printf("Error initializing nodeAPI: %v\n", err)
		return
	}

	grpcPort := ":50001"
	fmt.Printf("Started ListenAndServePayChAPI\n")
	if err := grpc.ListenAndServePayChAPI(nodeAPI, grpcPort); err != nil {
		log.Printf("server returned with error: %v", err)
	}
}
