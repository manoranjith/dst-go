package session

import (
	"context"
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

		id        string
		pchannel  *pclient.Channel
		lockState chLockState
		currency  string
		parts     []string
		// Store a clone of current state of the channel.
		// Because the channel mutex in sdk will be locked during handle update function and the state cannot be read then.
		currState *pchannel.State

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

func NewChannel(pch *pclient.Channel, currency string, parts []string) *channel {
	channelID := pch.ID()
	ch := &channel{
		id:                 BytesToHex(channelID[:]),
		pchannel:           pch,
		lockState:          open,
		currState:          pch.State().Clone(),
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

func (ch *channel) SendChUpdate(stateUpdater perun.StateUpdater) error {
	ch.Logger.Debug("Received request channel.sendChUpdate")
	ch.Lock()
	defer ch.Unlock()

	err := ch.pchannel.UpdateBy(context.TODO(), stateUpdater)
	if err != nil {
		ch.Logger.Error("Sending channel update:", err)
		return perun.GetAPIError(err)
	}
	ch.currState = ch.pchannel.State().Clone()
	return nil
}

func (ch *channel) SubChUpdates(notifier perun.ChUpdateNotifier) error {
	ch.Logger.Debug("Received request channel.subChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier != nil {
		return perun.ErrSubAlreadyExists
	}
	ch.chUpdateNotifier = notifier

	// Send all cached notifications
	// TODO: (mano) This works for gRPC, but change to send in background.
	for i := len(ch.chUpdateNotifCache) - 1; i >= 0; i-- {
		ch.chUpdateNotifier(ch.chUpdateNotifCache[0])
		ch.chUpdateNotifCache = ch.chUpdateNotifCache[1 : i+1]
	}
	return nil
}

func (ch *channel) UnsubChUpdates() error {
	ch.Logger.Debug("Received request channel.unSubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier == nil {
		return perun.ErrNoActiveSub
	}
	ch.chUpdateNotifier = nil
	return nil
}

func (ch *channel) RespondChUpdate(chUpdateID string, accept bool) error {
	ch.Logger.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	entry, ok := ch.chUpdateResponders[chUpdateID]
	if !ok {
		ch.Logger.Error(perun.ErrUnknownUpdateID, chUpdateID)
		return perun.ErrUnknownUpdateID
	}
	// TODO: Check if delete or defer delete
	delete(ch.chUpdateResponders, chUpdateID)
	if entry.expiry < time.Now().UTC().Unix() {
		ch.Logger.Error(perun.ErrRespTimeoutExpired)
		return perun.ErrRespTimeoutExpired
	}

	switch accept {
	case true:
		err := entry.responder.Accept(context.TODO())
		if err != nil {
			ch.Logger.Error("Accepting channel update", err)
			return perun.GetAPIError(err)
		}
		ch.currState = ch.pchannel.State().Clone()

	case false:
		err := entry.responder.Reject(context.TODO(), "rejected by user")
		if err != nil {
			ch.Logger.Error("Rejecting channel update", err)
			return perun.GetAPIError(err)
		}
	}

	if ch.lockState == finalized {
		// Init close, wait to see how to do this.
	}
	return nil
}

func (ch *channel) GetInfo() perun.ChannelInfo {
	ch.Logger.Debug("Received request channel.RespondChUpdate")
	ch.RLock()
	defer ch.RUnlock()
	return ch.getChInfo()
}

// This function assumes that caller has already locked the channel.
func (ch *channel) getChInfo() perun.ChannelInfo {
	return perun.ChannelInfo{
		ChannelID: ch.id,
		Currency:  ch.currency,
		State:     ch.pchannel.State().Clone(),
		Parts:     ch.parts,
	}
}

func (ch *channel) Close() (perun.ChannelInfo, error) {
	ch.Logger.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	// Try to finalize state, so that channel can be settled collaboratively.
	// If this fails, channel will still be settled but by registering the state on-chain
	// and waiting for challenge duration to expire.
	if err := ch.pchannel.UpdateBy(context.TODO(), func(_ *pchannel.State) {}); err != nil {
		ch.Logger.Info("Error when trying to finalize state for closing:", err)
		ch.Logger.Info("Opting for non collaborative close")
	}

	err := ch.pchannel.Settle(context.TODO())

	if cerr := ch.pchannel.Close(); err != nil {
		ch.Logger.Error("Settling channel", err)
		return perun.ChannelInfo{}, perun.GetAPIError(err)
	} else if cerr != nil {
		ch.Logger.Error("Closing channel", cerr)
	}
	return ch.getChInfo(), nil
}
