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
	"context"
	"time"

	"github.com/abiosoft/ishell"

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

var (
	nodeCmd = &ishell.Cmd{
		Name: "node",
		Help: "Node command. Usage: node [command]",
		Func: node,
	}

	nodeTimeCmd = &ishell.Cmd{
		Name: "time",
		Help: "Print node time. Usage: node time",
		Func: nodeTime,
	}
	nodeConfigCmd = &ishell.Cmd{
		Name: "config",
		Help: "Print node config. Usage: node config",
		Func: nodeConfig,
	}
)

func init() {
	nodeCmd.AddCmd(nodeTimeCmd)
	nodeCmd.AddCmd(nodeConfigCmd)
}

func node(c *ishell.Context) {
	c.Println(c.Cmd.HelpText())
}

func nodeTime(c *ishell.Context) {
	noArgsReq := 0
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	timeReq := pb.TimeReq{}
	timeResp, err := client.Time(context.Background(), &timeReq)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	c.Printf("%s\n\n", greenf("Perun node time: %s", time.Unix(timeResp.Time, 0)))
}

func nodeConfig(c *ishell.Context) {
	noArgsReq := 0
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	getConfigReq := pb.GetConfigReq{}
	getConfigResp, err := client.GetConfig(context.Background(), &getConfigReq)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	c.Printf("%s\n\n", greenf("Perun node config:\n%v", prettify(getConfigResp)))
}
