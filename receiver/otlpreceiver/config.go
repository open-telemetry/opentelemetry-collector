// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpreceiver // import "go.opentelemetry.io/collector/receiver/otlpreceiver"

import (
	"encoding"
	"errors"
	"fmt"
	"net/url"
	"path"

	"go.opentelemetry.io/collector/component"
)

var _ encoding.TextUnmarshaler = (*SanitizedURLPath)(nil)

func (s *SanitizedURLPath) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err != nil {
		return fmt.Errorf("invalid HTTP URL path set for signal: %w", err)
	}

	if !path.IsAbs(u.Path) {
		u.Path = "/" + u.Path
	}

	*s = SanitizedURLPath(u.Path)
	return nil
}

var _ component.Config = (*Config)(nil)

// Validate checks the receiver configuration is valid
func validateProtocols(p *Protocols) error {
	if !p.GRPC.HasValue() && !p.HTTP.HasValue() {
		return errors.New("must specify at least one protocol when using the OTLP receiver")
	}
	return nil
}
