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
	"sync"

	"github.com/abiosoft/ishell"
	"github.com/fatih/color"

	grpclib "google.golang.org/grpc"

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

// ishell is a wrapper around the shell type that includes a mutex.
// the mutex will be locked by subscription go-routines when printing
// the received notification.

type shell struct {
	*ishell.Shell
	sync.Mutex
}

// singleton instance of ishell that can be accessed throught this program.
var sh *shell

var (
	grpcPort = ":50001"

	// singleton instance of client and context that will be used for all tests.
	client pb.Payment_APIClient

	// SPrintf style functions that produce colored text.
	redf, greenf func(format string, a ...interface{}) string
)

func init() {
	redf = color.New(color.FgRed).SprintfFunc()
	greenf = color.New(color.FgGreen).SprintfFunc()
}

func defaultHandler(c *ishell.Context) {
	c.Printf("Got command %v with args %v\n", c.Cmd.Name, c.Args)
}

func main() {
	// New shell includes help, clear, exit commands by default.
	sh = &shell{
		Shell: ishell.New(),
	}
	// Read and write history to $HOME/.ishell_history
	sh.SetHomeHistoryPath(".ishell_history")

	sh.AddCmd(nodeCmd)
	sh.AddCmd(sessCmd)
	sh.AddCmd(propCmd)
	sh.AddCmd(chCmd)

	sh.Printf("Perun node cli application.\n\n")

	conn, err := grpclib.Dial(grpcPort, grpclib.WithInsecure())
	client = pb.NewPayment_APIClient(conn)
	if err != nil {
		sh.Printf("Error connecting to perun node at %s: %v", grpcPort, err)
	}
	sh.Printf("Connected to perun node at %s\n\n", grpcPort)

	sh.Run()
}
