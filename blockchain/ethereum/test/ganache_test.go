// +build integration

package ethereumtest_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ethereumtest "github.com/direct-state-transfer/dst-go/blockchain/ethereum/test"
)

func Test_NewHDWalletAccs(t *testing.T) {
	prng := rand.New(rand.NewSource(1729))
	cntAccs := 5
	accs := ethereumtest.NewHDWalletAccs(t, prng.Int63(), cntAccs)
	require.Len(t, accs, cntAccs)

	_, err := accs[0].SignData([]byte("test-string"))
	require.NoError(t, err)
}

func Test_NewGanacheBackendSetup(t *testing.T) {
	cntAccs := 5
	setup1 := ethereumtest.NewGanacheBackendSetup(t, cntAccs)
	require.True(t, setup1.Running())
	require.True(t, ethereumtest.ActiveTCPListener(setup1.GanacheAddr, 5*time.Second))
	require.Len(t, setup1.Accs, cntAccs)

	setup2 := ethereumtest.NewGanacheBackendSetup(t, cntAccs)

	require.True(t, setup2.Running())
	require.True(t, ethereumtest.ActiveTCPListener(setup2.GanacheAddr, 5*time.Second))
	require.Len(t, setup2.Accs, cntAccs)

	assert.Equal(t, setup1.GanacheAddr, setup2.GanacheAddr)
	assert.NotEqual(t, setup1.Accs[0], setup2.Accs[0])

}
