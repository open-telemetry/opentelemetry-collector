// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/receiverhelper/internal"

import "errors"

// ErrDownstreamError indicates that the error occurred in the downstream consumer
var ErrDownstreamError = errors.New("downstream error")

// IsDownstreamError returns true if the error is a downstream error
func IsDownstreamError(err error) bool {
	return errors.Is(err, ErrDownstreamError)
}

// WrapDownstreamError wraps an error to indicate it's a downstream error
func WrapDownstreamError(err error) error {
	return errors.Join(ErrDownstreamError, err)
}
