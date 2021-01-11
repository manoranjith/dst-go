package idprovider

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	PeerIDNotFoundError         Error = "Peer ID not found"
	PeerAliasAlreadyUsedError   Error = "Peer alias is already used for another peer ID"
	PeerIDAlreadyRegistered     Error = "Peer ID already exists"
	ParsingOffChainAddressError Error = "Parsing off-chain address"
)
