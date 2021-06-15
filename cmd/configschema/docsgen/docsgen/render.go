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

package docsgen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"go.opentelemetry.io/collector/cmd/configschema/configschema"
)

func renderHeader(typ, group, doc string) []byte {
	return []byte(fmt.Sprintf(
		"# %q %s Reference\n\n%s\n\n",
		typ,
		strings.Title(group),
		doc,
	))
}

func renderTable(tmpl *template.Template, field *configschema.Field) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := executeTableTemplate(tmpl, field, buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func executeTableTemplate(tmpl *template.Template, field *configschema.Field, buf *bytes.Buffer) error {
	err := tmpl.Execute(buf, field)
	if err != nil {
		return err
	}
	for _, subField := range field.Fields {
		if subField.Fields == nil {
			continue
		}
		err = executeTableTemplate(tmpl, subField, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

const durationBlock = "### time-Duration \n" +
	"An optionally signed sequence of decimal numbers, " +
	"each with a unit suffix, such as `300ms`, `-1.5h`, " +
	"or `2h45m`. Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`."

func hasTimeDuration(f *configschema.Field) bool {
	if f.Type == "time.Duration" {
		return true
	}
	for _, sub := range f.Fields {
		if hasTimeDuration(sub) {
			return true
		}
	}
	return false
}
