package session

import (
	"context"

	"github.com/hyperledger-labs/perun-node"
	"github.com/pkg/errors"
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
	peers      []string
	// send notification.
	// Mechanism to create subscription ID ?.... What is it unique of... ?
	// This subscription is for a session, but any number of subscriptions can be made and all are identical.
	// So use channel ID  as the subscription ID. Later this can be changed.
	UpdateNotify     StateDecoder          // Handler for sending notifications
	UpdateResponders perun.UpdateResponder // Map of proposalIDs to ProposalResponders.

	UpdateCache *client.ChannelUpdate // There will be only one active update at a time... Document this clearly in the diagram.
	App         string                // App that runs in the channel
	AppParams   map[string]string     // App specific parameters
}
type ChannelAPI interface {
	SendChUpdate(f StateUpdater) error
	SubChUpdates(f StateDecoder) error // Err if subscription exists.
	UnsubChUpdates() error             // Err if there is no subscription.
	RespondToChUpdateNotif(accept bool) error
	GetState() *channel.State
	CloseCh() (*channel.State, error)
}

func (c *Channel) HasActiveSub() bool {
	return c.UpdateNotify != nil
}

// func (c *Channel) SendPayChUpdate(alias string, amount string) error {
// 	err := c.Controller.UpdateBy(nil, func(_ *channel.State) {})
// 	if err != nil {
// 		return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Sending state update"))
// 	}
// 	return nil
// }

func (c *Channel) SendChUpdate(f StateUpdater) error {
	err := c.Controller.UpdateBy(nil, f)
	if err != nil {
		return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Sending state update"))
	}
	return nil
}

func (c *Channel) SubChUpdates(f StateDecoder) error {
	if c.UpdateNotify != nil {
		return perun.NewAPIError(perun.ErrSubAlreadyExists, nil)
	}
	c.UpdateNotify = f
	return nil
}

func (c *Channel) UnsubChUpdates() error {
	if c.UpdateNotify == nil {
		return perun.NewAPIError(perun.ErrNoActiveSub, nil)
	}
	c.UpdateNotify = nil
	return nil
}

func (c *Channel) RespondToChUpdateNotif(accept bool) error {
	if c.UpdateResponders == nil {
		return errors.New("no response expected")
	}
	switch accept {
	case true:
		err := c.UpdateResponders.Accept(context.TODO())
		if err != nil {
			return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Accepting state update"))
		}
	case false:
		err := c.UpdateResponders.Reject(context.TODO(), "rejected by user")
		if err != nil {
			return perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Rejecting state update"))
		}
	}
	return nil
}

func (c *Channel) GetState() *channel.State {
	return c.Controller.State()
}

func (c *Channel) CloseCh() (*channel.State, error) {
	// Try to finalize state, so that channel can be settled collaboratively.
	// If this fails, channel will be settled non-collaboratively.
	// Non-Collaborative takes more on-chain txns and time.
	if err := c.Controller.UpdateBy(nil, func(_ *channel.State) {}); err != nil {
		_ = err
		// Log error
	}
	err := c.Controller.Settle(nil)
	if cerr := c.Controller.Close(); err != nil {
		return nil, perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Closing channel controller"))
	} else if cerr != nil {
		_ = cerr
		// log cerr
	}
	return c.Controller.State(), nil

}

func (ch *Channel) BalInfo() BalInfo {
	return BalInfo{}
}

type UpdateNotifier func(s channel.State)

// How to link functions defined here to Handlers registered in client.New ???
// Those handlers should passon the function to client.
// Those handlers should be provided from here ?
// Or they put the data in a channel, that reaches here...
// How do i connect the callback from here to those channels.
