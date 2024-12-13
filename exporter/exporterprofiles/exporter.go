// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/exporter/xexporter instead.
package exporterprofiles // import "go.opentelemetry.io/collector/exporter/exporterprofiles"

import "go.opentelemetry.io/collector/exporter/xexporter"

// Profiles is an exporter that can consume profiles.
// Deprecated: [0.116.0] Use xeporter.Profiles instead.
type Profiles = xexporter.Profiles

// Deprecated: [0.116.0] Use xeporter.Factory instead.
type Factory = xexporter.Factory

// FactoryOption apply changes to ReceiverOptions.
// Deprecated: [0.116.0] Use xeporter.FactoryOption instead.
type FactoryOption = xexporter.FactoryOption

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
// Deprecated: [0.116.0] Use xeporter.CreateProfilesFunc instead.
type CreateProfilesFunc = xexporter.CreateProfilesFunc

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xeporter.WithTraces instead.
var WithTraces = xexporter.WithTraces

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xeporter.WithMetrics instead.
var WithMetrics = xexporter.WithMetrics

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xeporter.WithLogs instead.
var WithLogs = xexporter.WithLogs

// WithProfiles overrides the default "error not supported" implementation for CreateProfilesExporter and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xeporter.WithProfiles instead.
var WithProfiles = xexporter.WithProfiles

// NewFactory returns a Factory.
// Deprecated: [0.116.0] Use xeporter.NewFactory instead.
var NewFactory = xexporter.NewFactory
