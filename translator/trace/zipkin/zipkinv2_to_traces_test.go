// Copyright 2020, OpenTelemetry Authors
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

package zipkin

import (
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestZipkinSpansToInternalTraces(t *testing.T) {
	tests := []struct {
		name string
		zs   []*zipkinmodel.SpanModel
		td   pdata.Traces
		err  error
	}{
		{
			name: "empty",
			zs:   make([]*zipkinmodel.SpanModel, 0),
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			td, err := V2SpansToTraces(test.zs)
			assert.EqualValues(t, test.err, err)
			assert.Equal(t, len(test.zs), td.SpanCount())
			assert.EqualValues(t, test.td, td)
		})
	}
}
