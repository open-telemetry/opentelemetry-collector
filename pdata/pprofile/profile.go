// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// switchDictionary updates the Profile, switching its indices from one
// dictionary to another.
func (ms Profile) switchDictionary(src, dst ProfilesDictionary) error {
	for i, v := range ms.Sample().All() {
		err := v.switchDictionary(src, dst)
		if err != nil {
			return fmt.Errorf("error switching dictionary for sample %d: %w", i, err)
		}
	}

	return nil
}
