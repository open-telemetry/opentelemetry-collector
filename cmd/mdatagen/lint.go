// Copyright The OpenTelemetry Authors
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
	"errors"
	"strings"
	"unicode"

	"go.opentelemetry.io/collector/cmd/mdatagen/third_party/golint"
)

// formatIdentifier variable in a go-safe way
func formatIdentifier(s string, exported bool) (string, error) {
	if s == "" {
		return "", errors.New("string cannot be empty")
	}
	// Convert various characters to . for strings.Title to operate on.
	replace := strings.NewReplacer("_", ".", "-", ".", "<", ".", ">", ".", "/", ".", ":", ".")
	str := replace.Replace(s)
	str = strings.Title(str)
	str = strings.ReplaceAll(str, ".", "")

	var word string
	var output string

	// Fixup acronyms to make lint happy.
	for idx, r := range str {
		if idx == 0 {
			if exported {
				r = unicode.ToUpper(r)
			} else {
				r = unicode.ToLower(r)
			}
		}

		if unicode.IsUpper(r) || unicode.IsNumber(r) {
			// If the current word is an acronym and it's either exported or it's not the
			// beginning of an unexported variable then upper case it.
			if golint.Acronyms[strings.ToUpper(word)] && (exported || output != "") {
				output += strings.ToUpper(word)
				word = string(r)
			} else {
				output += word
				word = string(r)
			}
		} else {
			word += string(r)
		}
	}

	if golint.Acronyms[strings.ToUpper(word)] && output != "" {
		output += strings.ToUpper(word)
	} else {
		output += word
	}

	// Remove white spaces
	output = strings.Join(strings.Fields(output), "")

	return output, nil
}
