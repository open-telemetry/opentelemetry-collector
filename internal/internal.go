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

package internal

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// DefaultEndpointFile is the location where the
// endpoint file is at on the current platform.
func DefaultEndpointFile() string {
	const f = "opencensus.endpoint"
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "opencensus", f)
	}
	return filepath.Join(guessUnixHomeDir(), ".config", f)
}

func guessUnixHomeDir() string {
	// Prefer $HOME over user.Current due to glibc bug: golang.org/issue/13470
	if v := os.Getenv("HOME"); v != "" {
		return v
	}
	// Else, fall back to user.Current:
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return ""
}
