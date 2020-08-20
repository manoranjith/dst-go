package session_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/session"
)

var dummyProposalNotifier = func(perun.ChProposalNotif) {}

func Test_Session_SubPayChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.NewEmptySession()
		require.NoError(t, ch.SubChProposals(dummyProposalNotifier))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := session.NewEmptySession()
		require.NoError(t, ch.SubChProposals(dummyProposalNotifier))
		err := ch.SubChProposals(dummyProposalNotifier)
		assert.True(t, errors.Is(err, perun.ErrSubAlreadyExists))
	})
}

func Test_Session_Sub_UnsubChProposals(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.NewEmptySession()
		require.NoError(t, ch.SubChProposals(dummyProposalNotifier))
		assert.NoError(t, ch.UnsubChProposals())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := session.NewEmptySession()
		err := ch.UnsubChProposals()
		assert.True(t, errors.Is(err, perun.ErrNoActiveSub))
	})
}

var dummyChCloseNotifier = func(perun.ChCloseNotif) {}

func Test_Session_SubChClose(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.NewEmptySession()
		assert.NoError(t, ch.SubChCloses(dummyChCloseNotifier))
	})
	t.Run("error_already_subscribed", func(t *testing.T) {
		ch := session.NewEmptySession()
		assert.NoError(t, ch.SubChCloses(dummyChCloseNotifier))
		err := ch.SubChCloses(dummyChCloseNotifier)
		assert.True(t, errors.Is(err, perun.ErrSubAlreadyExists))
	})
}

func Test_Session_Sub_UnsubChClose(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.NewEmptySession()
		assert.NoError(t, ch.SubChCloses(dummyChCloseNotifier))
		assert.NoError(t, ch.UnsubChCloses())
	})

	t.Run("error_not_subscribed", func(t *testing.T) {
		ch := session.NewEmptySession()
		err := ch.UnsubChCloses()
		assert.True(t, errors.Is(err, perun.ErrNoActiveSub))
	})
}
