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

	"github.com/hyperledger-labs/perun-node/app/payment"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

type PaymentAPI struct {
	n perun.NodeAPI
}

func NewPaymentAPI(n perun.NodeAPI) *PaymentAPI {
	return &PaymentAPI{n}
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
	balInfo := perun.BalInfo{
		Currency: req.OpeningBalance.Currency,
		Bals:     make(map[string]string, len(req.OpeningBalance.Balances)),
	}
	for _, aliasBalance := range req.OpeningBalance.Balances {
		for key, value := range aliasBalance.Value {
			balInfo.Bals[key] = value
		}
	}
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

func (a *PaymentAPI) GetPayChs(context.Context, *pb.GetPayChsReq) (*pb.GetPayChsResp, error) {
	return nil, nil
}

func (a *PaymentAPI) SubPayChProposals(*pb.SubPayChProposalsReq, pb.Payment_API_SubPayChProposalsServer) error {
	return nil
}

func (a *PaymentAPI) UnsubPayChProposals(context.Context, *pb.UnsubPayChProposalsReq) (*pb.UnsubPayChProposalsResp, error) {
	return nil, nil
}

func (a *PaymentAPI) RespondPayChProposal(context.Context, *pb.RespondPayChProposalReq) (*pb.RespondPayChProposalResp, error) {
	return nil, nil
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
