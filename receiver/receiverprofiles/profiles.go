// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use xreceiver module instead.
package receiverprofiles // import "go.opentelemetry.io/collector/receiver/receiverprofiles"

import "go.opentelemetry.io/collector/receiver/xreceiver"

// Profiles receiver receives profiles.
// Its purpose is to translate data from any format to the collector's internal profile format.
// Profiles receiver feeds a xconsumer.Profiles with data.
//
// For example, it could be a pprof data source which translates pprof profiles into pprofile.Profiles.
// Deprecated: [0.116.0] Use the xreceiver.Profiles instead.
type Profiles = xreceiver.Profiles

// Factory is a factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
// Deprecated: [0.116.0] Use the xreceiver.Factory instead.
type Factory = xreceiver.Factory

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles.
// Deprecated: [0.116.0] Use the xreceiver.CreateProfilesFunc instead.
type CreateProfilesFunc = xreceiver.CreateProfilesFunc

// FactoryOption apply changes to Factory.
// Deprecated: [0.116.0] Use the xreceiver.FactoryOption instead.
type FactoryOption = xreceiver.FactoryOption

// WithTraces overrides the default "error not supported" implementation for Factory.CreateTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use the xreceiver.WithTraces instead.
var WithTraces = xreceiver.WithTraces

// WithMetrics overrides the default "error not supported" implementation for Factory.CreateMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use the xreceiver.WithMetrics instead.
var WithMetrics = xreceiver.WithMetrics

// WithLogs overrides the default "error not supported" implementation for Factory.CreateLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use the xreceiver.WithLogs instead.
var WithLogs = xreceiver.WithLogs

// WithProfiles overrides the default "error not supported" implementation for Factory.CreateProfiles and the default "undefined" stability level.
// Deprecated: [0.116.0] Use the xreceiver.WithProfiles instead.
var WithProfiles = xreceiver.WithProfiles

// NewFactory returns a Factory.
// Deprecated: [0.116.0] Use the xreceiver.NewFactory instead.
var NewFactory = xreceiver.NewFactory
