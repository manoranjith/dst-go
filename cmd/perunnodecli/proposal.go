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

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
	"github.com/hyperledger-labs/perun-node/currency"
)

var (
	propCmd = &ishell.Cmd{
		Name: "prop",
		Help: "Channel proposal command",
		Func: prop,
	}
	propSendCmd = &ishell.Cmd{
		Name: "send",
		Help: "Send a channel proposal. Usage: prop send [sess alias] [peer alias] [own amount] [peer amount]",
		Func: propSend,
	}
	propSubCmd = &ishell.Cmd{
		Name: "sub",
		Help: "Subscribe to channel proposals. Usage: prop sub [sess alias]",
		Func: propSub,
	}
	propUnsubCmd = &ishell.Cmd{
		Name: "unsub",
		Help: "Unsubscribe from channel proposals. Usage: prop unsub [sess alias]",
		Func: propUnsub,
	}
	propAcceptCmd = &ishell.Cmd{
		Name: "accept",
		Help: "Accept channel proposals. Usage: prop accept [sess alias] [proposal alias]",
		Func: propAccept,
	}
	propRejectCmd = &ishell.Cmd{
		Name: "reject",
		Help: "Reject channel proposals. Usage: prop reject [sess alias] [proposal alias]",
		Func: propReject,
	}
	// propPendingCmd = &ishell.Cmd{
	// 	Name: "pending",
	// 	Help: "List channel proposals pending a response. Usage: prop list [sess alias]",
	// 	Func: propPending,
	// }
	chCounter = 0 // counter to track the number of channel opened to assign alias numbers.

	chMap    map[string]string // Map of channel alias to channel id.
	revChMap map[string]string // Map of channel id to channel alias.

	propCounter = 0 // counter to track the number of proposal opened to assign alias numbers.

	propMap    map[string]string // Map of proposal alias to proposal id.
	revPropMap map[string]string // Map of proposal id to proposal alias.
)

func init() {
	propCmd.AddCmd(propSendCmd)
	propCmd.AddCmd(propSubCmd)
	propCmd.AddCmd(propUnsubCmd)
	propCmd.AddCmd(propAcceptCmd)
	propCmd.AddCmd(propRejectCmd)
	// propCmd.AddCmd(propPendingCmd)

	propMap = make(map[string]string)
	revPropMap = make(map[string]string)

	chMap = make(map[string]string)
	revChMap = make(map[string]string)
}

// creates an alias for the channel id, adds it to the local map and returns the alias.
func addChID(id string) (alias string) {
	chCounter = chCounter + 1
	alias = fmt.Sprintf("c%d", chCounter)
	chMap[alias] = id
	revChMap[id] = alias
	return alias
}

// creates an alias for the proposal id, adds it to the local map and returns the alias.
func addPropID(id string) (alias string) {
	propCounter = propCounter + 1
	alias = fmt.Sprintf("p%d", propCounter)
	propMap[alias] = id
	revPropMap[id] = alias
	return alias
}

func prop(c *ishell.Context) {
	c.Println(c.Cmd.HelpText())
}

func propSend(c *ishell.Context) {
	// [sess alias] [peer alias] [own amount] [peer amount]
	noArgsReq := 4
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	sessID, ok := sessMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known session aliases:\n%v\n\n", prettify(sessMap)))
		return
	}

	req := pb.OpenPayChReq{
		SessionID: sessID,
		OpeningBalInfo: &pb.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{perun.OwnAlias, c.Args[1]},
			Bal:      []string{c.Args[2], c.Args[3]},
		},
		ChallengeDurSecs: 10,
	}
	resp, err := client.OpenPayCh(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.OpenPayChResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error opening channel : %v", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.OpenPayChResp_MsgSuccess_)
	chAlias := addChID(msg.MsgSuccess.OpenedPayChInfo.ChID)
	c.Printf("%s\n\n", greenf("Channel opened. Alias: %s.\nInitial Info:\n%s", chAlias,
		prettify(msg.MsgSuccess.OpenedPayChInfo)))
}

func propSub(c *ishell.Context) {
	// [sess alias]
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

	req := pb.SubPayChProposalsReq{
		SessionID: sessID,
	}
	sub, err := client.SubPayChProposals(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
	}
	go propNotifHandler(sub)
	c.Printf("%s\n\n", greenf("Subscribed to channel proposals in session %s (ID: %s)", c.Args[0], sessID))
}

func propNotifHandler(sub pb.Payment_API_SubPayChProposalsClient) {
	for {
		notifMsg, err := sub.Recv()
		if err != nil {
			sh.Lock()
			sh.Printf("%s\n\n", redf("Error receiving channel proposal notification: %v", err))
			sh.Unlock()
			return
		}
		msgErr, ok := notifMsg.Response.(*pb.SubPayChProposalsResp_Error)
		if ok {
			sh.Lock()
			sh.Printf("%s\n\n", redf("Error message received in proposal notification : %v", msgErr.Error.Error))
			sh.Unlock()
			return
		}
		notif, ok := notifMsg.Response.(*pb.SubPayChProposalsResp_Notify_)
		propAlias := addPropID(notif.Notify.ProposalID)
		sh.Lock()
		sh.Printf("%s\n\n", greenf("Channel proposal received. Alias: %s.\nInitial Info:\n%s", propAlias,
			prettify(notif.Notify)))
		sh.Unlock()
	}
}

func propUnsub(c *ishell.Context) {
	// [sess alias]
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	sessID, ok := sessMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known session aliases:\n%v\n\n", prettify(sessMap)))
		return
	}

	req := pb.UnsubPayChProposalsReq{
		SessionID: sessID,
	}
	resp, err := client.UnsubPayChProposals(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.UnsubPayChProposalsResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error unsubscribing from channel proposals : %v", msgErr.Error.Error))
		return
	}
	c.Printf("%s\n\n", greenf("Unsubscribed from channel proposals in session %s (ID: %s)", c.Args[0], sessID))
}

func propAccept(c *ishell.Context) {
	// [sess alias] [prop alias]
	noArgsReq := 2
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	sessID, ok := sessMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known session aliases:\n%v\n\n", prettify(sessMap)))
		return
	}
	propID, ok := propMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown proposal alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known proposal aliases:\n%v\n\n", prettify(propMap)))
		return
	}

	req := pb.RespondPayChProposalReq{
		SessionID:  sessID,
		ProposalID: propID,
		Accept:     true,
	}
	resp, err := client.RespondPayChProposal(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.RespondPayChProposalResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error unsubscribing from channel proposals : %v", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.RespondPayChProposalResp_MsgSuccess_)
	chAlias := addChID(msg.MsgSuccess.OpenedPayChInfo.ChID)
	c.Printf("%s\n\n", greenf("Channel opened. Alias: %s.\nInitial Info:\n%s", chAlias,
		prettify(msg.MsgSuccess.OpenedPayChInfo)))
	delete(propMap, propID)
}

func propReject(c *ishell.Context) {
	// [sess alias] [prop alias]
	noArgsReq := 2
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	sessID, ok := sessMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known session aliases:\n%v\n\n", prettify(sessMap)))
		return
	}
	propID, ok := propMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown proposal alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known proposal aliases:\n%v\n\n", prettify(propMap)))
		return
	}

	req := pb.RespondPayChProposalReq{
		SessionID:  sessID,
		ProposalID: propID,
		Accept:     false,
	}
	resp, err := client.RespondPayChProposal(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.RespondPayChProposalResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error unsubscribing from channel proposals : %v", msgErr.Error.Error))
		return
	}
	_, ok = resp.Response.(*pb.RespondPayChProposalResp_MsgSuccess_)
	c.Printf("%s\n\n", greenf("Channel proposal rejected successfully."))
	delete(propMap, propID)
}

func propPending(c *ishell.Context) {
	// [sess alias]
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	_, ok := sessMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known session aliases:\n%v\n\n", prettify(sessMap)))
		return
	}
	c.Printf("%s\n\n", greenf("Pending proposals:\n%v\n\n", prettify(propMap)))
	return

}
