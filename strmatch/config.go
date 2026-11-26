// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package strmatch // import "go.opentelemetry.io/collector/strmatch"

import (
	"errors"
	"regexp"
)

// Config configures the matching behavior of a Matcher.
type Config struct {
	Strict string `mapstructure:"strict"`
	Regex  string `mapstructure:"regexp"`
	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate checks that exactly one of Strict or Regex is set and that Regex compiles
func (c Config) Validate() error {
	if c.Strict == "" && c.Regex == "" {
		return errors.New("must specify either strict or regex")
	}
	if c.Strict != "" && c.Regex != "" {
		return errors.New("strict and regex cannot be used together")
	}

	if c.Regex != "" {
		_, err := regexp.Compile(c.Regex)
		if err != nil {
			return err
		}
	}

	return nil
}

type combinedMatcher struct {
	stricts map[any]struct{}
	regexes []*regexp.Regexp
}

// New creates a Matcher out of a set of Config configuration objects.
func New(configs []Config) Matcher {
	cf := &combinedMatcher{
		stricts: make(map[any]struct{}),
	}
	for _, config := range configs {
		if config.Strict != "" {
			cf.stricts[config.Strict] = struct{}{}
		}

		if config.Regex != "" {
			// Validate() call above ensures that the regex is valid.
			re := regexp.MustCompile(config.Regex)
			cf.regexes = append(cf.regexes, re)
		}
	}
	return cf
}

func (cf *combinedMatcher) Matches(toMatch any) bool {
	_, ok := cf.stricts[toMatch]
	if ok {
		return ok
	}
	if str, ok := toMatch.(string); ok {
		for _, re := range cf.regexes {
			if re.MatchString(str) {
				return true
			}
		}
	}
	return false
}
