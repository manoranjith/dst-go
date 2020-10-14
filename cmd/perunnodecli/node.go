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
	grpclib "google.golang.org/grpc"

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

var (
	nodeCmd = &ishell.Cmd{
		Name: "node",
		Help: "Node command. Usage: node [command]",
		Func: node,
	}

	nodeConnectCmd = &ishell.Cmd{
		Name: "connect",
		Help: "Connect to a running perun node instance. Usage: node connect [url]",
		Func: nodeConnect,
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
	nodeCmd.AddCmd(nodeConnectCmd)
	nodeCmd.AddCmd(nodeTimeCmd)
	nodeCmd.AddCmd(nodeConfigCmd)
}

func node(c *ishell.Context) {
	c.Println(c.Cmd.HelpText())
}

func nodeConnect(c *ishell.Context) {
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	nodeAddr := c.Args[0]
	conn, err := grpclib.Dial(nodeAddr, grpclib.WithInsecure())
	if err != nil {
		sh.Printf("Error connecting to perun node at %s: %v", nodeAddr, err)
	}
	client = pb.NewPayment_APIClient(conn)
	t, err := getNodeTime()
	if err != nil {
		c.Printf("%s\n\n", redf("Error connecting to perun node: %v", err))
		return
	}
	sh.Printf("Connected to perun node at %s. Node time is %v\n\n", nodeAddr, time.Unix(t, 0))
}

func nodeTime(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command"))
		return
	}
	noArgsReq := 0
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	t, err := getNodeTime()
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	c.Printf("%s\n\n", greenf("Perun node time: %s", time.Unix(t, 0)))
}

func getNodeTime() (int64, error) {
	timeReq := pb.TimeReq{}
	timeResp, err := client.Time(context.Background(), &timeReq)
	if err != nil {
		return 0, err
	}
	return timeResp.Time, err
}

func nodeConfig(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command"))
		return
	}
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
