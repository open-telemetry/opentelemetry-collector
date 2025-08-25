// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statustest // import "go.opentelemetry.io/collector/service/internal/status/statustest"

import (
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/service/internal/status"
)

// NewNopStatusReporter creates a status reporter that discards any events.
// TODO move to service/status
func NewNopStatusReporter() status.Reporter {
	return &nopStatusReporter{}
}

type nopStatusReporter struct{}

func (r *nopStatusReporter) Ready() {}

func (r *nopStatusReporter) ReportStatus(*status.InstanceID, *componentstatus.Event) {}

func (r *nopStatusReporter) ReportOKIfStarting(*status.InstanceID) {}
