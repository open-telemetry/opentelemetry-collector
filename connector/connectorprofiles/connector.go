// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/connector/xconnector instead.
package connectorprofiles // import "go.opentelemetry.io/collector/connector/connectorprofiles"

import "go.opentelemetry.io/collector/connector/xconnector"

// Deprecated: [0.116.0] Use xconnector.Factory instead.
type Factory = xconnector.Factory

// A Profiles connector acts as an exporter from a profiles pipeline and a receiver
// to one or more traces, metrics, logs, or profiles pipelines.
// Profiles feeds a consumer.Traces, consumer.Metrics, consumer.Logs, or xconsumer.Profiles with data.
//
// Examples:
//   - Profiles could be collected in one pipeline and routed to another profiles pipeline
//     based on criteria such as attributes or other content of the profile. The second
//     pipeline can then process and export the profile to the appropriate backend.
//   - Profiles could be summarized by a metrics connector that emits statistics describing
//     the number of profiles observed.
//   - Profiles could be analyzed by a logs connector that emits events when particular
//     criteria are met.
//
// Deprecated: [0.116.0] Use xconnector.Profiles instead.
type Profiles = xconnector.Profiles

// CreateTracesToProfilesFunc is the equivalent of Factory.CreateTracesToProfiles().
// Deprecated: [0.116.0] Use xconnector.CreateTracesToProfilesFunc instead.
type CreateTracesToProfilesFunc = xconnector.CreateTracesToProfilesFunc

// CreateMetricsToProfilesFunc is the equivalent of Factory.CreateMetricsToProfiles().
// Deprecated: [0.116.0] Use xconnector.CreateMetricsToProfilesFunc instead.
type CreateMetricsToProfilesFunc = xconnector.CreateMetricsToProfilesFunc

// CreateLogsToProfilesFunc is the equivalent of Factory.CreateLogsToProfiles().
// Deprecated: [0.116.0] Use xconnector.CreateLogsToProfilesFunc instead.
type CreateLogsToProfilesFunc = xconnector.CreateLogsToProfilesFunc

// CreateProfilesToProfilesFunc is the equivalent of Factory.CreateProfilesToProfiles().
// Deprecated: [0.116.0] Use xconnector.CreateProfilesToProfilesFunc instead.
type CreateProfilesToProfilesFunc = xconnector.CreateProfilesToProfilesFunc

// CreateProfilesToTracesFunc is the equivalent of Factory.CreateProfilesToTraces().
// Deprecated: [0.116.0] Use xconnector.CreateProfilesToTracesFunc instead.
type CreateProfilesToTracesFunc = xconnector.CreateProfilesToTracesFunc

// CreateProfilesToMetricsFunc is the equivalent of Factory.CreateProfilesToMetrics().
// Deprecated: [0.116.0] Use xconnector.CreateProfilesToMetricsFunc instead.
type CreateProfilesToMetricsFunc = xconnector.CreateProfilesToMetricsFunc

// CreateProfilesToLogsFunc is the equivalent of Factory.CreateProfilesToLogs().
// Deprecated: [0.116.0] Use xconnector.CreateProfilesToLogsFunc instead.
type CreateProfilesToLogsFunc = xconnector.CreateProfilesToLogsFunc

// FactoryOption apply changes to ReceiverOptions.
// Deprecated: [0.116.0] Use xconnector.FactoryOption instead.
type FactoryOption = xconnector.FactoryOption

// WithTracesToTraces overrides the default "error not supported" implementation for WithTracesToTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithTracesToTraces instead.
var WithTracesToTraces = xconnector.WithTracesToTraces

// WithTracesToMetrics overrides the default "error not supported" implementation for WithTracesToMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithTracesToMetrics instead.
var WithTracesToMetrics = xconnector.WithTracesToMetrics

// WithTracesToLogs overrides the default "error not supported" implementation for WithTracesToLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithTracesToLogs instead.
var WithTracesToLogs = xconnector.WithTracesToLogs

// WithMetricsToTraces overrides the default "error not supported" implementation for WithMetricsToTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithMetricsToTraces instead.
var WithMetricsToTraces = xconnector.WithMetricsToTraces

// WithMetricsToMetrics overrides the default "error not supported" implementation for WithMetricsToMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithMetricsToMetrics instead.
var WithMetricsToMetrics = xconnector.WithMetricsToMetrics

// WithMetricsToLogs overrides the default "error not supported" implementation for WithMetricsToLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithMetricsToLogs instead.
var WithMetricsToLogs = xconnector.WithMetricsToLogs

// WithLogsToTraces overrides the default "error not supported" implementation for WithLogsToTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithLogsToTraces instead.
var WithLogsToTraces = xconnector.WithLogsToTraces

// WithLogsToMetrics overrides the default "error not supported" implementation for WithLogsToMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithLogsToMetrics instead.
var WithLogsToMetrics = xconnector.WithLogsToMetrics

// WithLogsToLogs overrides the default "error not supported" implementation for WithLogsToLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithLogsToLogs instead.
var WithLogsToLogs = xconnector.WithLogsToLogs

// WithTracesToProfiles overrides the default "error not supported" implementation for WithTracesToProfiles and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithTracesToProfiles instead.
var WithTracesToProfiles = xconnector.WithTracesToProfiles

// WithMetricsToProfiles overrides the default "error not supported" implementation for WithMetricsToProfiles and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithMetricsToProfiles instead.
var WithMetricsToProfiles = xconnector.WithMetricsToProfiles

// WithLogsToProfiles overrides the default "error not supported" implementation for WithLogsToProfiles and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithLogsToProfiles instead.
var WithLogsToProfiles = xconnector.WithLogsToProfiles

// WithProfilesToProfiles overrides the default "error not supported" implementation for WithProfilesToProfiles and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithProfilesToProfiles instead.
var WithProfilesToProfiles = xconnector.WithProfilesToProfiles

// WithProfilesToTraces overrides the default "error not supported" implementation for WithProfilesToTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithProfilesToTraces instead.
var WithProfilesToTraces = xconnector.WithProfilesToTraces

// WithProfilesToMetrics overrides the default "error not supported" implementation for WithProfilesToMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithProfilesToMetrics instead.
var WithProfilesToMetrics = xconnector.WithProfilesToMetrics

// WithProfilesToLogs overrides the default "error not supported" implementation for WithProfilesToLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xconnector.WithProfilesToLogs instead.
var WithProfilesToLogs = xconnector.WithProfilesToLogs

// NewFactory returns a Factory.
// Deprecated: [0.116.0] Use xconnector.NewFactory instead.
var NewFactory = xconnector.NewFactory
