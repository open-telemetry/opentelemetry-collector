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

package exporter

import (
	"github.com/census-instrumentation/opencensus-service/consumer"
)

// TraceExporter composes TraceConsumer with some additional exporter-specific functions.
type TraceExporter interface {
	consumer.TraceConsumer

	// TraceExportFormat gets the name of the format in which this exporter sends its data.
	// For exporters that can export multiple signals it is recommended to encode the signal
	// as suffix (e.g. "oc_trace").
	TraceExportFormat() string
}

// MetricsExporter composes MetricsConsumer with some additional exporter-specific functions.
type MetricsExporter interface {
	consumer.MetricsConsumer

	// MetricsExportFormat gets the name of the format in which this exporter sends its data.
	// For exporters that can export multiple signals it is recommended to encode the signal
	// as suffix (e.g. "oc_metrics").
	MetricsExportFormat() string
}

// Exporter is union of trace and/or metrics exporter.
type Exporter interface {
	TraceExporter
	MetricsExporter
	Stop() error
}
