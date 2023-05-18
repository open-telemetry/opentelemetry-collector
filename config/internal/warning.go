// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/config/internal"

import (
	"net"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

func shouldWarn(endpoint string) bool {
	if endpoint == ":" {
		// : (aka 0.0.0.0:0)
		return true
	}

	if strings.HasPrefix(endpoint, ":") {
		// :<port> (aka 0.0.0.0:<port>)
		_, err := strconv.ParseInt(endpoint[1:], 10, 64)
		// If it's not a number, it's probably invalid, don't warn.
		return err == nil
	}

	// <host>:<port>
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil { // Probably invalid, don't warn.
		return false
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

// WarnOnUnspecifiedHost emits a warning if an endpoint has an unspecified host.
func WarnOnUnspecifiedHost(logger *zap.Logger, endpoint string) {
	if shouldWarn(endpoint) {
		logger.Warn(
			"Using the 0.0.0.0 address exposes this server to every network interface, which may facilitate Denial of Service attacks",
			zap.String(
				"documentation",
				"https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/security-best-practices.md#safeguards-against-denial-of-service-attacks",
			),
		)
	}
}
