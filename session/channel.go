package session

import (
	"context"
	"sync"
	"time"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	pclient "perun.network/go-perun/client"
	"perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/log"
)

const (
	ChannelOpen      ChannelLockState = "Open"
	ChannelFinalized ChannelLockState = "Finalized"
	ChannelClosed    ChannelLockState = "Closed"
)

type (
	Channel struct {
		log.Logger

		ID        string
		Channel   *client.Channel
		LockState ChannelLockState
		Currency  string
		Parts     []string
		// Store a clone of current state of the channel.
		// Because the channel mutex in sdk will be locked during handle update function and the state cannot be read then.
		CurrState *channel.State

		chUpdateNotifier   ChUpdateNotifier
		chUpdateNotifCache []ChUpdateNotif
		chUpdateResponders map[string]ChUpdateResponderEntry

		sync.RWMutex
	}

	ChannelLockState string

	ChUpdateNotifier func(ChUpdateNotif)

	ChUpdateNotif struct {
		UpdateID  string
		Currency  string
		CurrState *channel.State
		Update    *pclient.ChannelUpdate
		Parts     []string
		Expiry    int64
	}

	ChUpdateResponderEntry struct {
		chUpdateResponder ChUpdateResponder
		Expiry            int64
	}

	//go:generate mockery -name ProposalResponder -output ../internal/mocks

	// ChUpdaterResponder represents the methods on channel update responder that will be used the pern node.
	ChUpdateResponder interface {
		Accept(ctx context.Context) error
		Reject(ctx context.Context, reason string) error
	}

	App struct {
		Def  wallet.Address
		Data channel.Data
	}

	ChannelInfo struct {
		ChannelID string
		Currency  string
		State     *channel.State
		Parts     []string // List of Alias of channel participants.
	}

	BalInfo struct {
		Currency string
		Bals     map[string]string // Map of alias to balance.
	}

	StateUpdater func(*channel.State)
)

func NewChannel(pch *client.Channel, currency string, parts []string) *Channel {
	channelID := pch.ID()
	ch := &Channel{
		ID:                 BytesToHex(channelID[:]),
		Channel:            pch,
		LockState:          ChannelOpen,
		CurrState:          pch.State().Clone(),
		Currency:           currency,
		Parts:              parts,
		chUpdateResponders: make(map[string]ChUpdateResponderEntry),
	}
	ch.Logger = log.NewLoggerWithField("channel-id", ch.ID)
	return ch
}

func (ch *Channel) SendChUpdate(stateUpdater StateUpdater) error {
	ch.Logger.Debug("Received request channel.sendChUpdate")
	ch.Lock()
	defer ch.Unlock()

	err := ch.Channel.UpdateBy(context.TODO(), stateUpdater)
	if err != nil {
		ch.Logger.Error("Sending channel update:", err)
		return perun.GetAPIError(err)
	}
	ch.CurrState = ch.Channel.State().Clone()
	return nil
}

func (ch *Channel) SubChUpdates(notifier ChUpdateNotifier) error {
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

func (ch *Channel) UnsubChUpdates() error {
	ch.Logger.Debug("Received request channel.unSubChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier == nil {
		return perun.ErrNoActiveSub
	}
	ch.chUpdateNotifier = nil
	return nil
}

func (ch *Channel) RespondChUpdate(chUpdateID string, accept bool) error {
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
	if entry.Expiry < time.Now().UTC().Unix() {
		ch.Logger.Error(perun.ErrRespTimeoutExpired)
		return perun.ErrRespTimeoutExpired
	}

	switch accept {
	case true:
		err := entry.chUpdateResponder.Accept(context.TODO())
		if err != nil {
			ch.Logger.Error("Accepting channel update", err)
			return perun.GetAPIError(err)
		}
		ch.CurrState = ch.Channel.State().Clone()

	case false:
		err := entry.chUpdateResponder.Reject(context.TODO(), "rejected by user")
		if err != nil {
			ch.Logger.Error("Rejecting channel update", err)
			return perun.GetAPIError(err)
		}
	}

	if ch.LockState == ChannelFinalized {
		// Init close, wait to see how to do this.
	}
	return nil
}

func (ch *Channel) GetInfo() ChannelInfo {
	ch.Logger.Debug("Received request channel.RespondChUpdate")
	ch.RLock()
	defer ch.RUnlock()
	return ch.getChInfo()
}

// This function assumes that caller has already locked the channel.
func (ch *Channel) getChInfo() ChannelInfo {
	return ChannelInfo{
		ChannelID: ch.ID,
		Currency:  ch.Currency,
		State:     ch.Channel.State().Clone(),
		Parts:     ch.Parts,
	}
}

func (ch *Channel) Close() (ChannelInfo, error) {
	ch.Logger.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	// Try to finalize state, so that channel can be settled collaboratively.
	// If this fails, channel will still be settled but by registering the state on-chain
	// and waiting for challenge duration to expire.
	if err := ch.Channel.UpdateBy(nil, func(_ *channel.State) {}); err != nil {
		ch.Logger.Info("Error when trying to finalize state for closing:", err)
		ch.Logger.Info("Opting for non collaborative close")
	}

	err := ch.Channel.Settle(context.TODO())

	if cerr := ch.Channel.Close(); err != nil {
		ch.Logger.Error("Settling channel", err)
		return ChannelInfo{}, perun.GetAPIError(err)
	} else if cerr != nil {
		ch.Logger.Error("Closing channel", cerr)
	}
	return ch.getChInfo(), nil
}
