// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterset"
)

// Config defines configuration for Resource processor.
type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	Action                         Action       `mapstructure:"action"`
	Metrics                        MetricFilter `mapstructure:"metrics"`
	Traces                         TraceFilter  `mapstructure:"traces"`
}

// Action is the enum to specify what happens to metrics and
// traces that match the filter.
type Action string

const (
	// INCLUDE means metrics or traces matching the filter will be
	// included in further processing, all other data will be dropped.
	INCLUDE Action = "include"
	// EXCLUDE means metrics or traces matching the filter will be excluded
	// from further processing and dropped, all other data will continue to be processed.
	EXCLUDE Action = "exclude"
)

// MetricFilter filters by Metric properties.
type MetricFilter struct {
	// NameFilter filters by the Name specified in the Metric's MetricDescriptor.
	NameFilter filterset.FilterConfig `mapstructure:"names"`
}

// TraceFilter filters by Span properties.
type TraceFilter struct {
	// NameFilter filters the Name specified in the Span.
	NameFilter filterset.FilterConfig `mapstructure:"names"`
}
