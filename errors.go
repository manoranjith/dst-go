package perun

import (
	"fmt"

	"github.com/pkg/errors"
)

type APIError struct {
	Type string // The error should be one of the known errors.
	Info string // Info field contains additional information about the error.
}

func (e APIError) Error() string {
	return fmt.Sprintf("%s. Info: %s", e.Type, e.Info)
}

func NewAPIError(errType string, err error) error {
	if err == nil {
		return errors.WithStack(APIError{Type: errType})
	}
	return errors.WithStack(APIError{Type: errType, Info: err.Error()})
}

var (
	ErrUnknownSessionID  = "No session corresponding to the specified ID."
	ErrUnknownProposalID = "No proposal corresponding to the specified ID."
	ErrUnknownChannelID  = "No payment channel corresponding to the specified ID."
	ErrUnknownAlias      = "Peer corresponding to the specified ID not found in contacts provider." // Used
	ErrUnknownVersionID  = "No pending payment request with the specified version of state."
	ErrUnknownCurrency   = "Currency not supported by this node instance."

	ErrInsufficientBal     = "Insufficient balance in sender account."
	ErrInvalidAmount       = "Invalid amount string."
	ErrInvalidBalance      = "Unknown currency or invalid amount string."
	ErrInvalidConfig       = "Invalid configuration detected."
	ErrInvalidOffChainAddr = "Invalid off-chain address string." // Used

	ErrNoActiveSub      = "No active subscription was found."
	ErrSubAlreadyExists = "A subscription for this context already exists."

	ErrPeerAliasInUse = "Alias already used by another peer in the contacts." // Used
	ErrPeerExists     = "Peer already available in the contacts provider."    // Used

	// ErrPeerNotResponding   = "Peer did not respond within expected timeout."
	// ErrRespTimeoutExpired  = "Response to the notification was sent after the timeout has expired."
	ErrPeerRejected = "The request was rejected by the peer. Reason for rejection should be included in the error information."

	ErrUnclosedPayCh  = "Session cannot be closed (without force option as there are unclosed channels."
	ErrInternalServer = "Internal Server Error"
)
