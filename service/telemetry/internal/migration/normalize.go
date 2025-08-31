// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/internal/migration"

import "strings"

func normalizeEndpoint(endpoint string, insecure *bool) *string {
	if !strings.HasPrefix(endpoint, "https://") && !strings.HasPrefix(endpoint, "http://") {
		if insecure != nil && *insecure {
			endpoint = "http://" + endpoint
		} else {
			endpoint = "https://" + endpoint
		}
	}
	return &endpoint
}
