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

package internal // import "go.opentelemetry.io/collector/pdata/internal/cmd/pdatagen/internal"

var resourceFile = &File{
	Path:        "pdata",
	Name:        "resource",
	PackageName: "pcommon",
	imports: []string{
		`otlpresource "go.opentelemetry.io/collector/pdata/internal/data/protogen/resource/v1"`,
	},
	testImports: []string{
		`"testing"`,
		``,
		`"github.com/stretchr/testify/assert"`,
		``,
		`"go.opentelemetry.io/collector/pdata/internal"`,
	},
	structs: []baseStruct{
		resource,
	},
}

var resource = &messageValueStruct{
	structName:     "Resource",
	packageName:    "pcommon",
	description:    "// Resource is a message representing the resource information.",
	originFullName: "otlpresource.Resource",
	fields: []baseField{
		attributes,
		droppedAttributesCount,
	},
}

var resourceField = &messageValueField{
	fieldName:     "Resource",
	returnMessage: resource,
}
