// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package resource // import "go.opentelemetry.io/collector/service/internal/resource"

import (
	"context"
	"fmt"
	"sort"
	"strings"

	ec2detector "go.opentelemetry.io/contrib/detectors/aws/ec2/v2"
	"go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/contrib/detectors/aws/eks"
	"go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/detectors/azure/azurevm"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel/sdk/resource"
)

// DetectorFactory creates a resource.Detector based on the detector name.
type DetectorFactory func(context.Context) (resource.Detector, error)

// The declarative configuration schema currently documents container/host/process/service detectors:
// https://github.com/open-telemetry/opentelemetry-configuration/blob/ddc8c8f47085eb5637a4b1e084b65e384521f8b9/schema/resource.yaml#L74-L141
// The collector extends that list with additional Go SDK detectors (env, host, and cloud providers) to preserve parity
// with the resource detection features exposed directly by the SDKs.
var detectorRegistry = map[string]DetectorFactory{
	"env":  newEnvDetector,
	"host": newHostDetector,
	"aws": func(_ context.Context) (resource.Detector, error) {
		return &multiDetector{
			detectors: []resource.Detector{
				ec2detector.NewResourceDetector(),
				ecs.NewResourceDetector(),
				eks.NewResourceDetector(),
				lambda.NewResourceDetector(),
			},
		}, nil
	},
	"aws/ec2": func(_ context.Context) (resource.Detector, error) {
		return ec2detector.NewResourceDetector(), nil
	},
	"aws/ecs": func(_ context.Context) (resource.Detector, error) {
		return ecs.NewResourceDetector(), nil
	},
	"aws/eks": func(_ context.Context) (resource.Detector, error) {
		return eks.NewResourceDetector(), nil
	},
	"aws/lambda": func(_ context.Context) (resource.Detector, error) {
		return lambda.NewResourceDetector(), nil
	},
	"gcp": func(_ context.Context) (resource.Detector, error) {
		return gcp.NewDetector(), nil
	},
	"azure": func(_ context.Context) (resource.Detector, error) {
		return azurevm.New(), nil
	},
}

func newEnvDetector(_ context.Context) (resource.Detector, error) {
	return &envDetector{}, nil
}

type envDetector struct{}

// Detect returns resource information from environment variables.
func (e *envDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(ctx, resource.WithFromEnv())
	if err != nil {
		return nil, err
	}
	// Strip schema URL to avoid conflicts - it will be set at the resource level
	return resource.New(ctx, resource.WithAttributes(res.Attributes()...))
}

func newHostDetector(_ context.Context) (resource.Detector, error) {
	return &hostDetector{}, nil
}

type hostDetector struct{}

// Detect returns host resource information.
func (h *hostDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	res, err := resource.New(ctx, resource.WithHost())
	if err != nil {
		return nil, err
	}
	// Strip schema URL to avoid conflicts - it will be set at the resource level
	return resource.New(ctx, resource.WithAttributes(res.Attributes()...))
}

type multiDetector struct {
	detectors []resource.Detector
}

// Detect runs all detectors and merges their results.
func (m *multiDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx, resource.WithDetectors(m.detectors...))
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
