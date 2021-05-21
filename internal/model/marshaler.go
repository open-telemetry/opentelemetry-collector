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

package model

import (
	"fmt"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/model/serializer"
	"go.opentelemetry.io/collector/internal/model/translator"
)

// TracesMarshaler marshals pdata.Traces into bytes.
type TracesMarshaler struct {
	translate translator.TracesEncoder
	serialize serializer.TracesMarshaler
}

// Marshal pdata.Traces into bytes. On error []byte is nil.
func (t *TracesMarshaler) Marshal(td pdata.Traces) ([]byte, error) {
	model, err := t.translate.EncodeTraces(td)
	if err != nil {
		return nil, fmt.Errorf("converting pdata to model failed: %w", err)
	}
	buf, err := t.serialize.MarshalTraces(model)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	return buf, nil
}
