// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package sampling contains the interfaces and data types used to implement
// the various sampling policies.

package viperutils

import (
	"bytes"
	"flag"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ViperFromYAMLBytes unmarshals byte content in the YAML file format into
// a viper object.
func ViperFromYAMLBytes(yamlBlob []byte) (*viper.Viper, error) {
	v := viper.New()
	if err := LoadYAMLBytes(v, yamlBlob); err != nil {
		return nil, err
	}
	return v, nil
}

// LoadYAMLBytes sets the viper config type to yaml and loads the given blob.
func LoadYAMLBytes(v *viper.Viper, yamlBlob []byte) error {
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBuffer(yamlBlob)); err != nil {
		return err
	}
	return nil
}

// AddFlags adds the provided flags to the provided viper and cobra command.
func AddFlags(v *viper.Viper, command *cobra.Command, addFlagsFns ...func(*flag.FlagSet)) (*viper.Viper, *cobra.Command) {
	flagSet := new(flag.FlagSet)
	for _, addFlags := range addFlagsFns {
		addFlags(flagSet)
	}
	command.Flags().AddGoFlagSet(flagSet)

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.BindPFlags(command.Flags())
	return v, command
}
