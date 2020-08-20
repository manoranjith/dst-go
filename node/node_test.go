package node_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger-labs/perun-node/node"
)

// func Test_Integ_New(t *testing.T) {

// 	addr1 := "0x1681807986951466585898338058137657239292"
// 	addr2 := "0x2681807986951466585898338058137657239292"
// 	addr3 := "0x3681807986951466585898338058137657239292"
// 	// addr4 := "0x4681807986951466585898338058137657239292"
// 	n, err := node.New(addr1, addr2, addr3, "debug", "")
// 	require.NoError(t, err)
// 	require.NotNil(t, n)
// 	fmt.Printf("%+v", n)

// }

var (
	testdataDir     = "../testdata/node"
	validConfigFile = "valid.yaml"
)

func Test_ParseConfig(t *testing.T) {
	cfg, err := node.ParseConfig(filepath.Join(testdataDir, validConfigFile))
	require.NoError(t, err)
	fmt.Printf("%+v\n", cfg)
}
