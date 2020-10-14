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

type (
	// openChannelInfo stores the minimal information required by this app for processing
	// payment commands on an open channel.
	openChannelInfo struct {
		id        string
		peerAlias string
	}
)

var (
	channelCmd = &ishell.Cmd{
		Name: "channel",
		Help: "Use this command to open/close payment channels. Usage: channel [sub-command]",
		Func: channelFn,
	}
	channelSendCmd = &ishell.Cmd{
		Name: "send-opening-request",
		// nolint: lll
		Help: "Send a request to open a channel with the peer. Usage: channel send-opening-request [peer alias] [own amount] [peer amount]",
		Func: channelSendFn,
	}
	channelSubCmd = &ishell.Cmd{
		Name: "subscribe-opening-request",
		Help: "Subscribe to channel opening request notications from the peer. Usage: channel subscribe-opening-request",
		Func: channelSubFn,
	}
	channelUnsubCmd = &ishell.Cmd{
		Name: "unsubcribe-opening-request",
		Help: "Unsubscribe from channel opening request notications from the peer. Usage: channel unsubcribe-opening-request",
		Func: channelUnsubFn,
	}
	channelAcceptCmd = &ishell.Cmd{
		Name: "accept-opening-request",
		Help: "Accept channel opening request from the peer. Usage: channel accept-opening-request [channel notification alias]",
		Completer: func([]string) []string {
			return channelNotifList
		},
		Func: channelAcceptFn,
	}
	channelRejectCmd = &ishell.Cmd{
		Name: "reject-opening-request",
		Help: "Reject channel opening request from the peer. Usage: channel reject-opening-request [channel notification alias]",
		Completer: func([]string) []string {
			return channelNotifList
		},
		Func: channelRejectFn,
	}
	// channelListPending = &ishell.Cmd{
	// 	Name: "list-notifications-pending-response",
	// 	Help: "List channel opening requests notifications pending a response. Usage: channel list-notifications-pending-response",
	// 	Func: channelPending,
	// }
	// channelListAll = &ishell.Cmd{
	// 	Name: "list-open-channels",
	// 	Help: "List channels that are open for off-chain payments. Usage: channel list-open-channels",
	// 	Func: channelPending,
	// }
	openChannelsCounter = 0 // counter to track the number of channel opened to assign alias numbers.

	openChannelsMap    map[string]openChannelInfo // Map of open channel alias to open channel info.
	openChannelsRevMap map[string]string          // Map of open channel id to open channel alias.

	channelNotifCounter = 0 // counter to track the number of proposal opened to assign alias numbers.

	// List of channel notification ids for autocompletion.
	channelNotifList []string
	// Map of channel notification id to the notification payload.
	channelNotifMap map[string]*pb.SubPayChProposalsResp_Notify
)

func init() {
	channelCmd.AddCmd(channelSendCmd)
	channelCmd.AddCmd(channelSubCmd)
	channelCmd.AddCmd(channelUnsubCmd)
	channelCmd.AddCmd(channelAcceptCmd)
	channelCmd.AddCmd(channelRejectCmd)
	// propCmd.AddCmd(propPendingCmd)

	openChannelsMap = make(map[string]openChannelInfo)
	openChannelsRevMap = make(map[string]string)

	channelNotifMap = make(map[string]*pb.SubPayChProposalsResp_Notify)
}

// creates an alias for the channel id, adds the channel to the open channels map and returns the alias.
func addToOpenChannelMap(id, peer string) (alias string) {
	openChannelsCounter = openChannelsCounter + 1
	alias = fmt.Sprintf("ch_%s_%d", peer, openChannelsCounter)
	openChannelsMap[alias] = openChannelInfo{id, peer}
	openChannelsRevMap[id] = alias
	return alias
}

// creates an alias for the channel opening request notification proposal, adds it to the channel notifications
// map and returns the alias.
// It also adds the entry to channel notification list for autocompletetion.
func addToChannelNotifMap(notif *pb.SubPayChProposalsResp_Notify) (alias string) {
	channelNotifCounter = channelNotifCounter + 1
	alias = fmt.Sprintf("ch_N_%d", channelNotifCounter)
	channelNotifMap[alias] = notif

	channelNotifList = append(channelNotifList, alias)
	return alias
}

func removeFromChannelNotifMap(alias string) {
	delete(channelNotifMap, alias)
	aliasIdx := 0
	for idx := range channelNotifList {
		if channelNotifList[idx] == alias {
			aliasIdx = idx
			break
		}
	}
	channelNotifList[aliasIdx] = channelNotifList[len(channelNotifList)-1]
	channelNotifList[len(channelNotifList)-1] = ""
	channelNotifList = channelNotifList[:len(channelNotifList)-1]
}

func channelFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command."))
		return
	}
	c.Println(c.Cmd.HelpText())
}

func channelSendFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command."))
		return
	}

	// Usage: channel send [peer alias] [own amount] [peer amount]",
	noArgsReq := 3
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	req := pb.OpenPayChReq{
		SessionID: sessionID,
		OpeningBalInfo: &pb.BalInfo{
			Currency: currency.ETH,
			Parts:    []string{perun.OwnAlias, c.Args[0]},
			Bal:      []string{c.Args[1], c.Args[2]},
		},
		ChallengeDurSecs: challengeDurSecs,
	}
	resp, err := client.OpenPayCh(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v.", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.OpenPayChResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error opening channel : %v.", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.OpenPayChResp_MsgSuccess_)
	chAlias := addToOpenChannelMap(msg.MsgSuccess.OpenedPayChInfo.ChID, findPeerAlias(msg.MsgSuccess.OpenedPayChInfo))
	c.Printf("%s\n\n", greenf("Channel opened. Alias: %s.\nChannel Info:\n%s.", chAlias,
		prettifyPayChInfo(msg.MsgSuccess.OpenedPayChInfo)))
}

func channelSubFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command."))
		return
	}
	// Usage: channel sub
	noArgsReq := 0
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}
	channelSub(c)
}

func channelSub(c *ishell.Context) {
	req := pb.SubPayChProposalsReq{
		SessionID: sessionID,
	}
	sub, err := client.SubPayChProposals(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v.", err))
		return
	}
	go channelNotifHandler(sub)
	c.Printf("%s\n\n", greenf("Subscribed to channel opening request notifications in current session."))
}

func channelNotifHandler(sub pb.Payment_API_SubPayChProposalsClient) {
	for {
		notifMsg, err := sub.Recv()
		if err != nil {
			sh.Lock()
			sh.Printf("%s\n\n", redf("Error receiving channel channel opening request notification: %v.", err))
			sh.Unlock()
			return
		}
		msgErr, ok := notifMsg.Response.(*pb.SubPayChProposalsResp_Error)
		if ok {
			sh.Lock()
			sh.Printf("%s\n\n", redf("Error received in channel opening request notification: %v.", msgErr.Error.Error))
			sh.Unlock()
			return
		}
		notif, ok := notifMsg.Response.(*pb.SubPayChProposalsResp_Notify_)
		channelNotifAlias := addToChannelNotifMap(notif.Notify)
		nodeTime, _ := getNodeTime()
		sh.Lock()
		sh.Printf("%s\n\n", greenf("Channel opening request notification received. Notification Alias: %s.\n%s.\nExpires in %ds.",
			channelNotifAlias, prettifyChannelOpeningRequest(notif.Notify), notif.Notify.Expiry-nodeTime))
		sh.Unlock()
	}
}

func prettifyChannelOpeningRequest(data *pb.SubPayChProposalsResp_Notify) string {
	return fmt.Sprintf("ID: %s, Currency: %s, Balance %v",
		data.ProposalID, data.OpeningBalInfo.Currency, data.OpeningBalInfo.Bal)
}

func channelUnsubFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command."))
		return
	}
	// Usage: channel unsub
	noArgsReq := 0
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	channelUnsub(c)
}

func channelUnsub(c *ishell.Context) {
	req := pb.UnsubPayChProposalsReq{
		SessionID: sessionID,
	}
	resp, err := client.UnsubPayChProposals(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v.", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.UnsubPayChProposalsResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error unsubscribing from channel proposals : %v.", msgErr.Error.Error))
		return
	}
	c.Printf("%s\n\n", greenf("Unsubscribed from channel opening request notifications in current session."))
}

func channelAcceptFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command."))
		return
	}

	// Usage: channel accept [channel notification alias]",
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	channelNotif, ok := channelNotifMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel opening notification alias.%s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known proposal aliases:\n%v.\n\n", prettify(channelNotifList)))
		return
	}

	req := pb.RespondPayChProposalReq{
		SessionID:  sessionID,
		ProposalID: channelNotif.ProposalID,
		Accept:     true,
	}
	resp, err := client.RespondPayChProposal(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v.", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.RespondPayChProposalResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error responding accept to channel opening request : %v.", msgErr.Error.Error))
		return
	}
	msg, ok := resp.Response.(*pb.RespondPayChProposalResp_MsgSuccess_)
	chAlias := addToOpenChannelMap(msg.MsgSuccess.OpenedPayChInfo.ChID, findPeerAlias(msg.MsgSuccess.OpenedPayChInfo))
	c.Printf("%s\n\n", greenf("Channel opened. Alias: %s.\nChannel Info:\n%s.", chAlias,
		prettifyPayChInfo(msg.MsgSuccess.OpenedPayChInfo)))

	removeFromChannelNotifMap(c.Args[0])
}

func findPeerAlias(payChInfo *pb.PayChInfo) string {
	for idx := range payChInfo.BalInfo.Parts {
		if payChInfo.BalInfo.Parts[idx] != perun.OwnAlias {
			return payChInfo.BalInfo.Parts[idx]
		}
	}
	return ""
}

func channelRejectFn(c *ishell.Context) {
	if client == nil {
		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command"))
		return
	}

	// Usage: channel reject [channel notification alias]",
	noArgsReq := 1
	if len(c.Args) != noArgsReq {
		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
		return
	}

	channelNotif, ok := channelNotifMap[c.Args[0]]
	if !ok {
		c.Printf("%s\n\n", redf("Unknown channel opening notification alias.%s", c.Args[0]))
		c.Printf("%s\n\n", redf("Known proposal aliases:\n%v.\n\n", prettify(channelNotifList)))
		return
	}

	req := pb.RespondPayChProposalReq{
		SessionID:  sessionID,
		ProposalID: channelNotif.ProposalID,
		Accept:     false,
	}
	resp, err := client.RespondPayChProposal(context.Background(), &req)
	if err != nil {
		c.Printf("%s\n\n", redf("Error sending command to perun node: %v.", err))
		return
	}
	msgErr, ok := resp.Response.(*pb.RespondPayChProposalResp_Error)
	if ok {
		c.Printf("%s\n\n", redf("Error responding accept to channel opening request : %v.", msgErr.Error.Error))
		return
	}
	_, ok = resp.Response.(*pb.RespondPayChProposalResp_MsgSuccess_)
	c.Printf("%s\n\n", greenf("Channel proposal rejected successfully."))

	removeFromChannelNotifMap(c.Args[0])
}

// func channelPending(c *ishell.Context) {
// 	if client == nil {
// 		c.Printf("%s\n\n", redf("Not connected to perun node, connect using 'node connect' command"))
// 		return
// 	}
// 	// [sess alias]
// 	noArgsReq := 1
// 	if len(c.Args) != noArgsReq {
// 		c.Printf("%s\n\n", redf("Got %d arg(s). Want %d.", len(c.Args), noArgsReq))
// 		c.Printf("Command help:\t%s\n\n", c.Cmd.Help)
// 		return
// 	}
// 	_, ok := sessMap[c.Args[0]]
// 	if !ok {
// 		c.Printf("%s\n\n", redf("Unknown session alias %s", c.Args[0]))
// 		c.Printf("%s\n\n", redf("Known session aliases:\n%v\n\n", prettify(sessMap)))
// 		return
// 	}
// 	c.Printf("%s\n\n", greenf("Pending proposals:\n%v\n\n", prettify(channelNotifMap)))
// }

func prettifyPayChInfo(data *pb.PayChInfo) string {
	return fmt.Sprintf("ID: %s, Currency: %s, Balance %v, Version %s",
		data.ChID, data.BalInfo.Currency, data.BalInfo.Bal, data.Version)
}
