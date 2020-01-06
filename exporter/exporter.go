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

// TraceExporter composes TraceConsumer with some additional exporter-specific functions.
type TraceExporter interface {
	consumer.TraceConsumer
	Exporter
}

// MetricsExporter composes MetricsConsumer with some additional exporter-specific functions.
type MetricsExporter interface {
	consumer.MetricsConsumer
	Exporter
}
