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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/hyperledger-labs/perun-node"
	pngrpc "github.com/hyperledger-labs/perun-node/api/grpc"
	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
	"github.com/hyperledger-labs/perun-node/cmd/perunnode"
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
	var aliceAlias, bobAlias = "alice", "bob"

	t.Run("Node.OpenSession_Alice", func(t *testing.T) {
		aliceCfg := sessiontest.NewConfig(t, prng)
		openSessionReq := pb.OpenSessionReq{
			ConfigFile: sessiontest.NewConfigFile(t, aliceCfg),
		}
		openSessionResp, err := client.OpenSession(ctx, &openSessionReq)
		t.Logf("\nResponse: %+v, Error: %+v", openSessionResp, err)
		successResponse := openSessionResp.Response.(*pb.OpenSessionResp_MsgSuccess_)
		aliceSessionID = successResponse.MsgSuccess.SessionID
		t.Logf("Bob session id: %s", aliceSessionID)
	})

	t.Run("Node.OpenSession_Bob", func(t *testing.T) {
		bobCfg := sessiontest.NewConfig(t, prng)
		openSessionReq := pb.OpenSessionReq{
			ConfigFile: sessiontest.NewConfigFile(t, bobCfg),
		}
		openSessionResp, err := client.OpenSession(ctx, &openSessionReq)
		t.Logf("\nResponse: %+v, Error: %+v", openSessionResp, err)
		successResponse := openSessionResp.Response.(*pb.OpenSessionResp_MsgSuccess_)
		bobSessionID = successResponse.MsgSuccess.SessionID
		t.Logf("Bob session id: %s", bobSessionID)
	})

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
}

func StartServer(t *testing.T) {
	// Initialize a listener.
	listener, err := net.Listen("tcp", grpcPort)
	require.NoErrorf(t, err, "starting listener")

	// Initializr a grpc payment API/
	nodeAPI, err := perunnode.New(nodeCfg)
	require.NoErrorf(t, err, "initializing nodeAPI")
	grpcPaymentAPI := pngrpc.NewPaymentAPI(nodeAPI)

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
	return
}
