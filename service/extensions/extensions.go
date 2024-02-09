// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	internalextensions "go.opentelemetry.io/collector/service/internal/extensions"
)

// Extensions is a map of extensions created from extension configs.
// Deprecated: [v0.95.0] This type is deprecated and will be removed in a future release.
type Extensions = internalextensions.Extensions

// Settings holds configuration for building Extensions.
// Deprecated: [v0.95.0] This type is deprecated and will be removed in a future release.
type Settings = internalextensions.Settings

// Config represents the ordered list of extensions configured for the service.
// Deprecated: [v0.95.0] This type is deprecated and will be removed in a future release.
type Config = internalextensions.Config

// New creates a new Extensions from Config.
// Deprecated: [v0.95.0] This function is deprecated and will be removed in a future release.
var New = internalextensions.New //lint:ignore
