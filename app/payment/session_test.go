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
	pclient "perun.network/go-perun/client"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/app/payment"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
)

var (
	parts                                    []string
	bals, wantBals, wantUpdatedBals          map[string]string
	balInfo, wantBalInfo, wantUpdatedBalInfo perun.BalInfo
	version                                  uint64 = 1
	versionString                            string = "1"
	app                                      perun.App
	chInfo                                   perun.ChannelInfo
	challengeDurSecs                         uint64 = 10
	peerAlias                                       = "peer"
	chProposalNotif                          perun.ChProposalNotif
	chCloseNotif                             perun.ChCloseNotif

	amountToSend  = "0.5"
	chUpdateNotif perun.ChUpdateNotif
)

func init() {
	parts = []string{perun.OwnAlias, peerAlias}
	bals = make(map[string]string)
	bals[perun.OwnAlias] = "1"
	bals[peerAlias] = "2"
	balInfo = perun.BalInfo{
		Currency: "ETH",
		Bals:     bals,
	}
	app = perun.App{
		Def:  ppayment.AppDef(),
		Data: &ppayment.NoData{},
	}
	chInfo = perun.ChannelInfo{
		ChannelID: "channel1",
		Currency:  currency.ETH,
		State: &pchannel.State{
			Allocation: pchannel.Allocation{
				Balances: [][]*big.Int{{big.NewInt(1e18), big.NewInt(2e18)}},
			},
		},
		Parts: []string{perun.OwnAlias, peerAlias},
	}
	wantBals = make(map[string]string)
	wantBals[perun.OwnAlias] = "1.000000"
	wantBals[peerAlias] = "2.000000"
	wantBalInfo = perun.BalInfo{
		Currency: "ETH",
		Bals:     wantBals,
	}

	chProposalNotif = perun.ChProposalNotif{
		ProposalID: "proposal1",
		Currency:   currency.ETH,
		Proposal: &pclient.ChannelProposal{
			ChallengeDuration: challengeDurSecs,
			InitBals: &pchannel.Allocation{
				Balances: [][]*big.Int{{big.NewInt(1e18), big.NewInt(2e18)}},
			},
		},
		Parts:  parts,
		Expiry: 1597946401,
	}

	chCloseNotif = perun.ChCloseNotif{
		ChannelID: "channel1",
		Currency:  currency.ETH,
		ChState: &pchannel.State{
			Allocation: pchannel.Allocation{
				Balances: [][]*big.Int{{big.NewInt(1e18), big.NewInt(2e18)}},
			},
			Version: version,
		},
		Parts: parts,
		Error: "",
	}

	wantUpdatedBals = make(map[string]string)
	wantUpdatedBals[perun.OwnAlias] = "0.500000"
	wantUpdatedBals[peerAlias] = "2.500000"
	wantUpdatedBalInfo = perun.BalInfo{
		Currency: "ETH",
		Bals:     wantUpdatedBals,
	}

	chUpdateNotif = perun.ChUpdateNotif{
		UpdateID: "update1",
		Currency: currency.ETH,
		Update: &pclient.ChannelUpdate{
			State: &pchannel.State{
				Allocation: pchannel.Allocation{
					Balances: [][]*big.Int{{big.NewInt(0.5e18), big.NewInt(2.5e18)}},
				},
				IsFinal: true,
				Version: version,
			},
		},
		Parts:  parts,
		Expiry: 1597946401,
	}
}

func Test_OpenPayCh(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("OpenCh", context.Background(), peerAlias, balInfo, app, challengeDurSecs).Return(chInfo, nil)

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

func Test_GetPayChs(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("GetChInfos").Return([]perun.ChannelInfo{chInfo})

		gotPayChInfos := payment.GetPayChs(sessionAPI)
		require.Len(t, gotPayChInfos, 1)
		assert.Equal(t, "0", gotPayChInfos[0].Version)
		assert.Equal(t, wantBalInfo, gotPayChInfos[0].BalInfo)
		assert.NotZero(t, gotPayChInfos[0].ChannelID)
	})
}

func Test_SubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var notifier perun.ChProposalNotifier
		var notif payment.PayChProposalNotif
		dummyNotifier := func(gotNotif payment.PayChProposalNotif) {
			notif = gotNotif
		}
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("SubChProposals", mock.MatchedBy(func(gotNotifier perun.ChProposalNotifier) bool {
			notifier = gotNotifier
			return true
		})).Return(nil)

		gotErr := payment.SubPayChProposals(sessionAPI, dummyNotifier)
		require.NoError(t, gotErr)
		require.NotNil(t, notifier)

		notifier(chProposalNotif)
		require.NotZero(t, notif)
		assert.Equal(t, chProposalNotif.ProposalID, notif.ProposalID)
		assert.Equal(t, chProposalNotif.Currency, notif.Currency)
		assert.Equal(t, wantBalInfo, notif.OpeningBals)
		assert.Equal(t, chProposalNotif.Proposal.ChallengeDuration, notif.ChallengeDurSecs)
		assert.Equal(t, chProposalNotif.Expiry, notif.Expiry)
	})
	t.Run("error", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("SubChProposals", mock.Anything).Return(assert.AnError)

		dummyNotifier := func(notif payment.PayChProposalNotif) {}
		gotErr := payment.SubPayChProposals(sessionAPI, dummyNotifier)
		assert.Error(t, gotErr)
	})
}

// nolint: dupl	// not duplicate of Test_UnsubPayChCloses.
func Test_UnsubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("UnsubChProposals", mock.Anything).Return(nil)

		gotErr := payment.UnsubPayChProposals(sessionAPI)
		assert.NoError(t, gotErr)
	})
	t.Run("error", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("UnsubChProposals", mock.Anything).Return(assert.AnError)

		gotErr := payment.UnsubPayChProposals(sessionAPI)
		assert.Error(t, gotErr)
	})
}

// nolint: dupl	// not duplicate of Test_RespondPayChUpdate.
func Test_RespondPayChProposal(t *testing.T) {
	proposalID := "proposal-id-1"
	t.Run("happy_accept", func(t *testing.T) {
		accept := true
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("RespondChProposal", context.Background(), proposalID, accept).Return(nil)

		gotErr := payment.RespondPayChProposal(context.Background(), sessionAPI, proposalID, accept)
		assert.NoError(t, gotErr)
	})
	t.Run("happy_reject", func(t *testing.T) {
		accept := false
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("RespondChProposal", context.Background(), proposalID, accept).Return(nil)

		gotErr := payment.RespondPayChProposal(context.Background(), sessionAPI, proposalID, accept)
		assert.NoError(t, gotErr)
	})
	t.Run("error", func(t *testing.T) {
		accept := true
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("RespondChProposal", context.Background(), proposalID, accept).Return(assert.AnError)

		gotErr := payment.RespondPayChProposal(context.Background(), sessionAPI, proposalID, accept)
		assert.Error(t, gotErr)
	})
}

func Test_SubPayChCloses(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		var notifier perun.ChCloseNotifier
		var notif payment.PayChCloseNotif
		dummyNotifier := func(gotNotif payment.PayChCloseNotif) {
			notif = gotNotif
		}
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("SubChCloses", mock.MatchedBy(func(gotNotifier perun.ChCloseNotifier) bool {
			notifier = gotNotifier
			return true
		})).Return(nil)

		gotErr := payment.SubPayChCloses(sessionAPI, dummyNotifier)
		require.NoError(t, gotErr)
		require.NotNil(t, notifier)
		notifier(chCloseNotif)
		require.NotZero(t, notif)
		assert.Equal(t, chCloseNotif.ChannelID, notif.ClosingState.ChannelID)
		assert.Equal(t, wantBalInfo, notif.ClosingState.BalInfo)
		assert.Equal(t, versionString, notif.ClosingState.Version)
	})
	t.Run("error", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("SubChCloses", mock.Anything).Return(assert.AnError)

		dummyNotifier := func(notif payment.PayChCloseNotif) {}
		gotErr := payment.SubPayChCloses(sessionAPI, dummyNotifier)
		assert.Error(t, gotErr)
	})
}

// nolint: dupl	// not duplicate of Test_UnsubPayChProposals.
func Test_UnsubPayChCloses(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("UnsubChCloses", mock.Anything).Return(nil)

		gotErr := payment.UnsubPayChCloses(sessionAPI)
		assert.NoError(t, gotErr)
	})
	t.Run("error", func(t *testing.T) {
		sessionAPI := &mocks.SessionAPI{}
		sessionAPI.On("UnsubChCloses", mock.Anything).Return(assert.AnError)

		gotErr := payment.UnsubPayChCloses(sessionAPI)
		assert.Error(t, gotErr)
	})
}
