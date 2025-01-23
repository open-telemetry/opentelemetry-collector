// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/connector/internal"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pipeline"
)

func ErrDataTypes(id component.ID, from, to pipeline.Signal) error {
	return fmt.Errorf("connector %q cannot connect from %s to %s: %w", id, from, to, pipeline.ErrSignalNotSupported)
}
