// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher // import "go.opentelemetry.io/collector/exporter/exporterbatcher"

import (
	"fmt"
)

type SizerType struct {
	val string
}

const (
	sizerTypeItems = "items"
)

var SizerTypeItems = SizerType{val: sizerTypeItems}

// UnmarshalText implements TextUnmarshaler interface.
func (s *SizerType) UnmarshalText(text []byte) error {
	switch str := string(text); str {
	case sizerTypeItems:
		*s = SizerTypeItems
	default:
		return fmt.Errorf("invalid sizer: %q", str)
	}
	return nil
}
