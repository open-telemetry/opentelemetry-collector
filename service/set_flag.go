// Copyright  The OpenTelemetry Authors
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

package service

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"go.opentelemetry.io/collector/config"
)

const (
	setFlagName     = "set"
	setFlagFileType = "properties"
)

func addSetFlag(flagSet *pflag.FlagSet) {
	flagSet.StringArray(setFlagName, []string{}, "Set arbitrary component config property. The component has to be defined in the config file and the flag has a higher precedence. Array config properties are overridden and maps are joined, note that only a single (first) array property can be set e.g. -set=processors.attributes.actions.key=some_key. Example --set=processors.batch.timeout=2s")
}

// AddSetFlagProperties overrides properties from set flag(s) in supplied viper instance.
// The implementation reads set flag(s) from the cmd and passes the content to a new viper instance as .properties file.
// Then the properties from new viper instance are read and set to the supplied viper.
func AddSetFlagProperties(v *viper.Viper, cmd *cobra.Command) error {
	flagProperties, err := cmd.Flags().GetStringArray(setFlagName)
	if err != nil {
		return err
	}
	if len(flagProperties) == 0 {
		return nil
	}
	b := &bytes.Buffer{}
	for _, property := range flagProperties {
		property = strings.TrimSpace(property)
		if _, err := fmt.Fprintf(b, "%s\n", property); err != nil {
			return err
		}
	}
	viperFlags := config.NewViper()
	viperFlags.SetConfigType(setFlagFileType)
	if err := viperFlags.ReadConfig(b); err != nil {
		return fmt.Errorf("failed to read set flag config: %v", err)
	}

	// Viper implementation of v.MergeConfig(io.Reader) or v.MergeConfigMap(map[string]interface)
	// does not work properly.  This is b/c if it attempts to merge into a nil object it will fail here
	// https://github.com/spf13/viper/blob/3826be313591f83193f048520482a7b3cf17d506/viper.go#L1709

	// The workaround is to call v.Set(string, interface) on all root properties from the config file
	// this will correctly preserve the original config and set them up for viper to overlay them with
	// the --set params.  It should also be noted that setting the root keys is important.  This is
	// b/c the viper .AllKeys() method does not return empty objects.
	// For instance with the following yaml structure:
	// a:
	//   b:
	//   c: {}
	//
	// viper.AllKeys() would only return a.b, but not a.c.  However otel expects {} to behave
	// the same as nil object in its config file.  Therefore we extract and set the root keys only
	// to catch both a.b and a.c.

	rootKeys := map[string]struct{}{}
	for _, k := range viperFlags.AllKeys() {
		keys := strings.Split(k, config.ViperDelimiter)
		if len(keys) > 0 {
			rootKeys[keys[0]] = struct{}{}
		}
	}

	for k := range rootKeys {
		v.Set(k, v.Get(k))
	}

	// now that we've copied the config into the viper "overrides" copy the --set flags
	// as well
	for _, k := range viperFlags.AllKeys() {
		v.Set(k, viperFlags.Get(k))
	}

	return nil
}
