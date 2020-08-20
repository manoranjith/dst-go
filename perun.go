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

package perun

import (
	"context"
	"math/big"

	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/client"
	pclient "perun.network/go-perun/client"
	perunLog "perun.network/go-perun/log"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire"
	"perun.network/go-perun/wire/net"
)

// Peer represents any participant in the off-chain network that the user wants to transact with.
type Peer struct {
	// Name assigned by user for referring to this peer in api requests to the node.
	// It is unique within a session on the node.
	Alias string `yaml:"alias"`

	// Permanent identity used for authenticating the peer in the off-chain network.
	OffChainAddr wire.Address `yaml:"-"`
	// This field holds the string value of address for easy marshaling / unmarshaling.
	OffChainAddrString string `yaml:"offchain_address"`

	// Address for off-chain communication.
	CommAddr string `yaml:"comm_address"`
	// Type of off-chain communication protocol.
	CommType string `yaml:"comm_type"`
}

// OwnAlias is the alias for the entry of the users own peer details.
// It will be used when traslating addresses in incoming messages / proposals to aliases.
const OwnAlias = "self"

// ContactsReader represents a read only cached list of contacts.
type ContactsReader interface {
	ReadByAlias(alias string) (p Peer, contains bool)
	ReadByOffChainAddr(offChainAddr wire.Address) (p Peer, contains bool)
}

// Contacts represents a cached list of contacts backed by a storage. Read, Write and Delete methods act on the
// cache. The state of cached list can be written to the storage by using the UpdateStorage method.
type Contacts interface {
	ContactsReader
	Write(alias string, p Peer) error
	Delete(alias string) error
	UpdateStorage() error
}

//go:generate mockery -name CommBackend -output ./internal/mocks

// CommBackend defines the set of methods required for initializing components required for off-chain communication.
// This can be protocols such as tcp, websockets, MQTT.
type CommBackend interface {
	// Returns a listener that can listen for incoming messages at the specified address.
	NewListener(address string) (net.Listener, error)

	// Returns a dialer that can dial for new outgoing connections.
	// If timeout is zero, program will use no timeout, but standard OS timeouts may still apply.
	NewDialer() Dialer
}

//go:generate mockery -name Dialer -output ./internal/mocks

// Dialer extends net.Dialer with Registerer interface.
type Dialer interface {
	net.Dialer
	Registerer
}

//go:generate mockery -name Registerer -output ./internal/mocks

// Registerer is used to register the commAddr corresponding to an offChainAddr to the wire.Bus in runtime.
type Registerer interface {
	Register(offChainAddr wire.Address, commAddr string)
}

// Credential represents the parameters required to access the keys and make signatures for a given address.
type Credential struct {
	Addr     wallet.Address
	Wallet   wallet.Wallet
	Keystore string
	Password string
}

// User represents a participant in the off-chain network that uses a session on this node for sending transactions.
type User struct {
	Peer

	OnChain  Credential // Account for funding the channel and the on-chain transactions.
	OffChain Credential // Account (corresponding to off-chain address) used for signing authentication messages.

	// List of participant addresses for this user in each open channel.
	// OffChain credential is used for managing all these accounts.
	PartAddrs []wallet.Address
}

// Session provides a context for the user to interact with a node. It manages user data (such as IDs, contacts),
// and channel client.
//
// Once established, a user can establish and transact on state channels. All the channels within a session will use
// the same type and version of communication and state channel protocol. If a user desires to use multiple types or
// versions of any protocol, it should request a seprate session for each combination of type and version of those.
type Session struct {
	ID   string // ID uniquely identifies a session instance.
	User User

	ChannelClient ChannelClient
}

type (
	ChUpdateNotifier func(ChUpdateNotif)

	ChUpdateNotif struct {
		UpdateID  string
		Currency  string
		CurrState *pchannel.State
		Update    *pclient.ChannelUpdate
		Parts     []string
		Expiry    int64
	}

	App struct {
		Def  wallet.Address
		Data pchannel.Data
	}

	ChannelInfo struct {
		ChannelID string
		Currency  string
		State     *pchannel.State
		Parts     []string // List of Alias of channel participants.
	}

	BalInfo struct {
		Currency string
		Bals     map[string]string // Map of alias to balance.
	}

	StateUpdater func(*pchannel.State)
)

type ChannelAPI interface {
	ID() string
	SendChUpdate(stateUpdater StateUpdater) error
	SubChUpdates(notifier ChUpdateNotifier) error
	UnsubChUpdates() error
	RespondChUpdate(chUpdateID string, accept bool) error
	GetInfo() ChannelInfo
	Close() (ChannelInfo, error)
}

type (
	ChProposalNotifier func(ChProposalNotif)

	ChProposalNotif struct {
		ProposalID string
		Currency   string
		Proposal   *pclient.ChannelProposal
		Parts      []string
		Expiry     int64
	}

	ChCloseNotifier func(ChCloseNotif)

	ChCloseNotif struct {
		ChannelID string
		Currency  string
		ChState   *pchannel.State
		Parts     []string
		Error     string
	}
)

type SessionAPI interface {
	ID() string
	AddContact(peer Peer) error
	OpenCh(peerAlias string, openingBals BalInfo, app App, challengeDurSecs uint64) (ChannelInfo, error)
	HandleClose(chID string, err error)
	GetCh(channelID string) (ChannelAPI, error)
	GetChInfos() []ChannelInfo
	HandleUpdate(chUpdate pclient.ChannelUpdate, responder *pclient.UpdateResponder)
	HandleProposal(chProposal *pclient.ChannelProposal, responder *pclient.ProposalResponder)
	SubChProposals(notifier ChProposalNotifier) error
	UnsubChProposals() error
	RespondChProposal(chProposalID string, accept bool) error
	SubChCloses(notifier ChCloseNotifier) error
	UnsubChCloses() error
}

// Currency represents a parser than can convert between string represetation of a currency and
// their equivalent value in base unit represented as a big interger.
type Currency interface {
	Parse(string) (*big.Int, error)
	Print(*big.Int) string
}

//go:generate mockery -name ChannelClient -output ./internal/mocks

// ChannelClient allows the user to establish off-chain channels and transact on these channels.
//
// It allows the user to enable persistence, where all data pertaining to the lifecycle of a channel is
// persisted continuously. When it is enabled, the channel client can be stopped at any point of time and resumed later.
//
// However, the channel client is not responsible if any channel the user was participating in was closed
// with a wrong state when the channel client was not running.
// Hence it is highly recommended not to stop the channel client if there are open channels.
type ChannelClient interface {
	Registerer
	ProposeChannel(context.Context, *client.ChannelProposal) (*client.Channel, error)
	Handle(client.ProposalHandler, client.UpdateHandler)
	Channel(pchannel.ID) (*client.Channel, error)
	Close() error

	EnablePersistence(persistence.PersistRestorer)
	OnNewChannel(handler func(*client.Channel))
	Restore(context.Context) error

	Log() perunLog.Logger
}

//go:generate mockery -name WireBus -output ./internal/mocks

// WireBus is a an extension of the wire.Bus interface in go-perun to include a "Close" method.
// wire.Bus (in go-perun) is a central message bus over which all clients of a channel network
// communicate. It is used as the transport layer abstraction for the ChannelClient.
type WireBus interface {
	wire.Bus
	Close() error
}

// ChainBackend wraps the methods required for instantiating and using components for
// making on-chain transactions and reading on-chain values on a specific blockchain platform.
// The timeout for on-chain transaction should be implemented by the corresponding backend. It is
// upto the implementation to make the value user configurable.
//
// It defines methods for deploying contracts; validating deployed contracts and instantiating a funder, adjudicator.
type ChainBackend interface {
	DeployAdjudicator() (adjAddr wallet.Address, _ error)
	DeployAsset(adjAddr wallet.Address) (assetAddr wallet.Address, _ error)
	ValidateContracts(adjAddr, assetAddr wallet.Address) error
	NewFunder(assetAddr wallet.Address) pchannel.Funder
	NewAdjudicator(adjAddr, receiverAddr wallet.Address) pchannel.Adjudicator
}

// WalletBackend wraps the methods for instantiating wallets and accounts that are specific to a blockchain platform.
type WalletBackend interface {
	ParseAddr(string) (wallet.Address, error)
	NewWallet(keystore string, password string) (wallet.Wallet, error)
	UnlockAccount(wallet.Wallet, wallet.Address) (wallet.Account, error)
} // nolint:gofumpt // unknown error, maybe a false positive

type NodeAPI interface {
	Time() int64
	GetConfig() map[string]string
	Help() []string
	OpenSession(configFile string) (ID string, _ error)
}
