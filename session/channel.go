package session

import "perun.network/go-perun/client"

type ChannelLockState string

const (
	ChannelLocked    ChannelLockState = "Locked"
	ChannelFinalized ChannelLockState = "Finalized"
	ChannelClosed    ChannelLockState = "Closed"
)

type Channel struct {
	controller client.Channel
	LockState  ChannelLockState
	// send notification.
	// Mechanism to create subscription ID ?.... What is it unique of... ?
	// This subscription is for a session, but any number of subscriptions can be made and all are identical.
	// So use channel ID  as the subscription ID. Later this can be changed.
	UpdateNotify     PayChUpdateNotify       // Handler for sending notifications
	UpdateResponders *client.UpdateResponder // Map of proposalIDs to ProposalResponders.

	UpdateCache *client.ChannelUpdate // There will be only one active update at a time... Document this clearly in the diagram.
}

func (ch *Channel) BalInfo() BalInfo {
	return BalInfo{}
}

type PayChUpdateNotify func(proposalID string, alias string, initBals BalInfo, ChDurSecs uint64)

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
	SubPayChUpdates(PayChUpdateNotify) (subID string)
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
