// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package connectorprofiles // import "go.opentelemetry.io/collector/connector/connectorprofiles"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/internal"
)

// A Profiles connector acts as an exporter from a profiles pipeline and a receiver
// to one or more traces, metrics, logs, or profiles pipelines.
// Profiles feeds a consumer.Traces, consumer.Metrics, consumer.Logs, or consumerprofiles.Profiles with data.
//
// Examples:
//   - Profiles could be collected in one pipeline and routed to another profiles pipeline
//     based on criteria such as attributes or other content of the profile. The second
//     pipeline can then process and export the profile to the appropriate backend.
//   - Profiles could be summarized by a metrics connector that emits statistics describing
//     the number of profiles observed.
//   - Profiles could be analyzed by a logs connector that emits events when particular
//     criteria are met.
type Profiles = internal.Profiles

// CreateTracesToProfilesFunc is the equivalent of Factory.CreateTracesToProfiles().
type CreateTracesToProfilesFunc = internal.CreateTracesToProfilesFunc

// CreateMetricsToProfilesFunc is the equivalent of Factory.CreateMetricsToProfiles().
type CreateMetricsToProfilesFunc = internal.CreateMetricsToProfilesFunc

// CreateLogsToProfilesFunc is the equivalent of Factory.CreateLogsToProfiles().
type CreateLogsToProfilesFunc = internal.CreateLogsToProfilesFunc

// CreateProfilesToProfilesFunc is the equivalent of Factory.CreateProfilesToProfiles().
type CreateProfilesToProfilesFunc = internal.CreateProfilesToProfilesFunc

// CreateProfilesToTracesFunc is the equivalent of Factory.CreateProfilesToTraces().
type CreateProfilesToTracesFunc = internal.CreateProfilesToTracesFunc

// CreateProfilesToMetricsFunc is the equivalent of Factory.CreateProfilesToMetrics().
type CreateProfilesToMetricsFunc = internal.CreateProfilesToMetricsFunc

// CreateProfilesToLogsFunc is the equivalent of Factory.CreateProfilesToLogs().
type CreateProfilesToLogsFunc = internal.CreateProfilesToLogsFunc

// WithTracesToProfiles overrides the default "error not supported" implementation for WithTracesToProfiles and the default "undefined" stability level.
func WithTracesToProfiles(createTracesToProfiles CreateTracesToProfilesFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithTracesToProfiles(createTracesToProfiles, sl)
}

// WithMetricsToProfiles overrides the default "error not supported" implementation for WithMetricsToProfiles and the default "undefined" stability level.
func WithMetricsToProfiles(createMetricsToProfiles CreateMetricsToProfilesFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithMetricsToProfiles(createMetricsToProfiles, sl)
}

// WithLogsToProfiles overrides the default "error not supported" implementation for WithLogsToProfiles and the default "undefined" stability level.
func WithLogsToProfiles(createLogsToProfiles CreateLogsToProfilesFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithLogsToProfiles(createLogsToProfiles, sl)
}

// WithProfilesToProfiles overrides the default "error not supported" implementation for WithProfilesToProfiles and the default "undefined" stability level.
func WithProfilesToProfiles(createProfilesToProfiles CreateProfilesToProfilesFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithProfilesToProfiles(createProfilesToProfiles, sl)
}

// WithProfilesToTraces overrides the default "error not supported" implementation for WithProfilesToTraces and the default "undefined" stability level.
func WithProfilesToTraces(createProfilesToTraces CreateProfilesToTracesFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithProfilesToTraces(createProfilesToTraces, sl)
}

// WithProfilesToMetrics overrides the default "error not supported" implementation for WithProfilesToMetrics and the default "undefined" stability level.
func WithProfilesToMetrics(createProfilesToMetrics CreateProfilesToMetricsFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithProfilesToMetrics(createProfilesToMetrics, sl)
}

// WithProfilesToLogs overrides the default "error not supported" implementation for WithProfilesToLogs and the default "undefined" stability level.
func WithProfilesToLogs(createProfilesToLogs CreateProfilesToLogsFunc, sl component.StabilityLevel) connector.FactoryOption {
	return internal.WithProfilesToLogs(createProfilesToLogs, sl)
}
