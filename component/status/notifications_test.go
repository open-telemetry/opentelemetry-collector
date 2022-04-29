package status

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config"
)

func TestNotifications_PipelineReadyNotReady(t *testing.T) {
	notifications := NewNotifications()

	l := newListenerSpy(false)

	un := notifications.RegisterListener(
		WithPipelineReadyHandler(l.PipelineReadySpy.Func),
		WithPipelineNotReadyHandler(l.PipelineNotReadySpy.Func),
	)

	assert.False(t, l.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineReady")
	assert.False(t, l.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")

	notifications.Start()
	defer notifications.Shutdown()

	assert.NoError(t, notifications.PipelineReady())
	assert.True(t, l.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")

	assert.NoError(t, notifications.PipelineNotReady())
	assert.True(t, l.PipelineNotReadySpy.WasCalled(), "Expected call to PipelineNotReady")

	assert.NoError(t, un())
}

func TestNotifications_PipelineReadyNotReadyWithError(t *testing.T) {
	notifications := NewNotifications()

	l1 := newListenerSpy(true)
	l2 := newListenerSpy(false)

	un1 := notifications.RegisterListener(
		WithPipelineReadyHandler(l1.PipelineReadySpy.Func),
		WithPipelineNotReadyHandler(l1.PipelineNotReadySpy.Func),
	)

	un2 := notifications.RegisterListener(
		WithPipelineReadyHandler(l2.PipelineReadySpy.Func),
		WithPipelineNotReadyHandler(l2.PipelineNotReadySpy.Func),
	)

	assert.False(t, l1.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineReady")
	assert.False(t, l1.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")
	assert.False(t, l2.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineReady")
	assert.False(t, l2.PipelineReadySpy.WasCalled(), "Unexpected call to PipelineNotReady")

	notifications.Start()
	defer notifications.Shutdown()

	assert.Error(t, notifications.PipelineReady())
	assert.True(t, l1.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")
	assert.True(t, l2.PipelineReadySpy.WasCalled(), "Expected call to PipelineReady")

	assert.Error(t, notifications.PipelineNotReady())
	assert.True(t, l1.PipelineNotReadySpy.WasCalled(), "Expected call to PipelineNotReady")
	assert.True(t, l2.PipelineNotReadySpy.WasCalled(), "Expected call to PipelineNotReady")

	assert.NoError(t, un1())
	assert.NoError(t, un2())
}

func TestNotifications_ReportStatus(t *testing.T) {
	notifications := NewNotifications()

	eeSpy := newStatusEventSpy(false)

	un := notifications.RegisterListener(
		WithStatusEventHandler(eeSpy.Func),
	)

	notifications.Start()
	defer notifications.Shutdown()

	compID := config.NewComponentID("nop")

	notifications.ReportStatus(
		NewStatusEvent(EventError, compID, WithError(errors.New("err1"))),
	)
	assert.True(t, eeSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 1, eeSpy.CallCount)
	assert.Equal(t, "err1", eeSpy.LastArg.Error.Error())

	notifications.ReportStatus(
		NewStatusEvent(EventError, compID, WithError(errors.New("err2"))),
	)
	assert.True(t, eeSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, eeSpy.CallCount)
	assert.Equal(t, "err2", eeSpy.LastArg.Error.Error())

	assert.NoError(t, un())
}

func TestNotifications_StatusHandlerWithError(t *testing.T) {
	notifications := NewNotifications()

	l1 := newListenerSpy(true)
	l2 := newListenerSpy(false)

	un1 := notifications.RegisterListener(
		WithStatusEventHandler(l1.StatusEventSpy.Func),
	)

	un2 := notifications.RegisterListener(
		WithStatusEventHandler(l2.StatusEventSpy.Func),
	)

	notifications.Start()
	defer notifications.Shutdown()

	compID := config.NewComponentID("nop")

	assert.Error(t, notifications.ReportStatus(NewStatusEvent(EventOK, compID)))
	assert.Error(t, notifications.ReportStatus(NewStatusEvent(EventError, compID, WithError(errors.New("err")))))

	assert.True(t, l1.StatusEventSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, l1.StatusEventSpy.CallCount)
	assert.True(t, l2.StatusEventSpy.WasCalled(), "Expected call to status event handler")
	assert.Equal(t, 2, l2.StatusEventSpy.CallCount)

	assert.NoError(t, un1())
	assert.NoError(t, un2())
}

func TestNotifications_RegisterUnregister(t *testing.T) {
	l1 := newListenerSpy(false)
	l2 := newListenerSpy(false)

	notifications := NewNotifications()

	un1 := notifications.RegisterListener(
		WithPipelineReadyHandler(l1.PipelineReadySpy.Func),
		WithPipelineNotReadyHandler(l1.PipelineNotReadySpy.Func),
		WithStatusEventHandler(l1.StatusEventSpy.Func),
	)

	un2 := notifications.RegisterListener(
		WithPipelineReadyHandler(l2.PipelineReadySpy.Func),
		WithPipelineNotReadyHandler(l2.PipelineNotReadySpy.Func),
		WithStatusEventHandler(l2.StatusEventSpy.Func),
	)

	compID := config.NewComponentID("nop")

	notifications.Start()
	notifications.PipelineReady()
	notifications.ReportStatus(NewStatusEvent(EventError, compID, WithError(errors.New("err1"))))
	un1()

	notifications.ReportStatus(NewStatusEvent(EventError, compID, WithError(errors.New("err2"))))
	un2()

	notifications.ReportStatus(NewStatusEvent(EventOK, compID))
	notifications.PipelineNotReady()
	notifications.Shutdown()

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

type callCounter struct {
	CallCount int
}

func (cc *callCounter) WasCalled() bool {
	return cc.CallCount > 0
}

type pipelineEventSpy struct {
	callCounter
	Func PipelineEventFunc
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
	Func    StatusEventFunc
	LastArg StatusEvent
}

func newStatusEventSpy(returnErr bool) *statusEventSpy {
	var err error
	if returnErr {
		err = errors.New("error event func error")
	}

	spy := &statusEventSpy{}
	spy.Func = func(ev StatusEvent) error {
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
