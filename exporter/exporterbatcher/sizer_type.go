// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher // import "go.opentelemetry.io/collector/exporter/exporterbatcher"

import (
	"fmt"
)

type SizerType string

const (
	SizerTypeItems SizerType = "items"
	bytesSizerString = "bytes"
)

var (
	ItemsSizer = SizerType{itemsSizerString}
	BytesSizer = SizerType{bytesSizerString}
)

// String converts SizerType to a string
func (s *SizerType) String() string {
	return s.string
}

// NewSizer creates a new sizer. It returns an error if the input is invalid.
func NewSizer(text string) (SizerType, error) {
	switch text {
	case itemsSizerString:
		return ItemsSizer, nil
	case bytesSizerString:
		return BytesSizer, nil
	default:
		return SizerType{}, fmt.Errorf("invalid sizer: %q", text)
	}
}

// MustNewType creates a new sizer. It panics if the input is invalid.
func MustNewSizer(text string) SizerType {
	sizer, err := NewSizer(text)
	if err != nil {
		panic(err)
	}
	return sizer
}

// UnmarshalText implements TextUnmarshaler interface.
func (s *SizerType) UnmarshalText(text []byte) error {
	switch str := string(text); str {
	case itemsSizerString:
		s.string = itemsSizerString
	case bytesSizerString:
		s.string = bytesSizerString
	default:
		return fmt.Errorf("invalid sizer: %q", str)
	}
	return nil
}
