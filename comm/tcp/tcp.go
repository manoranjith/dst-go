// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/direct-state-transfer/dst-go
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

package tcp

import (
	"time"

	"perun.network/go-perun/peer"
	"perun.network/go-perun/peer/net"

	"github.com/direct-state-transfer/dst-go"
)

// Backend provices offchain communication adapter for tcp protocol.
// It stores any data required for configuration.
type Backend struct {
	dialerTimeout time.Duration
}

// NewListener returns a listener that can listen for incommig messages at
// the specified address using the tcp protocol.
func (b Backend) NewListener(addr string) (peer.Listener, error) {
	return net.NewListener("tcp", addr)
}

// NewDialer returns a tcp dialer that can dial for connections with a dial
// timeout configured during the initialization of the backend.
//
// If the duration was set to zero, this program will not use any timeout.
// However default timeouts based on the operating system will still apply.
func (b Backend) NewDialer() peer.Dialer {
	return net.NewDialer("tcp", b.dialerTimeout)
}

// NewBackend returns a offchain communication that uses tcp protocol.
// The provided dialerTimeout will be when dialing new outging connections.
// If the duration was set to zero, this program will not use any timeout.
// However default timeouts based on the operating system will still apply.
func NewBackend(dialerTimeout time.Duration) dst.CommBackend {
	return Backend{dialerTimeout: dialerTimeout}

}
