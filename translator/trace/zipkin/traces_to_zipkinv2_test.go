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

package zipkin

import (
	"errors"
	"io"
	"math/rand"
	"testing"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/internal/goldendataset"
)

func TestInternalTracesToZipkinSpans(t *testing.T) {
	tests := []struct {
		name string
		td   pdata.Traces
		zs   []*zipkinmodel.SpanModel
		err  error
	}{
		{
			name: "empty",
			td:   testdata.GenerateTraceDataEmpty(),
			err:  nil,
		},
		{
			name: "oneEmpty",
			td:   testdata.GenerateTraceDataOneEmptyResourceSpans(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneEmptyOneNil",
			td:   testdata.GenerateTraceDataOneEmptyOneNilResourceSpans(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "noLibs",
			td:   testdata.GenerateTraceDataNoLibraries(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneEmptyLib",
			td:   testdata.GenerateTraceDataOneEmptyInstrumentationLibrary(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneEmptyLibOneNilLib",
			td:   testdata.GenerateTraceDataOneEmptyOneNilInstrumentationLibrary(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  nil,
		},
		{
			name: "oneSpanNoResrouce",
			td:   testdata.GenerateTraceDataOneSpanNoResource(),
			zs:   make([]*zipkinmodel.SpanModel, 0),
			err:  errors.New("TraceID is nil"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			zss, err := InternalTracesToZipkinSpans(test.td)
			assert.EqualValues(t, test.err, err)
			if test.name == "empty" {
				assert.Nil(t, zss)
			} else {
				assert.Equal(t, len(test.zs), len(zss))
				assert.EqualValues(t, test.zs, zss)
			}
		})
	}
}

func TestInternalTracesToZipkinSpansAndBack(t *testing.T) {
	rscSpans, err := goldendataset.GenerateResourceSpans(
		"../../../internal/goldendataset/testdata/generated_pict_pairs_traces.txt",
		"../../../internal/goldendataset/testdata/generated_pict_pairs_spans.txt",
		io.Reader(rand.New(rand.NewSource(2004))))
	assert.NoError(t, err)
	for _, rs := range rscSpans {
		orig := make([]*otlptrace.ResourceSpans, 1)
		orig[0] = rs
		td := pdata.TracesFromOtlp(orig)
		zipkinSpans, err := InternalTracesToZipkinSpans(td)
		assert.NoError(t, err)
		assert.Equal(t, td.SpanCount(), len(zipkinSpans))
		tdFromZS, zErr := V2SpansToInternalTraces(zipkinSpans)
		assert.NoError(t, zErr)
		assert.NotNil(t, tdFromZS)
		assert.Equal(t, td.SpanCount(), tdFromZS.SpanCount())
	}
}
