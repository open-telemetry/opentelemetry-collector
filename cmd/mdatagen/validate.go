// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/multierr"
)

func (md *metadata) Validate() error {
	var errs error
	if err := md.validateType(); err != nil {
		errs = multierr.Append(errs, err)
	}
	if err := md.validateStatus(); err != nil {
		errs = multierr.Append(errs, err)
	}
	if err := md.validateResourceAttributes(); err != nil {
		errs = multierr.Append(errs, err)
	}
	if err := md.validateMetrics(); err != nil {
		errs = multierr.Append(errs, err)
	}
	return errs
}

func (md *metadata) validateType() error {
	if md.Type == "" {
		return errors.New("missing type")
	}
	return nil
}

func (md *metadata) validateStatus() error {
	if md.Parent != "" && md.Status == nil {
		// status is not required for subcomponents.
		return nil
	}

	var errs error
	if md.Status == nil {
		return errors.New("missing status")
	}
	if err := md.Status.validateClass(); err != nil {
		errs = multierr.Append(errs, err)
	}
	if md.Parent == "" {
		if err := md.Status.validateStability(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}

func (s *Status) validateClass() error {
	if s.Class == "" {
		return errors.New("missing class")
	}
	if s.Class != "receiver" && s.Class != "processor" && s.Class != "exporter" && s.Class != "connector" && s.Class != "extension" && s.Class != "cmd" && s.Class != "pkg" {
		return fmt.Errorf("invalid class: %v", s.Class)
	}
	return nil
}

func (s *Status) validateStability() error {
	var errs error
	if len(s.Stability) == 0 {
		return errors.New("missing stability")
	}
	for stability, component := range s.Stability {
		if stability != "development" && stability != "alpha" && stability != "beta" && stability != "stable" && stability != "deprecated" && stability != "unmaintained" {
			errs = multierr.Append(errs, fmt.Errorf("invalid stability: %v", stability))
		}
		if component == nil {
			errs = multierr.Append(errs, fmt.Errorf("missing component for stability: %v", stability))
		}
		for _, c := range component {
			if c != "metrics" &&
				c != "traces" &&
				c != "logs" &&
				c != "traces_to_traces" &&
				c != "traces_to_metrics" &&
				c != "traces_to_logs" &&
				c != "metrics_to_traces" &&
				c != "metrics_to_metrics" &&
				c != "metrics_to_logs" &&
				c != "logs_to_traces" &&
				c != "logs_to_metrics" &&
				c != "logs_to_logs" &&
				c != "extension" {
				errs = multierr.Append(errs, fmt.Errorf("invalid component: %v", c))
			}
		}
	}
	return errs
}

func (md *metadata) validateResourceAttributes() error {
	var errs error
	for name, attr := range md.ResourceAttributes {
		if attr.Description == "" {
			errs = multierr.Append(errs, fmt.Errorf("empty description for resource attribute: %v", name))
		}
		empty := ValueType{ValueType: pcommon.ValueTypeEmpty}
		if attr.Type == empty {
			errs = multierr.Append(errs, fmt.Errorf("empty type for resource attribute: %v", name))
		}
	}
	return errs
}

func (md *metadata) validateMetrics() error {
	var errs error
	usedAttrs := map[attributeName]bool{}
	for mn, m := range md.Metrics {
		if m.Sum == nil && m.Gauge == nil {
			errs = multierr.Append(errs, fmt.Errorf("metric %v doesn't have a metric type key, "+
				"one of the following has to be specified: sum, gauge", mn))
			continue
		}
		if m.Sum != nil && m.Gauge != nil {
			errs = multierr.Append(errs, fmt.Errorf("metric %v has more than one metric type keys, "+
				"only one of the following has to be specified: sum, gauge", mn))
			continue
		}
		if err := m.validate(); err != nil {
			errs = multierr.Append(errs, fmt.Errorf(`metric "%v": %w`, mn, err))
			continue
		}
		unknownAttrs := make([]attributeName, 0, len(m.Attributes))
		for _, attr := range m.Attributes {
			if _, ok := md.Attributes[attr]; ok {
				usedAttrs[attr] = true
			} else {
				unknownAttrs = append(unknownAttrs, attr)
			}
		}
		if len(unknownAttrs) > 0 {
			errs = multierr.Append(errs, fmt.Errorf(`metric "%v" refers to undefined attributes: %v`, mn, unknownAttrs))
		}
	}
	errs = multierr.Append(errs, md.validateAttributes(usedAttrs))
	return errs
}

func (m *metric) validate() error {
	var errs error
	if m.Description == "" {
		errs = multierr.Append(errs, errors.New(`missing metric description`))
	}
	if m.Unit == nil {
		errs = multierr.Append(errs, errors.New(`missing metric unit`))
	}
	if m.Sum != nil {
		errs = multierr.Append(errs, m.Sum.Validate())
	}
	if m.Gauge != nil {
		errs = multierr.Append(errs, m.Gauge.Validate())
	}
	return errs
}

func (mit MetricInputType) Validate() error {
	if mit.InputType != "" && mit.InputType != "string" {
		return fmt.Errorf("invalid `input_type` value \"%v\", must be \"\" or \"string\"", mit.InputType)
	}
	return nil
}

func (md *metadata) validateAttributes(usedAttrs map[attributeName]bool) error {
	var errs error
	unusedAttrs := make([]attributeName, 0, len(md.Attributes))
	for attrName, attr := range md.Attributes {
		if attr.Description == "" {
			errs = multierr.Append(errs, fmt.Errorf(`missing attribute description for: %v`, attrName))
		}
		empty := ValueType{ValueType: pcommon.ValueTypeEmpty}
		if attr.Type == empty {
			errs = multierr.Append(errs, fmt.Errorf("empty type for attribute: %v", attrName))
		}
		if !usedAttrs[attrName] {
			unusedAttrs = append(unusedAttrs, attrName)
		}
	}
	if len(unusedAttrs) > 0 {
		errs = multierr.Append(errs, fmt.Errorf("unused attributes: %v", unusedAttrs))
	}
	return errs
}
