package user_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
	ethereum "github.com/direct-state-transfer/dst-go/blockchain/ethereum/test"
	"github.com/direct-state-transfer/dst-go/user"
)

func Test_NewUnlockedUser(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	setup := ethereum.NewUserSetup(t, rng)

	type args struct {
		wb  dst.WalletBackend
		cfg user.Config
	}
	tests := []struct {
		name    string
		args    args
		want    dst.User
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				wb: setup.WalletBackend,
				cfg: user.Config{
					Alias: setup.User.Alias,
					OnChainWallet: user.WalletConfig{
						Addr:         setup.User.OnChainAcc.Address(),
						KeystorePath: setup.KeystorePath,
						Password:     ""},
					OffChainWallet: user.WalletConfig{
						Addr:         setup.User.OffchainID,
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
			wantErr: false,
		},
		{
			name: "invalid-onchain-password",
			args: args{
				wb: setup.WalletBackend,
				cfg: user.Config{
					Alias: setup.User.Alias,
					OnChainWallet: user.WalletConfig{
						Addr:         setup.User.OnChainAcc.Address(),
						KeystorePath: setup.KeystorePath,
						Password:     "invalid-password"},
					OffChainWallet: user.WalletConfig{
						Addr:         setup.User.OffchainID,
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
			wantErr: true,
		},
		{
			name: "valid-onchain-invalid-offchain-password",
			args: args{
				wb: setup.WalletBackend,
				cfg: user.Config{
					Alias: setup.User.Alias,
					OnChainWallet: user.WalletConfig{
						Addr:         setup.User.OnChainAcc.Address(),
						KeystorePath: setup.KeystorePath,
						Password:     ""},
					OffChainWallet: user.WalletConfig{
						Addr:         setup.User.OffchainID,
						KeystorePath: setup.KeystorePath,
						Password:     "invalid-pwd"}}},
			wantErr: true,
		},
		{
			name: "invalid-keystore-path",
			args: args{
				wb: setup.WalletBackend,
				cfg: user.Config{
					Alias: setup.User.Alias,
					OnChainWallet: user.WalletConfig{
						Addr:         setup.User.OnChainAcc.Address(),
						KeystorePath: "invalid-keystore-path",
						Password:     ""},
					OffChainWallet: user.WalletConfig{
						Addr:         setup.User.OffchainID,
						KeystorePath: setup.KeystorePath,
						Password:     ""}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := user.New(tt.args.wb, tt.args.cfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Zero(t, got)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, got)
			}
		})
	}
}
