package config

type CommProtocol string

const (
	TCPIP CommProtocol = "tcpip"
)

type PeerConfig struct {
	Alias       string
	PerunID     string
	HostName    string
	PortNumber  uint32
	CommAdapter CommProtocol
	OnChainID   string
	WalletPath  string
}

// ChannelTimeout specifies different timeouts required for channel operation.
// Timeouts are specified in seconds and can take a max value of 65535 ~=18 Hours.
// What timeouts are relevant ?
// Fund - The channel should be funded by all peers within this timeout.
// Response - Peer should respond to any request within this timeout.
//			- This includes receiving the request, informing the user via rpc and returning the response.
// Handle - Use Peer Response timeout ?
// Dial - Used for dialer - should it not be same as PeerResponseTimeout ?
//		- Because, a node necessarily need not accept all connections, it can also involve user approval.
// OnChainChallenge - Funds can be withdrawn after this timeout.
// Transaction -
// Settle - Use OnChainChallenge + 2 * Blockchain Tx timeout ?
// Because this involves in worst case (two on chain operations - register, withdraw
// and waiting out challenge duration
type ChannelTimeout struct {
	Fund             uint16
	Response         uint16
	OnChainChallenge uint16
}

type BlockchainConfig struct {
	NodeURL            string
	AdjudicatorAddr    string
	AssetHolderAddr    string
	TransactionTimeout uint16
}
type NodeConfig struct {
	ClientID string
	Owner    PeerConfig
}
