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

package grpc_test

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/hyperledger-labs/perun-node"
	pngrpc "github.com/hyperledger-labs/perun-node/api/grpc"
	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
	"github.com/hyperledger-labs/perun-node/cmd/perunnode"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

var (
	nodeCfg = perun.NodeConfig{
		LogFile:      "",
		LogLevel:     "debug",
		ChainURL:     "ws://127.0.0.1:8545",
		Adjudicator:  "0x9daEdAcb21dce86Af8604Ba1A1D7F9BFE55ddd63",
		Asset:        "0x5992089d61cE79B6CF90506F70DD42B8E42FB21d",
		CommTypes:    []string{"tcp"},
		ContactTypes: []string{"yaml"},
		Currencies:   []string{"ETH"},

		ChainConnTimeout: 30 * time.Second,
		OnChainTxTimeout: 10 * time.Second,
		ResponseTimeout:  10 * time.Second,
	}

	grpcPort = ":50001"
)

func Test_Integ_Role(t *testing.T) {
	StartServer(t)

	conn, err := grpc.Dial(grpcPort, grpc.WithInsecure())
	require.NoError(t, err, "dialing to grpc server")
	t.Log("connected to server")

	// Inititalize client.
	client := pb.NewPayment_APIClient(conn)
	ctx := context.Background()

	t.Run("Node.Time", func(t *testing.T) {
		timeReq := pb.TimeReq{}
		timeResp, err := client.Time(ctx, &timeReq)
		require.NoError(t, err)
		t.Logf("\nResponse: %+v, Error: %+v", timeResp, err)
	})

	t.Run("Node.GetConfig", func(t *testing.T) {
		getConfigReq := pb.GetConfigReq{}
		getConfigResp, err := client.GetConfig(ctx, &getConfigReq)
		require.NoError(t, err)
		t.Logf("\nResponse: %+v, Error: %+v", getConfigResp, err)
	})

	t.Run("Node.Help", func(t *testing.T) {
		helpReq := pb.HelpReq{}
		helpResp, err := client.Help(ctx, &helpReq)
		require.NoError(t, err)
		t.Logf("\nResponse: %+v, Error: %+v", helpResp, err)
	})

	prng := rand.New(rand.NewSource(1729))
	var aliceSessionID, bobSessionID string
	var alicePeer, bobPeer *pb.Peer
	aliceAlias, bobAlias := "alice", "bob"
	wg := &sync.WaitGroup{}

	// Run OpenSession for Alice, Bob in top level test, because cleaup functions
	// for removing the keystore directory, contacts file are registered to this
	// testing.T.

	// Alice Open Session
	aliceCfg := sessiontest.NewConfig(t, prng)
	aliceOpenSessionReq := pb.OpenSessionReq{
		ConfigFile: sessiontest.NewConfigFile(t, aliceCfg),
	}
	aliceOpenSessionResp, err := client.OpenSession(ctx, &aliceOpenSessionReq)
	t.Logf("\nResponse: %+v, Error: %+v", aliceOpenSessionResp, err)
	aliceSuccessResponse := aliceOpenSessionResp.Response.(*pb.OpenSessionResp_MsgSuccess_)
	aliceSessionID = aliceSuccessResponse.MsgSuccess.SessionID
	t.Logf("Alice session id: %s", aliceSessionID)

	// Bob Open Session
	bobCfg := sessiontest.NewConfig(t, prng)
	bobOpenSessionReq := pb.OpenSessionReq{
		ConfigFile: sessiontest.NewConfigFile(t, bobCfg),
	}
	bobOpenSessionResp, err := client.OpenSession(ctx, &bobOpenSessionReq)
	t.Logf("\nResponse: %+v, Error: %+v", bobOpenSessionResp, err)
	bobSuccessResponse := bobOpenSessionResp.Response.(*pb.OpenSessionResp_MsgSuccess_)
	bobSessionID = bobSuccessResponse.MsgSuccess.SessionID
	t.Logf("Bob session id: %s", bobSessionID)

	t.Run("Session.GetContact_Alice", func(t *testing.T) {
		getContactReq := pb.GetContactReq{
			SessionID: aliceSessionID,
			Alias:     perun.OwnAlias,
		}
		getContactResp, err := client.GetContact(ctx, &getContactReq)
		t.Logf("\nResponse: %+v, Error: %+v", getContactResp, err)
		successResponse, ok := getContactResp.Response.(*pb.GetContactResp_MsgSuccess_)
		if !ok {
			errorResponse := getContactResp.Response.(*pb.GetContactResp_Error)
			t.Errorf("Error response: %+v", errorResponse)
		} else {
			alicePeer = successResponse.MsgSuccess.Peer
			alicePeer.Alias = aliceAlias
			t.Logf("Alice Peer is: %+v", alicePeer)
		}
	})

	t.Run("Session.GetContact_Bob", func(t *testing.T) {
		getContactReq := pb.GetContactReq{
			SessionID: bobSessionID,
			Alias:     perun.OwnAlias,
		}
		getContactResp, err := client.GetContact(ctx, &getContactReq)
		t.Logf("\nResponse: %+v, Error: %+v", getContactResp, err)
		successResponse, ok := getContactResp.Response.(*pb.GetContactResp_MsgSuccess_)
		if !ok {
			errorResponse := getContactResp.Response.(*pb.GetContactResp_Error)
			t.Errorf("Error response: %+v", errorResponse)
		} else {
			bobPeer = successResponse.MsgSuccess.Peer
			bobPeer.Alias = bobAlias
			t.Logf("Alice Peer is: %+v", bobPeer)
		}
	})

	t.Run("Session.AddContact_Alice", func(t *testing.T) {
		addContactReq := pb.AddContactReq{
			SessionID: aliceSessionID,
			Peer:      bobPeer,
		}
		addContactResp, err := client.AddContact(ctx, &addContactReq)
		t.Logf("\nResponse: %+v, Error: %+v", addContactResp, err)
		_, ok := addContactResp.Response.(*pb.AddContactResp_MsgSuccess_)
		if !ok {
			errorResponse := addContactResp.Response.(*pb.AddContactResp_Error)
			t.Errorf("Error response: %+v", errorResponse)
		} else {
			t.Logf("Alice added bob to contacts")
		}
	})

	t.Run("Session.AddContact_Bob", func(t *testing.T) {
		addContactReq := pb.AddContactReq{
			SessionID: bobSessionID,
			Peer:      alicePeer,
		}
		addContactResp, err := client.AddContact(ctx, &addContactReq)
		t.Logf("\nResponse: %+v, Error: %+v", addContactResp, err)
		_, ok := addContactResp.Response.(*pb.AddContactResp_MsgSuccess_)
		if !ok {
			errorResponse := addContactResp.Response.(*pb.AddContactResp_Error)
			t.Errorf("Error response: %+v", errorResponse)
		} else {
			t.Logf("Bob added alice to contacts")
		}
	})

	var channel1ID string
	t.Run("Session.OpenPayCh_Sub_Unsub_Respond", func(t *testing.T) {
		wg.Add(1)
		// Alice proposes payment channel to bob.
		go func() {
			balInfo_ := perun.BalInfo{
				Currency: currency.ETH,
				Bals:     make(map[string]string),
			}
			balInfo_.Bals[perun.OwnAlias] = "1"
			balInfo_.Bals[bobAlias] = "2"
			openPayChReq := pb.OpenPayChReq{
				SessionID:        aliceSessionID,
				PeerAlias:        bobAlias,
				OpeningBalance:   pngrpc.ToGrpcBalInfo(balInfo_),
				ChallengeDurSecs: 10,
			}
			openPayChResp, err := client.OpenPayCh(ctx, &openPayChReq)
			t.Logf("\nResponse: %+v, Error: %+v", openPayChResp, err)
			successResponse, ok := openPayChResp.Response.(*pb.OpenPayChResp_MsgSuccess_)
			if !ok {
				errorResponse := openPayChResp.Response.(*pb.OpenPayChResp_Error)
				t.Errorf("Error response: %+v", errorResponse)
			} else {
				channel1ID = successResponse.MsgSuccess.Channel.ChannelID
				t.Logf("Bob added alice to contacts")
			}
			wg.Done()
		}()

		// Bob subscribes to channel proposal notifications.
		subPayChProposalsReq := pb.SubPayChProposalsReq{
			SessionID: bobSessionID,
		}
		payChProposalsSubClient, err := client.SubPayChProposals(ctx, &subPayChProposalsReq)
		require.NoErrorf(t, err, "subscribing to payment channel proposals")

		subPayChProposalsResp, err := payChProposalsSubClient.Recv()
		require.NoErrorf(t, err, "receiving payment channel proposal notification")
		notif, ok := subPayChProposalsResp.Response.(*pb.SubPayChProposalsResp_Notify_)
		if !ok {
			t.Errorf("Error receiving notifications")
		}
		t.Logf("Bob received payment channel proposal notification: %+v", notif.Notify)

		// Bob accepts channel proposal.
		respondChProposalReq := pb.RespondPayChProposalReq{
			SessionID:  bobSessionID,
			ProposalID: notif.Notify.ProposalID,
			Accept:     true,
		}
		_, err = client.RespondPayChProposal(ctx, &respondChProposalReq)
		require.NoErrorf(t, err, "responding to payment channel proposal")

		// Bob unsubscribes to channel proposal notifications.
		unsubPayChProposalsReq := pb.UnsubPayChProposalsReq{
			SessionID: bobSessionID,
		}
		_, err = client.UnsubPayChProposals(ctx, &unsubPayChProposalsReq)
		require.NoErrorf(t, err, "unsubscribing to payment channel proposals")

		wg.Wait()
	})

	t.Run("Channel.SendPayChUpdate_Sub_Unsub_Respond", func(t *testing.T) {
		wg.Add(1)
		// Bob sends payment channel to alice.
		go func() {
			sendPayChUpdateReq := pb.SendPayChUpdateReq{
				SessionID: bobSessionID,
				ChannelID: channel1ID,
				Payee:     aliceAlias,
				Amount:    "0.5",
			}
			sendPayChUpdateResp, err := client.SendPayChUpdate(ctx, &sendPayChUpdateReq)
			t.Logf("\nResponse: %+v, Error: %+v", sendPayChUpdateResp, err)
			_, ok := sendPayChUpdateResp.Response.(*pb.SendPayChUpdateResp_MsgSuccess_)
			if !ok {
				errorResponse := sendPayChUpdateResp.Response.(*pb.SendPayChUpdateResp_Error)
				t.Errorf("Error response: %+v", errorResponse)
			} else {
				t.Logf("Bob send payment to alice")
			}
			wg.Done()
		}()

		// Alice subscribes to channel proposal notifications.
		subpayChUpdatesReq := pb.SubpayChUpdatesReq{
			SessionID: aliceSessionID,
			ChannelID: channel1ID,
		}
		payChUpdatesSubClient, err := client.SubPayChUpdates(ctx, &subpayChUpdatesReq)
		require.NoErrorf(t, err, "subscribing to payment channel updates")

		subPayChUpdatesResp, err := payChUpdatesSubClient.Recv()
		require.NoErrorf(t, err, "receiving payment channel update notification")
		notif, ok := subPayChUpdatesResp.Response.(*pb.SubPayChUpdatesResp_Notify_)
		if !ok {
			t.Errorf("Error receiving notifications")
		}
		t.Logf("Bob received payment channel update notification: %+v", notif.Notify)

		// Alice accepts channel proposal.
		respondChUpdateReq := pb.RespondPayChUpdateReq{
			SessionID: aliceSessionID,
			UpdateID:  notif.Notify.UpdateID,
			ChannelID: channel1ID,
			Accept:    true,
		}
		_, err = client.RespondPayChUpdate(ctx, &respondChUpdateReq)
		require.NoErrorf(t, err, "responding to payment channel proposal")

		// Alice unsubscribes to channel proposal notifications.
		unsubPayChUpdatesReq := pb.UnsubPayChUpdatesReq{
			SessionID: aliceSessionID,
			ChannelID: channel1ID,
		}
		_, err = client.UnsubPayChUpdates(ctx, &unsubPayChUpdatesReq)
		require.NoErrorf(t, err, "unsubscribing to payment channel proposals")

		wg.Wait()
	})

	t.Run("Channel.ClosePayCh_Sub_Unsub", func(t *testing.T) {
		wg.Add(1)
		// Bob sends payment channel to alice.
		go func() {
			closePayChReq := pb.ClosePayChReq{
				SessionID: bobSessionID,
				ChannelID: channel1ID,
			}
			closePayChResp, err := client.ClosePayCh(ctx, &closePayChReq)
			t.Logf("\nResponse: %+v, Error: %+v", closePayChResp, err)
			_, ok := closePayChResp.Response.(*pb.ClosePayChResp_MsgSuccess_)
			if !ok {
				errorResponse := closePayChResp.Response.(*pb.ClosePayChResp_Error)
				t.Errorf("Error response: %+v", errorResponse)
			} else {
				t.Logf("Bob closed payment channel")
			}
			wg.Done()
		}()

		// Alice subscribes to channel close notifications.
		subpayChClosesReq := pb.SubPayChClosesReq{
			SessionID: aliceSessionID,
		}
		payChClosesSubClient, err := client.SubPayChCloses(ctx, &subpayChClosesReq)
		require.NoErrorf(t, err, "subscribing to payment channel updates")

		subPayChClosesResp, err := payChClosesSubClient.Recv()
		require.NoErrorf(t, err, "receiving payment channel update notification")
		notif, ok := subPayChClosesResp.Response.(*pb.SubPayChClosesResp_Notify_)
		if !ok {
			t.Errorf("Error receiving notifications")
		}
		t.Logf("Alice received payment channel close notification: %+v", notif.Notify)

		// Alice unsubscribes to channel close notifications.
		unsubPayChClosesReq := pb.UnsubPayChClosesReq{
			SessionID: aliceSessionID,
		}
		_, err = client.UnsubPayChClose(ctx, &unsubPayChClosesReq)
		require.NoErrorf(t, err, "unsubscribing to payment channel proposals")

		// This doesn't work on payment branch... but will work on develop for new channel close logic.
		// // Bob subscribes to channel close notifications.
		// subpayChClosesReq = pb.SubPayChClosesReq{
		// 	SessionID: bobSessionID,
		// }
		// payChClosesSubClient, err = client.SubPayChCloses(ctx, &subpayChClosesReq)
		// require.NoErrorf(t, err, "subscribing to payment channel updates")

		// subPayChClosesResp, err = payChClosesSubClient.Recv()
		// require.NoErrorf(t, err, "receiving payment channel update notification")
		// notif, ok = subPayChClosesResp.Response.(*pb.SubPayChClosesResp_Notify_)
		// if !ok {
		// 	t.Errorf("Error receiving notifications")
		// }
		// t.Logf("Bob received payment channel close notification: %+v", notif.Notify)

		// // Bob unsubscribes to channel close notifications.
		// unsubPayChClosesReq = pb.UnsubPayChClosesReq{
		// 	SessionID: bobSessionID,
		// }
		// _, err = client.UnsubPayChClose(ctx, &unsubPayChClosesReq)
		// require.NoErrorf(t, err, "unsubscribing to payment channel proposals")
		wg.Wait()
	})
}

func StartServer(t *testing.T) {
	// Initialize a listener.
	listener, err := net.Listen("tcp", grpcPort)
	require.NoErrorf(t, err, "starting listener")

	// Initializr a grpc payment API/
	nodeAPI, err := perunnode.New(nodeCfg)
	require.NoErrorf(t, err, "initializing nodeAPI")
	grpcPaymentAPI := pngrpc.NewGrpcPayChServer(nodeAPI)

	// create grpc server
	grpcServer := grpc.NewServer()
	pb.RegisterPayment_APIServer(grpcServer, grpcPaymentAPI)

	// Run Server in a go-routine.
	t.Log("Starting server")

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("failed to serve: %v", err)
		}
	}()
}
