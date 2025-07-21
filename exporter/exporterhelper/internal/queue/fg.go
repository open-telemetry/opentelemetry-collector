// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import "go.opentelemetry.io/collector/featuregate"

// PersistRequestContextFeatureGate controls whether request context should be preserved in the persistent queue.
var PersistRequestContextFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"exporter.PersistRequestContext",
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.128.0"),
	featuregate.WithRegisterDescription("controls whether context should be stored alongside requests in the persistent queue"),
)

// assign the feature gate to separate functions to make it possible to override the behavior in tests
// on write and read paths separately.
var (
	PersistRequestContextOnRead  = PersistRequestContextFeatureGate.IsEnabled
	PersistRequestContextOnWrite = PersistRequestContextFeatureGate.IsEnabled
)
