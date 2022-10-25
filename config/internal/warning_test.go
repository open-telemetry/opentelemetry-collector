// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/config/internal"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWarnOnUnspecifiedHost(t *testing.T) {
	tests := []struct {
		endpoint string
		warn     bool
		err      string
	}{
		{
			endpoint: "0.0.0.0:0",
			warn:     true,
		},
		{
			endpoint: "127.0.0.1:0",
		},
		{
			endpoint: "localhost:0",
		},
		{
			endpoint: "localhost::0",
			err:      "too many colons in address",
		},
	}
	for _, test := range tests {
		t.Run(test.endpoint, func(t *testing.T) {
			core, observed := observer.New(zap.DebugLevel)
			logger := zap.New(core)
			err := WarnOnUnspecifiedHost(logger, test.endpoint)
			if test.err != "" {
				assert.ErrorContains(t, err, test.err)
				return
			}

			require.NoError(t, err)

			var len int
			if test.warn {
				len = 1
			}
			require.Len(t, observed.FilterLevelExact(zap.WarnLevel).All(), len)
		})
	}

}
