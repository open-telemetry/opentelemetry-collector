// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregates // import "go.opentelemetry.io/collector/internal/featuregates"

import "go.opentelemetry.io/collector/featuregate"

var UseUnifiedEnvVarExpansionRules = featuregate.GlobalRegistry().MustRegister("confmap.unifyEnvVarExpansion",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.103.0"),
	featuregate.WithRegisterDescription("`${FOO}` will now be expanded as if it was `${env:FOO}` and no longer expands $ENV syntax. See https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/rfcs/env-vars.md for more details. When this feature gate is stable, expandconverter will be removed."))
