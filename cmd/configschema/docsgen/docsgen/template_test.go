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
	"encoding/json"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/cmd/configschema/configschema"
)

func TestTableTemplate(t *testing.T) {
	field := testDataField(t)
	tmpl, err := tableTemplate()
	require.NoError(t, err)
	bytes, err := renderTable(tmpl, field)
	require.NoError(t, err)
	require.NotNil(t, bytes)
}

func testDataField(t *testing.T) *configschema.Field {
	jsonBytes, err := ioutil.ReadFile(path.Join("testdata", "otlp-receiver.json"))
	require.NoError(t, err)
	field := configschema.Field{}
	err = json.Unmarshal(jsonBytes, &field)
	require.NoError(t, err)
	return &field
}
