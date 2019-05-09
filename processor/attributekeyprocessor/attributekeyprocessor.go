// Copyright 2019, OpenCensus Authors
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

package attributekeyprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/processor"
)

// KeyReplacement identifies a key and its respective replacement.
type KeyReplacement struct {
	// Key the attribute key to be replaced.
	Key string `mapstructure:"key"`
	// NewKey is the value that will be used as the new key for the attribute.
	NewKey string `mapstructure:"replacement"`
	// Overwrite is set to true to indicate that the replacement should be
	// performed even if the new key already exists on the attributes.
	// In this case the original value associated with the new key is lost.
	Overwrite bool `mapstructure:"overwrite"`
	// KeepOriginal is set to true to indicate that the original key
	// should not be removed from the attributes.
	KeepOriginal bool `mapstructure:"keep"`
}

type attributekeyprocessor struct {
	nextConsumer consumer.TraceConsumer
	replacements []KeyReplacement
}

var _ processor.TraceProcessor = (*attributekeyprocessor)(nil)

// NewTraceProcessor returns a processor.TraceProcessor
func NewTraceProcessor(nextConsumer consumer.TraceConsumer, replacements ...KeyReplacement) (processor.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, errors.New("nextConsumer is nil")
	}

	lenReplacements := len(replacements)
	if lenReplacements > 0 {
		seenKeys := make(map[string]bool, lenReplacements)
		for _, replacement := range replacements {
			if seenKeys[replacement.Key] {
				return nil, fmt.Errorf("replacement key %q already specified", replacement.Key)
			}
			seenKeys[replacement.Key] = true
			if seenKeys[replacement.NewKey] {
				return nil, fmt.Errorf("replacement new key %q is already a key being mapped", replacement.NewKey)
			}
		}
	}

	return &attributekeyprocessor{
		nextConsumer: nextConsumer,
		replacements: replacements,
	}, nil
}

func (akp *attributekeyprocessor) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	if len(akp.replacements) == 0 {
		return akp.nextConsumer.ConsumeTraceData(ctx, td)
	}
	for _, span := range td.Spans {
		if span == nil || span.Attributes == nil || len(span.Attributes.AttributeMap) == 0 {
			// Nothing to do
			continue
		}

		attribMap := span.Attributes.AttributeMap
		for _, replacement := range akp.replacements {
			if keyValue, oldKeyPresent := attribMap[replacement.Key]; oldKeyPresent {
				newKeyMapped := func() bool {
					_, newKeyPresent := attribMap[replacement.NewKey]
					return newKeyPresent
				}
				if replacement.Overwrite || !newKeyMapped() {
					attribMap[replacement.NewKey] = keyValue
					if !replacement.KeepOriginal {
						delete(attribMap, replacement.Key)
					}
				}
			}
		}
	}
	return akp.nextConsumer.ConsumeTraceData(ctx, td)
}
