// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package authtest // import "go.opentelemetry.io/collector/extension/auth/authtest"

import (
	"go.opentelemetry.io/collector/extension/extensionauth/extensionauthtest"
)

// MockClient provides a mock implementation of GRPCClient and HTTPClient interfaces
// Deprecated: [v0.121.0] Use extensionauthtest.MockClient instead.
type MockClient = extensionauthtest.MockClient
