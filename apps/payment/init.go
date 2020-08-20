package payment

import (
	"github.com/hyperledger-labs/perun-node/blockchain/ethereum"

	ppayment "perun.network/go-perun/apps/payment"
)

// init() initializes the payment app in go-perun.
func init() {
	wb := ethereum.NewWalletBackend()
	emptyAddr, err := wb.ParseAddr("0x0")
	if err != nil {
		panic("Error parsing zero address for app payment def: " + err.Error())
	}
	ppayment.SetAppDef(emptyAddr) // dummy app def.
}
