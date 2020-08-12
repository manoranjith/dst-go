package session

import (
	"context"
	"errors"

	"github.com/hyperledger-labs/perun-node"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

type ChannelLockState string

const (
	ChannelOpen      ChannelLockState = "Open"
	ChannelFinalized ChannelLockState = "Finalized"
	ChannelClosed    ChannelLockState = "Closed"
)

type Channel struct {
	ID         string
	Controller perun.Channel
	LockState  ChannelLockState
	// send notification.
	// Mechanism to create subscription ID ?.... What is it unique of... ?
	// This subscription is for a session, but any number of subscriptions can be made and all are identical.
	// So use channel ID  as the subscription ID. Later this can be changed.
	UpdateNotify     PayChUpdateNotify     // Handler for sending notifications
	UpdateResponders perun.UpdateResponder // Map of proposalIDs to ProposalResponders.

	UpdateCache *client.ChannelUpdate // There will be only one active update at a time... Document this clearly in the diagram.
}

func (c *Channel) HasActiveSub() bool {
	return c.UpdateNotify != nil
}

func (c *Channel) SendPayChUpdate(alias string, amount string) error {
	return c.Controller.UpdateBy(nil, func(_ *channel.State) {})
}

func (c *Channel) SubPayChUpdates(f PayChUpdateNotify) error {
	if c.UpdateNotify != nil {
		return errors.New("already subscribed")
	}
	c.UpdateNotify = f
	return nil
}

func (c *Channel) UnsubPayChUpdates() error {
	if c.UpdateNotify == nil {
		return errors.New("no active subscription")
	}
	c.UpdateNotify = nil
	return nil
}

func (c *Channel) RespondToPayChUpdateNotif(accept bool) error {
	if c.UpdateResponders == nil {
		return errors.New("no response expected")
	}
	if !accept {
		return c.UpdateResponders.Reject(context.TODO(), "rejected by user")
	}
	return c.UpdateResponders.Accept(context.TODO())
}

func (c *Channel) GetBalance() BalInfo {
	panic("not implemented")
}

func (c *Channel) ClosePayCh() (finalBals BalInfo, _ error) {
	// Try to finalize state, so that channel can be settled collaboratively.
	// If this fails, channel will be settled non-collaboratively.
	// Non-Collaborative takes more on-chain txns and time.
	if err := c.Controller.UpdateBy(nil, func(_ *channel.State) {}); err != nil {
		_ = err
		// Log error
	}
	err := c.Controller.Settle(nil)
	if cerr := c.Controller.Close(); err != nil {
		return finalBals, err
	} else if cerr != nil {
		_ = cerr
		// log cerr
	}
	return finalBals, nil

}

func (ch *Channel) BalInfo() BalInfo {
	return BalInfo{}
}

type PayChUpdateNotify interface {
	PayChUpdateNotify(alias string, bals BalInfo, ChannelgeDurSecs uint64)
}

type PayChState struct {
	channelID string
	BalInfo   BalInfo
	Version   string
}

type Currency string

const (
	CurrencyETH Currency = "ETH"
)

type BalInfo struct {
	Currency string
	bals     map[string]string // Map of alias to balance.
}

type ChannelAPI interface {
	SendPayChUpdate(alias string, amount string) error
	SubPayChUpdates(PayChUpdateNotify) error // Err if subscription exists.
	// SendPayChNotif(
	UnsubPayChUpdates() error // Err if there is no subscription.
	RespondToPayChUpdateNotif(accept bool) error
	GetBalance() BalInfo
	ClosePayCh() (finalBals BalInfo, _ error)
}

func NewChannel() {}

// How to link functions defined here to Handlers registered in client.New ???
// Those handlers should passon the function to client.
// Those handlers should be provided from here ?
// Or they put the data in a channel, that reaches here...
// How do i connect the callback from here to those channels.
