// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// Config represents the ordered list of extensions configured for the service.
type Config []component.ID

// Validate checks if the configuration is valid.
func (cfg Config) Validate() error {
	// Validate no extensions are duplicated.
	extSet := make(map[component.ID]struct{}, len(cfg))
	for _, ref := range cfg {
		if _, exists := extSet[ref]; exists {
			return fmt.Errorf("references extension %q multiple times", ref)
		}
		extSet[ref] = struct{}{}
	}
	return nil
}
