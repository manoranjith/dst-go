// Package dst defines domain types and services for the dst node.
package dst

import (
	"context"
	"net"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/channel/persistence"
	"perun.network/go-perun/client"
	perunLog "perun.network/go-perun/log"
	"perun.network/go-perun/peer"
	"perun.network/go-perun/wallet"
)

// Peer represents any participant in the off-chain network that the user wants to transact with.
type Peer struct {
	// Name assigned by user for referring to this peer in api requests to the node.
	// It is unique within a session on the node.
	Alias string

	OffchainID       peer.Address // Permanent identity used for authenticating the peer in the off-chain network.
	Transport        Transport    // Transport Layer protocol used for commmunicating with a peer in the off-chain network.
	TransportBackend              // Defines the methods required for using a particular transport protocol.
}

// Transport represents the transport layer protocol adapter required for off-chain comunication.
type Transport struct {
	Addr net.Addr // Transport layer protocol address
	Type string   // Type of transport layer protocol
}

// TransportBackend defines the set of methods required for communicating with the peer on any transport layer protocol.
type TransportBackend interface {
	// Parse a network address from string.
	ParseAddress(string) net.Addr

	// Initialize a listener for a the given network address.
	NewListener(net.Addr) (peer.Listener, error)

	// Initialize a dialer for the given network address.
	NewDialer(net.Addr) (peer.Dialer, error)
}

// User represents a participant in the off-chain network that uses a session on this node for sending transactions.
type User struct {
	Peer

	OnChainAcc    wallet.Account // Account for funding the channel and the on-chain transactions.
	OnChainWallet wallet.Wallet  // Wallet that stores the keys corresponding to on-chain account.

	OffchainAcc wallet.Account // Account (corresponding to off-chain ID) used for signing authentication messages.

	// Wallet that stores keys corresponding to off-chain account. It is also used for creating
	// and storing keys corresponding to ephemeral participant IDs (used for participating in a channel).
	// These keys will be used for signing state updates.
	OffChainWallet wallet.Wallet
}

// Session provides a context for the user to interact with a node. It manages user data (such as IDs, contacts),
// configured protocol services and backends.
//
// Once established, a user can establish and transact on state channels. All the channels within a session will use
// the saame type and version of protocol or service. If a user desires to use multiple types or versions of
// any protocol or service, it should request a seprate session for each combination of type and version of those.
type Session struct {
	ID   string //ID is the unique identifier each instance of session.
	User User

	OnChainTxBackend OnChainTxBackend
	ProtocolService  ProtocolService
}

// ProtocolService allows the user to establish off-chain channels and transact on these channels.
// A channel is closed when a close is initiated either by the user or other channel participants in the channel.
//
// This service allows the user to enable persistence, where all data pertaining to the lifecycle of a channel is
// persisted continuously. When it is enabled, the service can be stopped at any point of time and resumed later.
//
// However, the protocol service is not responsible if any channel the user was participating in was closed
// with a wrong state when the protocol service was not running.
// Hence it is highly recommended not to stop the protocol service if there are open channels.
type ProtocolService interface {
	Listen(listener peer.Listener)

	ProposeChannel(ctx context.Context, req *client.ChannelProposal) (*client.Channel, error)
	Handle(ph client.ProposalHandler, uh client.UpdateHandler)
	Channel(id channel.ID) (*client.Channel, error)
	Close() error

	EnablePersistence(pr persistence.PersistRestorer)
	OnNewChannel(handler func(*client.Channel))
	Reconnect(ctx context.Context) error

	Log() perunLog.Logger
}

// OnChainTxBackend wraps the methods required for instanting and using components for
// making on-chain transactions and reading on-chain values on a specific blockchain platform.
//
// It defines methods for deploying contracts; validating deployed contracts and instantiating a funder, adjudicator.
type OnChainTxBackend interface {
	DeployAdjudicator(ctx context.Context) (wallet.Address, error)
	DeployAsset(ctx context.Context, adjAddr wallet.Address) (wallet.Address, error)
	ValidateContracts(adjAddr, assetAddr wallet.Address) error
	NewFunder(assetAddr wallet.Address) channel.Funder
	NewAdjudicator(adjAddr, receiverAddr wallet.Address) channel.Adjudicator
}

// WalletBackend wraps the methods for instanting wallets and accounts that are specific to a blockchain platform.
type WalletBackend interface {
	NewWallet(keystore string, password string) (wallet.Wallet, error)
	NewAccount(wallet.Wallet, wallet.Address) (wallet.Account, error)
}
