// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofileotlp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var _ json.Unmarshaler = ExportRequest{}
var _ json.Marshaler = ExportRequest{}

var profilesRequestJSON = []byte(`
	{
		"resourceProfiles": [
		{
			"resource": {},
			"scopeProfiles": [
				{
					"scope": {},
					"profileRecords": [
						{
							"body": {
								"stringValue": "test_profile_record"
							},
							"traceId": "",
							"spanId": ""
						}
					]
				}
			]
		}
		]
	}`)

func TestRequestToPData(t *testing.T) {
	tr := NewExportRequest()
	assert.Equal(t, tr.Profiles().ProfileRecordCount(), 0)
	tr.Profiles().ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	assert.Equal(t, tr.Profiles().ProfileRecordCount(), 1)
}

func TestRequestJSON(t *testing.T) {
	lr := NewExportRequest()
	assert.NoError(t, lr.UnmarshalJSON(profilesRequestJSON))
	assert.Equal(t, "test_profile_record", lr.Profiles().ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0))

	got, err := lr.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, strings.Join(strings.Fields(string(profilesRequestJSON)), ""), string(got))
}
