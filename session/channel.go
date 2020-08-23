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
	"sync"
	"time"

	pchannel "perun.network/go-perun/channel"
	pclient "perun.network/go-perun/client"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/log"
)

const (
	open      chLockState = "open"
	finalized chLockState = "finalized"
	closed    chLockState = "closed"
)

type (
	channel struct {
		log.Logger

		id               string
		pchannel         *pclient.Channel
		lockState        chLockState
		currency         string
		parts            []string
		timeoutCfg       timeoutConfig
		currState        *pchannel.State
		challengeDurSecs uint64 // challenge duration for the channel in seconds.

		chUpdateNotifier   perun.ChUpdateNotifier
		chUpdateNotifCache []perun.ChUpdateNotif
		chUpdateResponders map[string]chUpdateResponderEntry

		sync.RWMutex
	}

	chLockState string

	chUpdateResponderEntry struct {
		responder chUpdateResponder
		expiry    int64
	}

	//go:generate mockery -name ProposalResponder -output ../internal/mocks

	// ChUpdaterResponder represents the methods on channel update responder that will be used the pern node.
	chUpdateResponder interface {
		Accept(ctx context.Context) error
		Reject(ctx context.Context, reason string) error
	}
)

// NewChannel sets up a channel object from the passed pchannel.
func NewChannel(pch *pclient.Channel, currency string, parts []string, challengeDurSecs uint64,
	tConf timeoutConfig) *channel {
	ch := &channel{
		id:                 fmt.Sprintf("%x", pch.ID()),
		pchannel:           pch,
		lockState:          open,
		currState:          pch.State().Clone(),
		challengeDurSecs:   challengeDurSecs,
		timeoutCfg:         tConf,
		currency:           currency,
		parts:              parts,
		chUpdateResponders: make(map[string]chUpdateResponderEntry),
	}
	ch.Logger = log.NewLoggerWithField("channel-id", ch.id)
	return ch
}

func (ch *channel) ID() string {
	return ch.id
}

func (ch *channel) SendChUpdate(pctx context.Context, updater perun.StateUpdater) error {
	ch.Debug("Received request channel.sendChUpdate")
	ch.Lock()
	defer ch.Unlock()

	if ch.lockState == finalized {
		ch.Error("Dropping update request as the channel is " + ch.lockState)
		return perun.ErrChFinalized
	} else if ch.lockState == closed {
		ch.Error("Dropping update request as the channel is " + ch.lockState)
		return perun.ErrChClosed
	}

	ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.chUpdate())
	defer cancel()
	err := ch.pchannel.UpdateBy(ctx, updater)
	if err != nil {
		ch.Error("Sending channel update:", err)
		return perun.GetAPIError(err)
	}
	prevChInfo := ch.getChInfo()
	ch.currState = ch.pchannel.State().Clone()
	ch.Debugf("State upated from %v to %v", prevChInfo, ch.getChInfo())
	return nil
}

func (ch *channel) SubChUpdates(notifier perun.ChUpdateNotifier) error {
	ch.Debug("Received request channel.subChUpdates")
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
	ch.Debug("Received request channel.unSubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier == nil {
		ch.Error(perun.ErrNoActiveSub)
		return perun.ErrNoActiveSub
	}
	ch.chUpdateNotifier = nil
	return nil
}

func (ch *channel) RespondChUpdate(pctx context.Context, updateID string, accept bool) error {
	ch.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	entry, ok := ch.chUpdateResponders[updateID]
	if !ok {
		ch.Error(perun.ErrUnknownUpdateID, updateID)
		return perun.ErrUnknownUpdateID
	}
	delete(ch.chUpdateResponders, updateID)
	currTime := time.Now().UTC().Unix()
	if entry.expiry < currTime {
		ch.Error("timeout:", entry.expiry, "received response at:", currTime)
		return perun.ErrRespTimeoutExpired
	}

	switch accept {
	case true:
		ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.respChUpdateAccept())
		defer cancel()
		err := entry.responder.Accept(ctx)
		if err != nil {
			ch.Logger.Error("Accepting channel update", err)
			return perun.GetAPIError(err)
		}
		ch.currState = ch.pchannel.State().Clone()

	case false:
		ctx, cancel := context.WithTimeout(pctx, ch.timeoutCfg.respChUpdateReject())
		defer cancel()
		err := entry.responder.Reject(ctx, "rejected by user")
		if err != nil {
			ch.Logger.Error("Rejecting channel update", err)
			return perun.GetAPIError(err)
		}
	}

	// TODO: (mano) Provide an option for user to config the node to close finalized channels automatically.
	// For now, it is upto the user to close a channel that has been set to finalized state.
	// if ch.lockState == finalized {
	// }
	return nil
}

func (ch *channel) GetInfo() perun.ChannelInfo {
	ch.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()
	return ch.getChInfo()
}

// This function assumes that caller has already locked the channel.
func (ch *channel) getChInfo() perun.ChannelInfo {
	return perun.ChannelInfo{
		ChannelID: ch.id,
		Currency:  ch.currency,
		State:     ch.currState,
		Parts:     ch.parts,
	}
}

func (ch *channel) Close(pctx context.Context) (perun.ChannelInfo, error) {
	ch.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	switch ch.lockState {
	case open:
		ch.lockState = closed
		// Try to finalize state, so that channel can be settled directly without waiting for challenge duration
		// to expire. If this fails, channel will still be settled but by registering the state on-chain
		// and waiting for challenge duration to expire.
		chFinalizer := func(state *pchannel.State) {
			state.IsFinal = true
		}
		upCtx, upCancel := context.WithTimeout(pctx, ch.timeoutCfg.chUpdate())
		defer upCancel()
		if err := ch.pchannel.UpdateBy(upCtx, chFinalizer); err != nil {
			ch.Logger.Info("Error when trying to finalize state for closing:", err)
			ch.Logger.Info("Opting for non collaborative close")
		} else {
			ch.currState = ch.pchannel.State().Clone()
		}
		fallthrough

	case finalized:
		ch.lockState = closed
		clCtx, clCancel := context.WithTimeout(pctx, ch.timeoutCfg.closeCh(ch.challengeDurSecs))
		defer clCancel()
		err := ch.pchannel.Settle(clCtx)

		if cerr := ch.pchannel.Close(); err != nil {
			ch.Logger.Error("Settling channel", err)
			return perun.ChannelInfo{}, perun.GetAPIError(err)
		} else if cerr != nil {
			ch.Logger.Error("Closing channel", cerr)
		}
		return ch.getChInfo(), nil

	case closed:
		return ch.getChInfo(), perun.ErrChClosed
	}
	ch.Error("Program reached unknonwn state")
	return ch.getChInfo(), perun.ErrInternalServer
}
