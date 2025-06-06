// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package partialsuccess // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/partialsuccess"

import "fmt"

var _ error = &PartialSuccessError{}

type PartialSuccessError struct {
	FailureCount int
	Reason       string
}

func (e *PartialSuccessError) Error() string {
	return fmt.Sprintf("partial success: %d failed: %s", e.FailureCount, e.Reason)
}
