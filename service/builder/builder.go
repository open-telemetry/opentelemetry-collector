// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"flag"
	"fmt"
)

const (
	// flags
	configCfg      = "config"
	memBallastFlag = "mem-ballast-size-mib"

	kindLogKey        = "component_kind"
	kindLogsReceiver  = "receiver"
	kindLogsProcessor = "processor"
	kindLogsExporter  = "exporter"
	kindLogExtension  = "extension"
	typeLogKey        = "component_type"
	nameLogKey        = "component_name"
)

var (
	configFile     *string
	memBallastSize *uint
)

// Flags adds flags related to basic building of the collector application to the given flagset.
func Flags(flags *flag.FlagSet) {
	configFile = flags.String(configCfg, "", "Path to the config file")
	memBallastSize = flags.Uint(memBallastFlag, 0,
		fmt.Sprintf("Flag to specify size of memory (MiB) ballast to set. Ballast is not used when this is not specified. "+
			"default settings: 0"))
}

// GetConfigFile gets the config file from the config file flag.
func GetConfigFile() string {
	return *configFile
}

// MemBallastSize returns the size of memory ballast to use in MBs
func MemBallastSize() int {
	return int(*memBallastSize)
}
