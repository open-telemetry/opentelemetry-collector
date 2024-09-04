// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"
import "go.opentelemetry.io/collector/exporter/internal"

// Request represents a single request that can be sent to an external endpoint.
// Experimental: All the following APIs are at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Request = internal.Request
type RequestErrorHandler = internal.RequestErrorHandler

func extractPartialRequest(req Request, err error) Request {
	return internal.ExtractPartialRequest(req, err)
}
