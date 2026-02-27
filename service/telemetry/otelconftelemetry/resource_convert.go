// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"fmt"
	"math"
	"math/bits"
	"net/url"
	"strconv"
	"strings"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

const maxInt64 = math.MaxInt64

func pcommonResourceFromConfig(resCfg config.Resource) (pcommon.Resource, error) {
	pcommonRes := pcommon.NewResource()
	attrs := pcommonRes.Attributes()
	merged, err := mergeResourceAttributes(resCfg.Attributes, resCfg.AttributesList)
	if err != nil {
		return pcommon.Resource{}, err
	}
	for _, attr := range merged {
		putPcommonAttribute(attrs, attr.Name, attr.Value)
	}
	return pcommonRes, nil
}

func mergeResourceAttributes(attrs []config.AttributeNameValue, attrsList *string) ([]config.AttributeNameValue, error) {
	listAttrs, err := parseAttributesList(attrsList)
	if err != nil {
		return nil, err
	}
	merged := make([]config.AttributeNameValue, 0, len(listAttrs)+len(attrs))
	merged = append(merged, listAttrs...)
	merged = append(merged, attrs...)
	return merged, nil
}

func parseAttributesList(attrsList *string) ([]config.AttributeNameValue, error) {
	if attrsList == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*attrsList)
	if raw == "" {
		return nil, nil
	}

	pairs := strings.Split(raw, ",")
	attrs := make([]config.AttributeNameValue, 0, len(pairs))
	var invalid []string
	for _, pair := range pairs {
		key, val, found := strings.Cut(pair, "=")
		if !found {
			invalid = append(invalid, pair)
			continue
		}
		name := strings.TrimSpace(key)
		value, err := url.PathUnescape(strings.TrimSpace(val))
		if err != nil {
			value = val
		}
		attrs = append(attrs, config.AttributeNameValue{
			Name:  name,
			Value: value,
		})
	}
	if len(invalid) > 0 {
		return attrs, fmt.Errorf("resource attributes_list has missing value for %v", invalid)
	}
	return attrs, nil
}

func putPcommonAttribute(attrs pcommon.Map, name string, value any) {
	switch val := value.(type) {
	case bool:
		attrs.PutBool(name, val)
	case int64:
		attrs.PutInt(name, val)
	case uint64:
		if val <= uint64(maxInt64) {
			attrs.PutInt(name, int64(val))
		} else {
			attrs.PutStr(name, strconv.FormatUint(val, 10))
		}
	case float64:
		attrs.PutDouble(name, val)
	case int8:
		attrs.PutInt(name, int64(val))
	case uint8:
		attrs.PutInt(name, int64(val))
	case int16:
		attrs.PutInt(name, int64(val))
	case uint16:
		attrs.PutInt(name, int64(val))
	case int32:
		attrs.PutInt(name, int64(val))
	case uint32:
		attrs.PutInt(name, int64(val))
	case float32:
		attrs.PutDouble(name, float64(val))
	case int:
		attrs.PutInt(name, int64(val))
	case uint:
		if bits.UintSize < 64 || uint64(val) <= uint64(maxInt64) {
			attrs.PutInt(name, int64(val))
		} else {
			attrs.PutStr(name, strconv.FormatUint(uint64(val), 10))
		}
	case string:
		attrs.PutStr(name, val)
	default:
		attrs.PutStr(name, fmt.Sprint(value))
	}
}

func pcommonValueToAttribute(k string, v pcommon.Value) attribute.KeyValue {
	switch v.Type() {
	case pcommon.ValueTypeBool:
		return attribute.Bool(k, v.Bool())
	case pcommon.ValueTypeInt:
		return attribute.Int64(k, v.Int())
	case pcommon.ValueTypeDouble:
		return attribute.Float64(k, v.Double())
	case pcommon.ValueTypeStr:
		return attribute.String(k, v.Str())
	case pcommon.ValueTypeBytes:
		return attribute.String(k, string(v.Bytes().AsRaw()))
	default:
		return attribute.String(k, fmt.Sprint(v.AsRaw()))
	}
}
