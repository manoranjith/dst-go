// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/hyperledger-labs/perun-node
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

package payment_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ppayment "perun.network/go-perun/apps/payment"
	pchannel "perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/app/payment"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
)

const peerAlias = "peer"

func Test_OpenPayCh(t *testing.T) {
	bals := make(map[string]string)
	bals[perun.OwnAlias] = "1"
	bals[peerAlias] = "2"
	balInfo := perun.BalInfo{
		Currency: "ETH",
		Bals:     bals,
	}
	app := perun.App{
		Def:  ppayment.AppDef(),
		Data: &ppayment.NoData{},
	}
	var challengeDurSecs uint64 = 10
	chInfo := perun.ChannelInfo{
		ChannelID: "channel1",
		Currency:  currency.ETH,
		State: &pchannel.State{
			Allocation: pchannel.Allocation{
				Balances: [][]*big.Int{{big.NewInt(1e18), big.NewInt(2e18)}},
			},
		},
		Parts: []string{perun.OwnAlias, peerAlias},
	}

	t.Run("happy", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("OpenCh", context.Background(), peerAlias, balInfo, app, challengeDurSecs).Return(chInfo, nil)

		wantBals := make(map[string]string)
		wantBals[perun.OwnAlias] = "1.000000"
		wantBals[peerAlias] = "2.000000"
		wantBalInfo := perun.BalInfo{
			Currency: "ETH",
			Bals:     wantBals,
		}

		gotPayChInfo, gotErr := payment.OpenPayCh(context.Background(), sessionAPI, peerAlias, balInfo, challengeDurSecs)
		require.NoError(t, gotErr)
		assert.Equal(t, wantBalInfo, gotPayChInfo.BalInfo)
		assert.Equal(t, "0", gotPayChInfo.Version)
		assert.NotZero(t, gotPayChInfo.ChannelID)
	})

	t.Run("error", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("OpenCh", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			perun.ChannelInfo{}, assert.AnError)

		_, gotErr := payment.OpenPayCh(context.Background(), sessionAPI, peerAlias, balInfo, challengeDurSecs)
		require.Error(t, gotErr)
	})
}
