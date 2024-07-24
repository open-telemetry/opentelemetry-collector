// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package globalgates // import "go.opentelemetry.io/collector/internal/globalgates"

import "go.opentelemetry.io/collector/featuregate"

var UseUnifiedEnvVarExpansionRules = featuregate.GlobalRegistry().MustRegister("confmap.unifyEnvVarExpansion",
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.103.0"),
	featuregate.WithRegisterDescription("`${FOO}` will now be expanded as if it was `${env:FOO}` and no longer expands $ENV syntax. See https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/env-vars.md for more details. When this feature gate is stable, expandconverter will be removed."))

const StrictlyTypedInputID = "confmap.strictlyTypedInput"

var StrictlyTypedInputGate = featuregate.GlobalRegistry().MustRegister(StrictlyTypedInputID,
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.103.0"),
	featuregate.WithRegisterDescription("Makes type casting rules during configuration unmarshaling stricter. See https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/env-vars.md for more details."),
)

var DisableOpenCensusBridge = featuregate.GlobalRegistry().MustRegister("service.disableOpenCensusBridge",
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.105.0"),
	featuregate.WithRegisterDescription("`Disables the OpenCensus bridge meaning any component still using the OpenCensus SDK will no longer be able to produce telemetry."))
