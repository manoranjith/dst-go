package payment

import (
	"context"
	"fmt"
	"math/big"

	ppayment "perun.network/go-perun/apps/payment"
	pchannel "perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/currency"
)

type (
	PayChProposalNotif struct {
		ProposalID       string
		Currency         string
		OpeningBals      perun.BalInfo
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

func OpenPayCh(pctx context.Context, s perun.SessionAPI, peerAlias string, openingBals perun.BalInfo, challengeDurSecs uint64) (PayChInfo, error) {
	paymentApp := perun.App{
		Def:  ppayment.AppDef(),
		Data: &ppayment.NoData{},
	}

	chInfo, err := s.OpenCh(pctx, peerAlias, openingBals, paymentApp, challengeDurSecs)
	if err != nil {
		return PayChInfo{}, err
	}
	return PayChInfo{
		ChannelID: chInfo.ChannelID,
		BalInfo:   balsFromState(chInfo.Currency, chInfo.State, chInfo.Parts),
		Version:   fmt.Sprintf("%d", chInfo.State.Version),
	}, nil
}

func GetPayChs(s perun.SessionAPI) []PayChInfo {
	chInfos := s.GetChInfos()

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

func SubPayChProposals(s perun.SessionAPI, notifier PayChProposalNotifier) error {
	return s.SubChProposals(func(notif perun.ChProposalNotif) {
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

func RespondPayChProposal(pctx context.Context, s perun.SessionAPI, proposalID string, accept bool) error {
	return s.RespondChProposal(pctx, proposalID, accept)
}

func UnsubPayChProposals(s perun.SessionAPI) error {
	return s.UnsubChProposals()
}

func SubPayChCloses(s perun.SessionAPI, notifier PayChCloseNotifier) error {
	return s.SubChCloses(func(notif perun.ChCloseNotif) {
		notifier(PayChCloseNotif{
			ClosingState: PayChInfo{
				ChannelID: notif.ChannelID,
				BalInfo:   balsFromState(notif.Currency, notif.ChState, notif.Parts),
				Version:   fmt.Sprintf("%d", notif.ChState.Version),
			},
		})
	})
}

func UnsubPayChCloses(s perun.SessionAPI) error {
	return s.UnsubChCloses()
}

func balsFromState(currency string, state *pchannel.State, parts []string) perun.BalInfo {
	return balsFromBigInt(currency, state.Balances[0], parts)
}

func balsFromBigInt(chCurrency string, bigInt []*big.Int, parts []string) perun.BalInfo {
	balInfo := perun.BalInfo{
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
