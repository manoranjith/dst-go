package payment

import (
	"perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/session"
)

type (
	PayChInfo struct {
		channelID string
		BalInfo   session.BalInfo
		Version   string
	}

	// PayChUpdateNotifier func(PayChUpdateNotif)

	// PayChUpdateNotif struct {

	// }
)

func SendPayChUpdate(ch *session.Channel, payee, amount string) error {

	f, err := newPayChStateUpdater(ch, ch.Currency, payee, amount)
	if err != nil {
		return err
	}
	return ch.SendChUpdate(f)
}

func newPayChStateUpdater(ch *session.Channel, chCurrency, payee, amount string) (session.StateUpdater, error) {
	parsedAmount, err := currency.NewParser(chCurrency).Parse(amount)
	if err != nil {
		ch.Logger.Error("Parsing amount", err)
		return nil, perun.ErrInvalidAmount
	}

	// find index
	var payerIdx, payeeIdx int
	if ch.Parts[0] == payee {
		payeeIdx = 0
	} else if ch.Parts[1] == payee {
		payeeIdx = 1
	} else {
		return nil, perun.ErrInvalidPayee
	}
	payerIdx = payeeIdx ^ 1

	// check sufficient balance
	bals := ch.Channel.State().Allocation.Clone().Balances[0]
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
