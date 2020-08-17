package session

import (
	"fmt"
	"math/big"

	"github.com/hyperledger-labs/perun-node"
	"github.com/pkg/errors"
	"perun.network/go-perun/apps/payment"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/client"
)

// Currency is a parameter for this app.
const currency = "currency"

type PaymentChannelAPI interface {
	SendPayChUpdate(payee, amount string) error
	SubPayChUpdates(PayChUpdateNotify) error // Err if subscription exists.
	UnsubPayChUpdates() error                // Err if there is no subscription.
	RespondToPayChUpdateNotif(accept bool) error
	GetBalance() BalInfo
	ClosePayCh() (finalBals BalInfo, _ error)
}

type PayChUpdateNotify interface {
	Notify(bals BalInfo, version string, final bool, expiry int64)
}

// Channel API

func SendPayChUpdate(ch *Channel, payee, amount string) error {
	updater, err := newPaymentStateUpdater(ch, payee, amount)
	if err != nil {
		return err
	}
	return ch.SendChUpdate(updater)
}

func SubPayChUpdates(ch *Channel, f PayChUpdateNotify) error {
	return ch.SubChUpdates(func(s *channel.State, expiry int64) {
		balInfo := balsFromState(ch.AppParams[currency], s, ch.peers)
		f.Notify(balInfo, fmt.Sprintf("%d", s.Version), s.IsFinal, expiry)
	})
}

func UnsubPayChUpdates(ch *Channel) error {
	return ch.UnsubChUpdates()
}

func RespondToPayChUpdateNotif(ch *Channel, accept bool) error {
	return ch.RespondToChUpdateNotif(accept)

}

func GetBalance(c *Channel) BalInfo {
	return balsFromState(ETH, c.GetState(), c.peers)
}

func ClosePayCh(ch *Channel) (BalInfo, error) {
	closingState, err := ch.CloseCh()
	if err != nil {
		return BalInfo{}, err
	}
	return balsFromState(ch.AppParams[currency], closingState, ch.peers), nil
}

type BalInfo struct {
	Currency string
	Bals     map[string]string // Map of alias to balance.
}

func balsFromState(currency string, state *channel.State, addrs []string) BalInfo {
	return balsFromBigInt(currency, state.Balances[0], addrs)
}

func balsFromBigInt(currency string, bigInt []*big.Int, addrs []string) BalInfo {
	balInfo := BalInfo{
		Currency: currency,
		Bals:     make(map[string]string, len(addrs)),
	}

	parser := NewParser(currency)
	for i := range addrs {
		balInfo.Bals[addrs[i]] = parser.Print(bigInt[i])
		balInfo.Bals[addrs[i]] = parser.Print(bigInt[i])
	}
	return balInfo
}

func newPaymentStateUpdater(ch *Channel, payee, amount string) (StateUpdater, error) {
	parsedAmount, err := NewParser(ch.AppParams[currency]).Parse(amount)
	if err != nil {
		return nil, perun.NewAPIError(perun.ErrInvalidAmount, err)
	}

	// find index
	var senderIdx, receiverIdx int
	if ch.peers[0] == payee {
		receiverIdx = 0
	} else {
		receiverIdx = 1
	}
	senderIdx = receiverIdx ^ 1

	// check sufficient balance
	bals := ch.Controller.State().Allocation.Clone().Balances[0]
	if bals == nil {
		return nil, perun.NewAPIError(perun.ErrInternalServer, errors.New("balances are nil"))
	}
	bals[senderIdx].Sub(bals[senderIdx], parsedAmount)
	bals[receiverIdx].Add((bals[receiverIdx]), parsedAmount)
	if bals[senderIdx].Sign() == -1 {
		return nil, perun.NewAPIError(perun.ErrInsufficientBal, nil)
	}

	// return updater func
	return func(state *channel.State) {
		state.Allocation.Balances[0][senderIdx] = bals[senderIdx]
		state.Allocation.Balances[0][receiverIdx] = bals[receiverIdx]
	}, nil

}

type StateUpdater func(*channel.State)
type StateDecoder func(_ *channel.State, expiry int64)

// Session API

type PayChState struct {
	channelID string
	BalInfo   BalInfo
	Version   string
}

func OpenPayCh(s *Session, peerAlias string, openingBals BalInfo, ChDurSecs uint64) (PayChState, error) {
	paymentApp := App{
		Def:  payment.AppDef(),
		Data: &payment.NoData{},
	}

	ch, err := s.OpenCh(peerAlias, openingBals, paymentApp, ChDurSecs)
	if err != nil {
		return PayChState{}, perun.NewAPIError(perun.ErrInternalServer, errors.Wrap(err, "Proposing Channel"))
	}
	// TODO: Use NewChannel function to prepare other factors
	return PayChState{
		channelID: ch.ID,
		BalInfo:   balsFromState(ch.AppParams[currency], ch.Controller.State(), ch.peers),
		Version:   fmt.Sprintf("%d", ch.Controller.State().Version),
	}, nil
}

type ProposalNotifier func(_ *client.ChannelProposal, expiry int64)

// func SubPayChProposals

func SubPayChProposals(s *Session, f PayChProposalNotifier) error {
	return s.SubChProposals(func(p *client.ChannelProposal, expiry int64) {
		proposalID := p.ProposalID()
		// if p.AppDef.Equals() // TODO: Check if AppDef is same.
		peers := make([]string, len(p.PeerAddrs))
		for i := range peers {
			peer, ok := s.Contacts.ReadByOffChainAddr("")
			if !ok {
				// Drop subscription and reject it with reason = unknown part
			}
			peers[i] = peer.Alias
		}
		// How to get the currency here ?
		balInfo := balsFromBigInt(currency, p.InitBals.Balances[0], peers)

		f.PayChProposalNotify(BytesToHex(proposalID[:]), BytesToHex(proposalID[:]), balInfo, p.ChallengeDuration, expiry)
	})
}

func UnSubPayChProposal(s *Session) error {
	return s.UnsubChProposals()
}

func RespondToChProposalNotif(s *Session, proposalID string, accept bool) error {
	return s.RespondToChProposalNotif(proposalID, accept)
}

func GetPayChs(s *Session) []PayChState {
	chs := s.GetChs()
	payChs := make([]PayChState, len(chs))
	for idx := range chs {
		payChs[idx] = PayChState{
			channelID: chs[idx].ID,
			BalInfo:   balsFromState(chs[idx].AppParams[currency], chs[idx].Controller.State(), chs[idx].peers),
			Version:   fmt.Sprintf("%d", chs[idx].Controller.State().Version),
		}
	}
	return payChs
}

func CloseSession(s *Session, persistOpenCh bool) ([]Channel, error) {
	_, err := s.CloseSession(persistOpenCh)
	if err != nil {
		return nil, err
	}
	// Send open pay chs here.
	return nil, nil

}
