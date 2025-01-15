// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

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

type flagValue struct {
	reg *featuregate.Registry
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

// Function to beautify and print the feature flags.
func featureflags(reg *featuregate.Registry)  {
	f := &flagValue{reg: reg}
	if f.reg == nil {
		return 
	}
	c := 1
	
	f.reg.VisitAll(func(g *featuregate.Gate) {
		str := ""
		id := g.ID()
		desc := g.Description()
		if !g.IsEnabled() {
			id = "-" + id
		}
		str += fmt.Sprint(c) + "." + id + "\n"
		str+= "Description: "+desc + "\n"
		ref := g.ReferenceURL()
		from := g.FromVersion()
		to := g.ToVersion()
		if(ref != ""){
			str+= "ReferenceURL: "+ref + "\n"
		}
		if(from != "v<nil>"){
			str+= "From version: "+from + "\n"
			
		}
		if(to != "v<nil>"){
			str+= "From version: "+to + "\n"
			
		}
		// str += "\n"
		c+=1
		fmt.Println(str)
	})
}

func getConfigFlag(flagSet *flag.FlagSet) []string {
	cfv := flagSet.Lookup(configFlag).Value.(*configFlagValue)
	return append(cfv.values, cfv.sets...)
}
