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
	// Enabled defines whether the event is enabled by default.
	Enabled bool `mapstructure:"enabled"`

	// Warnings that will be shown to user under specified conditions.
	Warnings Warnings `mapstructure:"warnings"`

	// Description of the event.
	Description string `mapstructure:"description"`

	// The stability level of the event.
	Stability Stability `mapstructure:"stability"`

	// Extended documentation of the event. If specified, this will be appended to the description used in generated documentation.
	ExtendedDocumentation string `mapstructure:"extended_documentation"`

	// Attributes is the list of attributes that the event emits.
	Attributes []AttributeName `mapstructure:"attributes"`
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
