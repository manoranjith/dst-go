package session

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Parser interface {
	Parse(string) (*big.Int, error)
	Print(*big.Int) string
}

const ETH = "ETH"

var parsers map[string]Parser

func init() {
	parsers = make(map[string]Parser)

	ethMultiplier := decimal.NewFromFloat(params.Ether)
	parsers[ETH] = ethParser{multiplier: ethMultiplier, placesToRound: 6}
	parsers["E"] = nil
}

func Exists(currency string) bool {
	p, ok := parsers[currency]
	return ok && p != nil
}

// New parser returns the currency parser. It returns nil if invalid cure is used.
// so check if exists before usage.
func NewParser(currency string) Parser {
	return parsers[currency]
}

type ethParser struct {
	multiplier    decimal.Decimal
	placesToRound int32
}

func (p ethParser) Parse(input string) (*big.Int, error) {
	amount, err := decimal.NewFromString(input)
	if err != nil {
		return nil, errors.Wrap(err, "invalid decimal string")
	}

	amountBaseUnit := amount.Mul(p.multiplier)
	if amountBaseUnit.LessThan(decimal.NewFromInt(1)) {
		return nil, errors.New("amount is too small, should be larger than 1e-18")
	}
	return amountBaseUnit.BigInt(), nil
}

func (p ethParser) Print(input *big.Int) string {
	amount := decimal.NewFromBigInt(input, 0)
	return amount.Div(p.multiplier).StringFixedBank(p.placesToRound)
}
