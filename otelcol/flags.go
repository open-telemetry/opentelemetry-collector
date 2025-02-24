// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"errors"
	"flag"
	"strings"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	configFlag        = "config"
	mergeStrategyFlag = "merge-strategy"
)

type mergeStrategy struct {
	value string
}

func (s *mergeStrategy) Set(val string) error {
	s.value = val
	return nil
}

func (s *mergeStrategy) String() string {
	return s.value
}

type configFlagValue struct {
	values []string
	sets   []string
}

func (s *configFlagValue) Set(val string) error {
	s.values = append(s.values, val)
	return nil
}

func (s *configFlagValue) String() string {
	return "[" + strings.Join(s.values, ", ") + "]"
}

func flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)

	cfgs := new(configFlagValue)
	flagSet.Var(cfgs, configFlag, "Locations to the config file(s), note that only a"+
		" single location can be set per flag entry e.g. `--config=file:/path/to/first --config=file:path/to/second`.")

	strategy := new(mergeStrategy)
	flagSet.Var(strategy, mergeStrategyFlag, "Merge strategy to be used while merging configurations"+
		" from multiple sources. This is currently experimental and can be enabled with confmap.enableMergeAppendOption feature gate")

	flagSet.Func("set",
		"Set arbitrary component config property. The component has to be defined in the config file and the flag"+
			" has a higher precedence. Array config properties are overridden and maps are joined. Example --set=processors.batch.timeout=2s",
		func(s string) error {
			idx := strings.Index(s, "=")
			if idx == -1 {
				// No need for more context, see TestSetFlag/invalid_set.
				return errors.New("missing equal sign")
			}
			cfgs.sets = append(cfgs.sets, "yaml:"+strings.TrimSpace(strings.ReplaceAll(s[:idx], ".", "::"))+": "+strings.TrimSpace(s[idx+1:]))
			return nil
		})

	reg.RegisterFlags(flagSet)
	return flagSet
}

func getConfigFlag(flagSet *flag.FlagSet) []string {
	cfv := flagSet.Lookup(configFlag).Value.(*configFlagValue)
	return append(cfv.values, cfv.sets...)
}

func getMergeStrategy(flagSet *flag.FlagSet) string {
	mfv := flagSet.Lookup(mergeStrategyFlag).Value.(*mergeStrategy)
	return mfv.value
}
