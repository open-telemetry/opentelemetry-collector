// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Deprecated: [0.116.0] Use go.opentelemetry.io/collector/processor/xprocessor instead.
package processorprofiles // import "go.opentelemetry.io/collector/processor/processorprofiles"

import "go.opentelemetry.io/collector/processor/xprocessor"

// Factory is a component.Factory interface for processors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewFactory to implement it.
// Deprecated: [0.116.0] Use xprocessor.Factory instead.
type Factory = xprocessor.Factory

// Profiles is a processor that can consume profiles.
// Deprecated: [0.116.0] Use xprocessor.Profiles instead.
type Profiles = xprocessor.Profiles

// CreateProfilesFunc is the equivalent of Factory.CreateProfiles().
// Deprecated: [0.116.0] Use xprocessor.CreateProfilesFunc instead.
type CreateProfilesFunc = xprocessor.CreateProfilesFunc

// FactoryOption apply changes to ReceiverOptions.
// Deprecated: [0.116.0] Use xprocessor.FactoryOption instead.
type FactoryOption = xprocessor.FactoryOption

// WithTraces overrides the default "error not supported" implementation for CreateTraces and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xprocessor.WithTraces instead.
var WithTraces = xprocessor.WithTraces

// WithMetrics overrides the default "error not supported" implementation for CreateMetrics and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xprocessor.WithMetrics instead.
var WithMetrics = xprocessor.WithMetrics

// WithLogs overrides the default "error not supported" implementation for CreateLogs and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xprocessor.WithLogs instead.
var WithLogs = xprocessor.WithLogs

// WithProfiles overrides the default "error not supported" implementation for CreateProfiles and the default "undefined" stability level.
// Deprecated: [0.116.0] Use xprocessor.WithProfiles instead.
var WithProfiles = xprocessor.WithProfiles

// NewFactory returns a Factory.
// Deprecated: [0.116.0] Use xprocessor.NewFactory instead.
var NewFactory = xprocessor.NewFactory
