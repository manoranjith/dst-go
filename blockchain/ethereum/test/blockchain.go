package ethereum

import (
	"context"
	"math/rand"
	"testing"
	"time"

	internal "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/ethereum"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethchanneltest "perun.network/go-perun/backend/ethereum/channel/test"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/wallet"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
)

// DefaultTxTimeout is the default transaction timeout for simulated backend.
const DefaultTxTimeout = 5 * time.Second

// OnChainTxBackendSetup is a test setup that uses a simulated blockchain backend (for details on this backend,
// see go-ethereum) with required contracts deployed on it and a UserSetup.
type OnChainTxBackendSetup struct {
	*UserSetup
	OnChainTxBackend   dst.OnChainTxBackend
	AdjAddr, AssetAddr wallet.Address
}

// NewOnChainTxBackendSetup initializes and returns a test OnChainTxBackend using the given randomness.
func NewOnChainTxBackendSetup(t *testing.T, rng *rand.Rand) (_ *OnChainTxBackendSetup) {
	userSetup := NewUserSetup(t, rng)

	cbEth := newSimContractBackend(userSetup.User, userSetup.Keystore)
	cb := &internal.OnChainTxBackend{Cb: &cbEth}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultTxTimeout)
	defer cancel()
	adjudicator, err := ethchannel.DeployAdjudicator(ctx, cbEth)
	require.NoError(t, err)
	asset, err := ethchannel.DeployETHAssetholder(ctx, cbEth, adjudicator)
	require.NoError(t, err)

	// No cleanup required.
	return &OnChainTxBackendSetup{
		UserSetup:        userSetup,
		OnChainTxBackend: cb,
		AdjAddr:          ethwallet.AsWalletAddr(adjudicator),
		AssetAddr:        ethwallet.AsWalletAddr(asset),
	}
}

func newSimContractBackend(user dst.User, ks *keystore.KeyStore) ethchannel.ContractBackend {
	simBackend := ethchanneltest.NewSimulatedBackend()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTxTimeout)
	defer cancel()
	simBackend.FundAddress(ctx, ethwallet.AsEthAddr(user.OnChainAcc.Address()))

	onchainAcc := &user.OnChainAcc.(*ethwallet.Account).Account
	contractBackend := ethchannel.NewContractBackend(simBackend, ks, onchainAcc)
	return contractBackend
}
