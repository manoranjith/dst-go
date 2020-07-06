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

package implementation

import (
	"context"
	"time"

	"github.com/pkg/errors"

	ethchannel "perun.network/go-perun/backend/ethereum/channel"
	ethwallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
)

// OnChainTxBackend provides ethereum specific contract backend functionality.
type OnChainTxBackend struct {
	// Cb is the instance of contract backend that will be used for all onchain communications.
	Cb *ethchannel.ContractBackend
}

// NewFunder initializes and returns an instance of ethereum funder.
func (cb *OnChainTxBackend) NewFunder(assetAddr wallet.Address) channel.Funder {
	return newFunder(*cb.Cb, assetAddr)
}

// NewAdjudicator initializes and returns an instance of ethereum adjudicator.
func (cb *OnChainTxBackend) NewAdjudicator(adjAddr, receiverAddr wallet.Address) channel.Adjudicator {
	return newAdjudicator(*cb.Cb, adjAddr, receiverAddr)
}

// ValidateContracts validates the integrity of given adjudicator and asset holder contracts.
func (cb *OnChainTxBackend) ValidateContracts(adjAddr, assetAddr wallet.Address) error {
	return validateContracts(*cb.Cb, adjAddr, assetAddr)
}

// DeployAdjudicator deploys the adjudicator contract.
func (cb *OnChainTxBackend) DeployAdjudicator(ctx context.Context) (wallet.Address, error) {
	return deployAdjudicator(ctx, *cb.Cb)
}

// DeployAsset deploys the asset holder contract, setting the adjudicator address to given value.
func (cb *OnChainTxBackend) DeployAsset(ctx context.Context, adjAddr wallet.Address) (wallet.Address, error) {
	return deployAsset(ctx, *cb.Cb, adjAddr)
}

func newFunder(cb ethchannel.ContractBackend, assetAddr wallet.Address) channel.Funder {
	return ethchannel.NewETHFunder(cb, ethwallet.AsEthAddr(assetAddr))
}

func newAdjudicator(cb ethchannel.ContractBackend, adjAddr, receiverAddr wallet.Address) channel.Adjudicator {
	return ethchannel.NewAdjudicator(cb, ethwallet.AsEthAddr(adjAddr), ethwallet.AsEthAddr(receiverAddr))
}

func validateContracts(cb ethchannel.ContractBackend, adjAddr, assetAddr wallet.Address) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Integrity of Adjudicator is implicitly done during validation of asset holder contract.
	err := ethchannel.ValidateAssetHolderETH(ctx, cb, ethwallet.AsEthAddr(assetAddr), ethwallet.AsEthAddr(adjAddr))
	if ethchannel.IsContractBytecodeError(err) {
		return errors.Wrap(err, "invalid contracts at given address")
	}
	return errors.Wrap(err, "validating contracts")
}

func deployAdjudicator(ctx context.Context, cb ethchannel.ContractBackend) (wallet.Address, error) {
	addr, err := ethchannel.DeployAdjudicator(ctx, cb)
	return ethwallet.AsWalletAddr(addr), err
}

func deployAsset(ctx context.Context, cb ethchannel.ContractBackend, adjAddr wallet.Address) (wallet.Address, error) {
	addr, err := ethchannel.DeployETHAssetholder(ctx, cb, ethwallet.AsEthAddr(adjAddr))
	return ethwallet.AsWalletAddr(addr), err
}
