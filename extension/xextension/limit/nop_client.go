// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limit // import "go.opentelemetry.io/collector/extension/xextension/limit"

import "context"

type nopClient struct{}

var nopClientInstance Client = &nopClient{}

// NewNopClient returns a nop client
func NewNopClient() Client {
	return nopClientInstance
}

// Acquire implements Client.
func (nopClient) Acquire(_ context.Context, _ uint64) (ReleaseFunc, error) {
	return func() {}, nil
}
