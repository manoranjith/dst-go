package node

// Config represents the configuratio parameters for the node.
type Config struct {
	LogLevel string
	LogFile  string

	ChainAddr       string   // Address of the default blockchain node used by the perun node.
	AdjudicatorAddr string   // Address of the default Adjudicator contract used by the perun node.
	AssetAddr       string   // Address of the default Asset Holder contract used by the perun node.
	CommTypes       []string // Communication protocols supported by the node for off-chain communication.
	ContactTypes    []string // Contacts Provider backends supported by the node.
	Currencies      []string // Currencies supported by the node.
}
