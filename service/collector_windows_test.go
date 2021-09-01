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

//go:build windows
// +build windows

package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/svc"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func TestWindowsService_Execute(t *testing.T) {
	os.Args = []string{"otelcol", "--config", "testdata/otelcol-config-minimal.yaml"}

	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	s := NewWindowsService(CollectorSettings{BuildInfo: component.DefaultBuildInfo(), Factories: factories})

	colDone := make(chan struct{})
	requests := make(chan svc.ChangeRequest)
	changes := make(chan svc.Status)
	go func() {
		defer close(colDone)
		ssec, errno := s.Execute([]string{"svc name"}, requests, changes)
		assert.Equal(t, uint32(0), errno)
		assert.False(t, ssec)
	}()

	assert.Equal(t, svc.StartPending, (<-changes).State)
	assert.Equal(t, svc.Running, (<-changes).State)
	requests <- svc.ChangeRequest{Cmd: svc.Interrogate, CurrentStatus: svc.Status{State: svc.Running}}
	assert.Equal(t, svc.Running, (<-changes).State)
	requests <- svc.ChangeRequest{Cmd: svc.Stop}
	assert.Equal(t, svc.StopPending, (<-changes).State)
	assert.Equal(t, svc.Stopped, (<-changes).State)
	<-colDone
}
