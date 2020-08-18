package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// internal tests only check for errors is init() function in currency.go.
// see external tests file for other tests.
func Test_Exists_NewParser(t *testing.T) {
	t.Run("Err_Missing", func(t *testing.T) {
		assert.False(t, IsSupported("missing_parser_for_test"))
		assert.Nil(t, NewParser("missing_parser_for_test"))
	})

	t.Run("Err_Exists_but_nil", func(t *testing.T) {
		testCurrency := "nil_parser_for_test"
		currencies[testCurrency] = nil
		t.Cleanup(func() {
			delete(currencies, testCurrency)
		})

		assert.False(t, IsSupported(testCurrency))
		assert.Nil(t, NewParser("missing_parser_for_test"))
	})
}
