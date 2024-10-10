// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"
import "go.opentelemetry.io/collector/exporter/internal"

// Request represents a single request that can be sent to an external endpoint.
//
// Deprecated: [v0.112.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so that we don't remove it.
type Request = internal.Request

// RequestErrorHandler is an optional interface that can be implemented by Request to provide a way handle partial
// temporary failures. For example, if some items failed to process and can be retried, this interface allows to
// return a new Request that contains the items left to be sent. Otherwise, the original Request should be returned.
// If not implemented, the original Request will be returned assuming the error is applied to the whole Request.
//
// Deprecated: [v0.112.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so that we don't remove it.
type RequestErrorHandler = internal.RequestErrorHandler
