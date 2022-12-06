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

// Package otelcol handles the command-line, configuration, and runs the
// OpenTelemetry Collector.
package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"go.opentelemetry.io/collector/service"
)

// State defines Collector's state.
type State = service.State //nolint:staticcheck

const (
	StateStarting = service.StateStarting //nolint:staticcheck
	StateRunning  = service.StateRunning  //nolint:staticcheck
	StateClosing  = service.StateClosing  //nolint:staticcheck
	StateClosed   = service.StateClosed   //nolint:staticcheck
)

// Collector represents a server providing the OpenTelemetry Collector service.
type Collector = service.Collector //nolint:staticcheck

// NewCollector creates and returns a new instance of Collector.
var NewCollector = service.New //nolint:staticcheck

// CollectorSettings holds configuration for creating a new Collector.
type CollectorSettings = service.CollectorSettings //nolint:staticcheck

// ConfigProvider provides the service configuration.
//
// The typical usage is the following:
//
//	cfgProvider.Get(...)
//	cfgProvider.Watch() // wait for an event.
//	cfgProvider.Get(...)
//	cfgProvider.Watch() // wait for an event.
//	// repeat Get/Watch cycle until it is time to shut down the Collector process.
//	cfgProvider.Shutdown()
type ConfigProvider = service.ConfigProvider //nolint:staticcheck

// ConfigProviderSettings are the settings to configure the behavior of the ConfigProvider.
type ConfigProviderSettings = service.ConfigProviderSettings //nolint:staticcheck

// NewConfigProvider returns a new ConfigProvider that provides the service configuration:
// * Initially it resolves the "configuration map":
//   - Retrieve the confmap.Conf by merging all retrieved maps from the given `locations` in order.
//   - Then applies all the confmap.Converter in the given order.
//
// * Then unmarshalls the confmap.Conf into the service Config.
var NewConfigProvider = service.NewConfigProvider //nolint:staticcheck

// NewCommand constructs a new cobra.Command using the given CollectorSettings.
var NewCommand = service.NewCommand //nolint:staticcheck

// Config defines the configuration for the various elements of collector or agent.
type Config = service.Config //nolint:staticcheck
