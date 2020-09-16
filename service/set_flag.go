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
	flagSet.StringArray(setFlagName, []string{}, "Set arbitrary component config property. The component has to be defined in the config file. The flag has a higher precedence over config file. The arrays are overridden. Example --set=processors.batch.timeout=2s")
}

// addSetFlagProperties sets/overrides properties from set flag(s) to supplied viper instance.
// The implementation reads set flag(s) from the cmd and passes the content to viper as .properties file.
func addSetFlagProperties(v *viper.Viper, cmd *cobra.Command) error {
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

	for _, k := range viperFlags.AllKeys() {
		v.Set(k, viperFlags.Get(k))
	}
	return nil
}
