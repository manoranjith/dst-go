package idprovider

// Error type is used to define error constants for this package.
type Error string

// Error implements error interface.
func (e Error) Error() string {
	return string(e)
}

// Definition of error constants for this package.
const (
	ErrPeerIDNotFound          Error = "Peer ID not found"
	ErrPeerAliasAlreadyUsed    Error = "Peer alias is already used for another peer ID"
	ErrPeerIDAlreadyRegistered Error = "Peer ID already regsitered"
	ErrParsingOffChainAddress  Error = "Parsing off-chain address"
)
