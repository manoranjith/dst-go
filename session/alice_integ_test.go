package session_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hyperledger-labs/perun-node/session"
	"github.com/hyperledger-labs/perun-node/session/sessiontest"
)

var (
	aliceAlias = "1"
	alicePort  = 4341

	bobAlias = "2"
	bobPort  = 4342
)

func newSession(t *testing.T, role string) *session.Session {
	prng := rand.New(rand.NewSource(1729))
	newPaymentAppDef(t)

	_, aliceUser := sessiontest.NewTestUser(t, prng, uint(0))
	aliceUser.Alias = aliceAlias
	aliceUser.CommType = "tcp"
	aliceUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", 4341)

	_, bobUser := sessiontest.NewTestUser(t, prng, uint(0))
	bobUser.Alias = bobAlias
	bobUser.CommAddr = fmt.Sprintf("127.0.0.1:%d", 4342)
	bobUser.CommType = "tcp"

	switch role {
	case aliceAlias:
		alice := newTestSession(t, aliceUser)
		return alice
	case bobAlias:
		bob := newTestSession(t, bobUser)
		return bob
	}
	return nil
}
