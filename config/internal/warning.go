// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internal // import "go.opentelemetry.io/collector/config/internal"

import (
	"fmt"
	"net"

	"go.uber.org/zap"
)

// WarnOnUnspecifiedHost emits a warning if an endpoint has an unspecified host.
func WarnOnUnspecifiedHost(logger *zap.Logger, endpoint string) error {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse endpoint: %w", err)
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		logger.Warn(
			"Using the 0.0.0.0 address exposes this server to every network interface, which may facilitate Denial of Service attacks",
			zap.String(
				"documentation",
				"https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/security-best-practices.md#safeguards-against-denial-of-service-attacks",
			),
		)
	}

	return nil
}
