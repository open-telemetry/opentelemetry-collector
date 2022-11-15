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
package components

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/status"
)

func TestNotifications_PipelineReadyNotReady(t *testing.T) {
	notifications := NewNotifications()

	l := newListenerSpy(false)

	un := notifications.RegisterListener(
		status.WithPipelineStatusHandler(l.PipelineStatusSpy.Func),
	)

	assert.False(t, l.PipelineStatusSpy.NotReadyCalled(), "Unexpected call to SetPipelineStatus")
	assert.False(t, l.PipelineStatusSpy.NotReadyCalled(), "Unexpected call to PipelineNotReady")

	assert.NoError(t, notifications.SetPipelineStatus(status.PipelineReady))
	assert.True(t, l.PipelineStatusSpy.ReadyCalled(), "Expected call to SetPipelineStatus")

	assert.NoError(t, notifications.SetPipelineStatus(status.PipelineNotReady))
	assert.True(t, l.PipelineStatusSpy.NotReadyCalled(), "Expected call to PipelineNotReady")

	assert.NoError(t, un())
	notifications.Shutdown()
}

func TestNotifications_PipelineReadyNotReadyWithError(t *testing.T) {
	notifications := NewNotifications()

	l1 := newListenerSpy(true)
	l2 := newListenerSpy(false)

	un1 := notifications.RegisterListener(
		status.WithPipelineStatusHandler(l1.PipelineStatusSpy.Func),
	)

	un2 := notifications.RegisterListener(
		status.WithPipelineStatusHandler(l2.PipelineStatusSpy.Func),
	)

	assert.False(t, l1.PipelineStatusSpy.ReadyCalled(), "Unexpected call to SetPipelineStatus")
	assert.False(t, l1.PipelineStatusSpy.NotReadyCalled(), "Unexpected call to PipelineNotReady")
	assert.False(t, l2.PipelineStatusSpy.ReadyCalled(), "Unexpected call to SetPipelineStatus")
	assert.False(t, l2.PipelineStatusSpy.NotReadyCalled(), "Unexpected call to PipelineNotReady")

	assert.Error(t, notifications.SetPipelineStatus(status.PipelineReady))
	assert.True(t, l1.PipelineStatusSpy.ReadyCalled(), "Expected call to SetPipelineStatus")
	assert.True(t, l2.PipelineStatusSpy.ReadyCalled(), "Expected call to SetPipelineStatus")

	assert.Error(t, notifications.SetPipelineStatus(status.PipelineNotReady))
	assert.True(t, l1.PipelineStatusSpy.NotReadyCalled(), "Expected call to PipelineNotReady")
	assert.True(t, l2.PipelineStatusSpy.NotReadyCalled(), "Expected call to PipelineNotReady")

	assert.NoError(t, un1())
	assert.NoError(t, un2())
	notifications.Shutdown()
}

func TestNotifications_ReportStatus(t *testing.T) {
	notifications := NewNotifications()

	eeSpy := newStatusEventSpy(false)

	un := notifications.RegisterListener(
		status.WithComponentEventHandler(eeSpy.Func),
	)

	ev1, err := status.NewComponentEvent(
		status.ComponentError,
		status.WithError(errors.New("err1")),
	)

	assert.NoError(t, err)
	assert.NoError(t, notifications.SetComponentStatus(ev1))
	assert.True(t, eeSpy.Called(), "Expected call to status event handler")
	assert.Equal(t, 1, eeSpy.CallCount)
	assert.Equal(t, "err1", eeSpy.LastArg.Err().Error())

	ev2, err := status.NewComponentEvent(
		status.ComponentError,
		status.WithError(errors.New("err2")),
	)

	assert.NoError(t, err)
	assert.NoError(t, notifications.SetComponentStatus(ev2))
	assert.True(t, eeSpy.Called(), "Expected call to status event handler")
	assert.Equal(t, 2, eeSpy.CallCount)
	assert.Equal(t, "err2", eeSpy.LastArg.Err().Error())

	assert.NoError(t, un())
	notifications.Shutdown()
}

func TestNotifications_AllHandlersOptional(t *testing.T) {
	notifications := NewNotifications()
	un := notifications.RegisterListener()

	assert.NoError(t, notifications.SetPipelineStatus(status.PipelineReady))
	ev, err := status.NewComponentEvent(
		status.ComponentOK,
	)
	assert.NoError(t, err)
	assert.NoError(t, notifications.SetComponentStatus(ev))

	assert.NoError(t, notifications.SetPipelineStatus(status.PipelineNotReady))
	assert.NoError(t, un())
	notifications.Shutdown()
}

func TestNotifications_StatusHandlerWithError(t *testing.T) {
	notifications := NewNotifications()

	l1 := newListenerSpy(true)
	l2 := newListenerSpy(false)

	un1 := notifications.RegisterListener(
		status.WithComponentEventHandler(l1.StatusEventSpy.Func),
	)

	un2 := notifications.RegisterListener(
		status.WithComponentEventHandler(l2.StatusEventSpy.Func),
	)

	ev1, err := status.NewComponentEvent(status.ComponentOK)
	assert.NoError(t, err)
	assert.Error(t, notifications.SetComponentStatus(ev1))

	ev2, err := status.NewComponentEvent(
		status.ComponentError,
		status.WithError(errors.New("err")),
	)
	assert.NoError(t, err)
	assert.Error(t, notifications.SetComponentStatus(ev2))

	assert.True(t, l1.StatusEventSpy.Called(), "Expected call to status event handler")
	assert.Equal(t, 2, l1.StatusEventSpy.CallCount)
	assert.True(t, l2.StatusEventSpy.Called(), "Expected call to status event handler")
	assert.Equal(t, 2, l2.StatusEventSpy.CallCount)

	assert.NoError(t, un1())
	assert.NoError(t, un2())
	notifications.Shutdown()
}

func TestNotifications_RegisterUnregister(t *testing.T) {
	l1 := newListenerSpy(false)
	l2 := newListenerSpy(false)

	notifications := NewNotifications()

	un1 := notifications.RegisterListener(
		status.WithPipelineStatusHandler(l1.PipelineStatusSpy.Func),
		status.WithComponentEventHandler(l1.StatusEventSpy.Func),
	)

	un2 := notifications.RegisterListener(
		status.WithPipelineStatusHandler(l2.PipelineStatusSpy.Func),
		status.WithComponentEventHandler(l2.StatusEventSpy.Func),
	)

	assert.NoError(t, notifications.SetPipelineStatus(status.PipelineReady))

	ev1, err := status.NewComponentEvent(
		status.ComponentError,
		status.WithError(errors.New("err1")),
	)
	assert.NoError(t, err)
	assert.NoError(t, notifications.SetComponentStatus(ev1))
	assert.NoError(t, un1())

	ev2, err := status.NewComponentEvent(
		status.ComponentError,
		status.WithError(errors.New("err2")),
	)
	assert.NoError(t, err)
	assert.NoError(t, notifications.SetComponentStatus(ev2))
	assert.NoError(t, un2())

	ev3, err := status.NewComponentEvent(
		status.ComponentOK,
	)
	assert.NoError(t, err)
	assert.NoError(t, notifications.SetComponentStatus(ev3))

	assert.NoError(t, notifications.SetPipelineStatus(status.PipelineNotReady))
	notifications.Shutdown()

	assert.True(t, l1.PipelineStatusSpy.ReadyCalled(), "Expected call to SetPipelineStatus")
	assert.True(t, l1.StatusEventSpy.Called(), "Expected call to status event handler")
	assert.Equal(t, 1, l1.StatusEventSpy.CallCount)
	assert.Equal(t, "err1", l1.StatusEventSpy.LastArg.Err().Error())

	assert.True(t, l2.PipelineStatusSpy.ReadyCalled(), "Exected call to SetPipelineStatus")
	assert.True(t, l2.StatusEventSpy.Called(), "Expected call to status event handler")
	assert.Equal(t, 2, l2.StatusEventSpy.CallCount)
	assert.Equal(t, "err2", l2.StatusEventSpy.LastArg.Err().Error())
}

func TestNotifications_LateUnregisterReturnsError(t *testing.T) {
	l1 := newListenerSpy(false)

	notifications := NewNotifications()

	unreg := notifications.RegisterListener(
		status.WithPipelineStatusHandler(l1.PipelineStatusSpy.Func),
	)

	// shutdown unregisters listeners
	notifications.Shutdown()
	assert.Error(t, unreg())
}

type callCounter struct {
	CallCount int
}

func (cc *callCounter) Called() bool {
	return cc.CallCount > 0
}

type readyCallCounter struct {
	ReadyCallCount    int
	NotReadyCallCount int
}

func (cc *readyCallCounter) ReadyCalled() bool {
	return cc.ReadyCallCount > 0
}

func (cc *readyCallCounter) NotReadyCalled() bool {
	return cc.NotReadyCallCount > 0
}

type pipelineStatusSpy struct {
	readyCallCounter
	Func status.PipelineStatusFunc
}

func newPipelineEventSpy(returnErr bool) *pipelineStatusSpy {
	var err error
	if returnErr {
		err = errors.New("pipeline func error")
	}

	spy := &pipelineStatusSpy{}
	spy.Func = func(s status.PipelineReadiness) error {
		if s == status.PipelineReady {
			spy.ReadyCallCount++
		} else {
			spy.NotReadyCallCount++
		}
		return err
	}
	return spy
}

type statusEventSpy struct {
	callCounter
	Func    status.ComponentEventFunc
	LastArg *status.ComponentEvent
}

func newStatusEventSpy(returnErr bool) *statusEventSpy {
	var err error
	if returnErr {
		err = errors.New("error event func error")
	}

	spy := &statusEventSpy{}
	spy.Func = func(ev *status.ComponentEvent) error {
		spy.LastArg = ev
		spy.CallCount++
		return err
	}
	return spy
}

type listenerSpy struct {
	PipelineStatusSpy *pipelineStatusSpy
	StatusEventSpy    *statusEventSpy
}

func newListenerSpy(returnErr bool) *listenerSpy {
	return &listenerSpy{
		PipelineStatusSpy: newPipelineEventSpy(returnErr),
		StatusEventSpy:    newStatusEventSpy(returnErr),
	}
}
