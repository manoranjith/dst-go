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
	"strings"
	"sync"
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
	open   chStatus = iota // Open for off-chain tx.
	closed                 // Closed for off-chain tx, settled on-chain and amount withdrawn.
)

type (
	// Channel implements perun.ChAPI.
	Channel struct {
		log.Logger

		id               string
		pch              perun.Channel
		status           chStatus
		currency         string
		parts            []string
		timeoutCfg       timeoutConfig
		challengeDurSecs uint64
		currState        *pchannel.State

		chUpdateNotifier   perun.ChUpdateNotifier
		chUpdateNotifCache []perun.ChUpdateNotif
		chUpdateResponders map[string]chUpdateResponderEntry

		watcherWg *sync.WaitGroup
		psync.Mutex
	}

	chStatus uint8

	chUpdateResponderEntry struct {
		notif       perun.ChUpdateNotif
		responder   ChUpdateResponder
		notifExpiry int64
	}

	// ChUpdateResponder represents the methods on channel update responder that will be used the perun node.
	ChUpdateResponder interface {
		Accept(ctx context.Context) error
		Reject(ctx context.Context, reason string) error
	}
)

//go:generate mockery --name ChUpdateResponder --output ../internal/mocks

// newCh sets up a channel object from the passed pchannel.
func newCh(pch perun.Channel, currency string, parts []string, timeoutCfg timeoutConfig,
	challengeDurSecs uint64) *Channel {
	ch := &Channel{
		id:                 fmt.Sprintf("%x", pch.ID()),
		pch:                pch,
		status:             open,
		currState:          pch.State().Clone(),
		timeoutCfg:         timeoutCfg,
		challengeDurSecs:   challengeDurSecs,
		currency:           currency,
		parts:              parts,
		chUpdateResponders: make(map[string]chUpdateResponderEntry),
		watcherWg:          &sync.WaitGroup{},
	}
	ch.watcherWg.Add(1)
	go func(ch *Channel) {
		err := ch.pch.Watch(ch)
		ch.watcherWg.Done()

		ch.HandleWatcherReturned(err)
	}(ch)
	return ch
}

func (ch *Channel) HandleAdjudicatorEvent(e pchannel.AdjudicatorEvent) {
	if _, ok := e.(*pchannel.ConcludedEvent); ok {
		ch.Infof("[%s] Got adjudcator event of type %T: %v", ch.pch.Phase().String(), e, e)
		func() {
			ch.Lock()
			defer ch.Unlock()
			err := e.Timeout().Wait(context.Background())
			if err != nil {
				ch.Errorf("Error waiting for timeout to elapse: %v", err)
				return
			}
			ch.Infof("[%s] Timeout elapsed", ch.pch.Phase().String())

			if ch.pch.Phase() == pchannel.Final {
				ch.Info("Starting secondary settle phase")
				err = ch.settleSecondary(context.Background())
				ch.Errorf("Error secondary settle: %v", err)
			}
			ch.Infof("[%s] Timeout elapsed 2", ch.pch.Phase().String())
			if ch.pch.Phase() == pchannel.Withdrawn {
				if ch.chUpdateNotifier == nil {
					ch.Debug("HandleWatcherReturned: Notification dropped as there is no active subscription")
					return
				}
				notif := makeChCloseNotif(ch.getChInfo(), err)
				ch.chUpdateNotifier(notif)
				ch.unsubChUpdates()
				ch.Debug("HandleWatcherReturned: Notification sent")
			}
		}()
	}

}

// ID returns the ID of the channel.
//
// Does not require a mutex lock, as the data will remain unchanged throughout the lifecycle of the channel.
func (ch *Channel) ID() string {
	return ch.id
}

// Currency returns the currency interpreter used in the channel.
//
// Does not require a mutex lock, as the data will remain unchanged throughout the lifecycle of the channel.
func (ch *Channel) Currency() string {
	return ch.currency
}

// Parts returns the list of aliases of the channel participants.
//
// Does not require a mutex lock, as the data will remain unchanged throughout the lifecycle of the channel.
func (ch *Channel) Parts() []string {
	return ch.parts
}

// ChallengeDurSecs returns the challenge duration for the channel (in seconds) for refuting when
// an invalid/older state is registered on the blockchain closing the channel.
//
// Does not require a mutex lock, as the data will remain unchanged throughout the lifecycle of the channel.
func (ch *Channel) ChallengeDurSecs() uint64 {
	return ch.challengeDurSecs
}

// SendChUpdate implements chAPI.SendChUpdate.
func (ch *Channel) SendChUpdate(pctx context.Context, updater perun.StateUpdater) (perun.ChInfo, error) {
	ch.Debug("Received request: channel.SendChUpdate")
	ch.Lock()
	defer ch.Unlock()

	if ch.status == closed {
		return ch.getChInfo(), perun.ErrChClosed
	}

	err := ch.sendChUpdate(pctx, updater)
	if err != nil {
		return perun.ChInfo{}, err
	}
	prevChInfo := ch.getChInfo()
	ch.currState = ch.pch.State().Clone()
	ch.Debugf("State upated from %v to %v", prevChInfo, ch.getChInfo())
	return ch.getChInfo(), nil
}

func (ch *Channel) sendChUpdate(pctx context.Context, updater perun.StateUpdater) error {
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.chUpdate())
	defer cancel()
	err := ch.pch.UpdateBy(ctx, updater)
	if err != nil {
		ch.Error("Sending channel update:", err)
		// TODO: (mano) Use errors.Is here once a sentinel error value is defined in the SDK.
		if strings.Contains(err.Error(), "rejected by user") {
			err = perun.ErrPeerRejected
		}
	}
	return perun.GetAPIError(err)
}

// HandleUpdate handles the incoming updates on an open channel. All updates are sent to a centralized
// update handler defined on the session. The centrazlied handler identifies the channel and then
// invokes this function to process the update.
func (ch *Channel) HandleUpdate(chUpdate pclient.ChannelUpdate, responder ChUpdateResponder) {
	ch.Lock()
	defer ch.Unlock()

	if ch.status == closed {
		ch.Error("Unexpected HandleUpdate call for closed channel")
		return
	}

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
	ch.sendChUpdateNotif(notif)
}

func (ch *Channel) sendChUpdateNotif(notif perun.ChUpdateNotif) {
	if ch.chUpdateNotifier == nil {
		ch.chUpdateNotifCache = append(ch.chUpdateNotifCache, notif)
		ch.Debug("HandleUpdate: Notification cached")
		return
	}
	go func() {
		ch.chUpdateNotifier(notif)
		ch.Debug("HandleUpdate: Notification sent")
	}()
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

// SubChUpdates implements chAPI.SubChUpdates.
func (ch *Channel) SubChUpdates(notifier perun.ChUpdateNotifier) error {
	ch.Debug("Received request: channel.SubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.status == closed {
		return perun.ErrChClosed
	}

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

// UnsubChUpdates implements chAPI.UnsubChUpdates.
func (ch *Channel) UnsubChUpdates() error {
	ch.Debug("Received request: channel.UnsubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.status == closed {
		return perun.ErrChClosed
	}

	if ch.chUpdateNotifier == nil {
		ch.Error(perun.ErrNoActiveSub)
		return perun.ErrNoActiveSub
	}
	ch.unsubChUpdates()
	return nil
}

func (ch *Channel) unsubChUpdates() {
	ch.chUpdateNotifier = nil
}

// RespondChUpdate implements chAPI.RespondChUpdate.
func (ch *Channel) RespondChUpdate(pctx context.Context, updateID string, accept bool) (perun.ChInfo, error) {
	ch.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	if ch.status == closed {
		return ch.getChInfo(), perun.ErrChClosed
	}

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
			// err = ch.settleSecondary(pctx)
			// Channel will be settled when the event in received in the watcher.
			// Because, currently there is no secondaryRegister function.
		}
	case false:
		err = ch.rejectChUpdate(pctx, entry, "rejected by user")
	}
	return ch.getChInfo(), err
}

func (ch *Channel) acceptChUpdate(pctx context.Context, entry chUpdateResponderEntry) error {
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

func (ch *Channel) rejectChUpdate(pctx context.Context, entry chUpdateResponderEntry, reason string) error {
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.respChUpdate())
	defer cancel()
	err := entry.responder.Reject(ctx, reason)
	if err != nil {
		ch.Error("Rejecting channel update", err)
	}
	return perun.GetAPIError(errors.Wrap(err, "rejecting update"))
}

// GetChInfo implements chAPI.GetChInfo.
func (ch *Channel) GetChInfo() perun.ChInfo {
	ch.Debug("Received request: channel.GetChInfo")
	ch.Lock()
	chInfo := ch.getChInfo()
	ch.Unlock()
	return chInfo
}

// This function assumes that caller has already locked the channel.
func (ch *Channel) getChInfo() perun.ChInfo {
	return makeChInfo(ch.ID(), ch.parts, ch.currency, ch.currState)
}

func makeChInfo(chID string, parts []string, curr string, state *pchannel.State) perun.ChInfo {
	if state == nil {
		return perun.ChInfo{}
	}
	return perun.ChInfo{
		ChID:    chID,
		BalInfo: makeBalInfoFromState(parts, curr, state),
		App:     makeApp(state.App, state.Data),
		Version: fmt.Sprintf("%d", state.Version),
	}
}

// makeApp returns perun.makeApp formed from the given app def and app data.
func makeApp(def pchannel.App, data pchannel.Data) perun.App {
	return perun.App{
		Def:  def,
		Data: data,
	}
}

// makeBalInfoFromState retrieves balance information from the channel state.
func makeBalInfoFromState(parts []string, curr string, state *pchannel.State) perun.BalInfo {
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

// HandleWatcherReturned is invoked when the watcher for this channel has returned.
// If the channel is open (happens when watcher refuted to a wrong state that was registered on-chain),
//		it will be marked closed.
// Then it sends a channel close notification if the channel is already subscribed.
// If the channel is not subscribed, notification will not be cached as it not possible for the user
// to subscribe to channel after it is closed.
func (ch *Channel) HandleWatcherReturned(err error) {
	ch.Lock()
	defer ch.Unlock()
	ch.Debug("Watch returned")

	if ch.status == open {
		ch.close()
	}

	if ch.chUpdateNotifier == nil {
		ch.Debug("HandleWatcherReturned: Notification dropped as there is no active subscription")
		return
	}
	notif := makeChCloseNotif(ch.getChInfo(), err)
	ch.chUpdateNotifier(notif)
	ch.unsubChUpdates()
	ch.Debug("HandleWatcherReturned: Notification sent")
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

// Close implements chAPI.Close.
func (ch *Channel) Close(pctx context.Context) (perun.ChInfo, error) {
	ch.Debug("Received request channel.Close")
	ch.Lock()
	defer ch.Unlock()

	if ch.status == closed {
		return ch.getChInfo(), perun.ErrChClosed
	}

	ch.finalize(pctx)
	err := ch.settlePrimary(pctx)
	return ch.getChInfo(), err
}

// finalize tries to finalize the channel offchain by sending an update with isFinal = true
// to all channel participants.
//
// If this succeeds, calling Settle consequently will close the channel collaboratively by directly settling
// the channel on the blockchain without registering or waiting for challenge duration to expire.
// If this fails, calling Settle consequently will close the channel non-collaboratively, by registering
// the state on-chain and waiting for challenge duration to expire.
func (ch *Channel) finalize(pctx context.Context) {
	chFinalizer := func(state *pchannel.State) error {
		state.IsFinal = true
		return nil
	}
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.chUpdate())
	defer cancel()
	err := ch.pch.UpdateBy(ctx, chFinalizer)
	if err != nil {
		ch.Info("Error when trying to finalize state", err)
	} else {
		ch.currState = ch.pch.State().Clone()
	}
}

// settlePrimary is used when the channel close initiated by the user.
func (ch *Channel) settlePrimary(pctx context.Context) error {
	// TODO (mano): Document what happens when a Settle fails, should channel close be called again ?
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.settleChPrimary(ch.challengeDurSecs))
	defer cancel()
	err := ch.pch.Register(ctx)
	if err != nil {
		ch.Error("Registering channel", err)
		return perun.GetAPIError(err)
	}
	err = ch.pch.Settle(ctx, false)
	if err != nil {
		ch.Error("Settling channel", err)
		return perun.GetAPIError(err)
	}
	// ch.close()
	return nil
}

// settleSecondary is used when the channel close is initiated after accepting a final update.
func (ch *Channel) settleSecondary(pctx context.Context) error {
	// TODO (mano): Document what happens when a Settle fails, should channel close be called again ?
	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.settleChSecondary(ch.challengeDurSecs))
	defer cancel()
	err := ch.pch.Register(ctx)
	if err != nil {
		ch.Error("Registering channel", err)
		return perun.GetAPIError(err)
	}
	err = ch.pch.Settle(ctx, true)
	if err != nil {
		ch.Error("Settling channel", err)
		return perun.GetAPIError(err)
	}
	ch.close()
	return nil
}

// Close the computing resources (listeners, subscriptions etc.,) of the channel.
// If it fails, this error can be ignored.
// It also removes the channel from the session.
func (ch *Channel) close() {
	ch.watcherWg.Wait()

	if err := ch.pch.Close(); err != nil {
		ch.Error("Closing channel", err)
	}
	ch.status = closed
}
