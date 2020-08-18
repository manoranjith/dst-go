package session

import (
	"sync"

	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
	"perun.network/go-perun/wallet"

	"github.com/hyperledger-labs/perun-node/log"
)

type Channel struct {
	log.Logger

	ID        string
	Channel   *client.Channel
	LockState ChannelLockState
	Currency  string

	sync.RWMutex
}

type ChannelLockState string

const (
	ChannelOpen      ChannelLockState = "Open"
	ChannelFinalized ChannelLockState = "Finalized"
	ChannelClosed    ChannelLockState = "Closed"
)

type App struct {
	Def  wallet.Address
	Data channel.Data
}

type BalInfo struct {
	Currency string
	Bals     map[string]string // Map of alias to balance.
}

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
