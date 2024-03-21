// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreportconfig // import "go.opentelemetry.io/collector/internal/obsreportconfig"

import (
	"go.opentelemetry.io/collector/featuregate"
)

// DisableHighCardinalityMetricsfeatureGate is the feature gate that controls whether the collector should enable
// potentially high cardinality metrics. The gate will be removed when the collector allows for view configuration.
var DisableHighCardinalityMetricsfeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableHighCardinalityMetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector should enable potentially high"+
		"cardinality metrics. The gate will be removed when the collector allows for view configuration."))

// UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate is the feature gate that controls whether the collector
// supports configuring the OpenTelemetry SDK via configuration
var UseOtelWithSDKConfigurationForInternalTelemetryFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.useOtelWithSDKConfigurationForInternalTelemetry",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether the collector supports extended OpenTelemetry"+
		"configuration for internal telemetry"))
