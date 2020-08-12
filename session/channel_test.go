package session_test

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/perun-node/internal/mocks"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type UpdateNotifier struct{}

func (un *UpdateNotifier) PayChUpdateNotify(alias string, bals session.BalInfo, ChannelgeDurSecs uint64) {
}

func Test_Channel_HasActiveSub(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		ch := session.Channel{
			UpdateNotify: &UpdateNotifier{},
		}
		assert.True(t, ch.HasActiveSub())
	})
	t.Run("false", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
	})
}

func Test_Channel_SendPayChUpdate(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		assert.NoError(t, ch.SendPayChUpdate("", ""))
	})

	t.Run("Error_UpdateBy", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(errors.New("test-error"))
		err := ch.SendPayChUpdate("", "")
		assert.Error(t, err)
		t.Log(err)
	})
}

func Test_Channel_SubPayChUpdates(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		assert.NoError(t, ch.SubPayChUpdates(&UpdateNotifier{}))
		assert.True(t, ch.HasActiveSub())
	})

	t.Run("error_multiple_calls", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		assert.NoError(t, ch.SubPayChUpdates(&UpdateNotifier{}))
		assert.True(t, ch.HasActiveSub())

		err := ch.SubPayChUpdates(&UpdateNotifier{})
		assert.Error(t, err)
		t.Log(err)
		assert.True(t, ch.HasActiveSub())
	})
}

func Test_Channel_Sub_UnsubPayChUpdates(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		assert.NoError(t, ch.SubPayChUpdates(&UpdateNotifier{}))
		assert.True(t, ch.HasActiveSub())

		assert.NoError(t, ch.UnsubPayChUpdates())
		assert.False(t, ch.HasActiveSub())
	})
	t.Run("error_no_active_subscription", func(t *testing.T) {
		ch := session.Channel{}
		assert.False(t, ch.HasActiveSub())
		err := ch.UnsubPayChUpdates()
		assert.Error(t, err)
		assert.False(t, ch.HasActiveSub())
	})
}

func Test_Channel_Sub_RespondToPayChUpdateNotif(t *testing.T) {
	t.Run("happy_accept", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		updateResponder.On("Accept", mock.Anything).Return(nil)
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		assert.NoError(t, ch.RespondToPayChUpdateNotif(true))
	})

	t.Run("happy_reject", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		// TODO: Check if first argument is a not nil context.
		updateResponder.On("Reject", mock.Anything, mock.AnythingOfType("string")).Return(nil)
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		assert.NoError(t, ch.RespondToPayChUpdateNotif(false))
	})

	t.Run("error_accept_no_responder", func(t *testing.T) {
		ch := session.Channel{}
		err := ch.RespondToPayChUpdateNotif(true)
		assert.Error(t, err)
	})

	t.Run("error_reject_no_responder", func(t *testing.T) {
		ch := session.Channel{}
		err := ch.RespondToPayChUpdateNotif(false)
		assert.Error(t, err)
	})

	t.Run("error_accept", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		updateResponder.On("Accept", mock.Anything).Return(errors.New("test error"))
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		err := ch.RespondToPayChUpdateNotif(true)
		assert.Error(t, err)
	})

	t.Run("error_reject", func(t *testing.T) {
		updateResponder := &mocks.UpdateResponder{}
		// TODO: Check if first argument is a not nil context.
		updateResponder.On("Reject", mock.Anything, mock.AnythingOfType("string")).Return(errors.New("test error"))
		ch := session.Channel{
			UpdateResponders: updateResponder,
		}
		err := ch.RespondToPayChUpdateNotif(false)
		assert.Error(t, err)
	})
}

func Test_Channel_GetBalance(t *testing.T) {}
func Test_Channel_ClosePayCh(t *testing.T) {
	t.Run("happy_update_no_error", func(t *testing.T) {
		controller := &mocks.Channel{}
		ch := session.Channel{
			Controller: controller,
		}
		controller.On("UpdateBy", mock.Anything, mock.Anything).Return(nil)
		controller.On("Settle", mock.Anything).Return(nil)
		controller.On("Close", mock.Anything).Return(nil)
		_, err := ch.ClosePayCh()
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
		_, err := ch.ClosePayCh()
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
		_, err := ch.ClosePayCh()
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
		_, err := ch.ClosePayCh()
		assert.Error(t, err)
	})
}
