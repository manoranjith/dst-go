package idprovider

// Error type is used to define error constants for this package.
type Error string

// Error implements error interface.
func (e Error) Error() string {
	return string(e)
}

// Definition of error constants for this package.
const (
	ErrPeerIDNotFound          Error = "peer id not found"
	ErrPeerAliasAlreadyUsed    Error = "peer alias is already used for another peer id"
	ErrPeerIDAlreadyRegistered Error = "peer id already regsitered"
	ErrParsingOffChainAddress  Error = "parsing off-chain address"
)
