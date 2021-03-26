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

package fluentforwardreceiver

import (
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

func parseHexDump(name string) []byte {
	_, file, _, _ := runtime.Caller(0)
	dir, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		panic("Failed to find absolute path of hex dump: " + err.Error())
	}

	path := filepath.Join(dir, name+".hexdump")
	dump, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		panic("failed to read hex dump file " + path + ": " + err.Error())
	}

	var hexStr string
	for _, line := range strings.Split(string(dump), "\n") {
		if len(line) == 0 {
			continue
		}
		line = strings.Split(line, "|")[0]
		hexStr += strings.Join(strings.Fields(line)[1:], "")
	}

	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		panic("failed to parse hex bytes: " + err.Error())
	}

	return bytes
}
