package payment

import (
	"fmt"
	"math/big"

	"perun.network/go-perun/apps/payment"
	"perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node/currency"
	"github.com/hyperledger-labs/perun-node/session"
)

type (
	PayChProposalNotif struct {
		ProposalID       string
		Currency         string
		OpeningBals      session.BalInfo
		ChallengeDurSecs uint64
		Expiry           int64
	}

	PayChProposalNotifier func(PayChProposalNotif)

	PayChCloseNotif struct {
		ClosingState PayChInfo
		Error        string
	}

	PayChCloseNotifier func(PayChCloseNotif)
)

func OpenPayCh(s *session.Session, peerAlias string, openingBals session.BalInfo, challengeDurSecs uint64) (PayChInfo, error) {
	paymentApp := session.App{
		Def:  payment.AppDef(),
		Data: &payment.NoData{},
	}

	chInfo, err := s.OpenCh(peerAlias, openingBals, paymentApp, challengeDurSecs)
	if err != nil {
		return PayChInfo{}, err
	}
	return PayChInfo{
		ChannelID: chInfo.ChannelID,
		BalInfo:   balsFromState(chInfo.Currency, chInfo.State, chInfo.Parts),
		Version:   fmt.Sprintf("%d", chInfo.State.Version),
	}, nil
}

func GetPayChs(s *session.Session) []PayChInfo {
	chInfos := s.GetChs()

	payChInfos := make([]PayChInfo, len(chInfos))
	for i := range chInfos {
		payChInfos[i] = PayChInfo{
			ChannelID: chInfos[i].ChannelID,
			BalInfo:   balsFromState(chInfos[i].Currency, chInfos[i].State, chInfos[i].Parts),
			Version:   fmt.Sprintf("%d", chInfos[i].State.Version),
		}

	}
	return payChInfos
}

func SubPayChProposals(s *session.Session, notifier PayChProposalNotifier) error {
	return s.SubChProposals(func(notif session.ChProposalNotif) {
		balsBigInt := notif.Proposal.InitBals.Balances[0]
		notifier(PayChProposalNotif{
			ProposalID:       notif.ProposalID,
			Currency:         notif.Currency,
			OpeningBals:      balsFromBigInt("ETH", balsBigInt, notif.Parts),
			ChallengeDurSecs: notif.Proposal.ChallengeDuration,
			Expiry:           notif.Expiry,
		})
	})
}

func RespondPayChProposal(s *session.Session, proposalID string, accept bool) error {
	return s.RespondChProposal(proposalID, accept)
}

func UnsubPayChProposals(s *session.Session) error {
	return s.UnsubChProposals()
}

func SubPayChCloses(s *session.Session, notifier PayChCloseNotifier) error {
	return s.SubChCloses(func(notif session.ChCloseNotif) {
		notifier(PayChCloseNotif{
			ClosingState: PayChInfo{
				ChannelID: notif.ChannelID,
				BalInfo:   balsFromState(notif.Currency, notif.ChState, notif.Parts),
				Version:   fmt.Sprintf("%d", notif.ChState.Version),
			},
		})
	})
}

func UnsubPayChCloses(s *session.Session) error {
	return s.UnsubChCloses()
}

func balsFromState(currency string, state *channel.State, parts []string) session.BalInfo {
	return balsFromBigInt(currency, state.Balances[0], parts)
}

func balsFromBigInt(chCurrency string, bigInt []*big.Int, parts []string) session.BalInfo {
	balInfo := session.BalInfo{
		Currency: chCurrency,
		Bals:     make(map[string]string, len(parts)),
	}

	parser := currency.NewParser(chCurrency)
	for i := range parts {
		balInfo.Bals[parts[i]] = parser.Print(bigInt[i])
		balInfo.Bals[parts[i]] = parser.Print(bigInt[i])
	}
	return balInfo
}
