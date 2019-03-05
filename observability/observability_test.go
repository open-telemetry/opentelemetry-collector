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

// observability_test instead of just observability to avoid dependency cycle between
// observability and observabilitytest
package observability_test

// This file contains helpers that are useful to add observability
// with metrics and tracing using OpenCensus to the various pieces
// of the service.

import (
	"context"
	"testing"

	"github.com/census-instrumentation/opencensus-service/observability"
	"github.com/census-instrumentation/opencensus-service/observability/observabilitytest"
)

const (
	receiverName = "fake_receiver"
	exporterName = "fake_exporter"
)

func TestTracePieplineRecordedMetrics(t *testing.T) {
	defer observabilitytest.SetupRecordedMetricsTest(t)()

	receiverCtx := observability.ContextWithReceiverName(context.Background(), receiverName)
	observability.RecordTraceReceiverMetrics(receiverCtx, 17, 13)
	exporterCtx := observability.ContextWithExporterName(receiverCtx, exporterName)
	observability.RecordTraceExporterMetrics(exporterCtx, 27, 23)
	observabilitytest.CheckValueViewReceiverReceivedSpans(t, receiverName, 17)
	observabilitytest.CheckValueViewReceiverDroppedSpans(t, receiverName, 13)
	observabilitytest.CheckValueViewExporterReceivedSpans(t, receiverName, exporterName, 27)
	observabilitytest.CheckValueViewExporterDroppedSpans(t, receiverName, exporterName, 23)
}
