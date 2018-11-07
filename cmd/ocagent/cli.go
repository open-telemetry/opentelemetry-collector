// Copyright 2018, OpenCensus Authors
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

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/census-instrumentation/opencensus-service/internal/version"
)

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().IntVarP(&ocReceiverPort, "oci-port", "p", 55678, "The port on which the OpenCensus receiver is run")
	rootCmd.PersistentFlags().StringVarP(&configYAMLFile, "config", "c", "config.yaml", "The YAML file with the configurations for the agent and various exporters")
}

var rootCmd = &cobra.Command{
	Use:   "ocagent",
	Short: "ocagent runs the OpenCensus service",
	Run: func(cmd *cobra.Command, args []string) {
		runOCAgent()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information for ocagent",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(version.Info())
	},
}
