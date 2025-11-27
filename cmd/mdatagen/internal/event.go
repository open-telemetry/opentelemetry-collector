// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

import (
	"errors"

	"go.opentelemetry.io/collector/confmap"
)

type (
	EventName string
)

func (ln EventName) Render() (string, error) {
	return FormatIdentifier(string(ln), true)
}

func (ln EventName) RenderUnexported() (string, error) {
	return FormatIdentifier(string(ln), false)
}

type Event struct {
	Signal `mapstructure:",squash"`
}

func (l *Event) validate() error {
	var errs error
	if l.Description == "" {
		errs = errors.Join(errs, errors.New(`missing event description`))
	}
	return errs
}

func (l *Event) Unmarshal(parser *confmap.Conf) error {
	if !parser.IsSet("enabled") {
		return errors.New("missing required field: `enabled`")
	}
	return parser.Unmarshal(l)
}
