// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/receiver/internal"

import "go.opentelemetry.io/collector/component"

// Logs receiver receives logs.
// Its purpose is to translate data from any format to the collector's internal logs data format.
// LogsReceiver feeds a consumer.Logs with data.
//
// For example, it could be a receiver that reads syslogs and convert them into plog.Logs.
type Logs interface {
	component.Component
}
