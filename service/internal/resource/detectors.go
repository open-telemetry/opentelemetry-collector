// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.opentelemetry.io/otel/sdk/resource"
)

// DetectorFactory creates a resource.Detector based on the detector name.
type DetectorFactory func(context.Context) (resource.Detector, error)

// The declarative configuration schema currently documents container/host/process/service detectors:
// https://github.com/open-telemetry/opentelemetry-configuration/blob/ddc8c8f47085eb5637a4b1e084b65e384521f8b9/schema/resource.yaml#L74-L141
// The collector extends that list with additional Go SDK detectors (env and host) to preserve parity
// with the resource detection features exposed directly by the SDKs.
var detectorRegistry = map[string]DetectorFactory{
	"env":  newEnvDetector,
	"host": newHostDetector,
}

func newEnvDetector(_ context.Context) (resource.Detector, error) {
	return newSDKDetector(resource.WithFromEnv), nil
}

func newHostDetector(_ context.Context) (resource.Detector, error) {
	return newSDKDetector(resource.WithHost), nil
}

func newSDKDetector(option func() resource.Option) resource.Detector {
	return &sdkDetector{
		newResource: resource.New,
		option:      option,
	}
}

type sdkDetector struct {
	newResource func(context.Context, ...resource.Option) (*resource.Resource, error)
	option      func() resource.Option
}

// Detect returns resource information from the configured SDK detector option.
func (d *sdkDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	res, err := d.newResource(ctx, d.option())
	if err != nil {
		return nil, err
	}
	// Strip schema URL to avoid conflicts - it will be set at the resource level.
	return resource.New(ctx, resource.WithAttributes(res.Attributes()...))
}

// GetDetector returns a resource.Detector for the given detector name.
// Returns an error if the detector name is not recognized.
func GetDetector(ctx context.Context, name string) (resource.Detector, error) {
	factory, ok := detectorRegistry[name]
	if !ok {
		supportedDetectors := make([]string, 0, len(detectorRegistry))
		for detectorName := range detectorRegistry {
			supportedDetectors = append(supportedDetectors, detectorName)
		}
		sort.Strings(supportedDetectors)
		return nil, fmt.Errorf("unknown detector: %q. Supported detectors: %s", name, strings.Join(supportedDetectors, ", "))
	}
	return factory(ctx)
}

// GetDetectors returns a slice of resource.Detector instances for the given detector names.
func GetDetectors(ctx context.Context, names []string) ([]resource.Detector, error) {
	if len(names) == 0 {
		return nil, nil
	}

	detectors := make([]resource.Detector, 0, len(names))
	for _, name := range names {
		detector, err := GetDetector(ctx, name)
		if err != nil {
			return nil, err
		}
		detectors = append(detectors, detector)
	}
	return detectors, nil
}
