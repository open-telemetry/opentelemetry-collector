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
package status

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/status"
	"go.opentelemetry.io/collector/config"
)

func TestNotifications_PipelineReadyNotReady(t *testing.T) {
	notifications := NewNotifications()

	l := newListenerSpy(false)

	un := notifications.RegisterListener(
		status.WithPipelineReadyHandler(l.PipelineReadySpy.Func),
		status.WithPipelineNotReadyHandler(l.PipelineNotReadySpy.Func),
	)

	assert.False(t, l.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineReady")
	assert.False(t, l.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")

	assert.NoError(t, notifications.Start())

	assert.NoError(t, notifications.PipelineReady())
	assert.True(t, l.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")

	assert.NoError(t, notifications.PipelineNotReady())
	assert.True(t, l.PipelineNotReadySpy.WasCalled(), "Expected call to PipelineNotReady")

	assert.NoError(t, un())
	assert.NoError(t, notifications.Shutdown())
}

func TestNotifications_PipelineReadyNotReadyWithError(t *testing.T) {
	notifications := NewNotifications()

	l1 := newListenerSpy(true)
	l2 := newListenerSpy(false)

	un1 := notifications.RegisterListener(
		status.WithPipelineReadyHandler(l1.PipelineReadySpy.Func),
		status.WithPipelineNotReadyHandler(l1.PipelineNotReadySpy.Func),
	)

	un2 := notifications.RegisterListener(
		status.WithPipelineReadyHandler(l2.PipelineReadySpy.Func),
		status.WithPipelineNotReadyHandler(l2.PipelineNotReadySpy.Func),
	)

	assert.False(t, l1.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineReady")
	assert.False(t, l1.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")
	assert.False(t, l2.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineReady")
	assert.False(t, l2.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")

	assert.NoError(t, notifications.Start())
	assert.Error(t, notifications.PipelineReady())
	assert.True(t, l1.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")
	assert.True(t, l2.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")

	assert.Error(t, notifications.PipelineNotReady())
	assert.True(t, l1.PipelineNotReadySpy.WasCalled(), "Expected call to PipelineNotReady")
	assert.True(t, l2.PipelineNotReadySpy.WasCalled(), "Expected call to PipelineNotReady")

	assert.NoError(t, un1())
	assert.NoError(t, un2())
	assert.NoError(t, notifications.Shutdown())
}

func TestNotifications_ReportStatus(t *testing.T) {
	notifications := NewNotifications()

	eeSpy := newStatusEventSpy(false)

	un := notifications.RegisterListener(
		status.WithStatusEventHandler(eeSpy.Func),
	)

	assert.NoError(t, notifications.Start())

	compID := config.NewComponentID("nop")

	assert.NoError(t,
		notifications.ReportStatus(
			status.Event{
				Type:        status.RecoverableError,
				ComponentID: compID,
				Error:       errors.New("err1"),
			},
		),
	)
	assert.True(t, eeSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 1, eeSpy.CallCount)
	assert.Equal(t, "err1", eeSpy.LastArg.Error.Error())

	assert.NoError(t,
		notifications.ReportStatus(
			status.Event{
				Type:        status.RecoverableError,
				ComponentID: compID,
				Error:       errors.New("err2"),
			},
		),
	)
	assert.True(t, eeSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, eeSpy.CallCount)
	assert.Equal(t, "err2", eeSpy.LastArg.Error.Error())

	assert.NoError(t, un())
	assert.NoError(t, notifications.Shutdown())
}

func TestNotifications_StatusHandlerWithError(t *testing.T) {
	notifications := NewNotifications()

	l1 := newListenerSpy(true)
	l2 := newListenerSpy(false)

	un1 := notifications.RegisterListener(
		status.WithStatusEventHandler(l1.StatusEventSpy.Func),
	)

	un2 := notifications.RegisterListener(
		status.WithStatusEventHandler(l2.StatusEventSpy.Func),
	)

	assert.NoError(t, notifications.Start())

	compID := config.NewComponentID("nop")

	assert.Error(t, notifications.ReportStatus(
		status.Event{Type: status.OK, ComponentID: compID}),
	)
	assert.Error(t,
		notifications.ReportStatus(
			status.Event{
				Type:        status.RecoverableError,
				ComponentID: compID,
				Error:       errors.New("err"),
			},
		),
	)

	assert.True(t, l1.StatusEventSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, l1.StatusEventSpy.CallCount)
	assert.True(t, l2.StatusEventSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, l2.StatusEventSpy.CallCount)

	assert.NoError(t, un1())
	assert.NoError(t, un2())
	assert.NoError(t, notifications.Shutdown())
}

func TestNotifications_RegisterUnregister(t *testing.T) {
	l1 := newListenerSpy(false)
	l2 := newListenerSpy(false)

	notifications := NewNotifications()

	un1 := notifications.RegisterListener(
		status.WithPipelineReadyHandler(l1.PipelineReadySpy.Func),
		status.WithPipelineNotReadyHandler(l1.PipelineNotReadySpy.Func),
		status.WithStatusEventHandler(l1.StatusEventSpy.Func),
	)

	un2 := notifications.RegisterListener(
		status.WithPipelineReadyHandler(l2.PipelineReadySpy.Func),
		status.WithPipelineNotReadyHandler(l2.PipelineNotReadySpy.Func),
		status.WithStatusEventHandler(l2.StatusEventSpy.Func),
	)

	compID := config.NewComponentID("nop")

	assert.NoError(t, notifications.Start())
	assert.NoError(t, notifications.PipelineReady())
	assert.NoError(t,
		notifications.ReportStatus(
			status.Event{
				Type:        status.RecoverableError,
				ComponentID: compID,
				Error:       errors.New("err1"),
			},
		),
	)

	assert.NoError(t, un1())

	assert.NoError(t,
		notifications.ReportStatus(
			status.Event{
				Type:        status.RecoverableError,
				ComponentID: compID,
				Error:       errors.New("err2"),
			},
		),
	)

	assert.NoError(t, un2())

	assert.NoError(t,
		notifications.ReportStatus(
			status.Event{
				Type:        status.OK,
				ComponentID: compID,
			},
		),
	)

	assert.NoError(t, notifications.PipelineNotReady())
	assert.NoError(t, notifications.Shutdown())

	assert.True(t, l1.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")
	assert.True(t, l1.StatusEventSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 1, l1.StatusEventSpy.CallCount)
	assert.Equal(t, "err1", l1.StatusEventSpy.LastArg.Error.Error())
	assert.False(t, l1.PipelineNotReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")

	assert.True(t, l2.PipelineReadySpy.WasCalled(), "Exected call to PipelineReady")
	assert.True(t, l2.StatusEventSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, l2.StatusEventSpy.CallCount)
	assert.Equal(t, "err2", l2.StatusEventSpy.LastArg.Error.Error())
	assert.False(t, l2.PipelineNotReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")
}

func TestNotifications_LateUnregisterReturnsError(t *testing.T) {
	l1 := newListenerSpy(false)

	notifications := NewNotifications()
	assert.NoError(t, notifications.Start())

	unreg := notifications.RegisterListener(
		status.WithPipelineNotReadyHandler(l1.PipelineNotReadySpy.Func),
	)

	// shutdown unregisters listeners
	assert.NoError(t, notifications.Shutdown())
	assert.Error(t, unreg())
}

type callCounter struct {
	CallCount int
}

func (cc *callCounter) WasCalled() bool {
	return cc.CallCount > 0
}

type pipelineEventSpy struct {
	callCounter
	Func status.PipelineFunc
}

func newPipelineEventSpy(returnErr bool) *pipelineEventSpy {
	var err error
	if returnErr {
		err = errors.New("pipeline func error")
	}

	spy := &pipelineEventSpy{}
	spy.Func = func() error {
		spy.CallCount++
		return err
	}
	return spy
}

type statusEventSpy struct {
	callCounter
	Func    status.EventFunc
	LastArg status.Event
}

func newStatusEventSpy(returnErr bool) *statusEventSpy {
	var err error
	if returnErr {
		err = errors.New("error event func error")
	}

	spy := &statusEventSpy{}
	spy.Func = func(ev status.Event) error {
		spy.LastArg = ev
		spy.CallCount++
		return err
	}
	return spy
}

type listenerSpy struct {
	PipelineReadySpy    *pipelineEventSpy
	PipelineNotReadySpy *pipelineEventSpy
	StatusEventSpy      *statusEventSpy
}

func newListenerSpy(returnErr bool) *listenerSpy {
	return &listenerSpy{
		PipelineReadySpy:    newPipelineEventSpy(returnErr),
		PipelineNotReadySpy: newPipelineEventSpy(returnErr),
		StatusEventSpy:      newStatusEventSpy(returnErr),
	}
}
