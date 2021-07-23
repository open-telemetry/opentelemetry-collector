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

package otlptext

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestTracesText(t *testing.T) {
	type args struct {
		td pdata.Traces
	}
	tests := []struct {
		name  string
		args  args
		empty bool
	}{
		{"empty traces", args{pdata.NewTraces()}, true},
		{"traces with two spans", args{testdata.GenerateTracesTwoSpansSameResource()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traces, err := NewTextTracesMarshaler().MarshalTraces(tt.args.td)
			assert.NoError(t, err)
			if !tt.empty {
				assert.NotEmpty(t, traces)
			}
		})
	}
}
