package session_test

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"perun.network/go-perun/client"
)

type ProposalNotifier struct{}

func (pn *ProposalNotifier) PayChProposalNotify(proposalID string, alias string, initBals session.BalInfo, ChallengeDurSecs uint64) {
}

type CloseNotifier struct{}

func (cn *CloseNotifier) PayChCloseNotify(finalBals session.BalInfo, _ error) {}

func Test_Interface_SessionAPI(t *testing.T) {
	// use this over assert.implements as this prints info on missing methods.
	var _ session.SessionAPI = &session.Session{}
}

func Test_Session_OpenPayCh(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		chClient := &mocks.ChannelClient{}
		sess := session.Session{
			ChClient: chClient,
		}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(&client.Channel{}, nil)
		ch, err := sess.OpenPayCh("", session.BalInfo{}, 0)
		assert.NoError(t, err)
		assert.NotNil(t, ch)
	})

	t.Run("error_proposeChannel", func(t *testing.T) {
		chClient := &mocks.ChannelClient{}
		sess := session.Session{
			ChClient: chClient,
		}
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(nil, errors.New("test-error"))
		_, err := sess.OpenPayCh("", session.BalInfo{}, 0)
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_SubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Session{}
		assert.NoError(t, ch.SubPayChProposals(&ProposalNotifier{}))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := session.Session{}
		assert.NoError(t, ch.SubPayChProposals(&ProposalNotifier{}))
		err := ch.SubPayChProposals(&ProposalNotifier{})
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_Sub_UnsubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Session{}
		require.NoError(t, ch.SubPayChProposals(&ProposalNotifier{}))
		assert.NoError(t, ch.UnsubPayChProposals())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := session.Session{}
		err := ch.UnsubPayChProposals()
		assert.Error(t, err)
	})
}

func Test_Session_Sub_RespondToPayChProposalNotif(t *testing.T) {
	t.Run("happy_accept", func(t *testing.T) {
		ch := &mocks.Channel{}
		ch.On("ID").Return([32]byte{1, 2, 3})

		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Accept", mock.Anything, mock.Anything).Return(ch, nil)
		sess := session.Session{
			Channels:        make(map[string]*session.Channel),
			PayChResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.PayChResponders[proposalID] = proposalResponder
		assert.NoError(t, sess.RespondToPayChProposalNotif(proposalID, true))
	})

	t.Run("happy_reject", func(t *testing.T) {
		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Reject", mock.Anything, mock.Anything).Return(nil)
		sess := session.Session{
			PayChResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.PayChResponders[proposalID] = proposalResponder
		assert.NoError(t, sess.RespondToPayChProposalNotif(proposalID, false))
	})

	t.Run("error_accept_no_responder", func(t *testing.T) {
		sess := session.Session{
			PayChResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		err := sess.RespondToPayChProposalNotif(proposalID, true)
		assert.Error(t, err)
	})

	t.Run("error_reject_no_responder", func(t *testing.T) {
		sess := session.Session{
			PayChResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		err := sess.RespondToPayChProposalNotif(proposalID, false)
		assert.Error(t, err)
	})

	t.Run("error_accept", func(t *testing.T) {
		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Accept", mock.Anything, mock.Anything).Return(nil, errors.New("test-error"))
		sess := session.Session{
			PayChResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.PayChResponders[proposalID] = proposalResponder
		err := sess.RespondToPayChProposalNotif(proposalID, true)
		assert.Error(t, err)
	})

	t.Run("error_reject", func(t *testing.T) {
		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Reject", mock.Anything, mock.Anything).Return(errors.New("test-error"))
		sess := session.Session{
			PayChResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.PayChResponders[proposalID] = proposalResponder
		err := sess.RespondToPayChProposalNotif(proposalID, false)
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_SubPayChClose(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Session{}
		assert.NoError(t, ch.SubPayChClose(&CloseNotifier{}))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := session.Session{}
		assert.NoError(t, ch.SubPayChClose(&CloseNotifier{}))
		err := ch.SubPayChClose(&CloseNotifier{})
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_Sub_UnsubPayChClose(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Session{}
		require.NoError(t, ch.SubPayChClose(&CloseNotifier{}))
		assert.NoError(t, ch.UnsubPayChClose())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := session.Session{}
		err := ch.UnsubPayChClose()
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_BytesToHex(t *testing.T) {
	t.Run("happy_byteSlice", func(t *testing.T) {
		input := []byte{1, 2, 3}
		wantOutout := "0x010203"
		assert.Equal(t, wantOutout, session.BytesToHex(input))
	})

	t.Run("happy_byteArray", func(t *testing.T) {
		input := [3]byte{1, 2, 3}
		wantOutout := "0x010203"
		assert.Equal(t, wantOutout, session.BytesToHex(input[:]))
	})
}
