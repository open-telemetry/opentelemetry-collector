// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package filter // import "go.opentelemetry.io/collector/filter"

import (
	"errors"
	"regexp"
)

func validateConfig(c *Config) error {
	if c.Strict == "" && c.Regexp == "" {
		return errors.New("must specify either strict or regex")
	}
	if c.Strict != "" && c.Regexp != "" {
		return errors.New("strict and regex cannot be used together")
	}

	if c.Regexp != "" {
		_, err := regexp.Compile(c.Regexp)
		if err != nil {
			return err
		}
	}

	return nil
}

type combinedFilter struct {
	stricts map[any]struct{}
	regexes []*regexp.Regexp
}

// CreateFilter creates a Filter out of a set of Config configuration objects.
func CreateFilter(configs []Config) Filter {
	cf := &combinedFilter{
		stricts: make(map[any]struct{}),
	}
	for _, config := range configs {
		if config.Strict != "" {
			cf.stricts[config.Strict] = struct{}{}
		}

		if config.Regexp != "" {
			// Validate() call above ensures that the regex is valid.
			re := regexp.MustCompile(config.Regexp)
			cf.regexes = append(cf.regexes, re)
		}
	}
	return cf
}

func (cf *combinedFilter) Matches(toMatch any) bool {
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
