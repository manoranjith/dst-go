package perun

import (
	"errors"
)

// APIError represents the errors that will communicated via the user API.
type APIError string

func (e APIError) Error() string {
	return string(e)
}

// GetAPIError returns the APIError contained in err if err is an APIError.
// If not, it returns ErrInternalServer API error.
func GetAPIError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		return ErrInternalServer
	}
	return apiErr
}

// Setinal Error values that are relevant for the end user of the node.
var (
	ErrUnknownSessionID  = APIError("No session corresponding to the specified ID.")
	ErrUnknownProposalID = APIError("No channel proposal corresponding to the specified ID.")
	ErrUnknownChannelID  = APIError("No channel corresponding to the specified ID.")
	ErrUnknownAlias      = APIError("No peer corresponding to the specified ID was found in contacts.")
	ErrUnknownUpdateID   = APIError("No response was expected for the given channel update ID")

	ErrUnsupportedCurrency     = APIError("Currency not supported by this node instance.")
	ErrUnsupportedContactsType = APIError("Contacts type not supported by this node instance.")
	ErrUnsupportedCommType     = APIError("Communication protocol not supported by this node instance.")

	ErrInsufficientBal     = APIError("Insufficient balance in sender account.")
	ErrInvalidAmount       = APIError("Invalid amount string")
	ErrInvalidConfig       = APIError("Invalid configuration detected.")
	ErrInvalidOffChainAddr = APIError("Invalid off-chain address string.")
	ErrInvalidPayee        = APIError("Invalid payee, no such participant in the channel.")

	ErrNoActiveSub      = APIError("No active subscription was found.")
	ErrSubAlreadyExists = APIError("A subscription for this context already exists.")

	ErrPeerAliasInUse     = APIError("Alias already used by another peer in the contacts.")
	ErrPeerExists         = APIError("Peer already available in the contacts provider.")
	ErrRespTimeoutExpired = APIError("Response to the notification was sent after the timeout has expired.")
	ErrPeerRejected       = APIError("The request was rejected by the peer.")

	ErrUnclosedCh     = APIError("Session cannot be closed (without force option as there are unclosed channels.")
	ErrInternalServer = APIError("Internal Server Error")
)
