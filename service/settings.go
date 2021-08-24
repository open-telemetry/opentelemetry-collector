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

package service

import (
	"go.opentelemetry.io/contrib/zpages"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/service/parserprovider"
)

// svcSettings holds configuration for building a new service.
type svcSettings struct {
	// Factories component factories.
	Factories component.Factories

	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// Config represents the configuration of the service.
	Config *config.Config

	// Logger represents the logger used for all the components.
	Logger *zap.Logger

	// TracerProvider represents the TracerProvider used for all the components.
	TracerProvider trace.TracerProvider

	// ZPagesSpanProcessor represents the SpanProcessor for tracez page.
	ZPagesSpanProcessor *zpages.SpanProcessor

	// AsyncErrorChannel is the channel that is used to report fatal errors.
	AsyncErrorChannel chan error
}

// CollectorSettings holds configuration for creating a new Collector.
type CollectorSettings struct {
	// Factories component factories.
	Factories component.Factories

	// BuildInfo provides collector start information.
	BuildInfo component.BuildInfo

	// DisableGracefulShutdown disables the automatic graceful shutdown
	// of the collector on SIGINT or SIGTERM.
	// Users who want to handle signals themselves can disable this behavior
	// and manually handle the signals to shutdown the collector.
	DisableGracefulShutdown bool

	// ParserProvider provides the configuration's Parser.
	// If it is not provided a default provider is used. The default provider loads the configuration
	// from a config file define by the --config command line flag and overrides component's configuration
	// properties supplied via --set command line flag.
	// If the provider is parserprovider.Watchable, collector
	// may reload the configuration upon error.
	ParserProvider parserprovider.ParserProvider

	// ConfigUnmarshaler unmarshalls the configuration's Parser into the service configuration.
	// If it is not provided a default unmarshaler is used.
	ConfigUnmarshaler configunmarshaler.ConfigUnmarshaler

	// LoggingOptions provides a way to change behavior of zap logging.
	LoggingOptions []zap.Option
}
