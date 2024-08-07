// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/connector/internal"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
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
type Profiles interface {
	component.Component
	consumerprofiles.Profiles
}
