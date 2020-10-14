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

	"github.com/hyperledger-labs/perun-node/api/grpc/pb"
)

// shell is a wrapper around the ishell.Shell type to includes a mutex. The
// mutex will be locked by subscription go-routines when printing the received
// notification.
type shell struct {
	*ishell.Shell
	sync.Mutex
}

var (
	// File that stores history of commands used in the interactive shell.
	// This will be preserved accross the multiple runs of perunnode cli.
	// It will be located in the home directory.
	historyFile = ".perunnode_history"

	// Singleton instance of ishell that is used throughout this program.
	// this will be initialized in main() and be accessed by subscription
	// handler go-routines that run concurrently to print the received
	// notification messages.
	sh *shell

	// Singleton instance of grpc payment channel client that will be
	// used by all functions in this program. This is safe for concurrent
	// access without a mutex.
	client pb.Payment_APIClient

	// Session ID for the currently active session. The cli application
	// allows only one session to be open at a time and all channel requests,
	// payments and payment requests are sent and received in this context of
	// this session.
	// It is set when a session is opened and closed when a session is closed.
	sessionID string

	// standard value of challenge duration for all outgoing channel open requests.
	challengeDurSecs uint64 = 10

	// SPrintf style functions that produce colored text.
	redf   = color.New(color.FgRed).SprintfFunc()
	greenf = color.New(color.FgGreen).SprintfFunc()
)

func main() {
	// New shell includes help, clear, exit commands by default.
	sh = &shell{
		Shell: ishell.New(),
	}

	// Read and write history to $HOME/historyFile
	sh.SetHomeHistoryPath(historyFile)

	sh.AddCmd(nodeCmd)
	sh.AddCmd(sessionCmd)
	sh.AddCmd(channelCmd)
	sh.AddCmd(paymentCmd)

	sh.Printf("Perun node cli application.\n\n")
	sh.Printf("%s\n\n", greenf("Connect to a perun node instance using 'node connect' for making any transactions."))

	sh.Run()
}
