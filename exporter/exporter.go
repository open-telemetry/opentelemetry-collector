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

// Package exporter contains interfaces that wraps trace/metrics exporter.
package exporter

import (
	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
)

// Exporter defines functions that trace and metric exporters must implement.
type Exporter interface {
	component.Component
}

// TraceExporter is a TraceConsumer that is also an Exporter.
type TraceExporter interface {
	consumer.TraceConsumer
	Exporter
}

// TraceExporterV2 is an TraceConsumerV2 that is also an Exporter.
type TraceExporterV2 interface {
	consumer.TraceConsumerV2
	Exporter
}

// MetricsExporter is a MetricsConsumer that is also an Exporter.
type MetricsExporter interface {
	consumer.MetricsConsumer
	Exporter
}

// MetricsExporterV2 is a MetricsConsumerV2 that is also an Exporter.
type MetricsExporterV2 interface {
	consumer.MetricsConsumerV2
	Exporter
}
