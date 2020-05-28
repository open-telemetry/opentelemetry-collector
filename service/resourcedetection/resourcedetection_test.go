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

package resourcedetection

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/sdk/resource"
)

type MockDetector struct {
	mock.Mock
}

func (p *MockDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	args := p.Called()
	return args.Get(0).(*resource.Resource), args.Error(1)
}

func TestDetectResource(t *testing.T) {
	md1 := &MockDetector{}
	md1.On("Detect").Return(resource.New(kv.String("a", "1"), kv.String("b", "2")), nil)

	md2 := &MockDetector{}
	md2.On("Detect").Return(resource.New(kv.String("a", "11"), kv.String("c", "3")), nil)

	rp := NewResourceProvider()
	rp.AddDetectors(md1, md2)

	got, err := rp.DetectResource(context.Background())
	require.NoError(t, err)

	want := resource.New(kv.String("a", "1"), kv.String("b", "2"), kv.String("c", "3"))
	assert.Equal(t, got, want)
}

func TestDetectResource_Error(t *testing.T) {
	md1 := &MockDetector{}
	md1.On("Detect").Return(resource.New(kv.String("a", "1"), kv.String("b", "2")), nil)

	md2 := &MockDetector{}
	md2.On("Detect").Return(resource.New(), errors.New("err1"))

	rp := NewResourceProvider()
	rp.AddDetectors(md1, md2)

	_, err := rp.DetectResource(context.Background())
	require.EqualError(t, err, "err1")
}
