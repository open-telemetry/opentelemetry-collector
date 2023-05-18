// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package sharedgate exposes a featuregate that is used by multiple packages.
package sharedgate // import "go.opentelemetry.io/collector/otelcol/internal/sharedgate"

import "go.opentelemetry.io/collector/featuregate"

var ConnectorsFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"service.connectors",
	featuregate.StageStable,
	featuregate.WithRegisterFromVersion("v0.71.0"),
	featuregate.WithRegisterDescription("Enables 'connectors', a new type of component for transmitting signals between pipelines."),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-collector/issues/2336"),
	featuregate.WithRegisterToVersion("v0.78.0"))
