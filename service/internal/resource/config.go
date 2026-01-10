// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"go.opentelemetry.io/collector/component"
)

// Config models the resource configuration defined by the declarative configuration schema.
// Unknown keys inside the resource block (legacy inline attributes) are captured in DeprecatedAttributes.
type Config struct {
	Attributes           []Attribute       `mapstructure:"attributes"`
	AttributesList       *string           `mapstructure:"attributes_list"`
	SchemaURL            *string           `mapstructure:"schema_url"`
	Detection            *DetectionConfig  `mapstructure:"-"`
	detectionWrapper     *detectionWrapper `mapstructure:"detection"`
	DeprecatedAttributes map[string]any    `mapstructure:",remain"`
}

type detectionWrapper struct {
	Development *DetectionConfig `mapstructure:"development"`
}

func (cfg *Config) detectionConfig() *DetectionConfig {
	if cfg == nil {
		return nil
	}
	if cfg.Detection != nil {
		return cfg.Detection
	}
	if cfg.detectionWrapper != nil {
		return cfg.detectionWrapper.Development
	}
	return nil
}

// Attribute represents a schema-compliant resource attribute entry.
type Attribute struct {
	Name  string `mapstructure:"name"`
	Value any    `mapstructure:"value"`
	Type  string `mapstructure:"type"`
}

// DetectionConfig mirrors the declarative schema resource detection block.
type DetectionConfig struct {
	Attributes *IncludeExclude `mapstructure:"attributes"`
	Detectors  []any           `mapstructure:"detectors"`
}

// IncludeExclude configures attribute allow/deny lists using wildcard patterns.
type IncludeExclude struct {
	Included []string `mapstructure:"included"`
	Excluded []string `mapstructure:"excluded"`
}

// New builds a resource for collector telemetry.
// Detectors run first (optionally filtered), followed by configured attributes, and finally default service.* attributes.
func New(ctx context.Context, buildInfo component.BuildInfo, cfg *Config) (*sdkresource.Resource, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	attrSet := newAttributeSet()

	if cfg.AttributesList != nil {
		list := strings.TrimSpace(*cfg.AttributesList)
		if list != "" {
			listValues, err := parseAttributesList(list)
			if err != nil {
				return nil, err
			}
			attrSet.merge(listValues)
		}
	}

	if len(cfg.DeprecatedAttributes) > 0 {
		legacy, err := convertLegacyAttributes(cfg.DeprecatedAttributes)
		if err != nil {
			return nil, err
		}
		attrSet.merge(legacy)
	}

	if len(cfg.Attributes) > 0 {
		custom, err := convertAttributeEntries(cfg.Attributes)
		if err != nil {
			return nil, err
		}
		attrSet.merge(custom)
	}

	attrSet.ensureDefault(string(semconv.ServiceNameKey), attribute.StringValue(buildInfo.Command))
	attrSet.ensureDefault(string(semconv.ServiceVersionKey), attribute.StringValue(buildInfo.Version))
	attrSet.ensureDefault(string(semconv.ServiceInstanceIDKey), attribute.StringValue(newInstanceID()))

	schemaURL := semconv.SchemaURL
	if cfg.SchemaURL != nil {
		schemaURL = *cfg.SchemaURL
	}

	opts := []sdkresource.Option{
		sdkresource.WithSchemaURL(schemaURL),
	}

	if detCfg := cfg.detectionConfig(); detCfg != nil {
		detectorNames, err := detCfg.detectorNames()
		if err != nil {
			return nil, err
		}
		if len(detectorNames) > 0 {
			detectors, err := GetDetectors(ctx, detectorNames)
			if err != nil {
				return nil, err
			}
			filter, err := newAttributeFilter(detCfg.Attributes)
			if err != nil {
				return nil, err
			}
			if filter != nil {
				detectors = wrapDetectorsWithFilter(detectors, filter)
			}
			opts = append(opts, sdkresource.WithDetectors(detectors...))
		}
	}

	if kvs := attrSet.keyValues(); len(kvs) > 0 {
		opts = append(opts, sdkresource.WithAttributes(kvs...))
	}

	return sdkresource.New(ctx, opts...)
}

func newInstanceID() string {
	instanceUUID, _ := uuid.NewRandom()
	return instanceUUID.String()
}

func parseAttributesList(raw string) (map[string]*attributeValue, error) {
	pairs := strings.Split(raw, ",")
	result := make(map[string]*attributeValue, len(pairs))
	var invalid []string
	for _, pair := range pairs {
		entry := strings.TrimSpace(pair)
		if entry == "" {
			continue
		}
		key, value, found := strings.Cut(entry, "=")
		if !found {
			invalid = append(invalid, entry)
			continue
		}
		attrKey := strings.TrimSpace(key)
		if attrKey == "" {
			invalid = append(invalid, entry)
			continue
		}
		val := strings.TrimSpace(value)
		unescaped, err := url.PathUnescape(val)
		if err != nil {
			otel.Handle(err)
			unescaped = val
		}
		result[attrKey] = &attributeValue{value: attribute.StringValue(unescaped)}
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("resource attributes_list entries must be key=value pairs: %v", invalid)
	}
	return result, nil
}

func convertLegacyAttributes(values map[string]any) (map[string]*attributeValue, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make(map[string]*attributeValue, len(values))
	for key, raw := range values {
		switch val := raw.(type) {
		case nil:
			out[key] = nil
		case string:
			out[key] = &attributeValue{value: attribute.StringValue(val)}
		default:
			return nil, fmt.Errorf("legacy resource attribute %q must be string or null, got %T", key, raw)
		}
	}
	return out, nil
}

func convertAttributeEntries(attrs []Attribute) (map[string]*attributeValue, error) {
	out := make(map[string]*attributeValue, len(attrs))
	for idx, attr := range attrs {
		if attr.Name == "" {
			return nil, fmt.Errorf("resource.attributes[%d] is missing name", idx)
		}
		val, err := convertAttributeValue(attr)
		if err != nil {
			return nil, err
		}
		out[attr.Name] = val
	}
	return out, nil
}

func convertAttributeValue(attr Attribute) (*attributeValue, error) {
	if attr.Value == nil {
		return nil, nil
	}

	attrType := strings.ToLower(strings.TrimSpace(attr.Type))
	if attrType == "" {
		attrType = inferAttributeType(attr.Value)
	}

	switch attrType {
	case "", "string":
		s, err := toString(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.StringValue(s)}, nil
	case "bool":
		b, err := toBool(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.BoolValue(b)}, nil
	case "int":
		i, err := toInt64(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.Int64Value(i)}, nil
	case "double":
		f, err := toFloat64(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.Float64Value(f)}, nil
	case "string_array":
		arr, err := toStringSlice(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.StringSliceValue(arr)}, nil
	case "bool_array":
		arr, err := toBoolSlice(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.BoolSliceValue(arr)}, nil
	case "int_array":
		arr, err := toInt64Slice(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.Int64SliceValue(arr)}, nil
	case "double_array":
		arr, err := toFloat64Slice(attr.Value)
		if err != nil {
			return nil, fmt.Errorf("resource attribute %q: %w", attr.Name, err)
		}
		return &attributeValue{value: attribute.Float64SliceValue(arr)}, nil
	default:
		return nil, fmt.Errorf("resource attribute %q: unsupported type %q", attr.Name, attrType)
	}
}

func inferAttributeType(val any) string {
	switch v := val.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "double"
	case []string:
		return "string_array"
	case []bool:
		return "bool_array"
	case []int, []int32, []int64:
		return "int_array"
	case []float32, []float64:
		return "double_array"
	case []any:
		return inferSliceType(v)
	default:
		return ""
	}
}

func inferSliceType(values []any) string {
	for _, item := range values {
		switch item.(type) {
		case string:
			return "string_array"
		case bool:
			return "bool_array"
		case int, int32, int64:
			return "int_array"
		case float32, float64:
			return "double_array"
		default:
			continue
		}
	}
	return ""
}

func toString(v any) (string, error) {
	str, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}
	return str, nil
}

func toBool(v any) (bool, error) {
	val, ok := v.(bool)
	if ok {
		return val, nil
	}
	return false, fmt.Errorf("expected bool, got %T", v)
}

func toInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		if math.Trunc(val) != val {
			return 0, fmt.Errorf("expected int, got float %v", val)
		}
		return int64(val), nil
	case float32:
		if math.Trunc(float64(val)) != float64(val) {
			return 0, fmt.Errorf("expected int, got float %v", val)
		}
		return int64(val), nil
	default:
		return 0, fmt.Errorf("expected int, got %T", v)
	}
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

func toStringSlice(v any) ([]string, error) {
	switch val := v.(type) {
	case []string:
		return val, nil
	case []any:
		out := make([]string, len(val))
		for i, item := range val {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string array, element %d is %T", i, item)
			}
			out[i] = str
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected string array, got %T", v)
	}
}

func toBoolSlice(v any) ([]bool, error) {
	switch val := v.(type) {
	case []bool:
		return val, nil
	case []any:
		out := make([]bool, len(val))
		for i, item := range val {
			b, ok := item.(bool)
			if !ok {
				return nil, fmt.Errorf("expected bool array, element %d is %T", i, item)
			}
			out[i] = b
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected bool array, got %T", v)
	}
}

func toInt64Slice(v any) ([]int64, error) {
	switch val := v.(type) {
	case []int:
		out := make([]int64, len(val))
		for i, item := range val {
			out[i] = int64(item)
		}
		return out, nil
	case []int32:
		out := make([]int64, len(val))
		for i, item := range val {
			out[i] = int64(item)
		}
		return out, nil
	case []int64:
		return val, nil
	case []any:
		out := make([]int64, len(val))
		for i, item := range val {
			casted, err := toInt64(item)
			if err != nil {
				return nil, fmt.Errorf("expected int array, element %d: %w", i, err)
			}
			out[i] = casted
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected int array, got %T", v)
	}
}

func toFloat64Slice(v any) ([]float64, error) {
	switch val := v.(type) {
	case []float64:
		return val, nil
	case []float32:
		out := make([]float64, len(val))
		for i, item := range val {
			out[i] = float64(item)
		}
		return out, nil
	case []int:
		out := make([]float64, len(val))
		for i, item := range val {
			out[i] = float64(item)
		}
		return out, nil
	case []any:
		out := make([]float64, len(val))
		for i, item := range val {
			casted, err := toFloat64(item)
			if err != nil {
				return nil, fmt.Errorf("expected double array, element %d: %w", i, err)
			}
			out[i] = casted
		}
		return out, nil
	default:
		return nil, fmt.Errorf("expected double array, got %T", v)
	}
}

type attributeValue struct {
	value attribute.Value
}

type attributeSet struct {
	values map[string]*attributeValue
}

func newAttributeSet() *attributeSet {
	return &attributeSet{values: make(map[string]*attributeValue)}
}

func (s *attributeSet) merge(values map[string]*attributeValue) {
	if len(values) == 0 {
		return
	}
	for key, val := range values {
		if val == nil {
			s.values[key] = nil
			continue
		}
		copyVal := *val
		s.values[key] = &copyVal
	}
}

func (s *attributeSet) ensureDefault(key string, val attribute.Value) {
	if _, exists := s.values[key]; exists {
		return
	}
	s.values[key] = &attributeValue{value: val}
}

func (s *attributeSet) keyValues() []attribute.KeyValue {
	keys := make([]string, 0, len(s.values))
	for key, val := range s.values {
		if val != nil {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	kvs := make([]attribute.KeyValue, 0, len(keys))
	for _, key := range keys {
		kvs = append(kvs, attribute.KeyValue{
			Key:   attribute.Key(key),
			Value: s.values[key].value,
		})
	}
	return kvs
}

func (cfg *DetectionConfig) detectorNames() ([]string, error) {
	if cfg == nil || len(cfg.Detectors) == 0 {
		return nil, nil
	}
	names := make([]string, 0, len(cfg.Detectors))
	for idx, entry := range cfg.Detectors {
		switch val := entry.(type) {
		case string:
			if strings.TrimSpace(val) == "" {
				return nil, fmt.Errorf("resource detection entry %d must specify a detector name", idx)
			}
			names = append(names, val)
		case map[string]any:
			if len(val) != 1 {
				return nil, fmt.Errorf("resource detection entry %d must contain exactly one detector", idx)
			}
			for key := range val {
				if strings.TrimSpace(key) == "" {
					return nil, fmt.Errorf("resource detection entry %d must specify a detector name", idx)
				}
				names = append(names, key)
			}
		case nil:
			return nil, fmt.Errorf("resource detection entry %d cannot be null", idx)
		default:
			return nil, fmt.Errorf("resource detection entry %d must be a detector name or object, got %T", idx, entry)
		}
	}
	return names, nil
}

func wrapDetectorsWithFilter(detectors []sdkresource.Detector, filter *attributeFilter) []sdkresource.Detector {
	if filter == nil || len(detectors) == 0 {
		return detectors
	}
	out := make([]sdkresource.Detector, len(detectors))
	for i, det := range detectors {
		out[i] = filteredDetector{
			delegate: det,
			filter:   filter,
		}
	}
	return out
}

type filteredDetector struct {
	delegate sdkresource.Detector
	filter   *attributeFilter
}

func (fd filteredDetector) Detect(ctx context.Context) (*sdkresource.Resource, error) {
	res, err := fd.delegate.Detect(ctx)
	if err != nil || fd.filter == nil || res == nil {
		return res, err
	}
	return filterResourceAttributes(res, fd.filter), err
}

func filterResourceAttributes(res *sdkresource.Resource, filter *attributeFilter) *sdkresource.Resource {
	if res == nil || filter == nil {
		return res
	}
	attrs := res.Attributes()
	filtered := make([]attribute.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
		if filter.allowed(string(kv.Key)) {
			filtered = append(filtered, kv)
		}
	}
	if len(filtered) == len(attrs) {
		return res
	}
	return sdkresource.NewWithAttributes(res.SchemaURL(), filtered...)
}

type attributeFilter struct {
	includes []compiledPattern
	excludes []compiledPattern
}

func newAttributeFilter(cfg *IncludeExclude) (*attributeFilter, error) {
	if cfg == nil {
		return nil, nil
	}
	filter := &attributeFilter{}
	for _, pattern := range cfg.Included {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		cp, err := compilePattern(pattern)
		if err != nil {
			return nil, err
		}
		filter.includes = append(filter.includes, cp)
	}
	for _, pattern := range cfg.Excluded {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		cp, err := compilePattern(pattern)
		if err != nil {
			return nil, err
		}
		filter.excludes = append(filter.excludes, cp)
	}
	if len(filter.includes) == 0 && len(filter.excludes) == 0 {
		return nil, nil
	}
	return filter, nil
}

func (f *attributeFilter) allowed(name string) bool {
	if f == nil {
		return true
	}
	if len(f.includes) > 0 {
		match := false
		for _, p := range f.includes {
			if p.matches(name) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	for _, p := range f.excludes {
		if p.matches(name) {
			return false
		}
	}
	return true
}

type compiledPattern struct {
	raw string
	re  *regexp.Regexp
}

func (p compiledPattern) matches(value string) bool {
	return p.re.MatchString(value)
}

func compilePattern(pattern string) (compiledPattern, error) {
	regex := globToRegex(pattern)
	re, err := regexp.Compile(regex)
	if err != nil {
		return compiledPattern{}, fmt.Errorf("invalid attribute filter pattern %q: %w", pattern, err)
	}
	return compiledPattern{raw: pattern, re: re}, nil
}

func globToRegex(pattern string) string {
	var builder strings.Builder
	builder.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			builder.WriteString(".*")
		case '?':
			builder.WriteByte('.')
		default:
			builder.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	builder.WriteString("$")
	return builder.String()
}
