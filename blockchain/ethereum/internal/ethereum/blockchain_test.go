package ethereum_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
	ethereum "github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/ethereum"
	ethereumtest "github.com/direct-state-transfer/dst-go/blockchain/ethereum/test"
)

func Test_OnChainTxBackend_Interface(t *testing.T) {
	assert.Implements(t, (*dst.OnChainTxBackend)(nil), new(ethereum.OnChainTxBackend))
}

func Test_OnChainTxBackend_Deploy(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	setup := ethereumtest.NewOnChainTxBackendSetup(t, rng)

	ctx, cancel := context.WithTimeout(context.Background(), ethereumtest.DefaultTxTimeout)
	defer cancel()

	adjAddr, err := setup.OnChainTxBackend.DeployAdjudicator(ctx)
	require.NoError(t, err)
	assetAddr, err := setup.OnChainTxBackend.DeployAsset(ctx, adjAddr)
	require.NoError(t, err)
	assert.NoError(t, setup.OnChainTxBackend.ValidateContracts(adjAddr, assetAddr))

}

func Test_OnChainTxBackend_ValidateContracts(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	setup := ethereumtest.NewOnChainTxBackendSetup(t, rng)

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, setup.OnChainTxBackend.ValidateContracts(setup.AdjAddr, setup.AssetAddr))
	})
	t.Run("invalid-random-addrs", func(t *testing.T) {
		randomAddr1 := ethereumtest.NewRandomAddress(rng)
		randomAddr2 := ethereumtest.NewRandomAddress(rng)
		assert.Error(t, setup.OnChainTxBackend.ValidateContracts(randomAddr1, randomAddr2))
	})
}
