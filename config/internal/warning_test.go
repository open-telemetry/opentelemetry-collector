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
)

func TestShouldWarn(t *testing.T) {
	tests := []struct {
		endpoint string
		warn     bool
	}{
		{
			endpoint: "0.0.0.0:0",
			warn:     true,
		},
		{
			endpoint: ":0",
			warn:     true,
		},
		{
			// Valid input for net.Listen
			endpoint: ":+0",
			warn:     true,
		},
		{
			// Valid input for net.Listen
			endpoint: ":-0",
			warn:     true,
		},
		{
			// Valid input for net.Listen, same as zero port.
			// https://github.com/golang/go/issues/13610
			endpoint: ":",
			warn:     true,
		},
		{
			endpoint: "127.0.0.1:0",
		},
		{
			endpoint: "localhost:0",
		},
		{
			// invalid, don't warn
			endpoint: "localhost::0",
		},
	}
	for _, test := range tests {
		t.Run(test.endpoint, func(t *testing.T) {
			assert.Equal(t, shouldWarn(test.endpoint), test.warn)
		})
	}

}
