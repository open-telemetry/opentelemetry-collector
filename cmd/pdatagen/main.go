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

package main

import (
	"os"

	"go.opentelemetry.io/collector/cmd/pdatagen/internal"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	for _, fp := range internal.AllFiles {
		f, err := os.Create("./model/pdata/generated_" + fp.Name + ".go")
		check(err)
		_, err = f.WriteString(fp.GenerateFile())
		check(err)
		check(f.Close())
		f, err = os.Create("./model/pdata/generated_" + fp.Name + "_test.go")
		check(err)
		_, err = f.WriteString(fp.GenerateTestFile())
		check(err)
		check(f.Close())
	}
}
