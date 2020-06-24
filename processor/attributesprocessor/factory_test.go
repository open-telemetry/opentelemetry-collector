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

package attributesprocessor

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestFactory_Type(t *testing.T) {
	factory := Factory{}
	assert.Equal(t, factory.Type(), configmodels.Type(typeStr))
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, cfg, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: typeStr,
			TypeVal: typeStr,
		},
	})
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestFactory_CreateTraceProcessor(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []ActionKeyValue{
		{Key: "a key", Action: DELETE},
	}

	tp, err := factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	assert.NotNil(t, tp)
	assert.NoError(t, err)

	tp, err = factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{}, nil, cfg)
	assert.Nil(t, tp)
	assert.NotNil(t, err)

	oCfg.Actions = []ActionKeyValue{
		{Action: DELETE},
	}
	tp, err = factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	assert.Nil(t, tp)
	assert.NotNil(t, err)
}

func TestFactory_CreateMetricsProcessor(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	mp, err := factory.CreateMetricsProcessor(
		context.Background(), component.ProcessorCreateParams{}, nil, cfg)
	require.Nil(t, mp)
	assert.Equal(t, err, configerror.ErrDataTypeIsNotSupported)
}

func TestFactory_validateActionsConfiguration(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)

	oCfg.Actions = []ActionKeyValue{
		{Key: "one", Action: "Delete"},
		{Key: "two", Value: 123, Action: "INSERT"},
		{Key: "three", FromAttribute: "two", Action: "upDaTE"},
		{Key: "five", FromAttribute: "two", Action: "upsert"},
		{Key: "two", RegexPattern: "^\\/api\\/v1\\/document\\/(?P<documentId>.*)\\/update$", Action: "EXTRact"},
	}
	output, err := buildAttributesConfiguration(*oCfg)
	require.NoError(t, err)
	av := pdata.NewAttributeValueInt(123)
	compiledRegex, err := regexp.Compile(`^\/api\/v1\/document\/(?P<documentId>.*)\/update$`)
	require.NoError(t, err)

	assert.Equal(t, []attributeAction{
		{Key: "one", Action: DELETE},
		{Key: "two", Action: INSERT,
			AttributeValue: &av,
		},
		{Key: "three", FromAttribute: "two", Action: UPDATE},
		{Key: "five", FromAttribute: "two", Action: UPSERT},
		{Key: "two", Regex: compiledRegex, AttrNames: []string{"", "documentId"}, Action: EXTRACT},
	}, output)

}

func TestFactory_validateActionsConfiguration_InvalidConfig(t *testing.T) {
	testcase := []struct {
		name        string
		actionLists []ActionKeyValue
		errorString string
	}{
		{
			name:        "empty action lists",
			actionLists: []ActionKeyValue{},
			errorString: "error creating \"attributes\" processor due to missing required field \"actions\" of processor \"attributes/error\"",
		},
		{
			name: "missing key",
			actionLists: []ActionKeyValue{
				{Key: "one", Action: DELETE},
				{Key: "", Value: 123, Action: UPSERT},
			},
			errorString: "error creating \"attributes\" processor due to missing required field \"key\" at the 1-th actions of processor \"attributes/error\"",
		},
		{
			name: "invalid action",
			actionLists: []ActionKeyValue{
				{Key: "invalid", Action: "invalid"},
			},
			errorString: "error creating \"attributes\" processor due to unsupported action \"invalid\" at the 0-th actions of processor \"attributes/error\"",
		},
		{
			name: "unsupported value",
			actionLists: []ActionKeyValue{
				{Key: "UnsupportedValue", Value: []int{}, Action: UPSERT},
			},
			errorString: "error unsupported value type \"[]int\"",
		},
		{
			name: "missing value or from attribute",
			actionLists: []ActionKeyValue{
				{Key: "MissingValueFromAttributes", Action: INSERT},
			},
			errorString: "error creating \"attributes\" processor. Either field \"value\" or \"from_attribute\" setting must be specified for 0-th action of processor \"attributes/error\"",
		},
		{
			name: "both set value and from attribute",
			actionLists: []ActionKeyValue{
				{Key: "BothSet", Value: 123, FromAttribute: "aa", Action: UPSERT},
			},
			errorString: "error creating \"attributes\" processor due to both fields \"value\" and \"from_attribute\" being set at the 0-th actions of processor \"attributes/error\"",
		},
		{
			name: "pattern shouldn't be specified",
			actionLists: []ActionKeyValue{
				{Key: "key", RegexPattern: "(?P<operation_website>.*?)$", FromAttribute: "aa", Action: INSERT},
			},
			errorString: "error creating \"attributes\" processor. Action \"insert\" does not use the \"pattern\" field. This must not be specified for 0-th action of processor \"attributes/error\"",
		},
		{
			name: "missing rule for extract",
			actionLists: []ActionKeyValue{
				{Key: "aa", Action: EXTRACT},
			},
			errorString: "error creating \"attributes\" processor due to missing required field \"pattern\" for action \"extract\" at the 0-th action of processor \"attributes/error\"",
		},
		{name: "set value for extract",
			actionLists: []ActionKeyValue{
				{Key: "Key", RegexPattern: "(?P<operation_website>.*?)$", Value: "value", Action: EXTRACT},
			},
			errorString: "error creating \"attributes\" processor. Action \"extract\" does not use \"value\" or \"from_attribute\" field. These must not be specified for 0-th action of processor \"attributes/error\"",
		},
		{
			name: "set from attribute for extract",
			actionLists: []ActionKeyValue{
				{Key: "key", RegexPattern: "(?P<operation_website>.*?)$", FromAttribute: "aa", Action: EXTRACT},
			},
			errorString: "error creating \"attributes\" processor. Action \"extract\" does not use \"value\" or \"from_attribute\" field. These must not be specified for 0-th action of processor \"attributes/error\"",
		},
		{
			name: "invalid regex",
			actionLists: []ActionKeyValue{
				{Key: "aa", RegexPattern: "(?P<invalid.regex>.*?)$", Action: EXTRACT},
			},
			errorString: "error creating \"attributes\" processor. Field \"pattern\" has invalid pattern: \"(?P<invalid.regex>.*?)$\" to be set at the 0-th actions of processor \"attributes/error\"",
		},
		{
			name: "delete with regex",
			actionLists: []ActionKeyValue{
				{RegexPattern: "(?P<operation_website>.*?)$", Key: "ab", Action: DELETE},
			},
			errorString: "error creating \"attributes\" processor. Action \"delete\" does not use \"value\", \"pattern\" or \"from_attribute\" field. These must not be specified for 0-th action of processor \"attributes/error\"",
		},
		{
			name: "regex with unnamed capture group",
			actionLists: []ActionKeyValue{
				{Key: "aa", RegexPattern: ".*$", Action: EXTRACT},
			},
			errorString: "error creating \"attributes\" processor. Field \"pattern\" contains no named matcher groups at the 0-th actions of processor \"attributes/error\"",
		},
		{
			name: "regex with one unnamed capture groups",
			actionLists: []ActionKeyValue{
				{Key: "aa", RegexPattern: "^\\/api\\/v1\\/document\\/(?P<new_user_key>.*)\\/update\\/(.*)$", Action: EXTRACT},
			},
			errorString: "error creating \"attributes\" processor. Field \"pattern\" contains at least one unnamed matcher group at the 0-th actions of processor \"attributes/error\"",
		},
	}
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.NameVal = "attributes/error"
	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			oCfg.Actions = tc.actionLists
			output, err := buildAttributesConfiguration(*oCfg)
			assert.Nil(t, output)
			assert.Equal(t, err.Error(), tc.errorString)
		})
	}
}
