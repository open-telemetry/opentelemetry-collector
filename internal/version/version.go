// Copyright 2019, OpenTelemetry Authors
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

package version

import (
	"bytes"
	"fmt"
	"runtime"
)

// Version variable will be replaced at link time after `make` has been run.
var Version = "latest"

// GitHash variable will be replaced at link time after `make` has been run.
var GitHash = "<NOT PROPERLY GENERATED>"

// Info returns a formatted string, with linebreaks, intended to be displayed
// on stdout.
func Info() string {
	buf := new(bytes.Buffer)
	rows := [][2]string{
		{"Version", Version},
		{"GitHash", GitHash},
		{"Goversion", runtime.Version()},
		{"OS", runtime.GOOS},
		{"Architecture", runtime.GOARCH},
		// Add other valuable build-time information here.
	}
	maxRow1Alignment := 0
	for _, row := range rows {
		if cl0 := len(row[0]); cl0 > maxRow1Alignment {
			maxRow1Alignment = cl0
		}
	}

	for _, row := range rows {
		// Then finally print them with left alignment
		fmt.Fprintf(buf, "%*s %s\n", -maxRow1Alignment, row[0], row[1])
	}
	return buf.String()
}
