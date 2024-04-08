// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmaptest // import "go.opentelemetry.io/collector/confmap/confmaptest"

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/confmap"
)

func NewNopProviderSettings() confmap.ProviderSettings {
	return confmap.ProviderSettings{Logger: zap.NewNop()}
}

func NewLoggingProviderSettings() (confmap.ProviderSettings, *observer.ObservedLogs) {
	core, ol := observer.New(zap.InfoLevel) // todo how do i get an appropriate logging level in here?
	return confmap.ProviderSettings{Logger: zap.New(core)}, ol

}
