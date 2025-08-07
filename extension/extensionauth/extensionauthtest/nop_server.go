// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensionauthtest // import "go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
)

var (
	_ extension.Extension                 = (*nopServer)(nil)
	_ extensioncapabilities.Authenticator = (*nopServer)(nil)
)

type nopServer struct {
	component.StartFunc
	component.ShutdownFunc
}

// Authenticate implements extensioncapabilities.Authenticator.
func (n *nopServer) Authenticate(ctx context.Context, _ map[string][]string) (context.Context, error) {
	return ctx, nil
}

// NewNopServer returns a new extension.Extension that implements the extensioncapabilities.Authenticator.
func NewNopServer() extension.Extension {
	return &nopServer{}
}
