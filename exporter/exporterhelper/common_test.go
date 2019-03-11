// Copyright 2019, OpenCensus Authors
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
	if g, w := errToStatus(nil), okStatus; g != w {
		t.Fatalf("Status: Want %v Got %v", w, g)
	}

	errorStatus := trace.Status{Code: trace.StatusCodeUnknown, Message: "my_error"}
	if g, w := errToStatus(errors.New("my_error")), errorStatus; g != w {
		t.Fatalf("Status: Want %v Got %v", w, g)
	}
}

func checkRecordMetrics(t *testing.T, opts ExporterOptions, recordMetrics bool) {
	if opts.recordMetrics != recordMetrics {
		t.Errorf("Wrong recordMetrics Want: %t Got: %t", opts.recordMetrics, recordMetrics)
		return
	}
}

func checkSpanName(t *testing.T, opts ExporterOptions, spanName string) {
	if opts.spanName != spanName {
		t.Errorf("Wrong spanName Want: %s Got: %s", opts.spanName, spanName)
		return
	}
}
