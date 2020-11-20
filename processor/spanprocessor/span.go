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

package spanprocessor

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
)

type spanProcessor struct {
	config           Config
	toAttributeRules []toAttributeRule
	include          filterspan.Matcher
	exclude          filterspan.Matcher
}

// toAttributeRule is the compiled equivalent of config.ToAttributes field.
type toAttributeRule struct {
	// Compiled regexp.
	re *regexp.Regexp

	// Attribute names extracted from the regexp's subexpressions.
	attrNames []string
}

// newSpanProcessor returns the span processor.
func newSpanProcessor(config Config) (*spanProcessor, error) {
	include, err := filterspan.NewMatcher(config.Include)
	if err != nil {
		return nil, err
	}
	exclude, err := filterspan.NewMatcher(config.Exclude)
	if err != nil {
		return nil, err
	}

	sp := &spanProcessor{
		config:  config,
		include: include,
		exclude: exclude,
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

func (sp *spanProcessor) ProcessTraces(_ context.Context, td pdata.Traces) (pdata.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		ilss := rs.InstrumentationLibrarySpans()
		resource := rs.Resource()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			spans := ils.Spans()
			library := ils.InstrumentationLibrary()
			for k := 0; k < spans.Len(); k++ {
				s := spans.At(k)
				if filterspan.SkipSpan(sp.include, sp.exclude, s, resource, library) {
					continue
				}
				sp.processFromAttributes(s)
				sp.processToAttributes(s)
			}
		}
	}
	return td, nil
}

func (sp *spanProcessor) processFromAttributes(span pdata.Span) {
	if len(sp.config.Rename.FromAttributes) == 0 {
		// There is FromAttributes rule.
		return
	}

	attrs := span.Attributes()
	if attrs.Len() == 0 {
		// There are no attributes to create span name from.
		return
	}

	// Note: There was a separate proposal for creating the string.
	// With benchmarking, strings.Builder is faster than the proposal.
	// For full context, refer to this PR comment:
	// https://go.opentelemetry.io/collector/pull/301#discussion_r318357678
	var sb strings.Builder
	for i, key := range sp.config.Rename.FromAttributes {
		attr, found := attrs.Get(key)

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

		switch attr.Type() {
		case pdata.AttributeValueSTRING:
			sb.WriteString(attr.StringVal())
		case pdata.AttributeValueBOOL:
			sb.WriteString(strconv.FormatBool(attr.BoolVal()))
		case pdata.AttributeValueDOUBLE:
			sb.WriteString(strconv.FormatFloat(attr.DoubleVal(), 'f', -1, 64))
		case pdata.AttributeValueINT:
			sb.WriteString(strconv.FormatInt(attr.IntVal(), 10))
		default:
			sb.WriteString("<unknown-attribute-type>")
		}
	}
	span.SetName(sb.String())
}

func (sp *spanProcessor) processToAttributes(span pdata.Span) {
	if span.Name() == "" {
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
		oldName := span.Name()

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

		attrs := span.Attributes()

		// TODO: Pre-allocate len(submatches) space in the attributes.

		// Start from index 1, which is the first submatch (index 0 is the entire match).
		// We will go over submatches and will simultaneously build a new span name,
		// replacing matched subexpressions by attribute names.
		for i := 1; i < len(submatches); i++ {
			attrs.UpsertString(rule.attrNames[i], submatches[i])

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
		span.SetName(sb.String())

		if sp.config.Rename.ToAttributes.BreakAfterMatch {
			// Stop processing, break after first match is requested.
			break
		}
	}
}
