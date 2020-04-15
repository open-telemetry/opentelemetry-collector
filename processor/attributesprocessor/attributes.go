// Copyright 2019, OpenTelemetry Authors
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

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/filterspan"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type attributesProcessor struct {
	nextConsumer consumer.TraceConsumer
	config       attributesConfig
}

// This structure is very similar to the config for attributes processor
// with the value in the converted attribute format instead of the
// raw format from the configuration.
type attributesConfig struct {
	actions []attributeAction
	include filterspan.Matcher
	exclude filterspan.Matcher
}

type attributeAction struct {
	Key           string
	FromAttribute string
	// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/296
	// Do benchmark testing between having action be of type string vs integer.
	// The reason is attributes processor will most likely be commonly used
	// and could impact performance.
	Action         Action
	AttributeValue *pdata.AttributeValue
}

// newTraceProcessor returns a processor that modifies attributes of a span.
// To construct the attributes processors, the use of the factory methods are required
// in order to validate the inputs.
func newTraceProcessor(nextConsumer consumer.TraceConsumer, config attributesConfig) (component.TraceProcessor, error) {
	if nextConsumer == nil {
		return nil, componenterror.ErrNilNextConsumer
	}
	ap := &attributesProcessor{
		nextConsumer: nextConsumer,
		config:       config,
	}
	return ap, nil
}

func (a *attributesProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		if rs.IsNil() {
			continue
		}
		serviceName := processor.ServiceNameForResource(rs.Resource())
		ilss := rss.At(i).InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			if ils.IsNil() {
				continue
			}
			spans := ils.Spans()
			for k := 0; k < spans.Len(); k++ {
				a.processSpan(spans.At(k), serviceName)
			}
		}
	}
	return a.nextConsumer.ConsumeTraces(ctx, td)
}

func (a *attributesProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (a *attributesProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (a *attributesProcessor) Shutdown(context.Context) error {
	return nil
}

func (a *attributesProcessor) processSpan(span pdata.Span, serviceName string) {
	if span.IsNil() {
		// Do not create empty spans just to add attributes
		return
	}

	if a.skipSpan(span, serviceName) {
		return
	}

	attrs := span.Attributes()
	for _, action := range a.config.actions {
		// TODO https://github.com/open-telemetry/opentelemetry-collector/issues/296
		// Do benchmark testing between having action be of type string vs integer.
		// The reason is attributes processor will most likely be commonly used
		// and could impact performance.
		switch action.Action {
		case DELETE:
			attrs.Delete(action.Key)
		case INSERT:
			av, found := getSourceAttributeValue(action, attrs)
			if !found {
				continue
			}
			attrs.Insert(action.Key, av)
		case UPDATE:
			av, found := getSourceAttributeValue(action, attrs)
			if !found {
				continue
			}
			attrs.Update(action.Key, av)
		case UPSERT:
			av, found := getSourceAttributeValue(action, attrs)
			if !found {
				continue
			}
			attrs.Upsert(action.Key, av)
		case HASH:
			hashAttribute(action, attrs)
		}
	}
}

func getSourceAttributeValue(action attributeAction, attrs pdata.AttributeMap) (pdata.AttributeValue, bool) {
	// Set the key with a value from the configuration.
	if action.AttributeValue != nil {
		return *action.AttributeValue, true
	}

	return attrs.Get(action.FromAttribute)
}

func hashAttribute(action attributeAction, attrs pdata.AttributeMap) {
	if value, exists := attrs.Get(action.Key); exists {
		SHA1AttributeHasher(value)
	}
}

// skipSpan determines if a span should be processed.
// True is returned when a span should be skipped.
// False is returned when a span should not be skipped.
// The logic determining if a span should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func (a *attributesProcessor) skipSpan(span pdata.Span, serviceName string) bool {
	if a.config.include != nil {
		// A false returned in this case means the span should not be processed.
		if include := a.config.include.MatchSpan(span, serviceName); !include {
			return true
		}
	}

	if a.config.exclude != nil {
		// A true returned in this case means the span should not be processed.
		if exclude := a.config.exclude.MatchSpan(span, serviceName); exclude {
			return true
		}
	}

	return false
}
