package session_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"perun.network/go-perun/channel"

	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
)

type UpdateNotifier struct{}

func (un *UpdateNotifier) PayChUpdateNotify(alias string, bals session.BalInfo, ChannelgeDurSecs uint64) {
}

func Test_Interface_ChannelAPI(t *testing.T) {
	// use this over assert.implements as this prints info on missing methods.
	var _ session.ChannelAPI = &session.Channel{}
}

func Test_Channel_HasActiveSub(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		ch := session.Channel{
			UpdateNotify: func(_ *channel.State, expiry int64) {},
		}
		assert.True(t, ch.HasActiveSub())
	})
	t.Run("false", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
	})
}

func Test_Channel_SendChUpdate(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		assert.NoError(t, ch.SendChUpdate(func(state *channel.State) {}))
	})

	t.Run("Error_UpdateBy", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(errors.New("test-error"))
		err := ch.SendChUpdate(func(state *channel.State) {})
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Channel_SubChUpdates(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		assert.NoError(t, ch.SubChUpdates(func(state *channel.State, expiry int64) {}))
		assert.True(t, ch.HasActiveSub())
	})

	t.Run("error_multiple_calls", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		assert.NoError(t, ch.SubChUpdates(func(state *channel.State, expiry int64) {}))
		assert.True(t, ch.HasActiveSub())

		err := ch.SubChUpdates(func(state *channel.State, expiry int64) {})
		assert.Error(t, err)
		t.Log(err)
		assert.True(t, ch.HasActiveSub())
	})
}

func Test_Channel_Sub_UnsubChUpdates(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		assert.NoError(t, ch.SubChUpdates(func(state *channel.State, expiry int64) {}))
		assert.True(t, ch.HasActiveSub())

		assert.NoError(t, ch.UnsubChUpdates())
		assert.False(t, ch.HasActiveSub())
	})
	t.Run("error_no_active_subscription", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		err := ch.UnsubChUpdates()
		assert.Error(t, err)
		assert.False(t, ch.HasActiveSub())
	})
}

func Test_Channel_Sub_RespondToChUpdateNotif(t *testing.T) {
	t.Run("happy_accept", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		updateResponder.On("Accept", mock.Anything).Return(nil)
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		assert.NoError(t, ch.RespondToChUpdateNotif(true))
	})

	t.Run("happy_reject", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		// TODO: Check if first argument is a not nil context.
		updateResponder.On("Reject", mock.Anything, mock.AnythingOfType("string")).Return(nil)
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		assert.NoError(t, ch.RespondToChUpdateNotif(false))
	})

	t.Run("error_accept_no_responder", func(t *testing.T) {
		ch := session.Channel{}
		err := ch.RespondToChUpdateNotif(true)
		assert.Error(t, err)
	})

	t.Run("error_reject_no_responder", func(t *testing.T) {
		ch := session.Channel{}
		err := ch.RespondToChUpdateNotif(false)
		assert.Error(t, err)
	})

	t.Run("error_accept", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		updateResponder.On("Accept", mock.Anything).Return(errors.New("test error"))
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		err := ch.RespondToChUpdateNotif(true)
		assert.Error(t, err)
	})

	t.Run("error_reject", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		// TODO: Check if first argument is a not nil context.
		updateResponder.On("Reject", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("test error"))
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		err := ch.RespondToChUpdateNotif(false)
		assert.Error(t, err)
	})
}

func Test_Channel_GetBalance(t *testing.T) {}
func Test_Channel_CloseCh(t *testing.T) {
	t.Run("happy_update_no_error", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		controller.On("Settle", mock.Anything).Return(nil)
		controller.On("Close", mock.Anything).Return(nil)
		controller.On("State", mock.Anything).Return(&channel.State{})
		_, err := ch.CloseCh()
		assert.NoError(t, err)
	})

	t.Run("happy_update_error", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(errors.New("test-error"))
		controller.On("Settle", mock.Anything).Return(nil)
		controller.On("Close", mock.Anything).Return(nil)
		controller.On("State", mock.Anything).Return(&channel.State{})
		_, err := ch.CloseCh()
		assert.NoError(t, err)
	})

	t.Run("happy_close_error", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		controller.On("Settle", mock.Anything).Return(nil)
		controller.On("Close", mock.Anything).Return(errors.New("test-error"))
		controller.On("State", mock.Anything).Return(&channel.State{})
		_, err := ch.CloseCh()
		assert.NoError(t, err)
	})

	t.Run("error_settle", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		controller.On("Settle", mock.Anything).Return(errors.New("test-error"))
		controller.On("Close", mock.Anything).Return(nil)
		controller.On("State", mock.Anything).Return(&channel.State{})
		_, err := ch.CloseCh()
		assert.Error(t, err)
	})
}
