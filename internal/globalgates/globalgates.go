// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package globalgates // import "go.opentelemetry.io/collector/internal/globalgates"

import "go.opentelemetry.io/collector/featuregate"

var DisableOpenCensusBridge = featuregate.GlobalRegistry().MustRegister("service.disableOpenCensusBridge",
	featuregate.StageStable,
	featuregate.WithRegisterFromVersion("v0.105.0"),
	featuregate.WithRegisterToVersion("v0.109.0"),
	featuregate.WithRegisterDescription("`Disables the OpenCensus bridge meaning any component still using the OpenCensus SDK will no longer be able to produce telemetry."))

var NoopTracerProvider = featuregate.GlobalRegistry().MustRegister("service.noopTracerProvider",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.107.0"),
	featuregate.WithRegisterToVersion("v0.109.0"),
	featuregate.WithRegisterDescription("Sets a Noop OpenTelemetry TracerProvider to reduce memory allocations. This featuregate is incompatible with the zPages extension."))
