// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsreport // import "go.opentelemetry.io/collector/obsreport"

import "go.opentelemetry.io/collector/receiver/receiverhelper"

// Receiver is a helper to add observability to a receiver.
//
// Deprecated: [0.86.0] Use receiverhelper.ObsReport instead.
type Receiver = receiverhelper.ObsReport

// ReceiverSettings are settings for creating an Receiver.
//
// Deprecated: [0.86.0] Use receiverhelper.ObsReportSettings instead.
type ReceiverSettings = receiverhelper.ObsReportSettings

// NewReceiver creates a new Receiver.
//
// Deprecated: [0.86.0] Use receiverhelper.NewObsReport instead.
func NewReceiver(cfg ReceiverSettings) (*Receiver, error) {
	return receiverhelper.NewObsReport(cfg)
}
