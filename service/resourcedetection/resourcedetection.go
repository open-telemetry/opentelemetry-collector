// Copyright The OpenTelemetry Authors
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

// Package resourcedetection contains an interface for detecting resource
// information, and a mechanism to add custom detectors.
//
// It also includes a default implementation to detect resource information
// from the OT_RESOURCE environment variable.
package resourcedetection

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
)

type ResourceDetector interface {
	Detect(ctx context.Context) (*resource.Resource, error)
}

type ResourceProvider struct {
	detectors []ResourceDetector
}

func NewResourceProvider() *ResourceProvider {
	return &ResourceProvider{detectors: []ResourceDetector{&FromEnv{}}}
}

func (rp *ResourceProvider) AddDetectors(detectors ...ResourceDetector) {
	rp.detectors = append(rp.detectors, detectors...)
}

func (rp *ResourceProvider) DetectResource(ctx context.Context) (*resource.Resource, error) {
	var res *resource.Resource
	if rp == nil {
		return res, nil
	}

	for _, detector := range rp.detectors {
		r, err := detector.Detect(ctx)
		if err != nil {
			return nil, err
		}
		res = resource.Merge(res, r)
	}
	return res, nil
}
