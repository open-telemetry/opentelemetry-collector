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

package goldendataset

import (
	"math/rand"
	"testing"

	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
)

func TestGenerateParentSpan(t *testing.T) {
	traceID := generateTestTraceID()
	span := GenerateSpan(traceID, "", nil, "/gotest-parent", SpanKindServer,
		SpanAttrHTTPServer, SpanChildCountTwo, SpanChildCountOne, SpanStatusOk)
	assert.Equal(t, traceID, span.TraceId)
	assert.Nil(t, span.ParentSpanId)
	assert.Equal(t, 11, len(span.Attributes))
	assert.Equal(t, otlptrace.Status_Ok, span.Status.Code)
}

func TestGenerateChildSpan(t *testing.T) {
	traceID := generateTestTraceID()
	parentID := generateTestSpanID()
	span := GenerateSpan(traceID, "", parentID, "get_test_info", SpanKindClient,
		SpanAttrDatabaseSQL, SpanChildCountEmpty, SpanChildCountNil, SpanStatusOk)
	assert.Equal(t, traceID, span.TraceId)
	assert.Equal(t, parentID, span.ParentSpanId)
	assert.Equal(t, 8, len(span.Attributes))
	assert.Equal(t, otlptrace.Status_Ok, span.Status.Code)
}

func generateTestTraceID() []byte {
	var r [16]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return r[:]
}

func generateTestSpanID() []byte {
	var r [8]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return r[:]
}
