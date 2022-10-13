// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service // import "go.opentelemetry.io/collector/service"

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	configFlag = "config"
)

var (
	// Command-line flag that control the configuration file.
	gatesList = featuregate.FlagValue{}
)

type stringArrayValue struct {
	values []string
}

func (s *stringArrayValue) Set(val string) error {
	s.values = append(s.values, val)
	return nil
}

func (s *stringArrayValue) String() string {
	return "[" + strings.Join(s.values, ", ") + "]"
}

func flags() *flag.FlagSet {
	flagSet := new(flag.FlagSet)

	cfgs := new(stringArrayValue)
	flagSet.Var(cfgs, configFlag, "Locations to the config file(s), note that only a"+
		" single location can be set per flag entry e.g. `--config=file:/path/to/first --config=file:path/to/second`.")

	flagSet.Func("set",
		`Deprecated: use "--config=yaml:processors::batch::timeout: 2s" instead of "--set=processors.batch.timeout=2s"`, func(s string) error {
			idx := strings.Index(s, "=")
			if idx == -1 {
				// No need for more context, see TestSetFlag/invalid_set.
				return errors.New("missing equal sign")
			}
			cfgFlag := "yaml:" + strings.TrimSpace(strings.ReplaceAll(s[:idx], ".", "::")) + ": " + strings.TrimSpace(s[idx+1:])
			_, _ = fmt.Fprint(flagSet.Output(), "--set flag is deprecated, replace `--set=", s, "` with `--config=", cfgFlag, "`\n")
			return cfgs.Set(cfgFlag)
		})

	flagSet.Var(
		gatesList,
		"feature-gates",
		"Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature. '+' or no prefix will enable the feature.")

	return flagSet
}

func getConfigFlag(flagSet *flag.FlagSet) []string {
	return flagSet.Lookup(configFlag).Value.(*stringArrayValue).values
}
