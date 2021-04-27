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
	"text/template"

	"go.opentelemetry.io/collector/cmd/schemagen/configschema"
)

func renderHeader(tmpl *template.Template, f *configschema.Field) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := tmpl.Execute(buf, f)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderTable(tmpl *template.Template, field *configschema.Field) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := traverseFields(tmpl, field, buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func traverseFields(tmpl *template.Template, field *configschema.Field, buf *bytes.Buffer) error {
	err := tmpl.Execute(buf, field)
	if err != nil {
		return err
	}
	for _, subField := range field.Fields {
		if subField.Fields == nil {
			continue
		}
		err = traverseFields(tmpl, subField, buf)
		if err != nil {
			return err
		}
	}
	return nil
}
