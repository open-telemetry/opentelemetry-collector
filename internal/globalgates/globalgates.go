// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package globalgates // import "go.opentelemetry.io/collector/internal/globalgates"

import "go.opentelemetry.io/collector/featuregate"

var NoopTracerProvider = featuregate.GlobalRegistry().MustRegister("service.noopTracerProvider",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.107.0"),
	featuregate.WithRegisterToVersion("v0.109.0"),
	featuregate.WithRegisterDescription("Sets a Noop OpenTelemetry TracerProvider to reduce memory allocations. This featuregate is incompatible with the zPages extension."))

const MergeComponentsAppendID = "confmap.MergeComponentsAppend"

var MergeComponentsAppend = featuregate.GlobalRegistry().MustRegister(MergeComponentsAppendID,
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.105.0"),
	featuregate.WithRegisterDescription("Overrides default koanf merging strategy and combines slices."))
