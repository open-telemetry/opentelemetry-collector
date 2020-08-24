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

package tracetranslator

import (
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
)

// OCAttributeKeyExist returns true if a key in attribute of an OC Span exists.
// It returns false, if attributes is nil, the map itself is nil or the key wasn't found.
func OCAttributeKeyExist(ocAttributes *tracepb.Span_Attributes, key string) bool {
	if ocAttributes == nil || ocAttributes.AttributeMap == nil {
		return false
	}
	_, foundKey := ocAttributes.AttributeMap[key]
	return foundKey
}
