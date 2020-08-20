package client

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/perun-node"
)

func Test_ChannelClient_Interface(t *testing.T) {
	assert.Implements(t, (*perun.ChannelClient)(nil), new(client))
}
