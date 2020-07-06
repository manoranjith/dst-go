// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/direct-state-transfer/dst-go
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package implementation_test

import (
	"context"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/direct-state-transfer/dst-go"
	"github.com/direct-state-transfer/dst-go/blockchain/ethereum/internal/implementation"
	ethereumtest "github.com/direct-state-transfer/dst-go/blockchain/ethereum/test"
)

func Test_OnChainTxBackend_Interface(t *testing.T) {
	assert.Implements(t, (*dst.OnChainTxBackend)(nil), new(implementation.OnChainTxBackend))
}

func Test_OnChainTxBackend_Deploy(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	setup := ethereumtest.NewOnChainTxBackendSetup(t, rng, 1)

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
	setup := ethereumtest.NewOnChainTxBackendSetup(t, rng, 1)

	t.Run("happy", func(t *testing.T) {
		assert.NoError(t, setup.OnChainTxBackend.ValidateContracts(setup.AdjAddr, setup.AssetAddr))
	})
	t.Run("invalid-random-addrs", func(t *testing.T) {
		randomAddr1 := ethereumtest.NewRandomAddress(rng)
		randomAddr2 := ethereumtest.NewRandomAddress(rng)
		assert.Error(t, setup.OnChainTxBackend.ValidateContracts(randomAddr1, randomAddr2))
	})
}
