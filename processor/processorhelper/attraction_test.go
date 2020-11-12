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

package processorhelper

import (
	"crypto/sha1" // #nosec
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

// Common structure for all the Tests
type testCase struct {
	name               string
	inputAttributes    map[string]pdata.AttributeValue
	expectedAttributes map[string]pdata.AttributeValue
}

// runIndividualTestCase is the common logic of passing trace data through a configured attributes processor.
func runIndividualTestCase(t *testing.T, tt testCase, ap *AttrProc) {
	t.Run(tt.name, func(t *testing.T) {
		attrMap := pdata.NewAttributeMap().InitFromMap(tt.inputAttributes)
		ap.Process(attrMap)
		attrMap.Sort()
		require.Equal(t, pdata.NewAttributeMap().InitFromMap(tt.expectedAttributes).Sort(), attrMap)
	})
}

func TestAttributes_InsertValue(t *testing.T) {
	testCases := []testCase{
		// Ensure `attribute1` is set for spans with no attributes.
		{
			name:            "InsertEmptyAttributes",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		// Ensure `attribute1` is set.
		{
			name: "InsertKeyNoExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"anotherkey": pdata.NewAttributeValueString("bob"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"anotherkey": pdata.NewAttributeValueString("bob"),
				"attribute1": pdata.NewAttributeValueInt(123),
			},
		},
		// Ensures no insert is performed because the keys `attribute1` already exists.
		{
			name: "InsertKeyExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueString("bob"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"attribute1": pdata.NewAttributeValueString("bob"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "attribute1", Action: INSERT, Value: 123},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_InsertFromAttribute(t *testing.T) {

	testCases := []testCase{
		// Ensure no attribute is inserted because because attributes do not exist.
		{
			name:               "InsertEmptyAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure no attribute is inserted because because from_attribute `string_key` does not exist.
		{
			name: "InsertMissingFromAttribute",
			inputAttributes: map[string]pdata.AttributeValue{
				"bob": pdata.NewAttributeValueInt(1),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"bob": pdata.NewAttributeValueInt(1),
			},
		},
		// Ensure `string key` is set.
		{
			name: "InsertAttributeExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"anotherkey": pdata.NewAttributeValueInt(8892342),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"anotherkey": pdata.NewAttributeValueInt(8892342),
				"string key": pdata.NewAttributeValueInt(8892342),
			},
		},
		// Ensures no insert is performed because the keys `string key` already exist.
		{
			name: "InsertKeysExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"anotherkey": pdata.NewAttributeValueInt(8892342),
				"string key": pdata.NewAttributeValueString("here"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"anotherkey": pdata.NewAttributeValueInt(8892342),
				"string key": pdata.NewAttributeValueString("here"),
			},
		},
	}
	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "string key", Action: INSERT, FromAttribute: "anotherkey"},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_UpdateValue(t *testing.T) {

	testCases := []testCase{
		// Ensure no changes to the span as there is no attributes map.
		{
			name:               "UpdateNoAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure no changes to the span as the key does not exist.
		{
			name: "UpdateKeyNoExist",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("foo"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("foo"),
			},
		},
		// Ensure the attribute `db.secret` is updated.
		{
			name: "UpdateAttributes",
			inputAttributes: map[string]pdata.AttributeValue{
				"db.secret": pdata.NewAttributeValueString("password1234"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"db.secret": pdata.NewAttributeValueString("redacted"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "db.secret", Action: UPDATE, Value: "redacted"},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_UpdateFromAttribute(t *testing.T) {

	testCases := []testCase{
		// Ensure no changes to the span as there is no attributes map.
		{
			name:               "UpdateNoAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure the attribute `boo` isn't updated because attribute `foo` isn't present in the span.
		{
			name: "UpdateKeyNoExistFromAttribute",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("bob"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("bob"),
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			name: "UpdateKeyNoExistMainAttributed",
			inputAttributes: map[string]pdata.AttributeValue{
				"foo": pdata.NewAttributeValueString("over there"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"foo": pdata.NewAttributeValueString("over there"),
			},
		},
		// Ensure no updates as the target key `boo` doesn't exists.
		{
			name: "UpdateKeyFromExistingAttribute",
			inputAttributes: map[string]pdata.AttributeValue{
				"foo": pdata.NewAttributeValueString("there is a party over here"),
				"boo": pdata.NewAttributeValueString("not here"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"foo": pdata.NewAttributeValueString("there is a party over here"),
				"boo": pdata.NewAttributeValueString("there is a party over here"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "boo", Action: UPDATE, FromAttribute: "foo"},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_UpsertValue(t *testing.T) {
	testCases := []testCase{
		// Ensure `region` is set for spans with no attributes.
		{
			name:            "UpsertNoAttributes",
			inputAttributes: map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{
				"region": pdata.NewAttributeValueString("planet-earth"),
			},
		},
		// Ensure `region` is inserted for spans with some attributes(the key doesn't exist).
		{
			name: "UpsertAttributeNoExist",
			inputAttributes: map[string]pdata.AttributeValue{
				"mission": pdata.NewAttributeValueString("to mars"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"mission": pdata.NewAttributeValueString("to mars"),
				"region":  pdata.NewAttributeValueString("planet-earth"),
			},
		},
		// Ensure `region` is updated for spans with the attribute key `region`.
		{
			name: "UpsertAttributeExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"mission": pdata.NewAttributeValueString("to mars"),
				"region":  pdata.NewAttributeValueString("solar system"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"mission": pdata.NewAttributeValueString("to mars"),
				"region":  pdata.NewAttributeValueString("planet-earth"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "region", Action: UPSERT, Value: "planet-earth"},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_Extract(t *testing.T) {
	testCases := []testCase{
		// Ensure `new_user_key` is not set for spans with no attributes.
		{
			name:               "UpsertEmptyAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure `new_user_key` is not inserted for spans with missing attribute `user_key`.
		{
			name: "No extract with no target key",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `new_user_key` is not inserted for spans with missing attribute `user_key`.
		{
			name: "No extract with non string target key",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo":      pdata.NewAttributeValueString("ghosts are scary"),
				"user_key": pdata.NewAttributeValueInt(1234),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo":      pdata.NewAttributeValueString("ghosts are scary"),
				"user_key": pdata.NewAttributeValueInt(1234),
			},
		},
		// Ensure `new_user_key` is not updated for spans with attribute
		// `user_key` because `user_key` does not match the regular expression.
		{
			name: "No extract with no pattern matching",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key": pdata.NewAttributeValueString("does not match"),
				"boo":      pdata.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key": pdata.NewAttributeValueString("does not match"),
				"boo":      pdata.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `new_user_key` is not updated for spans with attribute
		// `user_key` because `user_key` does not match all of the regular
		// expression.
		{
			name: "No extract with no pattern matching",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key": pdata.NewAttributeValueString("/api/v1/document/12345678/update"),
				"boo":      pdata.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key": pdata.NewAttributeValueString("/api/v1/document/12345678/update"),
				"boo":      pdata.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `new_user_key` and `version` is inserted for spans with attribute `user_key`.
		{
			name: "Extract insert new values.",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key": pdata.NewAttributeValueString("/api/v1/document/12345678/update/v1"),
				"foo":      pdata.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueString("/api/v1/document/12345678/update/v1"),
				"new_user_key": pdata.NewAttributeValueString("12345678"),
				"version":      pdata.NewAttributeValueString("v1"),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
		},
		// Ensure `new_user_key` and `version` is updated for spans with attribute `user_key`.
		{
			name: "Extract updates existing values ",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueString("/api/v1/document/12345678/update/v1"),
				"new_user_key": pdata.NewAttributeValueString("2321"),
				"version":      pdata.NewAttributeValueString("na"),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueString("/api/v1/document/12345678/update/v1"),
				"new_user_key": pdata.NewAttributeValueString("12345678"),
				"version":      pdata.NewAttributeValueString("v1"),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
		},
		// Ensure `new_user_key` is updated and `version` is inserted for spans with attribute `user_key`.
		{
			name: "Extract upserts values",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueString("/api/v1/document/12345678/update/v1"),
				"new_user_key": pdata.NewAttributeValueString("2321"),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueString("/api/v1/document/12345678/update/v1"),
				"new_user_key": pdata.NewAttributeValueString("12345678"),
				"version":      pdata.NewAttributeValueString("v1"),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{

			{Key: "user_key", RegexPattern: "^\\/api\\/v1\\/document\\/(?P<new_user_key>.*)\\/update\\/(?P<version>.*)$", Action: EXTRACT},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_UpsertFromAttribute(t *testing.T) {

	testCases := []testCase{
		// Ensure `new_user_key` is not set for spans with no attributes.
		{
			name:               "UpsertEmptyAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure `new_user_key` is not inserted for spans with missing attribute `user_key`.
		{
			name: "UpsertFromAttributeNoExist",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `new_user_key` is inserted for spans with attribute `user_key`.
		{
			name: "UpsertFromAttributeExistsInsert",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key": pdata.NewAttributeValueInt(2245),
				"foo":      pdata.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueInt(2245),
				"new_user_key": pdata.NewAttributeValueInt(2245),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
		},
		// Ensure `new_user_key` is updated for spans with attribute `user_key`.
		{
			name: "UpsertFromAttributeExistsUpdate",
			inputAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueInt(2245),
				"new_user_key": pdata.NewAttributeValueInt(5422),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"user_key":     pdata.NewAttributeValueInt(2245),
				"new_user_key": pdata.NewAttributeValueInt(2245),
				"foo":          pdata.NewAttributeValueString("casper the friendly ghost"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "new_user_key", Action: UPSERT, FromAttribute: "user_key"},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_Delete(t *testing.T) {
	testCases := []testCase{
		// Ensure the span contains no changes.
		{
			name:               "DeleteEmptyAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure the span contains no changes because the key doesn't exist.
		{
			name: "DeleteAttributeNoExist",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("ghosts are scary"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("ghosts are scary"),
			},
		},
		// Ensure `duplicate_key` is deleted for spans with the attribute set.
		{
			name: "DeleteAttributeExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"duplicate_key": pdata.NewAttributeValueDouble(3245.6),
				"original_key":  pdata.NewAttributeValueDouble(3245.6),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"original_key": pdata.NewAttributeValueDouble(3245.6),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "duplicate_key", Action: DELETE},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_HashValue(t *testing.T) {

	intVal := int64(24)
	intBytes := make([]byte, int64ByteSize)
	binary.LittleEndian.PutUint64(intBytes, uint64(intVal))

	doubleVal := 2.4
	doubleBytes := make([]byte, float64ByteSize)
	binary.LittleEndian.PutUint64(doubleBytes, math.Float64bits(doubleVal))

	testCases := []testCase{
		// Ensure no changes to the span as there is no attributes map.
		{
			name:               "HashNoAttributes",
			inputAttributes:    map[string]pdata.AttributeValue{},
			expectedAttributes: map[string]pdata.AttributeValue{},
		},
		// Ensure no changes to the span as the key does not exist.
		{
			name: "HashKeyNoExist",
			inputAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("foo"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"boo": pdata.NewAttributeValueString("foo"),
			},
		},
		// Ensure string data types are hashed correctly
		{
			name: "HashString",
			inputAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueString("foo"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueString(sha1Hash([]byte("foo"))),
			},
		},
		// Ensure int data types are hashed correctly
		{
			name: "HashInt",
			inputAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueInt(intVal),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueString(sha1Hash(intBytes)),
			},
		},
		// Ensure double data types are hashed correctly
		{
			name: "HashDouble",
			inputAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueDouble(doubleVal),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueString(sha1Hash(doubleBytes)),
			},
		},
		// Ensure bool data types are hashed correctly
		{
			name: "HashBoolTrue",
			inputAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueBool(true),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueString(sha1Hash([]byte{1})),
			},
		},
		// Ensure bool data types are hashed correctly
		{
			name: "HashBoolFalse",
			inputAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueBool(false),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"updateme": pdata.NewAttributeValueString(sha1Hash([]byte{0})),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "updateme", Action: HASH},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestAttributes_FromAttributeNoChange(t *testing.T) {
	tc := testCase{
		name: "FromAttributeNoChange",
		inputAttributes: map[string]pdata.AttributeValue{
			"boo": pdata.NewAttributeValueString("ghosts are scary"),
		},
		expectedAttributes: map[string]pdata.AttributeValue{
			"boo": pdata.NewAttributeValueString("ghosts are scary"),
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "boo", Action: INSERT, FromAttribute: "boo"},
			{Key: "boo", Action: UPDATE, FromAttribute: "boo"},
			{Key: "boo", Action: UPSERT, FromAttribute: "boo"},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	runIndividualTestCase(t, tc, ap)
}

func TestAttributes_Ordering(t *testing.T) {
	testCases := []testCase{
		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. insert `svc.operation`: `default`
		// 3. delete `operation`.
		{
			name: "OrderingApplyAllSteps",
			inputAttributes: map[string]pdata.AttributeValue{
				"foo": pdata.NewAttributeValueString("casper the friendly ghost"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"foo":           pdata.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": pdata.NewAttributeValueString("default"),
			},
		},
		// For this example, the operations performed are
		// 1. do nothing for the first action of insert `operation`: `default`
		// 2. insert `svc.operation`: `arithmetic`
		// 3. delete `operation`.
		{
			name: "OrderingOperationExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"foo":       pdata.NewAttributeValueString("casper the friendly ghost"),
				"operation": pdata.NewAttributeValueString("arithmetic"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"foo":           pdata.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": pdata.NewAttributeValueString("arithmetic"),
			},
		},

		// For this example, the operations performed are
		// 1. insert `operation`: `default`
		// 2. update `svc.operation` to `default`
		// 3. delete `operation`.
		{
			name: "OrderingSvcOperationExists",
			inputAttributes: map[string]pdata.AttributeValue{
				"foo":           pdata.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": pdata.NewAttributeValueString("some value"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"foo":           pdata.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": pdata.NewAttributeValueString("default"),
			},
		},

		// For this example, the operations performed are
		// 1. do nothing for the first action of insert `operation`: `default`
		// 2. update `svc.operation` to `arithmetic`
		// 3. delete `operation`.
		{
			name: "OrderingBothAttributesExist",
			inputAttributes: map[string]pdata.AttributeValue{
				"foo":           pdata.NewAttributeValueString("casper the friendly ghost"),
				"operation":     pdata.NewAttributeValueString("arithmetic"),
				"svc.operation": pdata.NewAttributeValueString("add"),
			},
			expectedAttributes: map[string]pdata.AttributeValue{
				"foo":           pdata.NewAttributeValueString("casper the friendly ghost"),
				"svc.operation": pdata.NewAttributeValueString("arithmetic"),
			},
		},
	}

	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "operation", Action: INSERT, Value: "default"},
			{Key: "svc.operation", Action: UPSERT, FromAttribute: "operation"},
			{Key: "operation", Action: DELETE},
		},
	}

	ap, err := NewAttrProc(cfg)
	require.Nil(t, err)
	require.NotNil(t, ap)

	for _, tt := range testCases {
		runIndividualTestCase(t, tt, ap)
	}
}

func TestInvalidConfig(t *testing.T) {
	testcase := []struct {
		name        string
		actionLists []ActionKeyValue
		errorString string
	}{
		{
			name: "missing key",
			actionLists: []ActionKeyValue{
				{Key: "one", Action: DELETE},
				{Key: "", Value: 123, Action: UPSERT},
			},
			errorString: "error creating AttrProc due to missing required field \"key\" at the 1-th actions",
		},
		{
			name: "invalid action",
			actionLists: []ActionKeyValue{
				{Key: "invalid", Action: "invalid"},
			},
			errorString: "error creating AttrProc due to unsupported action \"invalid\" at the 0-th actions",
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
			errorString: "error creating AttrProc. Either field \"value\" or \"from_attribute\" setting must be specified for 0-th action",
		},
		{
			name: "both set value and from attribute",
			actionLists: []ActionKeyValue{
				{Key: "BothSet", Value: 123, FromAttribute: "aa", Action: UPSERT},
			},
			errorString: "error creating AttrProc due to both fields \"value\" and \"from_attribute\" being set at the 0-th actions",
		},
		{
			name: "pattern shouldn't be specified",
			actionLists: []ActionKeyValue{
				{Key: "key", RegexPattern: "(?P<operation_website>.*?)$", FromAttribute: "aa", Action: INSERT},
			},
			errorString: "error creating AttrProc. Action \"insert\" does not use the \"pattern\" field. This must not be specified for 0-th action",
		},
		{
			name: "missing rule for extract",
			actionLists: []ActionKeyValue{
				{Key: "aa", Action: EXTRACT},
			},
			errorString: "error creating AttrProc due to missing required field \"pattern\" for action \"extract\" at the 0-th action",
		},
		{name: "set value for extract",
			actionLists: []ActionKeyValue{
				{Key: "Key", RegexPattern: "(?P<operation_website>.*?)$", Value: "value", Action: EXTRACT},
			},
			errorString: "error creating AttrProc. Action \"extract\" does not use \"value\" or \"from_attribute\" field. These must not be specified for 0-th action",
		},
		{
			name: "set from attribute for extract",
			actionLists: []ActionKeyValue{
				{Key: "key", RegexPattern: "(?P<operation_website>.*?)$", FromAttribute: "aa", Action: EXTRACT},
			},
			errorString: "error creating AttrProc. Action \"extract\" does not use \"value\" or \"from_attribute\" field. These must not be specified for 0-th action",
		},
		{
			name: "invalid regex",
			actionLists: []ActionKeyValue{
				{Key: "aa", RegexPattern: "(?P<invalid.regex>.*?)$", Action: EXTRACT},
			},
			errorString: "error creating AttrProc. Field \"pattern\" has invalid pattern: \"(?P<invalid.regex>.*?)$\" to be set at the 0-th actions",
		},
		{
			name: "delete with regex",
			actionLists: []ActionKeyValue{
				{RegexPattern: "(?P<operation_website>.*?)$", Key: "ab", Action: DELETE},
			},
			errorString: "error creating AttrProc. Action \"delete\" does not use \"value\", \"pattern\" or \"from_attribute\" field. These must not be specified for 0-th action",
		},
		{
			name: "regex with unnamed capture group",
			actionLists: []ActionKeyValue{
				{Key: "aa", RegexPattern: ".*$", Action: EXTRACT},
			},
			errorString: "error creating AttrProc. Field \"pattern\" contains no named matcher groups at the 0-th actions",
		},
		{
			name: "regex with one unnamed capture groups",
			actionLists: []ActionKeyValue{
				{Key: "aa", RegexPattern: "^\\/api\\/v1\\/document\\/(?P<new_user_key>.*)\\/update\\/(.*)$", Action: EXTRACT},
			},
			errorString: "error creating AttrProc. Field \"pattern\" contains at least one unnamed matcher group at the 0-th actions",
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			ap, err := NewAttrProc(&Settings{Actions: tc.actionLists})
			assert.Nil(t, ap)
			assert.EqualValues(t, errors.New(tc.errorString), err)
		})
	}
}

func TestValidConfiguration(t *testing.T) {
	cfg := &Settings{
		Actions: []ActionKeyValue{
			{Key: "one", Action: "Delete"},
			{Key: "two", Value: 123, Action: "INSERT"},
			{Key: "three", FromAttribute: "two", Action: "upDaTE"},
			{Key: "five", FromAttribute: "two", Action: "upsert"},
			{Key: "two", RegexPattern: "^\\/api\\/v1\\/document\\/(?P<documentId>.*)\\/update$", Action: "EXTRact"},
		},
	}
	ap, err := NewAttrProc(cfg)
	require.NoError(t, err)

	av := pdata.NewAttributeValueInt(123)
	compiledRegex := regexp.MustCompile(`^\/api\/v1\/document\/(?P<documentId>.*)\/update$`)
	assert.Equal(t, []attributeAction{
		{Key: "one", Action: DELETE},
		{Key: "two", Action: INSERT,
			AttributeValue: &av,
		},
		{Key: "three", FromAttribute: "two", Action: UPDATE},
		{Key: "five", FromAttribute: "two", Action: UPSERT},
		{Key: "two", Regex: compiledRegex, AttrNames: []string{"", "documentId"}, Action: EXTRACT},
	}, ap.actions)

}

func sha1Hash(b []byte) string {
	// #nosec
	h := sha1.New()
	h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
