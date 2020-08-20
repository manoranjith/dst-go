package main

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
	"github.com/hyperledger-labs/perun-node/node"
)

type PaymentAPI struct {
	n perun.NodeAPI
}

func (a *PaymentAPI) GetConfig(context.Context, *pb.GetConfigReq) (*pb.GetConfigResp, error) {
	fmt.Println("Received request: GetConfig")
	cfg := a.n.GetConfig()
	return &pb.GetConfigResp{
		ChainAddress:       cfg.ChainAddr,
		AdjudicatorAddress: cfg.AdjudicatorAddr,
		AssetAddress:       cfg.AssetAddr,
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
func (a *PaymentAPI) AddContact(context.Context, *pb.AddContactReq) (*pb.AddContactResp, error) {
	return nil, nil
}
func (a *PaymentAPI) GetContact(context.Context, *pb.GetContactReq) (*pb.GetContactResp, error) {
	return nil, nil
}
func (a *PaymentAPI) OpenPayCh(context.Context, *pb.OpenPayChReq) (*pb.OpenPayChResp, error) {
	return nil, nil
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

func main() {
	// create listiner
	lis, err := net.Listen("tcp", ":50001")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}

	nodeCfg, err := node.ParseConfig("../../../testdata/node/valid.yaml")
	if err != nil {
		fmt.Println("error readind node config", err)
		return
	}
	nodeCfg.LogFile = ""
	n, err := node.New(nodeCfg)
	if err != nil {
		fmt.Println("error init node ", err)
		return
	}
	payServer := &PaymentAPI{
		n: n,
	}

	// create grpc server
	s := grpc.NewServer()
	pb.RegisterPayment_APIServer(s, payServer)
	// pb.RegisterNode_APIServer(s, server{})
	// fmt.Println("Server started")
	// // s1 := grpc.NewServer()
	// pb.RegisterSession_APIServer(s, server{})
	// go Stream()

	fmt.Println("Started listening")
	if err := s.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v", err)
		return
	}
	s.Serve(lis)
}
