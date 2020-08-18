package session_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node/log"
	"github.com/hyperledger-labs/perun-node/session"
)

var dummyProposalNotifier = func(session.ChProposalNotif) {}

func Test_Session_SubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := newEmptySession()
		require.NoError(t, ch.SubChProposals(dummyProposalNotifier))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := newEmptySession()
		require.NoError(t, ch.SubChProposals(dummyProposalNotifier))
		err := ch.SubChProposals(dummyProposalNotifier)
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_Sub_UnsubChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := newEmptySession()
		require.NoError(t, ch.SubChProposals(dummyProposalNotifier))
		assert.NoError(t, ch.UnsubChProposals())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := newEmptySession()
		err := ch.UnsubChProposals()
		assert.Error(t, err)
	})
}

var dummyChCloseNotifier = func(session.ChCloseNotif) {}

func Test_Session_SubChClose(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := newEmptySession()
		assert.NoError(t, ch.SubChCloses(dummyChCloseNotifier))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := newEmptySession()
		assert.NoError(t, ch.SubChCloses(dummyChCloseNotifier))
		err := ch.SubChCloses(dummyChCloseNotifier)
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Session_Sub_UnsubChClose(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := newEmptySession()
		assert.NoError(t, ch.SubChCloses(dummyChCloseNotifier))
		assert.NoError(t, ch.UnsubChCloses())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := newEmptySession()
		err := ch.UnsubChCloses()
		assert.Error(t, err)
		t.Log(err)
	})
}

func newEmptySession() session.Session {
	return session.Session{
		Logger: log.NewLoggerWithField("for", "test"),
	}
}
