// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statusutil // import "go.opentelemetry.io/collector/internal/statusutil"

import (
	"net/http"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewStatusFromMsgAndHTTPCode returns a gRPC status based on an error message string and a http status code.
// This function is shared between the http receiver and http exporter for error propagation.
func NewStatusFromMsgAndHTTPCode(errMsg string, statusCode int) *status.Status {
	var c codes.Code
	// Mapping based on https://github.com/grpc/grpc/blob/master/doc/http-grpc-status-mapping.md
	// 429 mapping to ResourceExhausted and 400 mapping to StatusBadRequest are exceptions.
	switch statusCode {
	case http.StatusBadRequest:
		c = codes.InvalidArgument
	case http.StatusUnauthorized:
		c = codes.Unauthenticated
	case http.StatusForbidden:
		c = codes.PermissionDenied
	case http.StatusNotFound:
		c = codes.Unimplemented
	case http.StatusTooManyRequests:
		c = codes.ResourceExhausted
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		c = codes.Unavailable
	default:
		c = codes.Unknown
	}
	return status.New(c, errMsg)
}

// GetRetryInfo returns the RetryInfo detail attached to the status, or nil if there is none.
func GetRetryInfo(status *status.Status) *errdetails.RetryInfo {
	// The details may come from a remote peer. Do not use status.Details(): it decodes
	// every detail, whatever its type and size, only to discard all but RetryInfo.
	for _, detail := range status.Proto().GetDetails() {
		if !detail.MessageIs((*errdetails.RetryInfo)(nil)) {
			continue
		}
		retryInfo := &errdetails.RetryInfo{}
		if err := detail.UnmarshalTo(retryInfo); err == nil {
			return retryInfo
		}
	}
	return nil
}
