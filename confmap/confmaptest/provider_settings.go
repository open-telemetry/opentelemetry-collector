// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmaptest // import "go.opentelemetry.io/collector/confmap/confmaptest"

import (
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/confmap"
)

func NewNopProviderSettings() confmap.ProviderSettings {
	return confmap.ProviderSettings{Logger: zap.NewNop()}
}
