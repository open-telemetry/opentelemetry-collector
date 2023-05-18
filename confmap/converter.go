// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"context"
)

// Converter is a converter interface for the confmap.Conf that allows distributions
// (in the future components as well) to build backwards compatible config converters.
type Converter interface {
	// Convert applies the conversion logic to the given "conf".
	Convert(ctx context.Context, conf *Conf) error
}
