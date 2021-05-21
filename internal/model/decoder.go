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

// TracesDecoder decodes bytes into pdata.Traces.
type TracesDecoder struct {
	translate translator.TracesDecoder
	serialize serializer.TracesUnmarshaler
}

// Decode bytes into pdata.Traces. On error pdata.Traces is invalid.
func (t *TracesDecoder) Decode(buf []byte) (pdata.Traces, error) {
	model, err := t.serialize.UnmarshalTraces(buf)
	if err != nil {
		return pdata.Traces{}, fmt.Errorf("unmarshal failed: %w", err)
	}
	td, err := t.translate.DecodeTraces(model)
	if err != nil {
		return pdata.Traces{}, fmt.Errorf("converting model to pdata failed: %w", err)
	}
	return td, nil
}
