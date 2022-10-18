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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/featuregate"
	"gopkg.in/yaml.v3"
	"strings"
)

const (
	configFlag    = "config"
	buildInfoFlag = "build-info"
)

var (
	// Command-line flag that control the configuration file.
	gatesList = featuregate.FlagValue{}
	BuildFlag bool
)

type configFlagValue struct {
	values []string
	sets   []string
}

type componentsOutput struct {
	Version string

	Receivers []config.Type

	Processors []config.Type

	Exporters []config.Type

	Extensions []config.Type
}

func (s *configFlagValue) Set(val string) error {
	s.values = append(s.values, val)
	return nil
}

func (s *configFlagValue) String() string {
	return "[" + strings.Join(s.values, ", ") + "]"
}

func flags() *flag.FlagSet {
	flagSet := new(flag.FlagSet)

	cfgs := new(configFlagValue)
	flagSet.Var(cfgs, configFlag, "Locations to the config file(s), note that only a"+
		" single location can be set per flag entry e.g. `--config=file:/path/to/first --config=file:path/to/second`.")

	flagSet.Func("set",
		"Set arbitrary component config property. The component has to be defined in the config file and the flag"+
			" has a higher precedence. Array config properties are overridden and maps are joined, note that only a single"+
			" (first) array property can be set e.g. --set=processors.attributes.actions.key=some_key. Example --set=processors.batch.timeout=2s",
		func(s string) error {
			idx := strings.Index(s, "=")
			if idx == -1 {
				// No need for more context, see TestSetFlag/invalid_set.
				return errors.New("missing equal sign")
			}
			cfgs.sets = append(cfgs.sets, "yaml:"+strings.TrimSpace(strings.ReplaceAll(s[:idx], ".", "::"))+": "+strings.TrimSpace(s[idx+1:]))
			return nil
		})

	flagSet.Var(
		gatesList,
		"feature-gates",
		"Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature. '+' or no prefix will enable the feature.")

	flagSet.BoolVar(&BuildFlag, buildInfoFlag, false,
		"Displays list of components available in collector distribution in yaml format",
	)

	return flagSet
}

func getConfigFlag(flagSet *flag.FlagSet) []string {
	cfv := flagSet.Lookup(configFlag).Value.(*configFlagValue)
	return append(cfv.values, cfv.sets...)
}

func getBuildInfo(set CollectorSettings) (error, []byte) {
	components := componentsOutput{}
	for ext, _ := range set.Factories.Extensions {
		components.Extensions = append(components.Extensions, ext)
	}
	for prs, _ := range set.Factories.Processors {
		components.Processors = append(components.Processors, prs)
	}
	for rcv, _ := range set.Factories.Receivers {
		components.Receivers = append(components.Receivers, rcv)
	}
	for exp, _ := range set.Factories.Exporters {
		components.Exporters = append(components.Exporters, exp)
	}
	components.Version = set.BuildInfo.Version

	yamlData, err := yaml.Marshal(components)

	if err != nil {
		return err, nil
	}

	fmt.Println(string(yamlData))
	return nil, yamlData
}
