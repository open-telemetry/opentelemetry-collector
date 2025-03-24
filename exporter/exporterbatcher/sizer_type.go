// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher // import "go.opentelemetry.io/collector/exporter/exporterbatcher"

import (
	"encoding"
	"fmt"
)

var (
	_ encoding.TextMarshaler   = (*SizerType)(nil)
	_ encoding.TextUnmarshaler = (*SizerType)(nil)
)

type SizerType struct {
	val string
}

const (
	sizerTypeBytes    = "bytes"
	sizerTypeItems    = "items"
	sizerTypeRequests = "requests"
)

var (
	SizerTypeBytes    = SizerType{val: sizerTypeBytes}
	SizerTypeItems    = SizerType{val: sizerTypeItems}
	SizerTypeRequests = SizerType{val: sizerTypeRequests}
)

// UnmarshalText implements TextUnmarshaler interface.
func (s *SizerType) UnmarshalText(text []byte) error {
	switch str := string(text); str {
	case sizerTypeItems:
		*s = SizerTypeItems
	case sizerTypeBytes:
		*s = SizerTypeBytes
	case sizerTypeRequests:
		*s = SizerTypeRequests
	default:
		return fmt.Errorf("invalid sizer: %q", str)
	}
	return nil
}

func (s *SizerType) MarshalText() ([]byte, error) {
	return []byte(s.val), nil
}
