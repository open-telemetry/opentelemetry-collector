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

package goldendataset

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/data"
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
)

func TestGenerateParentSpan(t *testing.T) {
	random := rand.Reader
	traceID := generateTraceID(random)
	spanInputs := &PICTSpanInputs{
		Parent:     SpanParentRoot,
		Tracestate: TraceStateEmpty,
		Kind:       SpanKindServer,
		Attributes: SpanAttrHTTPServer,
		Events:     SpanChildCountTwo,
		Links:      SpanChildCountOne,
		Status:     SpanStatusOk,
	}
	span := GenerateSpan(traceID, data.NewSpanID([8]byte{}), "/gotest-parent", spanInputs, random)
	assert.Equal(t, traceID, span.TraceId)
	assert.True(t, span.ParentSpanId.IsEmpty())
	assert.Equal(t, 11, len(span.Attributes))
	assert.Equal(t, otlptrace.Status_STATUS_CODE_OK, span.Status.Code)
}

func TestGenerateChildSpan(t *testing.T) {
	random := rand.Reader
	traceID := generateTraceID(random)
	parentID := generateSpanID(random)
	spanInputs := &PICTSpanInputs{
		Parent:     SpanParentChild,
		Tracestate: TraceStateEmpty,
		Kind:       SpanKindClient,
		Attributes: SpanAttrDatabaseSQL,
		Events:     SpanChildCountEmpty,
		Links:      SpanChildCountNil,
		Status:     SpanStatusOk,
	}
	span := GenerateSpan(traceID, parentID, "get_test_info", spanInputs, random)
	assert.Equal(t, traceID, span.TraceId)
	assert.Equal(t, parentID, span.ParentSpanId)
	assert.Equal(t, 12, len(span.Attributes))
	assert.Equal(t, otlptrace.Status_STATUS_CODE_OK, span.Status.Code)
}

func TestGenerateSpans(t *testing.T) {
	random := rand.Reader
	count1 := 16
	spans, nextPos, err := GenerateSpans(count1, 0, "testdata/generated_pict_pairs_spans.txt", random)
	assert.Nil(t, err)
	assert.Equal(t, count1, len(spans))
	count2 := 256
	spans, nextPos, err = GenerateSpans(count2, nextPos, "testdata/generated_pict_pairs_spans.txt", random)
	assert.Nil(t, err)
	assert.Equal(t, count2, len(spans))
	count3 := 118
	spans, _, err = GenerateSpans(count3, nextPos, "testdata/generated_pict_pairs_spans.txt", random)
	assert.Nil(t, err)
	assert.Equal(t, count3, len(spans))
}
