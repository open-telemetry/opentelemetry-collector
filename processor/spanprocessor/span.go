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

package spanprocessor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/processor/span"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
	"github.com/open-telemetry/opentelemetry-collector/processor"
)

type spanProcessor struct {
	nextConsumer     consumer.TraceConsumerOld
	config           Config
	toAttributeRules []toAttributeRule
	include          span.Matcher
	exclude          span.Matcher
}

// toAttributeRule is the compiled equivalent of config.ToAttributes field.
type toAttributeRule struct {
	// Compiled regexp.
	re *regexp.Regexp

	// Attribute names extracted from the regexp's subexpressions.
	attrNames []string
}

// newSpanProcessor returns the span processor.
func newSpanProcessor(nextConsumer consumer.TraceConsumerOld, config Config) (*spanProcessor, error) {
	if nextConsumer == nil {
		return nil, oterr.ErrNilNextConsumer
	}

	include, err := span.NewMatcher(config.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := span.NewMatcher(config.Exclude)
	if err != nil {
		return nil, err
	}

	sp := &spanProcessor{
		nextConsumer: nextConsumer,
		config:       config,
		include:      include,
		exclude:      exclude,
	}

	// Compile ToAttributes regexp and extract attributes names.
	if config.Rename.ToAttributes != nil {
		for _, pattern := range config.Rename.ToAttributes.Rules {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regexp pattern %s", pattern)
			}

			rule := toAttributeRule{
				re: re,
				// Subexpression names will become attribute names during extraction.
				attrNames: re.SubexpNames(),
			}

			sp.toAttributeRules = append(sp.toAttributeRules, rule)
		}
	}

	return sp, nil
}

func (sp *spanProcessor) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	serviceName := processor.ServiceNameForNode(td.Node)
	for _, span := range td.Spans {
		if span == nil {
			continue
		}

		if sp.skipSpan(span, serviceName) {
			continue
		}
		sp.processFromAttributes(span)
		sp.processToAttributes(span)
	}
	return sp.nextConsumer.ConsumeTraceData(ctx, td)
}

func (sp *spanProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

// Start is invoked during service startup.
func (sp *spanProcessor) Start(host component.Host) error {
	return nil
}

// Shutdown is invoked during service shutdown.
func (sp *spanProcessor) Shutdown() error {
	return nil
}

func (sp *spanProcessor) processFromAttributes(span *tracepb.Span) {
	if len(sp.config.Rename.FromAttributes) == 0 {
		// There is FromAttributes rule.
		return
	}

	if span.Attributes == nil || len(span.Attributes.AttributeMap) == 0 {
		// There are no attributes to create span name from.
		return
	}

	// Note: There was a separate proposal for creating the string.
	// With benchmarking, strings.Builder is faster than the proposal.
	// For full context, refer to this PR comment:
	// https://github.com/open-telemetry/opentelemetry-collector/pull/301#discussion_r318357678
	var sb strings.Builder
	for i, key := range sp.config.Rename.FromAttributes {
		attribute, found := span.Attributes.AttributeMap[key]

		// If one of the keys isn't found, the span name is not updated.
		if !found {
			return
		}

		// Note: WriteString() always return a nil error so there is no error checking
		// for this method call.
		// https://golang.org/src/strings/builder.go?s=3425:3477#L110

		// Include the separator before appending an attribute value if:
		// this isn't the first value(ie i == 0) loop through the FromAttributes
		// and
		// the separator isn't an empty string.
		if i > 0 && sp.config.Rename.Separator != "" {
			sb.WriteString(sp.config.Rename.Separator)
		}

		// Ideally with proto converting to the internal format for attributes
		// there shouldn't be any map entries with a nil value. However,
		// if there is a bad translation, this might be possible.
		if attribute == nil {
			sb.WriteString("<nil-attribute-value>")
			continue
		}

		switch value := attribute.Value.(type) {
		case *tracepb.AttributeValue_StringValue:
			sb.WriteString(value.StringValue.GetValue())
		case *tracepb.AttributeValue_BoolValue:
			sb.WriteString(strconv.FormatBool(value.BoolValue))
		case *tracepb.AttributeValue_DoubleValue:
			sb.WriteString(strconv.FormatFloat(value.DoubleValue, 'f', -1, 64))
		case *tracepb.AttributeValue_IntValue:
			sb.WriteString(strconv.FormatInt(value.IntValue, 10))
		default:
			sb.WriteString("<unknown-attribute-type>")
		}
	}
	span.Name = &tracepb.TruncatableString{Value: sb.String()}
}

func (sp *spanProcessor) processToAttributes(span *tracepb.Span) {
	if span.Name == nil || span.Name.Value == "" {
		// There is no span name to work on.
		return
	}

	if sp.config.Rename.ToAttributes == nil {
		// No rules to apply.
		return
	}

	// Process rules one by one. Store results of processing in the span
	// so that each subsequent rule works on the span name that is the output
	// after processing the previous rule.
	for _, rule := range sp.toAttributeRules {
		re := rule.re
		oldName := span.Name.Value

		// Match the regular expression and extract matched subexpressions.
		submatches := re.FindStringSubmatch(oldName)
		if submatches == nil {
			continue
		}
		// There is a match. We will also need positions of subexpression matches.
		submatchIdxPairs := re.FindStringSubmatchIndex(oldName)

		// A place to accumulate new span name.
		var sb strings.Builder

		// Index in the oldName until which we traversed.
		var oldNameIndex = 0

		// Ensure we have a place to create attributes.
		if span.Attributes == nil {
			span.Attributes = &tracepb.Span_Attributes{}
		}
		attrs := span.Attributes

		// Create a new map if one does not exist and size it to the number of
		// attributes that we plan to extract.
		if attrs.AttributeMap == nil {
			attrs.AttributeMap = make(map[string]*tracepb.AttributeValue, len(rule.attrNames))
		}

		// Start from index 1, which is the first submatch (index 0 is the entire match).
		// We will go over submatches and will simultaneously build a new span name,
		// replacing matched subexpressions by attribute names.
		for i := 1; i < len(submatches); i++ {
			// Create a string attribute from extracted submatch.
			attrValue := &tracepb.AttributeValue{Value: &tracepb.AttributeValue_StringValue{
				StringValue: &tracepb.TruncatableString{Value: submatches[i]},
			}}
			attrs.AttributeMap[rule.attrNames[i]] = attrValue

			// Add part of span name from end of previous match to start of this match
			// and then add attribute name wrapped in curly brackets.
			matchStartIndex := submatchIdxPairs[i*2] // start of i'th submatch.
			sb.WriteString(oldName[oldNameIndex:matchStartIndex] + "{" + rule.attrNames[i] + "}")

			// Advance the index to the end of current match.
			oldNameIndex = submatchIdxPairs[i*2+1] // end of i'th submatch.
		}
		if oldNameIndex < len(oldName) {
			// Append the remainder, from the end of last match until end of span name.
			sb.WriteString(oldName[oldNameIndex:])
		}

		// Set new span name.
		span.Name = &tracepb.TruncatableString{Value: sb.String()}

		if sp.config.Rename.ToAttributes.BreakAfterMatch {
			// Stop processing, break after first match is requested.
			break
		}
	}
}

// skipSpan determines if a span should be processed.
// True is returned when a span should be skipped.
// False is returned when a span should not be skipped.
// The logic determining if a span should be processed is set
// in the attribute configuration with the include and exclude settings.
// Include properties are checked before exclude settings are checked.
func (sp *spanProcessor) skipSpan(span *tracepb.Span, serviceName string) bool {
	if sp.include != nil {
		// A false returned in this case means the span should not be processed.
		if include := sp.include.MatchSpan(span, serviceName); !include {
			return true
		}
	}

	if sp.exclude != nil {
		// A true returned in this case means the span should not be processed.
		if exclude := sp.exclude.MatchSpan(span, serviceName); exclude {
			return true
		}
	}

	return false
}
