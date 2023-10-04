// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/featuregate"
)

const useLocalHostAsDefaultHostID = "component.UseLocalHostAsDefaultHost"

// UseLocalHostAsDefaultHostfeatureGate is the feature gate that controls whether
// server-like receivers and extensions such as the OTLP receiver use localhost as the default host for their endpoints.
var UseLocalHostAsDefaultHostfeatureGate = featuregate.GlobalRegistry().MustRegister(
	useLocalHostAsDefaultHostID,
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("controls whether server-like receivers and extensions such as the OTLP receiver use localhost as the default host for their endpoints"))

// EndpointForPort gets the endpoint for a given port using localhost or 0.0.0.0 depending on the feature gate.
func EndpointForPort(port int) string {
	host := "0.0.0.0"
	if UseLocalHostAsDefaultHostfeatureGate.IsEnabled() {
		host = "localhost"
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// LogAboutUseLocalHostAsDefault logs about the upcoming change from 0.0.0.0 to localhost on server-like components.
func LogAboutUseLocalHostAsDefault(logger *zap.Logger) {
	if !UseLocalHostAsDefaultHostfeatureGate.IsEnabled() {
		logger.Info(
			"The default endpoint(s) for this component will change in a future version to use localhost instead of 0.0.0.0. Use the feature gate to preview the new default.",
			zap.String("feature gate ID", useLocalHostAsDefaultHostID),
		)
	}
}
