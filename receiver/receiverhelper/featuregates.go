// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import "go.opentelemetry.io/collector/receiver/receiverhelper/internal/metadata"

// NewReceiverMetricsGate is the feature gate that controls whether to distinguish downstream errors from internal errors in pipeline telemetry.
// This feature gate is used in OTLP receiver tests, and therefore needs to be public.
var NewReceiverMetricsGate = metadata.ReceiverhelperNewReceiverMetricsFeatureGate
