package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// internal tests only check for errors is init() function in currency.go.
// see external tests file for other tests.
func Test_Exists_NewParser(t *testing.T) {
	t.Run("Err_Missing", func(t *testing.T) {
		assert.False(t, Exists("missing_parser_for_test"))
		assert.Nil(t, NewParser("missing_parser_for_test"))
	})

	t.Run("Err_Exists_but_nil", func(t *testing.T) {
		testCurrency := "nil_parser_for_test"
		parsers[testCurrency] = nil
		t.Cleanup(func() {
			delete(parsers, testCurrency)
		})

		assert.False(t, Exists(testCurrency))
		assert.Nil(t, NewParser("missing_parser_for_test"))
	})
}
