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

package session

import (
	"context"
	"fmt"
	"math/big"
	"time"

	pchannel "perun.network/go-perun/channel"
	pclient "perun.network/go-perun/client"
	psync "perun.network/go-perun/pkg/sync"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/log"
)

const (
	open   chStatus = "open"
	closed chStatus = "closed"
)

type (
	channel struct {
		log.Logger

		id               string
		pch              *pclient.Channel
		lockState        chStatus
		currency         string
		parts            []string
		timeoutCfg       timeoutConfig
		challengeDurSecs uint64
		currState        *pchannel.State

		chUpdateNotifier   perun.ChUpdateNotifier
		chUpdateNotifCache []perun.ChUpdateNotif
		chUpdateResponders map[string]chUpdateResponderEntry

		psync.Mutex
	}

	chStatus string

	chUpdateResponderEntry struct {
		notif       perun.ChUpdateNotif
		responder   chUpdateResponder
		notifExpiry int64
	}

	//go:generate mockery --name ProposalResponder --output ../internal/mocks

	// ChUpdaterResponder represents the methods on channel update responder that will be used the perun node.
	chUpdateResponder interface {
		Accept(ctx context.Context) error
		Reject(ctx context.Context, reason string) error
	}
)

// newCh sets up a channel object from the passed pchannel.
func newCh(pch *pclient.Channel, currency string, parts []string, timeoutCfg timeoutConfig,
	challengeDurSecs uint64) *channel {
	ch := &channel{
		id:                 fmt.Sprintf("%x", pch.ID()),
		pch:                pch,
		lockState:          open,
		currState:          pch.State().Clone(),
		timeoutCfg:         timeoutCfg,
		challengeDurSecs:   challengeDurSecs,
		currency:           currency,
		parts:              parts,
		chUpdateResponders: make(map[string]chUpdateResponderEntry),
	}
	go func(ch *channel) {
		ch.Debug("Started channel watcher")
		err := ch.pch.Watch()
		ch.HandleClose(err)
	}(ch)
	return ch
}

func (ch *channel) ID() string {
	return ch.id
}

func (ch *channel) SendChUpdate(pctx context.Context, updater perun.StateUpdater) (perun.ChInfo, error) {
	ch.Debug("Received request: channel.SendChUpdate")
	ch.Lock()
	defer ch.Unlock()

	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.chUpdate())
	defer cancel()
	err := ch.pch.UpdateBy(ctx, ch.pch.Idx(), updater)
	if err != nil {
		ch.Error("Sending channel update:", err)
		return perun.ChInfo{}, perun.GetAPIError(err)
	}
	prevChInfo := ch.getChInfo()
	ch.currState = ch.pch.State().Clone()
	ch.Debugf("State upated from %v to %v", prevChInfo, ch.getChInfo())
	return ch.getChInfo(), nil
}

func (ch *channel) HandleUpdate(chUpdate pclient.ChannelUpdate, responder *pclient.UpdateResponder) {
	ch.Lock()
	defer ch.Unlock()

	expiry := time.Now().UTC().Add(ch.timeoutCfg.response).Unix()
	notif := makeChUpdateNotif(ch.getChInfo(), chUpdate.State, expiry)
	entry := chUpdateResponderEntry{
		notif:       notif,
		responder:   responder,
		notifExpiry: expiry,
	}

	// Need not store entries for notification with expiry = 0, as these update requests have
	// already been rejected by the perun node. Hence no response is expected for these notifications.
	if expiry != 0 {
		ch.chUpdateResponders[notif.UpdateID] = entry
	}

	if ch.chUpdateNotifier == nil {
		ch.chUpdateNotifCache = append(ch.chUpdateNotifCache, notif)
		ch.Debug("HandleUpdate: Notification cached")
	} else {
		go ch.chUpdateNotifier(notif)
		ch.Debug("HandleUpdate: Notification sent")
	}
}

func makeChUpdateNotif(currChInfo perun.ChInfo, proposedState *pchannel.State, expiry int64) perun.ChUpdateNotif {
	var chUpdateType perun.ChUpdateType
	switch proposedState.IsFinal {
	case true:
		chUpdateType = perun.ChUpdateTypeFinal
	case false:
		chUpdateType = perun.ChUpdateTypeOpen
	}
	return perun.ChUpdateNotif{
		UpdateID:       fmt.Sprintf("%s_%d", currChInfo.ChID, proposedState.Version),
		CurrChInfo:     currChInfo,
		ProposedChInfo: makeChInfo(currChInfo.ChID, currChInfo.BalInfo.Parts, currChInfo.BalInfo.Currency, proposedState),
		Type:           chUpdateType,
		Expiry:         expiry,
		Error:          "",
	}
}

func (ch *channel) SubChUpdates(notifier perun.ChUpdateNotifier) error {
	ch.Debug("Received request: channel.SubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier != nil {
		ch.Error(perun.ErrSubAlreadyExists)
		return perun.ErrSubAlreadyExists
	}
	ch.chUpdateNotifier = notifier

	// Send all cached notifications
	for i := len(ch.chUpdateNotifCache); i > 0; i-- {
		go ch.chUpdateNotifier(ch.chUpdateNotifCache[0])
		ch.chUpdateNotifCache = ch.chUpdateNotifCache[1:i]
	}
	return nil
}

func (ch *channel) UnsubChUpdates() error {
	ch.Debug("Received request: channel.UnsubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier == nil {
		ch.Error(perun.ErrNoActiveSub)
		return perun.ErrNoActiveSub
	}
	ch.chUpdateNotifier = nil
	return nil
}

func (ch *channel) RespondChUpdate(pctx context.Context, updateID string, accept bool) (perun.ChInfo, error) {
	ch.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	entry, ok := ch.chUpdateResponders[updateID]
	if !ok {
		ch.Error(perun.ErrUnknownUpdateID, updateID)
		return perun.ChInfo{}, perun.ErrUnknownUpdateID
	}
	delete(ch.chUpdateResponders, updateID)

	currTime := time.Now().UTC().Unix()
	if entry.notifExpiry < currTime {
		ch.Error("timeout:", entry.notifExpiry, "received response at:", currTime)
		return perun.ChInfo{}, perun.ErrRespTimeoutExpired
	}

	var err error
	switch accept {
	case true:
		err = ch.acceptChUpdate(pctx, entry)
		if err == nil && entry.notif.Type == perun.ChUpdateTypeFinal {
			ch.Info("Responded to update successfully, settling the state as it was final update.")
			err = ch.settleSecondary(pctx)
		}
	case false:
		err = ch.rejectChUpdate(pctx, entry, "rejected by user")
	}
	return ch.getChInfo(), err
}

func (ch *channel) acceptChUpdate(pctx context.Context, entry chUpdateResponderEntry) error {
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.respChUpdate())
	defer cancel()
	err := entry.responder.Accept(ctx)
	if err != nil {
		ch.Error("Accepting channel update", err)
	} else {
		ch.currState = ch.pch.State().Clone()
	}
	return perun.GetAPIError(errors.Wrap(err, "accepting update"))
}

func (ch *channel) rejectChUpdate(pctx context.Context, entry chUpdateResponderEntry, reason string) error {
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.respChUpdate())
	defer cancel()
	err := entry.responder.Reject(ctx, reason)
	if err != nil {
		ch.Error("Rejecting channel update", err)
	}
	return perun.GetAPIError(errors.Wrap(err, "rejecting update"))
}

func (ch *channel) GetChInfo() perun.ChInfo {
	ch.Debug("Received request: channel.GetChInfo")
	ch.Lock()
	defer ch.Unlock()
	return ch.getChInfo()
}

// This function assumes that caller has already locked the channel.
func (ch *channel) getChInfo() perun.ChInfo {
	return makeChInfo(ch.ID(), ch.parts, ch.currency, ch.currState)
}

func makeChInfo(chID string, parts []string, curr string, state *pchannel.State) perun.ChInfo {
	return perun.ChInfo{
		ChID:    chID,
		BalInfo: makeBalInfoFromState(parts, curr, state),
		App:     makeApp(state.App, state.Data),
		Version: fmt.Sprintf("%d", state.Version),
	}
}

// makeApp returns perun.makeApp formed from the given add def and app data.
func makeApp(def pchannel.App, data pchannel.Data) perun.App {
	return perun.App{
		Def:  def,
		Data: data,
	}
}

// makeBalInfoFromState retrieves balance information from the channel state.
func makeBalInfoFromState(parts []string, curr string, state *pchannel.State) perun.BalInfo {
	if state == nil {
		return perun.BalInfo{}
	}
	return makeBalInfoFromRawBal(parts, curr, state.Balances[0])
}

// makeBalInfoFromRawBal retrieves balance information from the raw balance.
func makeBalInfoFromRawBal(parts []string, curr string, rawBal []*big.Int) perun.BalInfo {
	balInfo := perun.BalInfo{
		Currency: curr,
		Parts:    parts,
		Bal:      make([]string, len(rawBal)),
	}

	parser := currency.NewParser(curr)
	for i := range rawBal {
		balInfo.Bal[i] = parser.Print(rawBal[i])
	}
	return balInfo
}

func (ch *channel) HandleClose(err error) {
	ch.Debug("SDK Callback: Channel watcher returned.")
	ch.Lock()
	defer ch.Unlock()

	if ch.lockState == open {
		ch.lockState = closed
	}

	notif := makeChCloseNotif(ch.getChInfo(), err)

	if ch.chUpdateNotifier == nil {
		ch.chUpdateNotifCache = append(ch.chUpdateNotifCache, notif)
		ch.Debug("HandleClose: Notification cached")
	} else {
		go ch.chUpdateNotifier(notif)
		ch.Debug("HandleClose: Notification sent")
	}
}

func makeChCloseNotif(currChInfo perun.ChInfo, err error) perun.ChUpdateNotif {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	return perun.ChUpdateNotif{
		UpdateID:       fmt.Sprintf("%s_%s_%s", currChInfo.ChID, currChInfo.Version, "closed"),
		CurrChInfo:     currChInfo,
		ProposedChInfo: perun.ChInfo{},
		Type:           perun.ChUpdateTypeClosed,
		Expiry:         0,
		Error:          errMsg,
	}
}

func (ch *channel) Close(pctx context.Context) (perun.ChInfo, error) {
	ch.Debug("Received request channel.Close")
	ch.Lock()
	defer ch.Unlock()

	if ch.lockState == closed {
		return ch.getChInfo(), perun.ErrChClosed
	}

	ch.finalize(pctx)
	return ch.getChInfo(), ch.settlePrimary(pctx)
}

// finalize tries to finalize the channel offchain by sending an update with isFinal = true
// to all channel participants.
//
// If this suceeds, calling Settle consequently will close the channel collaboratively by directly settling
// the channel on the blockchain without registering or waiting for challenge duration to expire.
// If this fails, calling Settle consequently will close the channel non-collaboratively, by registering
// the state on-chain and waiting for challenge duration to expire.
func (ch *channel) finalize(pctx context.Context) error {
	if ch.lockState == closed {
		return perun.ErrChClosed
	}

	chFinalizer := func(state *pchannel.State) {
		state.IsFinal = true
	}
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.chUpdate())
	defer cancel()
	err := ch.pch.UpdateBy(ctx, ch.pch.Idx(), chFinalizer)
	if err != nil {
		ch.Info("Error when trying to finalize state", err)
	} else {
		ch.currState = ch.pch.State().Clone()
	}
	return err
}

// settlePrimary is used when the channel close initiated by the user.
func (ch *channel) settlePrimary(pctx context.Context) error {
	ch.lockState = closed

	// TODO (mano): Document what happens when a Settle fails, should channel close be called again ?
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.settleChPrimary(ch.challengeDurSecs))
	defer cancel()
	err := ch.pch.Settle(ctx)
	if err != nil {
		ch.Error("Settling channel", err)
		return perun.GetAPIError(err)
	}
	ch.close()
	return nil
}

// settleSecondary is used when the channel close is initiated after accepting a final update.
func (ch *channel) settleSecondary(pctx context.Context) error {
	ch.lockState = closed

	// TODO (mano): Document what happens when a Settle fails, should channel close be called again ?
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.settleChSecondary(ch.challengeDurSecs))
	defer cancel()
	err := ch.pch.SettleSecondary(ctx)
	if err != nil {
		ch.Error("Settling channel", err)
		return perun.GetAPIError(err)
	}
	ch.close()
	return nil
}

// Close the computing resources (listeners, subscriptions etc.,) of the channel.
// If it fails, this error can be ignored.
func (ch *channel) close() {
	if err := ch.pch.Close(); err != nil {
		ch.Error("Closing channel", err)
	}
}
