// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadata"

// assign the feature gate to separate functions to make it possible to override the behavior in tests
// on write and read paths separately.
var (
	PersistRequestContextOnRead  = metadata.ExporterPersistRequestContextFeatureGate.IsEnabled
	PersistRequestContextOnWrite = metadata.ExporterPersistRequestContextFeatureGate.IsEnabled
)
