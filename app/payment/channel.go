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

package payment

import (
	"context"
	"fmt"

	pchannel "perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
)

type (
	// PayChInfo represents the interpretation of channelInfo for payment app.
	PayChInfo struct {
		ChannelID string
		BalInfo   perun.BalInfo
		Version   string
	}
	// PayChUpdateNotifier represents the channel update notification function for payment app.
	PayChUpdateNotifier func(PayChUpdateNotif)

	// PayChUpdateNotif represents the channel update notification data for payment app.
	PayChUpdateNotif struct {
		UpdateID     string
		ProposedBals perun.BalInfo
		Version      string
		Final        bool
		Currency     string
		Parts        []string
		Timeout      int64
	}
)

// GetBalInfo returns the balance information for this channel.
func GetBalInfo(ch perun.ChannelAPI) perun.BalInfo {
	chInfo := ch.GetInfo()
	return balsFromState(chInfo.Currency, chInfo.State, chInfo.Parts)
}

// SendPayChUpdate send the given amount to the payee. Payee should be one of the channel participants.
// Use "self" to request payments.
func SendPayChUpdate(pctx context.Context, ch perun.ChannelAPI, payee, amount string) error {
	chInfo := ch.GetInfo()
	f, err := newUpdater(chInfo.State, chInfo.Parts, chInfo.Currency, payee, amount)
	if err != nil {
		return err
	}
	return ch.SendChUpdate(pctx, f)
}

func newUpdater(currState *pchannel.State, parts []string, chCurrency, payee, amount string) (
	perun.StateUpdater, error) {
	parsedAmount, err := currency.NewParser(chCurrency).Parse(amount)
	if err != nil {
		return nil, perun.ErrInvalidAmount
	}

	// find index
	var payerIdx, payeeIdx int
	if parts[0] == payee {
		payeeIdx = 0
	} else if parts[1] == payee {
		payeeIdx = 1
	} else {
		return nil, perun.ErrInvalidPayee
	}
	payerIdx = payeeIdx ^ 1

	// check sufficient balance
	bals := currState.Allocation.Clone().Balances[0]
	bals[payerIdx].Sub(bals[payerIdx], parsedAmount)
	bals[payeeIdx].Add((bals[payeeIdx]), parsedAmount)
	if bals[payerIdx].Sign() == -1 {
		return nil, perun.ErrInsufficientBal
	}

	// return updater func
	return func(state *pchannel.State) {
		state.Allocation.Balances[0][payerIdx] = bals[payerIdx]
		state.Allocation.Balances[0][payeeIdx] = bals[payeeIdx]
	}, nil
}

// SubPayChUpdates registers a subscription for updates on this channel.
func SubPayChUpdates(ch perun.ChannelAPI, notifier PayChUpdateNotifier) error {
	return ch.SubChUpdates(func(notif perun.ChUpdateNotif) {
		notifier(PayChUpdateNotif{
			UpdateID:     notif.UpdateID,
			ProposedBals: balsFromState(notif.Currency, notif.Update.State, notif.Parts),
			Version:      fmt.Sprintf("%d", notif.Update.State.Version),
			Final:        notif.Update.State.IsFinal,
			Timeout:      notif.Expiry,
		})
	})
}

// UnsubPayChUpdates unregisters a subscription for updates on this channel.
func UnsubPayChUpdates(ch perun.ChannelAPI) error {
	return ch.UnsubChUpdates()
}

// RespondPayChUpdate sends a response for a channel update notification.
func RespondPayChUpdate(pctx context.Context, ch perun.ChannelAPI, updateID string, accept bool) error {
	return ch.RespondChUpdate(pctx, updateID, accept)
}

// ClosePayCh closes the payment channel.
func ClosePayCh(pctx context.Context, ch perun.ChannelAPI) (PayChInfo, error) {
	chInfo, err := ch.Close(pctx)
	if err != nil {
		return PayChInfo{}, err
	}
	return PayChInfo{
		ChannelID: chInfo.ChannelID,
		BalInfo:   balsFromState(chInfo.Currency, chInfo.State, chInfo.Parts),
		Version:   fmt.Sprintf("%d", chInfo.State.Version),
	}, nil
}
