// +build integration

package ethereumtest

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	ethwallet "perun.network/go-perun/backend/ethereum/wallet"

	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"perun.network/go-perun/wallet"
)

// DefaultHDWalletRootPath is the default path used for deriving keys in the hierarchial
// deterministic wallet.
const defaultHDPath = "m/44'/60'/0'/0/"

// defaultInstance holds the single instance of ganache setup that will be used across
// all integration tests.
var defaultInstance GanacheBackendSetup

// MaxAccs is the number of the funded accounts that will be created in ganache-cli and hd wallet
// for a single instance of setup. It is only an arbitrary choice. Can be changed if required.
const MaxAccs = 100

type (
	GanacheBackendSetup struct {
		GanacheAddr string
		Running     func() bool
		Accs        []wallet.Account
		Logfile     string
		cntUsedAccs int
	}

	// Account represents an account held in the HD wallet.
	// TODO: hdwallet.Wallet appears to be safe for concurrent use. But test should be added to verify this.
	Account struct {
		wallet *hdwallet.Wallet
		idx    int
	}
)

// Address implements wallet.Account.
func (a *Account) Address() wallet.Address {
	return ethwallet.AsWalletAddr(a.wallet.Accounts()[a.idx].Address)
}

// SignData implements wallet.Account.
func (a *Account) SignData(data []byte) ([]byte, error) {
	return a.wallet.SignData(a.wallet.Accounts()[a.idx], "", data)
}

// InitHDWallet initialized a hierarchial deterministic. It uses the given seed to derive
// the mnemoic for key generation and uses "m/44'/60'/0'/0/" as root path
//
// When used with ganache cli, the same seed and path.
func NewHDWalletAccs(t *testing.T, seed int64, n int) []wallet.Account {
	//rand package is directly used to Read function safe for concurrent use, while rand.Rand.Read method is not.
	rand.Seed(seed)
	walletSeed := make([]byte, 20)
	_, err := rand.Read(walletSeed)
	require.NoError(t, err)
	mnemonic, err := hdwallet.NewMnemonicFromEntropy(walletSeed)
	require.NoError(t, err)

	w, err := hdwallet.NewFromMnemonic(mnemonic)
	require.NoError(t, err)

	accs := make([]wallet.Account, n)
	for i := 0; i < n; i++ {
		path, err := hdwallet.ParseDerivationPath(fmt.Sprintf("%s%d", defaultHDPath, i))
		require.NoError(t, err)
		_, err = w.Derive(path, true)
		require.NoError(t, err)

		accs[i] = &Account{wallet: w, idx: i}
	}
	return accs
}

// NewGanacheBackendSetup returns a ganacheBackendSetup with n funded accounts, each with 100 ethers.
// It initializes the setup only once during first call with MaxAccs number of accounts. Each successive call
// returns a derived setup with same ganache instance and n unique accounts from the initial set.
//
// Requires a working instance of ganache-cli and corresponding dependecies to be installed on the system.
// Output from ganache are return to ganache_<unix_timestamp>.logs file in os temp dir.
//
// Singleton approach is preferred over separate ganche instances for speed (initializing an instance takes ~2s).
// TODO: Figure out a way to shutdown the singleton instance.
func NewGanacheBackendSetup(t *testing.T, n int) GanacheBackendSetup {
	require.LessOrEqualf(t, n+defaultInstance.cntUsedAccs, MaxAccs,
		fmt.Sprintf("max accs (%d) in ganache backend setup reached. increase limit in test helper pkg", MaxAccs))

	if defaultInstance.GanacheAddr == "" {
		logs, err := os.Create(filepath.Join(os.TempDir(), fmt.Sprintf("ganache_%d.log", time.Now().UnixNano())))
		require.NoError(t, err)

		prng := rand.New(rand.NewSource(1729))
		commonSeed := prng.Int63()
		defaultInstance = GanacheBackendSetup{
			Logfile:     logs.Name(),
			Accs:        NewHDWalletAccs(t, commonSeed, MaxAccs),
			cntUsedAccs: 0,
		}
		defaultInstance.GanacheAddr, defaultInstance.Running = newGanacheBackend(t, commonSeed, MaxAccs, logs)
	}

	oldCnt := defaultInstance.cntUsedAccs
	accs := defaultInstance.Accs[oldCnt : oldCnt+n]
	defaultInstance.cntUsedAccs = oldCnt + n
	return GanacheBackendSetup{
		GanacheAddr: defaultInstance.GanacheAddr,
		Running:     defaultInstance.Running,
		Accs:        accs,
		Logfile:     defaultInstance.Logfile,
	}
}

// NewGanacheBackend starts a ganache instance at an arbitrary free port and returns a func to
// check if is running. It also generates n funded accounts with 100 ethers each (default value in ganache-cli).
// A function to close the ganache instance is added to t.Cleanup.
//
// The accounts are generated deterministically by derving a mnemonic from the given seed and
// "m/44'/60'/0'/0/" as root path. Use a hd wallet with same path and mnemonic to access the keys.
func newGanacheBackend(t *testing.T, seed int64, n int, log io.Writer) (addr string, running func() bool) {
	//rand package is directly used to Read function safe for concurrent use, while rand.Rand.Read method is not.
	rand.Seed(seed)
	walletSeed := make([]byte, 20)
	_, err := rand.Read(walletSeed)
	require.NoError(t, err)
	mnemonic, err := hdwallet.NewMnemonicFromEntropy(walletSeed)
	require.NoError(t, err)

	port, err := freeport.GetFreePort()
	require.NoErrorf(t, err, "cannot find free ports for starting ganache")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	args := []string{"-p", strconv.Itoa(port), "--hd_path", defaultHDPath, "-m", mnemonic, "-a", strconv.Itoa(n), "-v"}
	cmd := exec.CommandContext(ctx, "ganache-cli", args...)
	cmd.Stderr = log
	cmd.Stdout = log

	ganacheErr := make(chan error, 1)
	running = func() bool {
		select {
		case <-ganacheErr:
			return false
		default:
			return true
		}
	}

	go func() {
		ganacheErr <- cmd.Run()
	}()

	addr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	require.True(t, ActiveTCPListener(addr, 5*time.Second))
	return
}

// ActiveTCPListener returns true if any program is listening for tcp connections at the given address.
// It retries every 100 ms until the given timeout expires.
func ActiveTCPListener(addr string, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	for {
		select {
		case <-timer.C:
			return false
		default:
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err == nil {
				conn.Close()
				return true
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
