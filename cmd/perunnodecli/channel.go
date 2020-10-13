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
	chCmd = &ishell.Cmd{
		Name: "ch",
		Help: "Channel command",
		Func: ch,
	}
	chSendCmd = &ishell.Cmd{
		Name: "send",
		Help: "Send a channel update",
		Func: chSend,
	}
	chSubCmd = &ishell.Cmd{
		Name: "sub",
		Help: "Subscribe to channel updates",
		Func: chSub,
	}
	chUnsubCmd = &ishell.Cmd{
		Name: "unsub",
		Help: "Unsubscribe from channel updates",
		Func: chUnsub,
	}
	chAcceptCmd = &ishell.Cmd{
		Name: "accept",
		Help: "Accept a channel update",
		Func: chAccept,
	}
	chRejectCmd = &ishell.Cmd{
		Name: "reject",
		Help: "Reject a channel update",
		Func: chReject,
	}
	// chPendingCmd = &ishell.Cmd{
	// 	Name: "pending",
	// 	Help: "List channel update notifications pending a response",
	// 	Func: defaultHandler,
	// }
	chCloseCmd = &ishell.Cmd{
		Name: "close",
		Help: "Close channel",
		Func: chClose,
	}

	updateCounter = 0 // counter to track the number of update opened to assign alias numbers.

	updateMap    map[string]string // Map of update alias to update id.
	revUpdateMap map[string]string // Map of update id to update alias.
)

func init() {
	chCmd.AddCmd(chSendCmd)
	chCmd.AddCmd(chSubCmd)
	chCmd.AddCmd(chUnsubCmd)
	chCmd.AddCmd(chAcceptCmd)
	chCmd.AddCmd(chRejectCmd)
	// chCmd.AddCmd(chPendingCmd)
	chCmd.AddCmd(chCloseCmd)

	updateMap = make(map[string]string)
	revUpdateMap = make(map[string]string)
}

// creates an alias for the channel id, adds it to the local map and returns the alias.
func addUpdateID(id string) (alias string) {
	updateCounter = updateCounter + 1
	alias = fmt.Sprintf("u%d", updateCounter)
	updateMap[alias] = id
	revUpdateMap[id] = alias
	return alias
}

func ch(c *ishell.Context) {
	c.Println(c.Cmd.HelpText())
}

func chSend(c *ishell.Context) {
	// [sess alias] [ch alias] [peer alias] [amount]
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
	chID, ok := chMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known channel aliases:\n%v\n\n", prettify(chMap)))
		return
	}

	req := pb.SendPayChUpdateReq{
		SessionID: sessID,
		ChID:      chID,
		Payee:     c.Args[2],
		Amount:    c.Args[3],
	}
	resp, err := client.SendPayChUpdate(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.SendPayChUpdateResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error sending channel update: %v", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.SendPayChUpdateResp_MsgSuccess_)
	chAlias := revChMap[msg.MsgSuccess.UpdatedPayChInfo.ChID]
	c.Printf("%s\n\n", greenf("Channel updated. Alias: %s.\nUpdated Info:\n%s", chAlias,
		prettify(msg.MsgSuccess.UpdatedPayChInfo)))
}

func chSub(c *ishell.Context) {
	// [sess alias] [ch alias]
	noArgsReq := 2
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
	chID, ok := chMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known channel aliases:\n%v\n\n", prettify(chMap)))
		return
	}

	req := pb.SubpayChUpdatesReq{
		SessionID: sessID,
		ChID:      chID,
	}
	sub, err := client.SubPayChUpdates(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
	}
	go chNotifHandler(sub)
	c.Printf("%s\n\n", greenf("Subscribed to channel updates for channel %s (ID: %s) in session %s (ID: %s)",
		c.Args[1], chID, c.Args[0], sessID))
}

func chNotifHandler(sub pb.Payment_API_SubPayChUpdatesClient) {
	for {
		notifMsg, err := sub.Recv()
		if err != nil {
			sh.Lock()
			sh.Printf("%s\n\n", redf("Error receiving channel update notification: %v", err))
			sh.Unlock()
			return
		}
		msgErr, ok := notifMsg.Response.(*pb.SubPayChUpdatesResp_Error)
		if ok {
			sh.Lock()
			sh.Printf("%s\n\n", redf("Error message received in update notification : %v", msgErr.Error.Error))
			sh.Unlock()
			return
		}
		notif, ok := notifMsg.Response.(*pb.SubPayChUpdatesResp_Notify_)
		updateAlias := addUpdateID(notif.Notify.UpdateID)
		sh.Lock()
		sh.Printf("%s\n\n", greenf("Channel update received. Alias: %s.\nProposed Info:\n%s", updateAlias,
			prettify(notif.Notify)))
		sh.Unlock()
	}
}

func chUnsub(c *ishell.Context) {
	// [sess alias] [ch alias]
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
	chID, ok := chMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known channel aliases:\n%v\n\n", prettify(chMap)))
		return
	}

	req := pb.UnsubPayChUpdatesReq{
		SessionID: sessID,
		ChID:      chID,
	}
	resp, err := client.UnsubPayChUpdates(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.UnsubPayChUpdatesResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error unsubscribing from channel updates : %v", msgErr.Error.Error))
		return
	}
	c.Printf("%s\n\n", greenf("Unsubscribed from channel updates for channel %s [ID: %s]  in session %s (ID: %s)",
		c.Args[1], chID, c.Args[0], sessID))
}

func chAccept(c *ishell.Context) {
	// [sess alias] [channel alias] [update alias]
	noArgsReq := 3
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
	chID, ok := chMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known channel aliases:\n%v\n\n", prettify(chMap)))
		return
	}
	updateID, ok := updateMap[c.Args[2]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown update alias %s", c.Args[2]))
		c.Printf("%s\n\n", redf("Known proposal aliases:\n%v\n\n", prettify(propMap)))
		return
	}
	req := pb.RespondPayChUpdateReq{
		SessionID: sessID,
		ChID:      chID,
		UpdateID:  updateID,
		Accept:    true,
	}
	resp, err := client.RespondPayChUpdate(context.Background(), &req)
	if err != nil {
		sh.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.RespondPayChUpdateResp_Error)
	if ok {
		sh.Printf("%s\n\n", redf("Error accepting channel update: %v", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.RespondPayChUpdateResp_MsgSuccess_)
	chAlias := revChMap[chID]
	sh.Printf("%s\n\n", greenf("Channel updated. Alias: %s.\nUpdated Info:\n%s", chAlias,
		prettify(msg.MsgSuccess.UpdatedPayChInfo)))
}

func chReject(c *ishell.Context) {
	// [sess alias] [channel alias] [update alias]
	noArgsReq := 3
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
	chID, ok := chMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known channel aliases:\n%v\n\n", prettify(chMap)))
		return
	}
	updateID, ok := updateMap[c.Args[2]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown update alias %s", c.Args[2]))
		c.Printf("%s\n\n", redf("Known proposal aliases:\n%v\n\n", prettify(propMap)))
		return
	}
	req := pb.RespondPayChUpdateReq{
		SessionID: sessID,
		ChID:      chID,
		UpdateID:  updateID,
		Accept:    false,
	}
	resp, err := client.RespondPayChUpdate(context.Background(), &req)
	if err != nil {
		sh.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.RespondPayChUpdateResp_Error)
	if ok {
		sh.Printf("%s\n\n", redf("Error rejecting channel update: %v", msgErr.Error.Error))
		return
	}
	_, ok = resp.Response.(*pb.RespondPayChUpdateResp_MsgSuccess_)
	sh.Printf("%s\n\n", greenf("Channel update rejected successfully."))
}

func chClose(c *ishell.Context) {
	// [sess alias] [channel alias]
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
	chID, ok := chMap[c.Args[1]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel alias %s", c.Args[1]))
		c.Printf("%s\n\n", redf("Known channel aliases:\n%v\n\n", prettify(chMap)))
		return
	}
	req := pb.ClosePayChReq{
		SessionID: sessID,
		ChID:      chID,
	}
	resp, err := client.ClosePayCh(context.Background(), &req)
	if err != nil {
		sh.Printf("%s\n\n", redf("Error sending command to perun node: %v", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.ClosePayChResp_Error)
	if ok {
		sh.Printf("%s\n\n", redf("Error closing channel update: %v", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.ClosePayChResp_MsgSuccess_)
	sh.Printf("%s\n\n", greenf("Channel closed. Alias: %s.\nUpdated Info:\n%s", c.Args[1],
		prettify(msg.MsgSuccess.ClosedPayChInfo)))
}
