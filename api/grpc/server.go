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

package grpc

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	psync "perun.network/go-perun/pkg/sync"

	"github.com/hyperledger-labs/perun-node/app/payment"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

type PaymentAPI struct {
	n perun.NodeAPI

	// The mutex should be used when accessing the map data structures in this API.
	psync.Mutex

	// chProposalsNotif holds a unbuffered boolean channel for each active subscription.
	// When a subscription is registered, subsription routine will add an entry to this map
	// with the session ID as they key. It will then wait indefinitely on this channel.
	//
	// The unsubscription call should retreive the channel from the map and close it, which
	// will signal the subscription routine to end.
	chProposalsNotif map[string]chan bool
}

func NewPaymentAPI(n perun.NodeAPI) *PaymentAPI {
	return &PaymentAPI{
		n:                n,
		chProposalsNotif: make(map[string]chan bool),
	}
}

func (a *PaymentAPI) GetConfig(context.Context, *pb.GetConfigReq) (*pb.GetConfigResp, error) {
	fmt.Println("Received request: GetConfig")
	cfg := a.n.GetConfig()
	return &pb.GetConfigResp{
		ChainAddress:       cfg.ChainURL,
		AdjudicatorAddress: cfg.Adjudicator,
		AssetAddress:       cfg.Asset,
		CommTypes:          cfg.CommTypes,
		ContactTypes:       cfg.ContactTypes,
	}, nil
}

func (a *PaymentAPI) OpenSession(ctx context.Context, req *pb.OpenSessionReq) (*pb.OpenSessionResp, error) {
	fmt.Println("Received request: OpenSession")
	sessionID, err := a.n.OpenSession(req.ConfigFile)
	if err != nil {
		return &pb.OpenSessionResp{
			Response: &pb.OpenSessionResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}

	return &pb.OpenSessionResp{
		Response: &pb.OpenSessionResp_MsgSuccess_{
			MsgSuccess: &pb.OpenSessionResp_MsgSuccess{
				SessionID: sessionID,
			},
		},
	}, nil
}

func (a *PaymentAPI) Time(context.Context, *pb.TimeReq) (*pb.TimeResp, error) {
	fmt.Println("Received request: Time")
	return &pb.TimeResp{
		Time: a.n.Time(),
	}, nil
}

func (a *PaymentAPI) Help(context.Context, *pb.HelpReq) (*pb.HelpResp, error) {
	fmt.Println("Received request: Help")
	return &pb.HelpResp{
		Apis: a.n.Help(),
	}, nil
}

func (a *PaymentAPI) AddContact(ctx context.Context, req *pb.AddContactReq) (*pb.AddContactResp, error) {
	fmt.Println("Received request: AddContact")
	sess, err := a.n.GetSession(req.SessionID)
	if err != nil {
		return &pb.AddContactResp{
			Response: &pb.AddContactResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	peer := perun.Peer{
		Alias:              req.Peer.Alias,
		OffChainAddrString: req.Peer.OffChainAddress,
		CommAddr:           req.Peer.CommAddress,
		CommType:           req.Peer.CommType,
	}
	err = sess.AddContact(peer)
	if err != nil {
		return &pb.AddContactResp{
			Response: &pb.AddContactResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	return &pb.AddContactResp{
		Response: &pb.AddContactResp_MsgSuccess_{
			MsgSuccess: &pb.AddContactResp_MsgSuccess{
				Success: true,
			},
		},
	}, nil
}

func (a *PaymentAPI) GetContact(ctx context.Context, req *pb.GetContactReq) (*pb.GetContactResp, error) {
	fmt.Println("Received request: GetContact")
	sess, err := a.n.GetSession(req.SessionID)
	if err != nil {
		return &pb.GetContactResp{
			Response: &pb.GetContactResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	peer, err := sess.GetContact(req.Alias)
	peer_ := pb.Peer{
		Alias:           peer.Alias,
		OffChainAddress: peer.OffChainAddrString,
		CommAddress:     peer.CommAddr,
		CommType:        peer.CommType,
	}
	if err != nil {
		return &pb.GetContactResp{
			Response: &pb.GetContactResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	return &pb.GetContactResp{
		Response: &pb.GetContactResp_MsgSuccess_{
			MsgSuccess: &pb.GetContactResp_MsgSuccess{
				Peer: &peer_,
			},
		},
	}, nil
}

func (a *PaymentAPI) OpenPayCh(ctx context.Context, req *pb.OpenPayChReq) (*pb.OpenPayChResp, error) {
	fmt.Println("Received request: OpenPayCh")
	sess, err := a.n.GetSession(req.SessionID)
	if err != nil {
		return &pb.OpenPayChResp{
			Resp: &pb.OpenPayChResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	balInfo := FromGrpcBalInfo(req.OpeningBalance)
	payChInfo, err := payment.OpenPayCh(ctx, sess, req.PeerAlias, balInfo, req.ChallengeDurSecs)
	payChInfo_ := pb.PaymentChannel{
		ChannelID: payChInfo.ChannelID,
		Version:   payChInfo.Version,
	}
	if err != nil {
		return &pb.OpenPayChResp{
			Resp: &pb.OpenPayChResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	return &pb.OpenPayChResp{
		Resp: &pb.OpenPayChResp_MsgSuccess_{
			MsgSuccess: &pb.OpenPayChResp_MsgSuccess{
				Channel: &payChInfo_,
			},
		},
	}, nil
}

func FromGrpcBalInfo(src *pb.BalanceInfo) perun.BalInfo {
	balInfo := perun.BalInfo{
		Currency: src.Currency,
		Bals:     make(map[string]string, len(src.Balances)),
	}
	for _, aliasBalance := range src.Balances {
		for key, value := range aliasBalance.Value {
			balInfo.Bals[key] = value
		}
	}
	return balInfo
}

func ToGrpcBalInfo(src perun.BalInfo) *pb.BalanceInfo {
	balInfo := &pb.BalanceInfo{
		Currency: src.Currency,
		Balances: make([]*pb.BalanceInfo_AliasBalance, len(src.Bals)),
	}
	i := 0
	for key, value := range src.Bals {
		balInfo.Balances[i] = &pb.BalanceInfo_AliasBalance{
			Value: make(map[string]string),
		}
		balInfo.Balances[i].Value[key] = value
		i++
	}
	return balInfo
}

func (a *PaymentAPI) GetPayChs(context.Context, *pb.GetPayChsReq) (*pb.GetPayChsResp, error) {
	return nil, nil
}

func (a *PaymentAPI) SubPayChProposals(req *pb.SubPayChProposalsReq, srv pb.Payment_API_SubPayChProposalsServer) error {
	fmt.Println("Received request: SubPayChProposals")
	sess, err := a.n.GetSession(req.SessionID)
	if err != nil {
		// TODO: (mano) Return a error response and not a protocol error
		return errors.WithMessage(err, "cannot register subscription")
	}

	notifier := func(notif payment.PayChProposalNotif) {
		notif_ := pb.SubPayChProposalsResp_Notify_{
			Notify: &pb.SubPayChProposalsResp_Notify{
				ProposalID:       notif.ProposalID,
				OpeningBalance:   ToGrpcBalInfo(notif.OpeningBals),
				ChallengeDurSecs: notif.ChallengeDurSecs,
				Expiry:           notif.Expiry,
			},
		}
		notifResponse := pb.SubPayChProposalsResp{Response: &notif_}
		err := srv.Send(&notifResponse)
		if err != nil {
			// TODO: (mano) Error handling when sending notification.
			fmt.Println("Error sending notification")
		}
	}

	err = payment.SubPayChProposals(sess, notifier)
	if err != nil {
		return err
	}
	signal := make(chan bool)
	a.Lock()
	a.chProposalsNotif[req.SessionID] = signal
	a.Unlock()

	<-signal
	fmt.Println("Channel Proposal Subscription ended for" + req.SessionID)
	return nil
}

func (a *PaymentAPI) UnsubPayChProposals(ctx context.Context, req *pb.UnsubPayChProposalsReq) (*pb.UnsubPayChProposalsResp, error) {
	fmt.Println("Received request: UnsubPayChProposals")
	sess, err := a.n.GetSession(req.SessionID)
	if err != nil {
		return &pb.UnsubPayChProposalsResp{
			Response: &pb.UnsubPayChProposalsResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	err = payment.UnsubPayChProposals(sess)
	if err != nil {
		return &pb.UnsubPayChProposalsResp{
			Response: &pb.UnsubPayChProposalsResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}

	a.Lock()
	signal := a.chProposalsNotif[req.SessionID]
	a.Unlock()

	close(signal)
	return &pb.UnsubPayChProposalsResp{
		Response: &pb.UnsubPayChProposalsResp_MsgSuccess_{
			MsgSuccess: &pb.UnsubPayChProposalsResp_MsgSuccess{
				Success: true,
			},
		},
	}, nil
}

func (a *PaymentAPI) RespondPayChProposal(ctx context.Context, req *pb.RespondPayChProposalReq) (*pb.RespondPayChProposalResp, error) {
	fmt.Println("Received request: RespondPayChProposal")
	sess, err := a.n.GetSession(req.SessionID)
	if err != nil {
		return &pb.RespondPayChProposalResp{
			Response: &pb.RespondPayChProposalResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	err = payment.RespondPayChProposal(ctx, sess, req.ProposalID, req.Accept)
	if err != nil {
		return &pb.RespondPayChProposalResp{
			Response: &pb.RespondPayChProposalResp_Error{
				Error: &pb.MsgError{
					Error: err.Error(),
				},
			},
		}, nil
	}
	return &pb.RespondPayChProposalResp{
		Response: &pb.RespondPayChProposalResp_MsgSuccess_{
			MsgSuccess: &pb.RespondPayChProposalResp_MsgSuccess{
				Success: true,
			},
		},
	}, nil
}

func (a *PaymentAPI) SubPayChCloses(*pb.SubPayChClosesReq, pb.Payment_API_SubPayChClosesServer) error {
	return nil
}

func (a *PaymentAPI) UnsubPayChClose(context.Context, *pb.UnsubPayChClosesReq) (*pb.UnsubPayChClosesResp, error) {
	return nil, nil
}

func (a *PaymentAPI) CloseSession(context.Context, *pb.CloseSessionReq) (*pb.CloseSessionResp, error) {
	return nil, nil
}

func (a *PaymentAPI) SendPayChUpdate(context.Context, *pb.SendPayChUpdateReq) (*pb.SendPayChUpdateResp, error) {
	return nil, nil
}

func (a *PaymentAPI) SubPayChUpdates(*pb.SubpayChUpdatesReq, pb.Payment_API_SubPayChUpdatesServer) error {
	return nil
}

func (a *PaymentAPI) UnsubPayChUpdates(context.Context, *pb.UnsubPayChUpdatesReq) (*pb.UnsubPayChUpdatesResp, error) {
	return nil, nil
}

func (a *PaymentAPI) RespondPayChUpdate(context.Context, *pb.RespondPayChUpdateReq) (*pb.RespondPayChUpdateResp, error) {
	return nil, nil
}

func (a *PaymentAPI) GetPayChBalance(context.Context, *pb.GetPayChBalanceReq) (*pb.GetPayChBalanceResp, error) {
	return nil, nil
}

func (a *PaymentAPI) ClosePayCh(context.Context, *pb.ClosePayChReq) (*pb.ClosePayChResp, error) {
	return nil, nil
}
