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

import "encoding/json"

// AnagramicSignature is useful for comparing inputs such as roundtrip serialized
// JSON or encodings whose iteration orders change as object are serialized and deserialized
// for example to verify that we have the same output for:
//    ZipkinJava(any other implementation)-->ZipkinInterceptor-->AgentCore-->ZipkinExporter
// we can then compare say:
//  intercepted JSON:: [{"timestamp":1472470996199000,"duration":207000}]
//  roundtrip output:  [{"duration":207000,"timestamp":1472470996199000}]
func AnagramicSignature(ss string) string {
	mp := make(map[rune]int64)
	for _, s := range ss {
		mp[s] += 1
	}
	blob, _ := json.Marshal(mp)
	return string(blob)
}
