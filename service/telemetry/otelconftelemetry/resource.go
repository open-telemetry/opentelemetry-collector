// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	xotelconf "go.opentelemetry.io/contrib/otelconf/x"
	otelsdkresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/service/telemetry"
)

var errMissingCollectorResource = errors.New("collector resource must be initialized before creating telemetry providers")

const (
	otelConfigFileEnv             = "OTEL_CONFIG_FILE"
	otelExperimentalConfigFileEnv = "OTEL_EXPERIMENTAL_CONFIG_FILE"
	internalSchemaURLAttributeKey = "otelcol.internal.resource.schema_url"
)

// defaultAttributeValues is a variable so tests can stub the generated defaults.
var defaultAttributeValues = func(buildInfo component.BuildInfo) (map[string]string, error) {
	instanceUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	return map[string]string{
		string(semconv.ServiceNameKey):       buildInfo.Command,
		string(semconv.ServiceVersionKey):    buildInfo.Version,
		string(semconv.ServiceInstanceIDKey): instanceUUID.String(),
	}, nil
}

var (
	newExperimentalSDK   = xotelconf.NewSDK
	experimentalSDKEnvMu sync.Mutex
)

var putAttributeValue = func(attrs pcommon.Map, key string, value any) error {
	return attrs.PutEmpty(key).FromRaw(value)
}

func createInitialResourceConfig(_ context.Context, _ component.BuildInfo, cfg *ResourceConfig) (*xotelconf.Resource, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &xotelconf.Resource{
		DetectionDevelopment: cfg.DetectionDevelopment,
	}, nil
}

func includeDetectedAttribute(key string, filters *xotelconf.IncludeExclude) bool {
	if filters == nil {
		return true
	}
	if len(filters.Included) > 0 && !matchesAnyPattern(key, filters.Included) {
		return false
	}
	if len(filters.Excluded) > 0 && matchesAnyPattern(key, filters.Excluded) {
		return false
	}
	return true
}

func matchesAnyPattern(key string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := path.Match(pattern, key)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func createFixedResourceConfig(cfg *ResourceConfig, res *pcommon.Resource) (*otelconf.Resource, error) {
	if res == nil {
		return nil, errMissingCollectorResource
	}

	providerConfig := &otelconf.Resource{
		Attributes: make([]otelconf.AttributeNameValue, 0, res.Attributes().Len()),
	}
	schemaURLSource := cfg.SchemaUrl

	res.Attributes().Range(func(key string, value pcommon.Value) bool {
		if key == internalSchemaURLAttributeKey {
			schemaURL := value.Str()
			schemaURLSource = &schemaURL
			return true
		}
		providerConfig.Attributes = append(providerConfig.Attributes, otelconf.AttributeNameValue{
			Name:  key,
			Value: value.AsRaw(),
		})
		return true
	})
	if schemaURLSource != nil {
		// Preserve the configured schema URL separately because it is not exposed by pcommon.Resource.
		schemaURL := *schemaURLSource
		providerConfig.SchemaUrl = &schemaURL
	}

	return providerConfig, nil
}

func createResource(
	ctx context.Context,
	set telemetry.Settings,
	componentConfig component.Config,
) (pcommon.Resource, error) {
	cfg := &componentConfig.(*Config).Resource
	sdkCfg, err := createInitialResourceConfig(ctx, set.BuildInfo, &componentConfig.(*Config).Resource)
	if err != nil {
		return pcommon.Resource{}, err
	}
	sdk, effectiveResourceCfg, err := newResourceSDK(ctx, sdkCfg)
	if err != nil {
		return pcommon.Resource{}, err
	}

	defaults, err := defaultAttributeValues(set.BuildInfo)
	if err != nil {
		return pcommon.Resource{}, err
	}
	envResource := otelsdkresource.Environment()
	suppressed := suppressedLegacyAttributes(cfg.LegacyAttributes)
	configuredAttributes, err := configuredResourceAttributes(effectiveResourceCfg)
	if err != nil {
		return pcommon.Resource{}, err
	}
	configuredAttributeListValues, err := resourceAttributeListValues(effectiveResourceCfg)
	if err != nil {
		return pcommon.Resource{}, err
	}
	explicitConfiguredAttributes := configuredExplicitResourceAttributes(effectiveResourceCfg)

	sdkResource := sdk.Resource()
	pcommonResource := pcommon.NewResource()
	pcommonAttributes := pcommonResource.Attributes()

	pcommonAttributes.EnsureCapacity(len(defaults) + len(cfg.LegacyAttributes) + len(cfg.Attributes) + sdkResource.Len() + envResource.Len() + len(configuredAttributeListValues))

	for name, value := range defaults {
		if _, ok := suppressed[name]; ok && !configuredAttributes[name] {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, name, normalizeAttributeValue(value)); err != nil {
			return pcommon.Resource{}, err
		}
	}

	envIterator := envResource.Iter()
	for envIterator.Next() {
		kv := envIterator.Attribute()
		if _, ok := suppressed[string(kv.Key)]; ok && !configuredAttributes[string(kv.Key)] {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, string(kv.Key), kv.Value.AsInterface()); err != nil {
			return pcommon.Resource{}, err
		}
	}

	for key, value := range configuredAttributeListValues {
		if _, ok := suppressed[key]; ok && !configuredAttributes[key] {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, key, value); err != nil {
			return pcommon.Resource{}, err
		}
	}

	sdkIterator := sdkResource.Iter()

	for sdkIterator.Next() {
		kv := sdkIterator.Attribute()
		key := string(kv.Key)
		if effectiveResourceCfg != nil &&
			effectiveResourceCfg.DetectionDevelopment != nil &&
			!configuredAttributes[key] &&
			!includeDetectedAttribute(key, effectiveResourceCfg.DetectionDevelopment.Attributes) {
			continue
		}
		if _, ok := suppressed[key]; ok && !configuredAttributes[key] {
			continue
		}
		if _, exists := pcommonAttributes.Get(key); exists && !explicitConfiguredAttributes[key] {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, key, kv.Value.AsInterface()); err != nil {
			return pcommon.Resource{}, err
		}
	}

	for key, value := range cfg.LegacyAttributes {
		if configuredAttributes[key] {
			continue
		}
		if value == nil {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, key, normalizeAttributeValue(value)); err != nil {
			return pcommon.Resource{}, err
		}
	}

	for _, attr := range cfg.Attributes {
		if configuredAttributes[attr.Name] {
			continue
		}
		if err := putAttributeValue(pcommonAttributes, attr.Name, normalizeAttributeValue(attr.Value)); err != nil {
			return pcommon.Resource{}, err
		}
	}

	if effectiveResourceCfg != nil &&
		effectiveResourceCfg.SchemaUrl != nil &&
		(cfg.SchemaUrl == nil || *cfg.SchemaUrl != *effectiveResourceCfg.SchemaUrl) {
		pcommonAttributes.PutStr(internalSchemaURLAttributeKey, *effectiveResourceCfg.SchemaUrl)
	}

	return pcommonResource, nil
}

func configuredExplicitResourceAttributes(resourceCfg *xotelconf.Resource) map[string]bool {
	if resourceCfg == nil {
		return nil
	}

	configured := make(map[string]bool, len(resourceCfg.Attributes))
	for _, attr := range resourceCfg.Attributes {
		configured[attr.Name] = true
	}
	return configured
}

func newResourceSDK(ctx context.Context, resourceCfg *xotelconf.Resource) (xotelconf.SDK, *xotelconf.Resource, error) {
	if filename, ok := os.LookupEnv(otelExperimentalConfigFileEnv); ok && filename != "" {
		cfg, err := parseOpenTelemetryConfigFile(filename)
		if err != nil {
			return xotelconf.SDK{}, nil, err
		}
		cfg.Resource = mergeResourceOverride(cfg.Resource, resourceCfg)

		sdk, err := callExperimentalSDK(
			xotelconf.WithContext(ctx),
			xotelconf.WithOpenTelemetryConfiguration(*cfg),
		)
		return sdk, cfg.Resource, err
	}

	filename, ok := os.LookupEnv(otelConfigFileEnv)
	if !ok || filename == "" {
		sdk, err := callExperimentalSDK(
			xotelconf.WithContext(ctx),
			xotelconf.WithOpenTelemetryConfiguration(xotelconf.OpenTelemetryConfiguration{Resource: resourceCfg}),
		)
		return sdk, resourceCfg, err
	}

	cfg, err := parseOpenTelemetryConfigFile(filename)
	if err != nil {
		return xotelconf.SDK{}, nil, err
	}

	cfg.Resource = mergeResourceOverride(cfg.Resource, resourceCfg)

	sdk, err := callExperimentalSDK(
		xotelconf.WithContext(ctx),
		xotelconf.WithOpenTelemetryConfiguration(*cfg),
	)
	return sdk, cfg.Resource, err
}

func callExperimentalSDK(opts ...xotelconf.ConfigurationOption) (xotelconf.SDK, error) {
	filename, ok := os.LookupEnv(otelExperimentalConfigFileEnv)
	if !ok {
		return newExperimentalSDK(opts...)
	}

	experimentalSDKEnvMu.Lock()
	defer experimentalSDKEnvMu.Unlock()

	if err := os.Unsetenv(otelExperimentalConfigFileEnv); err != nil {
		return xotelconf.SDK{}, err
	}
	defer func() {
		_ = os.Setenv(otelExperimentalConfigFileEnv, filename)
	}()

	return newExperimentalSDK(opts...)
}

func parseOpenTelemetryConfigFile(filename string) (*xotelconf.OpenTelemetryConfiguration, error) {
	// #nosec G304 -- filename is a user-provided configuration file path
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return xotelconf.ParseYAML(b)
}

func configuredResourceAttributes(resourceCfg *xotelconf.Resource) (map[string]bool, error) {
	if resourceCfg == nil {
		return nil, nil
	}

	configured := make(map[string]bool, len(resourceCfg.Attributes))
	for _, attr := range resourceCfg.Attributes {
		configured[attr.Name] = true
	}
	keys, err := resourceAttributeListKeys(resourceCfg.AttributesList)
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		configured[key] = true
	}
	return configured, nil
}

func resourceAttributeListKeys(attributesList *string) ([]string, error) {
	values, err := resourceAttributeListValuesFromString(attributesList)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys, nil
}

func resourceAttributeListValues(resourceCfg *xotelconf.Resource) (map[string]string, error) {
	if resourceCfg == nil {
		return nil, nil
	}
	return resourceAttributeListValuesFromString(resourceCfg.AttributesList)
}

func resourceAttributeListValuesFromString(attributesList *string) (map[string]string, error) {
	if attributesList == nil || *attributesList == "" {
		return nil, nil
	}

	pairs := strings.Split(*attributesList, ",")
	values := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		if !found {
			return nil, fmt.Errorf("invalid resource.attributes_list entry %q: missing '='", strings.TrimSpace(pair))
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		if decoded, err := url.PathUnescape(value); err == nil {
			value = decoded
		}
		values[key] = value
	}
	return values, nil
}

func mergeResourceOverride(base, overlay *xotelconf.Resource) *xotelconf.Resource {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}
	if overlay.DetectionDevelopment != nil {
		base.DetectionDevelopment = overlay.DetectionDevelopment
	}
	return base
}

func suppressedLegacyAttributes(legacy map[string]any) map[string]struct{} {
	suppressed := make(map[string]struct{}, len(legacy))
	for name := range legacy {
		suppressed[name] = struct{}{}
	}
	return suppressed
}

func normalizeAttributeValue(v any) any {
	switch val := v.(type) {
	case bool, int64, float64, string:
		return val
	case uint64:
		return strconv.FormatUint(val, 10)
	case int8:
		return int64(val)
	case uint8:
		return int64(val)
	case int16:
		return int64(val)
	case uint16:
		return int64(val)
	case int32:
		return int64(val)
	case uint32:
		return int64(val)
	case float32:
		return float64(val)
	case int:
		return int64(val)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	default:
		return fmt.Sprint(v)
	}
}
