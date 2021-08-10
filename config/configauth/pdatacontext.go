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

package configauth

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/model/pdata"
)

type ctxKey struct{}

type AuthContext interface {
	Equal(other interface{}) bool
	GetAttribute(attrName string) interface{}
}

func InjectAuthContext(ctx context.Context, attrs map[interface{}]interface{}) context.Context {
	ac := &authC{delegate: attrs}
	return context.WithValue(ctx, ctxKey{}, ac)
}

func ExtractAuthContext(ctx context.Context) (AuthContext, bool) {
	ac, ok := ctx.Value(ctxKey{}).(*authC)
	if !ok {
		return nil, false
	}
	return ac, true
}

func InjectPDataContext(pda pdata.PDataContext, ac AuthContext) {
	pda.Set(ctxKey{}, ac)
}

func ExtractPDataContext(pda pdata.PDataContext) AuthContext {
	return pda.Get(ctxKey{}).(AuthContext)
}

type authC struct {
	delegate map[interface{}]interface{}
}

func (ac authC) Equal(other interface{}) bool {
	if other == nil {
		return false
	}
	otherAuthC, ok := other.(*authC)
	if !ok {
		return false
	}
	return reflect.DeepEqual(ac.delegate, otherAuthC.delegate)
}

func (ac authC) GetAttribute(attrName string) interface{} {
	return ac.delegate[attrName]
}

func (ac authC) String() string {
	return fmt.Sprintf("Auth Context: %v", ac.delegate)
}

// MarshalJSON serializes our context into a JSON blob. Note that interface{} is converted to string or []string.
func (ac *authC) MarshalJSON() ([]byte, error) {
	strmap := make(map[string]interface{})
	for k, v := range ac.delegate {
		strmap[fmt.Sprintf("%v", k)] = v
	}
	return json.Marshal(strmap)
}

// UnmarshalJSON deserializes our context from a JSON blob. Note that we cannot infer the original type before the serialization,
// so all entries are either string or []string.
func (ac *authC) UnmarshalJSON(data []byte) error {
	strmap := make(map[string]interface{})
	if err := json.Unmarshal(data, &strmap); err != nil {
		return err
	}

	// converts the map[string]interface{} to map[interface{}]interface{}
	delegate := make(map[interface{}]interface{})
	for k, v := range strmap {
		delegate[k] = v
	}
	ac.delegate = delegate

	return nil
}
