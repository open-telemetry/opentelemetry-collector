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

package config // import "go.opentelemetry.io/collector/config"

import (
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

// Service defines the configurable components of the service.
// Deprecated: [v0.52.0] Use service.ConfigService
type Service struct {
	// Telemetry is the configuration for collector's own telemetry.
	Telemetry ServiceTelemetry `mapstructure:"telemetry"`

	// Extensions are the ordered list of extensions configured for the service.
	Extensions []ComponentID `mapstructure:"extensions"`

	// Pipelines are the set of data pipelines configured for the service.
	Pipelines map[ComponentID]*Pipeline `mapstructure:"pipelines"`
}

// ServiceTelemetry defines the configurable settings for service telemetry.
// Deprecated: [v0.52.0] Use service.ConfigServiceTelemetry
type ServiceTelemetry struct {
	Logs    ServiceTelemetryLogs    `mapstructure:"logs"`
	Metrics ServiceTelemetryMetrics `mapstructure:"metrics"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// For metrics the attributes become Prometheus labels.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `mapstructure:"resource"`
}

// ServiceTelemetryLogs defines the configurable settings for service telemetry logs.
// This MUST be compatible with zap.Config. Cannot use directly zap.Config because
// the collector uses mapstructure and not yaml tags.
// Deprecated: [v0.52.0] Use service.ConfigServiceTelemetryLogs
type ServiceTelemetryLogs struct {
	// Level is the minimum enabled logging level.
	// (default = "INFO")
	Level zapcore.Level `mapstructure:"level"`

	// Development puts the logger in development mode, which changes the
	// behavior of DPanicLevel and takes stacktraces more liberally.
	// (default = false)
	Development bool `mapstructure:"development"`

	// Encoding sets the logger's encoding.
	// Example values are "json", "console".
	Encoding string `mapstructure:"encoding"`

	// DisableCaller stops annotating logs with the calling function's file
	// name and line number. By default, all logs are annotated.
	// (default = false)
	DisableCaller bool `mapstructure:"disable_caller"`

	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	// (default = false)
	DisableStacktrace bool `mapstructure:"disable_stacktrace"`

	// OutputPaths is a list of URLs or file paths to write logging output to.
	// The URLs could only be with "file" schema or without schema.
	// The URLs with "file" schema must be an absolute path.
	// The URLs without schema are treated as local file paths.
	// "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
	// see details at Open in zap/writer.go.
	// (default = ["stderr"])
	OutputPaths []string `mapstructure:"output_paths"`

	// ErrorOutputPaths is a list of URLs or file paths to write zap internal logger errors to.
	// The URLs could only be with "file" schema or without schema.
	// The URLs with "file" schema must use absolute paths.
	// The URLs without schema are treated as local file paths.
	// "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
	// see details at Open in zap/writer.go.
	//
	// Note that this setting only affects the zap internal logger errors.
	// (default = ["stderr"])
	ErrorOutputPaths []string `mapstructure:"error_output_paths"`

	// InitialFields is a collection of fields to add to the root logger.
	// Example:
	//
	// 		initial_fields:
	//	   		foo: "bar"
	//
	// By default, there is no initial field.
	InitialFields map[string]interface{} `mapstructure:"initial_fields"`
}

// ServiceTelemetryMetrics exposes the common Telemetry configuration for one component.
// Experimental: *NOTE* this structure is subject to change or removal in the future.
// Deprecated: [v0.52.0] Use service.ConfigServiceTelemetryMetrics
type ServiceTelemetryMetrics struct {
	// Level is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	Level configtelemetry.Level `mapstructure:"level"`

	// Address is the [address]:port that metrics exposition should be bound to.
	Address string `mapstructure:"address"`
}

// Pipeline defines a single pipeline.
// Deprecated: [v0.52.0] Use service.ConfigServicePipeline
type Pipeline struct {
	Receivers  []ComponentID `mapstructure:"receivers"`
	Processors []ComponentID `mapstructure:"processors"`
	Exporters  []ComponentID `mapstructure:"exporters"`
}

// Deprecated: [v0.52.0] will be removed soon.
type Pipelines = map[ComponentID]*Pipeline
