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

	"github.com/abiosoft/ishell"

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

var (
	sessionCmd = &ishell.Cmd{
		Name: "session",
		Help: "Use this command to open and close sessions. Usage: session [sub-command]",
		Func: sessionFn,
	}

	sessionOpenCmd = &ishell.Cmd{
		Name: "open",
		Help: "Open a new session. Usage: session open [session config file]",
		Func: sessionOpenFn,
	}
	sessionCloseCmd = &ishell.Cmd{
		Name: "close",
		Help: "Close the current session. Usage: session close",
		Func: sessionCloseFn,
	}
)

func init() {
	sessionCmd.AddCmd(sessionOpenCmd)
	sessionCmd.AddCmd(sessionCloseCmd)
}

func sessionFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command"))
		return
	}
	c.Println(c.Cmd.HelpText())
}

func sessionOpenFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command"))
		return
	}
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
	sessionID = msg.MsgSuccess.SessionID
	c.Printf("%s\n\n", greenf("Session opened. ID: %s. %s", sessionID))

	// Automatically subscribe to channel opening request notifications in this session.
	channelSub(c)
}

func sessionCloseFn(c *ishell.Context) {
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

	channelUnsub(c) // Close the channel opening request subscriptions before closing the session.

	req := pb.CloseSessionReq{
		SessionID: sessionID,
		Force:     false,
	}
	resp, err := client.CloseSession(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.CloseSessionResp_Error)
	if ok {
		channelSub(c) // If there is an error in session close, re-subscribe to channel opening request notifications.

		c.Printf("%s\n\n", redf("Error closing session : %v", msgErr.Error.Error))
		return
	}
	_, ok = resp.Response.(*pb.CloseSessionResp_MsgSuccess_)
	c.Printf("%s\n\n", greenf("Session closed. ID: %s."))
}
