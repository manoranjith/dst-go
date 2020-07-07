package user

import "perun.network/go-perun/wallet"

// WalletConfig defines the parameters required to configure a wallet.
type WalletConfig struct {
	Addr         wallet.Address
	KeystorePath string
	Password     string
}

// Config defines the parameters required to configure a user.
type Config struct {
	Alias string

	OnChainWallet  WalletConfig
	OffChainWallet WalletConfig
}
