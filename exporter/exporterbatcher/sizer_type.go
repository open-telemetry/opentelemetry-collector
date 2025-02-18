// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher // import "go.opentelemetry.io/collector/exporter/exporterbatcher"

import (
	"fmt"
)

type SizerType string

const SizerTypeItems SizerType = "items"
const SizerTypeBytes SizerType = "bytes"

// UnmarshalText implements TextUnmarshaler interface.
func (s *SizerType) UnmarshalText(text []byte) error {
	switch str := string(text); str {
	case string(SizerTypeItems):
		*s = SizerTypeItems
	case string(SizerTypeBytes):
		*s = SizerTypeBytes
	default:
		return fmt.Errorf("invalid sizer: %q", str)
	}
	return nil
}
