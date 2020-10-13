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
	"fmt"

	"github.com/abiosoft/ishell"

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

var (
	sessCmd = &ishell.Cmd{
		Name: "sess",
		Help: "Session Command. Usage: sess [command]",
		Func: sess,
	}

	sessOpenCmd = &ishell.Cmd{
		Name: "open",
		Help: "Open a new session. Usage: sess open [session config file].",
		Func: sessOpen,
	}
	sessCloseCmd = &ishell.Cmd{
		Name: "close",
		Help: "Close the current session",
		Func: sessClose,
	}
	sessCounter = 0 // counter to track the number of sessions opened to assign alias numbers.

	sessMap    map[string]string // Map of session alias to session id.
	revSessMap map[string]string // Map of session id to session alias.
)

func init() {
	sessCmd.AddCmd(sessOpenCmd)
	sessCmd.AddCmd(sessCloseCmd)

	sessMap = make(map[string]string)
	revSessMap = make(map[string]string)
}

// creates an alias for the session id, adds it to the local map and returns the alias.
func addSessID(id string) (alias string) {
	sessCounter = sessCounter + 1
	alias = fmt.Sprintf("s%d", sessCounter)
	sessMap[alias] = id
	revSessMap[id] = alias
	return alias
}

func sess(c *ishell.Context) {
	c.Println(c.Cmd.HelpText())
}

func sessOpen(c *ishell.Context) {
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	req := pb.OpenSessionReq{
		ConfigFile: c.Args[0],
	}
	resp, err := client.OpenSession(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.OpenSessionResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error opening session : %v", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.OpenSessionResp_MsgSuccess_)
	sessAlias := addSessID(msg.MsgSuccess.SessionID)
	c.Printf("%s\n\n", greenf("Session opened. ID: %s. Alias: %s", msg.MsgSuccess.SessionID, sessAlias))
}

func sessClose(c *ishell.Context) {
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	sessID, ok := sessMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known session aliases:\n%v", prettify(sessMap)))
		return
	}

	req := pb.CloseSessionReq{
		SessionID: sessID,
		Force:     false,
	}
	resp, err := client.CloseSession(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.CloseSessionResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error closing session : %v", msgErr.Error.Error))
		return
	}
	_, ok = resp.Response.(*pb.CloseSessionResp_MsgSuccess_)
	c.Printf("%s\n\n", greenf("Session closed. ID: %s. Alias: %s", sessID, c.Args[0]))
}
