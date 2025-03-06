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
	sizerTypeBytes = "bytes"
)

var (
	SizerTypeItems = SizerType{val: sizerTypeItems}
	SizerTypeBytes = SizerType{val: sizerTypeBytes}
)

// UnmarshalText implements TextUnmarshaler interface.
func (s *SizerType) UnmarshalText(text []byte) error {
	switch str := string(text); str {
	case sizerTypeItems:
		*s = SizerTypeItems
	// TODO support setting sizer to SizerTypeBytes when all logs, traces, and metrics batching support it
	default:
		return fmt.Errorf("invalid sizer: %q", str)
	}
	return nil
}
