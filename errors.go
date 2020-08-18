package perun

import (
	"fmt"
)

// Setinal Error values that are relevant for the end user of the node.
var (
	ErrUnknownSessionID  = fmt.Errorf("No session corresponding to the specified ID.")
	ErrUnknownProposalID = fmt.Errorf("No channel proposal corresponding to the specified ID.")
	ErrUnknownChannelID  = fmt.Errorf("No channel corresponding to the specified ID.")
	ErrUnknownAlias      = fmt.Errorf("No peer corresponding to the specified ID was found in contacts.")
	ErrUnknownUpdateID   = fmt.Errorf("No response was expected for the given channel update ID")

	ErrUnsupportedCurrency     = fmt.Errorf("Currency not supported by this node instance.")
	ErrUnsupportedContactsType = fmt.Errorf("Contacts type not supported by this node instance.")
	ErrUnsupportedCommType     = fmt.Errorf("Communication protocol not supported by this node instance.")

	ErrInsufficientBal     = fmt.Errorf("Insufficient balance in sender account.")
	ErrInvalidAmount       = fmt.Errorf("Invalid amount string.")
	ErrInvalidConfig       = fmt.Errorf("Invalid configuration detected.")
	ErrInvalidOffChainAddr = fmt.Errorf("Invalid off-chain address string.")

	ErrNoActiveSub      = fmt.Errorf("No active subscription was found.")
	ErrSubAlreadyExists = fmt.Errorf("A subscription for this context already exists.")

	ErrPeerAliasInUse = fmt.Errorf("Alias already used by another peer in the contacts.")
	ErrPeerExists     = fmt.Errorf("Peer already available in the contacts provider.")

	// ErrPeerNotResponding   = "Peer did not respond within expected timeout."
	// ErrRespTimeoutExpired  = "Response to the notification was sent after the timeout has expired."
	ErrPeerRejected = fmt.Errorf("The request was rejected by the peer.")

	ErrUnclosedPayCh  = fmt.Errorf("Session cannot be closed (without force option as there are unclosed channels.")
	ErrInternalServer = fmt.Errorf("Internal Server Error")
)
