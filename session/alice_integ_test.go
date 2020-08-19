package session_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hyperledger-labs/perun-node"
	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

var (
	aliceAlias = "alice"
	alicePort  = 4341

	bobAlias = "bob"
	bobPort  = 4342
)

func newSession(t *testing.T, role string) (*session.Session, perun.Peer) {
	prng := rand.New(rand.NewSource(1729))
	newPaymentAppDef(t)

	_, aliceUser := sessiontest.NewTestUser(t, prng, uint(0))
	aliceUser.Alias = aliceAlias
	aliceUser.CommType = "tcp"
	aliceUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", 4341)
	aliceUser.OffChainAddrString = aliceUser.OffChainAddr.String()

	_, bobUser := sessiontest.NewTestUser(t, prng, uint(0))
	bobUser.Alias = bobAlias
	bobUser.CommType = "tcp"
	bobUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", 4342)
	bobUser.OffChainAddrString = bobUser.OffChainAddr.String()

	switch role {
	case aliceAlias:
		alice := newTestSession(t, aliceUser)
		return alice, bobUser.Peer
	case bobAlias:
		bob := newTestSession(t, bobUser)
		return bob, aliceUser.Peer
	}
	return nil, perun.Peer{}
}

// func Test_Integ_Role_Alice(t *testing.T) {

// 	alice, gotBobContact := newSession(t, aliceAlias)
// }
