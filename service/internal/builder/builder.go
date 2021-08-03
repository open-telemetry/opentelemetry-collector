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
	memBallastFlag = "mem-ballast-size-mib"

	zapKindKey         = "kind"
	zapKindReceiver    = "receiver"
	zapKindProcessor   = "processor"
	zapKindLogExporter = "exporter"
	zapKindExtension   = "extension"
	zapNameKey         = "name"
)

var (
	memBallastSize *uint
)

// Flags adds flags related to basic building of the collector server to the given flagset.
// Deprecated: keep this flag for preventing the breaking change. Use `ballast extension` instead.
func Flags(flags *flag.FlagSet) {
	memBallastSize = flags.Uint(memBallastFlag, 0,
		fmt.Sprintf("Flag to specify size of memory (MiB) ballast to set. Ballast is not used when this is not specified. "+
			"default settings: 0"))
}

// MemBallastSize returns the size of memory ballast to use in MBs
func MemBallastSize() int {
	return int(*memBallastSize)
}
