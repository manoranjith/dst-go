package payment

import (
	"fmt"

	"perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/session"
)

type (
	PayChInfo struct {
		ChannelID string
		BalInfo   session.BalInfo
		Version   string
	}

	PayChUpdateNotifier func(PayChUpdateNotif)

	PayChUpdateNotif struct {
		UpdateID     string
		ProposedBals session.BalInfo
		Version      string
		Final        bool
		Currency     string
		Parts        []string
		Timeout      int64
	}
)

func SendPayChUpdate(ch *session.Channel, payee, amount string) error {
	chInfo := ch.GetInfo()
	f, err := newUpdater(chInfo.State, chInfo.Parts, chInfo.Currency, payee, amount)
	if err != nil {
		return err
	}
	return ch.SendChUpdate(f)
}

func RespondPayChUpdate(ch *session.Channel, updateID string, accept bool) error {
	return ch.RespondChUpdate(updateID, accept)
}

func GetBalance(ch *session.Channel) session.BalInfo {
	chInfo := ch.GetInfo()
	return balsFromState(chInfo.Currency, chInfo.State, chInfo.Parts)
}

func ClosePayCh(ch *session.Channel) (session.BalInfo, error) {
	chInfo, err := ch.Close()
	if err != nil {
		return session.BalInfo{}, err
	}
	return balsFromState(chInfo.Currency, chInfo.State, chInfo.Parts), nil
}

func newUpdater(currState *channel.State, parts []string, chCurrency, payee, amount string) (session.StateUpdater, error) {
	parsedAmount, err := currency.NewParser(chCurrency).Parse(amount)
	if err != nil {
		return nil, perun.ErrInvalidAmount
	}

	// find index
	var payerIdx, payeeIdx int
	if parts[0] == payee {
		payeeIdx = 0
	} else if parts[1] == payee {
		payeeIdx = 1
	} else {
		return nil, perun.ErrInvalidPayee
	}
	payerIdx = payeeIdx ^ 1

	// check sufficient balance
	bals := currState.Allocation.Clone().Balances[0]
	bals[payerIdx].Sub(bals[payerIdx], parsedAmount)
	bals[payeeIdx].Add((bals[payeeIdx]), parsedAmount)
	if bals[payerIdx].Sign() == -1 {
		return nil, perun.ErrInsufficientBal
	}

	// return updater func
	return func(state *channel.State) {
		state.Allocation.Balances[0][payerIdx] = bals[payerIdx]
		state.Allocation.Balances[0][payeeIdx] = bals[payeeIdx]
	}, nil

}

func SubPayChUpdates(ch *session.Channel, notifier PayChUpdateNotifier) error {
	return ch.SubChUpdates(func(notif session.ChUpdateNotif) {
		notifier(PayChUpdateNotif{
			UpdateID:     notif.UpdateID,
			ProposedBals: balsFromState(notif.Currency, notif.Update.State, notif.Parts),
			Version:      fmt.Sprintf("%d", notif.Update.State.Version),
			Final:        notif.Update.State.IsFinal,
			Timeout:      notif.Expiry,
		})
	})
}

func UnSubPayChUpdates(ch *session.Channel) error {
	return ch.UnsubChUpdates()
}

// TODO: Add a hook
// func ValidateUpdate(current, proposed *channel.State) error {

// 	// check 1:
// 	var oldSum, newSum *big.Int
// 	oldBals := current.Allocation.Balances[0]
// 	oldSum.Add(oldBals[0], oldBals[1])
// 	newBals := proposed.Allocation.Balances[0]
// 	newSum.Add(newBals[0], newBals[1])

// 	if newSum.Cmp(oldSum) != 0 {
// 		// return errors.New("invalid update: sum of balances is not constant")
// 	}

// 	if newBals[0].Sign() == -1 || newBals[1].Sign() == -1 {
// 		// return errrors.New("this update results in negative balance, hence not allowed")
// 	}
// }
