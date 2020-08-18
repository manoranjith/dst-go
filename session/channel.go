package session

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	pclient "perun.network/go-perun/client"
	"perun.network/go-perun/wallet"

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

		chUpdateNotifier   ChUpdateNotifier
		chUpdateNotifCache []ChUpdateNotif
		chUpdateResponders map[string]ChUpdateResponderEntry

		sync.RWMutex
	}

	ChannelLockState string

	ChUpdateNotifier func(ChUpdateNotif)

	ChUpdateNotif struct {
		UpdateID string
		Update   *pclient.ChannelUpdate
		Expiry   int64
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

	BalInfo struct {
		Currency string
		Bals     map[string]string // Map of alias to balance.
	}

	StateUpdater func(*channel.State)
)

func NewChannel(pch *client.Channel) *Channel {
	channelID := pch.ID()
	ch := &Channel{
		ID:        BytesToHex(channelID[:]),
		Channel:   pch,
		LockState: ChannelOpen,
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
		return errors.Wrap(err, "")
	}
	return nil
}

func (ch *Channel) SubChUpdates(notifier ChUpdateNotifier) error {
	ch.Logger.Debug("Received request channel.subChUpdates")
	ch.Lock()
	defer ch.Unlock()

	if ch.chUpdateNotifier != nil {
		return errors.New("")
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
		return errors.New("")
	}
	ch.chUpdateNotifier = nil
	return nil
}

func (ch *Channel) RespondChUpdate(chUpdateID string, accept bool) error {
	ch.Logger.Debug("Received request channel.RespondChUpdate")
	ch.Lock()
	defer ch.Unlock()

	entry, ok := ch.chUpdateResponders[chUpdateID]
	delete(ch.chUpdateResponders, chUpdateID)
	if !ok {
		return errors.New("")
	}
	if entry.Expiry > time.Now().UTC().Unix() {
		return errors.New("")
	}

	switch accept {
	case true:
		err := entry.chUpdateResponder.Accept(context.TODO())
		if err != nil {
			return errors.New("")
		}

	case false:
		err := entry.chUpdateResponder.Reject(context.TODO(), "rejected by user")
		if err != nil {
			return errors.New("")
		}
	}

	if ch.LockState == ChannelFinalized {
		// Init close, wait to see how to do this.
	}
	return nil
}
