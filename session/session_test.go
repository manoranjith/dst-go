package session_test

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/contacts/contactstest"
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
		contacts, err := contactstest.NewProvider(1, contactstest.WalletBackend)
		require.NoError(t, err)
		registerer := &mocks.Registerer{}
		sess := session.Session{
			ChClient: chClient,
			Contacts: contacts,
			Dialer:   registerer,
			Channels: make(map[string]*session.Channel),
		}
		registerer.On("Register", mock.Anything, mock.Anything).Return()
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(&client.Channel{}, nil)
		testBalInfo := session.BalInfo{Currency: session.ETH, Bals: make(map[string]string)}
		testBalInfo.Bals["self"] = "5"
		testBalInfo.Bals["1"] = "10"
		_, err = sess.OpenCh("1", testBalInfo, session.App{}, 0)
		assert.NoError(t, err)
	})

	t.Run("error_proposeChannel", func(t *testing.T) {
		chClient := &mocks.ChannelClient{}
		contacts, err := contactstest.NewProvider(1, contactstest.WalletBackend)
		require.NoError(t, err)
		registerer := &mocks.Registerer{}
		sess := session.Session{
			ChClient: chClient,
			Contacts: contacts,
			Dialer:   registerer,
			Channels: make(map[string]*session.Channel),
		}
		registerer.On("Register", mock.Anything, mock.Anything).Return()
		chClient.On("ProposeChannel", mock.Anything, mock.Anything).Return(nil, errors.New("test-error"))
		testBalInfo := session.BalInfo{Currency: session.ETH, Bals: make(map[string]string)}
		testBalInfo.Bals["self"] = "5"
		testBalInfo.Bals["1"] = "10"
		_, err = sess.OpenCh("1", testBalInfo, session.App{}, 0)
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_SubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Session{}
		assert.NoError(t, ch.SubChProposals(func(*client.ChannelProposal, int64) {}))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := session.Session{}
		assert.NoError(t, ch.SubChProposals(func(*client.ChannelProposal, int64) {}))
		err := ch.SubChProposals(func(*client.ChannelProposal, int64) {})
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_Sub_UnsubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Session{}
		require.NoError(t, ch.SubChProposals(func(*client.ChannelProposal, int64) {}))
		assert.NoError(t, ch.UnsubChProposals())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := session.Session{}
		err := ch.UnsubChProposals()
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
			ProposalResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.ProposalResponders[proposalID] = proposalResponder
		assert.NoError(t, sess.RespondToChProposalNotif(proposalID, true))
	})

	t.Run("happy_reject", func(t *testing.T) {
		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Reject", mock.Anything, mock.Anything).Return(nil)
		sess := session.Session{
			ProposalResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.ProposalResponders[proposalID] = proposalResponder
		assert.NoError(t, sess.RespondToChProposalNotif(proposalID, false))
	})

	t.Run("error_accept_no_responder", func(t *testing.T) {
		sess := session.Session{
			ProposalResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		err := sess.RespondToChProposalNotif(proposalID, true)
		assert.Error(t, err)
	})

	t.Run("error_reject_no_responder", func(t *testing.T) {
		sess := session.Session{
			ProposalResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		err := sess.RespondToChProposalNotif(proposalID, false)
		assert.Error(t, err)
	})

	t.Run("error_accept", func(t *testing.T) {
		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Accept", mock.Anything, mock.Anything).Return(nil, errors.New("test-error"))
		sess := session.Session{
			ProposalResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.ProposalResponders[proposalID] = proposalResponder
		err := sess.RespondToChProposalNotif(proposalID, true)
		assert.Error(t, err)
	})

	t.Run("error_reject", func(t *testing.T) {
		proposalResponder := &mocks.ProposalResponder{}
		proposalResponder.On("Reject", mock.Anything, mock.Anything).Return(errors.New("test-error"))
		sess := session.Session{
			ProposalResponders: make(map[string]perun.ProposalResponder),
		}
		proposalID := "prop-1"
		sess.ProposalResponders[proposalID] = proposalResponder
		err := sess.RespondToChProposalNotif(proposalID, false)
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
