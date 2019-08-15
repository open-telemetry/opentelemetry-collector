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
	"github.com/open-telemetry/opentelemetry-service/consumer"
)

// TraceExporter composes TraceConsumer with some additional exporter-specific functions.
type TraceExporter interface {
	consumer.TraceConsumer

	// Name gets the name of the trace exporter.
	Name() string

	// Shutdown is invoked during service shutdown.
	Shutdown() error
}

// MetricsExporter composes MetricsConsumer with some additional exporter-specific functions.
type MetricsExporter interface {
	consumer.MetricsConsumer

	// Name gets the name of the metrics exporter.
	Name() string

	// Shutdown is invoked during service shutdown.
	Shutdown() error
}
