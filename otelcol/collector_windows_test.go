// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package otelcol

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows/svc"

	"go.opentelemetry.io/collector/component"
)

func TestNewSvcHandler(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	filePath := filepath.Join("testdata", "otelcol-nop.yaml")
	os.Args = []string{"otelcol", "--config", filePath}

	s := NewSvcHandler(CollectorSettings{BuildInfo: component.NewDefaultBuildInfo(), Factories: nopFactories, ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filePath})})

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
