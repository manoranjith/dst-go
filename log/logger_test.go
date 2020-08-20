// Copyright (c) 2020 - for information on the respective copyright owner
// see the NOTICE file and/or the repository at
// https://github.com/hyperledger-labs/perun-node
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/perun-node/log"
)

func Test_Loggger_Supported_LogLevels(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		for _, level := range []string{"error", "info", "debug"} {
			l, err := log.NewLogger(level, "")
			assert.NoError(t, err)
			assert.NotNil(t, l)
		}
	})

	t.Run("invalid levels", func(t *testing.T) {
		for _, level := range []string{"panic", "fatal", "warn"} {
			l, err := log.NewLogger(level, "")
			assert.Error(t, err)
			assert.Nil(t, l)
		}
	})
}
