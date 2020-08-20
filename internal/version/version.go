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

package version

import (
	"bytes"
	"fmt"
	"runtime"
)

const (
	buildDev     = "dev"
	buildRelease = "release"
)

// Version variable will be replaced at link time after `make` has been run.
var Version = "latest"

// GitHash variable will be replaced at link time after `make` has been run.
var GitHash = "<NOT PROPERLY GENERATED>"

// BuildType should be one of (dev, release).
var BuildType = buildDev

// IsDevBuild returns true if this is a development (local) build.
func IsDevBuild() bool {
	return BuildType == buildDev
}

// IsReleaseBuild returns true if this is a release build.
func IsReleaseBuild() bool {
	return BuildType == buildRelease
}

// InfoVar is a singleton instance of the Info struct.
var InfoVar = Info([][2]string{
	{"Version", Version},
	{"GitHash", GitHash},
	{"BuildType", BuildType},
	{"Goversion", runtime.Version()},
	{"OS", runtime.GOOS},
	{"Architecture", runtime.GOARCH},
	// Add other valuable build-time information here.
})

// Info has properties about the build and runtime.
type Info [][2]string

// String returns a formatted string, with linebreaks, intended to be displayed
// on stdout.
func (i Info) String() string {
	buf := new(bytes.Buffer)
	maxRow1Alignment := 0
	for _, prop := range i {
		if cl0 := len(prop[0]); cl0 > maxRow1Alignment {
			maxRow1Alignment = cl0
		}
	}

	for _, prop := range i {
		// Then finally print them with left alignment
		fmt.Fprintf(buf, "%*s %s\n", -maxRow1Alignment, prop[0], prop[1])
	}
	return buf.String()
}
