package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

func main() {
	conn, err := grpc.Dial(":50001", grpc.WithInsecure())

	if err != nil {
		fmt.Printf("can not connect with server %v", err)
	} else {
		fmt.Printf("connected to server \n")
	}

	// Init
	client := pb.NewPayment_APIClient(conn)
	ctx := context.Background()

	// Node.Time
	timeReq := pb.TimeReq{}
	timeResp, err := client.Time(ctx, &timeReq)
	fmt.Printf("\nResponse: %+v, Error: %+v", timeResp, err)

	// Node.GetConfig
	getConfigReq := pb.GetConfigReq{}
	getConfigResp, err := client.GetConfig(ctx, &getConfigReq)
	fmt.Printf("\nResponse: %+v, Error: %+v", getConfigResp, err)

	// Node.Help
	helpReq := pb.HelpReq{}
	helpResp, err := client.Help(ctx, &helpReq)
	fmt.Printf("\nResponse: %+v, Error: %+v", helpResp, err)

	// Node.OpenSession
	openSessionReq := pb.OpenSessionReq{
		ConfigFile: "../../../testdata/role/alice_session.yaml",
	}
	openSessionResp, err := client.OpenSession(ctx, &openSessionReq)
	fmt.Printf("\nResponse: %+v, Error: %+v", openSessionResp, err)

	// var p *pb.OpenSessionRep_MsgSuccess_ = new(pb.OpenSessionRep_MsgSuccess_)
	// client := pb.NewNode_APIClient(conn)
	// request := pb.OpenSessionReq{ConfigFile: "Api config"}
	// resp, err := client.OpenSession(context.Background(), &request)
	// messageSuccess, ok := resp.Response.(*pb.OpenSessionRep_MsgSuccess_)

	// if ok {
	//     fmt.Printf("Success %v \n", messageSuccess.MsgSuccess)
	// } else {
	//     messageError, ok := resp.Response.(*pb.OpenSessionRep_Error)
	//     if ok {
	//         fmt.Printf("errore occured %v", messageError.Error)
	//     }
	// }
}
