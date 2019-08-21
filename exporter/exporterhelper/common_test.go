// Copyright 2019, OpenTelemetry Authors
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
package exporterhelper

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opencensus.io/trace"
)

func TestDefaultOptions(t *testing.T) {
	checkRecordMetrics(t, newExporterOptions(), false)
	checkSpanName(t, newExporterOptions(), "")
}

func TestWithRecordMetrics(t *testing.T) {
	checkRecordMetrics(t, newExporterOptions(WithRecordMetrics(true)), true)
	checkRecordMetrics(t, newExporterOptions(WithRecordMetrics(false)), false)
}

func TestWithSpanName(t *testing.T) {
	checkSpanName(t, newExporterOptions(WithSpanName("my_span")), "my_span")
	checkSpanName(t, newExporterOptions(WithSpanName("")), "")
}

func TestErrorToStatus(t *testing.T) {
	require.Equal(t, okStatus, errToStatus(nil))
	require.Equal(t, trace.Status{Code: trace.StatusCodeUnknown, Message: "my_error"}, errToStatus(errors.New("my_error")))
}

func checkRecordMetrics(t *testing.T, opts ExporterOptions, recordMetrics bool) {
	assert.Equalf(t, opts.recordMetrics, recordMetrics, "Wrong recordMetrics Want: %t Got: %t", opts.recordMetrics, recordMetrics)
}

func checkSpanName(t *testing.T, opts ExporterOptions, spanName string) {
	assert.Equalf(t, opts.spanName, spanName, "Wrong spanName Want: %s Got: %s", opts.spanName, spanName)
}
