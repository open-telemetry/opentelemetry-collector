// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v7"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/xpdata/pref"
	"go.opentelemetry.io/collector/pipeline"
)

// OrderedPosition tells the ordered queue whether another item continues the
// current stream or establishes a boundary after its write phase.
type OrderedPosition uint8

const (
	OrderedPositionContinue OrderedPosition = iota
	OrderedPositionEnd
)

// OrderedLogsDescriptor is one child of an atomically admitted logs request.
type OrderedLogsDescriptor struct {
	Request      plog.Logs
	PartitionKey string
	Position     OrderedPosition
	QueueID      [16]byte
}

// OrderedLogsDispatch is the metadata passed to an asynchronous ordered sender.
type OrderedLogsDispatch struct {
	OrderedLogsDescriptor
	GroupItems int
	// StreamContext is canceled when this attempt ends or is fenced.
	StreamContext context.Context
	StreamAttempt uint64
	Recovery      bool
}

// OrderedLogsCompletion separates release of same-partition write order from
// final retirement of queue ownership.
type OrderedLogsCompletion interface {
	Release()
	Succeed()
	Fail(error)
}

type (
	OrderedLogsConverterFunc func(context.Context, plog.Logs) ([]OrderedLogsDescriptor, error)
	OrderedLogsConsumeFunc   func(context.Context, OrderedLogsDispatch, OrderedLogsCompletion) error
)

// OrderedLogsSettings bounds the request-group and coordinator memory.
type OrderedLogsSettings struct {
	// MaxConcurrentWrites bounds concurrently dispatched partitions. Queue
	// consumption remains single-reader so queue insertion order is preserved.
	MaxConcurrentWrites  int
	MaxStaged            int
	MaxActivePartitions  int
	MaxReleasedRequests  int
	MaxReleasedBytes     int
	MaxRecoveryTailBytes int
	MaxGroupRequests     int
	MaxGroupItems        int
	MaxGroupBytes        int
	MaxPartitionKeyBytes int
}

// Validate rejects unbounded or internally inconsistent stream limits.
func (s OrderedLogsSettings) Validate() error {
	if s.MaxStaged <= 0 || s.MaxActivePartitions <= 0 || s.MaxReleasedRequests <= 0 ||
		s.MaxReleasedBytes <= 0 || s.MaxRecoveryTailBytes <= 0 || s.MaxGroupRequests <= 0 ||
		s.MaxGroupItems <= 0 || s.MaxGroupBytes <= 0 || s.MaxPartitionKeyBytes <= 0 {
		return errors.New("ordered stream limits must all be positive")
	}
	if s.MaxGroupRequests > s.MaxStaged || s.MaxGroupBytes > s.MaxReleasedBytes ||
		s.MaxRecoveryTailBytes > s.MaxReleasedBytes || s.MaxPartitionKeyBytes > s.MaxGroupBytes {
		return errors.New("ordered stream limits exceed their enclosing bounds")
	}
	return nil
}

type orderedLogsGroup struct {
	children []OrderedLogsDescriptor
	items    int
	bytes    int

	mu         sync.Mutex
	completion func(error)
	remaining  int
	completed  bool
	firstErr   error
}

var (
	_ request.Request                 = (*orderedLogsGroup)(nil)
	_ request.DeferredQueueCompletion = (*orderedLogsGroup)(nil)
)

func (g *orderedLogsGroup) QueueRequestsCount() int64 { return int64(len(g.children)) }

func (g *orderedLogsGroup) SetQueueCompletion(done func(error)) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.completion != nil || g.completed {
		return false
	}
	g.completion = done
	g.remaining = len(g.children)
	if g.remaining == 0 {
		g.completed = true
		go done(nil)
	}
	return true
}

func (g *orderedLogsGroup) finishChild(err error) {
	g.mu.Lock()
	if g.completed {
		g.mu.Unlock()
		return
	}
	if err != nil && (g.firstErr == nil || experr.IsShutdownErr(err)) {
		g.firstErr = err
	}
	g.remaining--
	if g.remaining == 0 {
		g.completed = true
		done := g.completion
		finalErr := g.firstErr
		g.mu.Unlock()
		if done != nil {
			done(finalErr)
		}
		return
	}
	g.mu.Unlock()
}

func (g *orderedLogsGroup) ItemsCount() int { return g.items }
func (g *orderedLogsGroup) BytesSize() int  { return g.bytes }
func (g *orderedLogsGroup) MergeSplit(context.Context, int, request.SizerType, request.Request) ([]request.Request, error) {
	return nil, errors.New("ordered request groups cannot be batched or split")
}

var orderedLogsWire = &plog.ProtoMarshaler{}

type orderedLogsReferenceCounter struct{}

func (orderedLogsReferenceCounter) Ref(req request.Request) {
	for _, child := range req.(*orderedLogsGroup).children {
		pref.RefLogs(child.Request)
	}
}

func (orderedLogsReferenceCounter) Unref(req request.Request) {
	for _, child := range req.(*orderedLogsGroup).children {
		pref.UnrefLogs(child.Request)
	}
}

func NewOrderedLogsQueueBatchSettings() queuebatch.Settings[request.Request] {
	return queuebatch.Settings[request.Request]{
		ReferenceCounter: orderedLogsReferenceCounter{},
	}
}

type orderedLogTask struct {
	desc         OrderedLogsDescriptor
	group        *orderedLogsGroup
	ctx          context.Context
	dispatched   bool
	released     bool
	committed    bool
	bytes        int
	tailCharge   int
	cancel       context.CancelFunc
	timer        *time.Timer
	recoveryTail bool
}

type orderedLogsPartition struct {
	key                  string
	tasks                []*orderedLogTask
	tail                 *orderedLogTask
	recoveryPending      *orderedLogTask
	active               bool
	activeTask           *orderedLogTask
	attemptCtx           context.Context
	attemptCancel        context.CancelFunc
	epoch                uint64
	stream               uint64
	streamOpen           bool
	retryStarted         time.Time
	backoff              *backoff.ExponentialBackOff
	retirementPending    bool
	retrying             bool
	waitingForTailBudget bool
}

type orderedLogsCoordinator struct {
	mu               sync.Mutex
	changed          *sync.Cond
	parts            map[string]*orderedLogsPartition
	staged           int
	releasedRequests int
	releasedBytes    int
	tailBytes        int
	activeWrites     int
	nextStream       uint64
	maxWorkers       int
	settings         OrderedLogsSettings
	consume          OrderedLogsConsumeFunc
	retryCfg         configretry.BackOffConfig
	timeout          time.Duration
	logger           *zap.Logger
	stop             chan struct{}
	stopped          bool
}

func newOrderedLogsCoordinator(set OrderedLogsSettings, consume OrderedLogsConsumeFunc, retry configretry.BackOffConfig, timeout time.Duration, maxWorkers int, logger *zap.Logger) *orderedLogsCoordinator {
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	c := &orderedLogsCoordinator{parts: make(map[string]*orderedLogsPartition), settings: set, consume: consume, retryCfg: retry, timeout: timeout, maxWorkers: maxWorkers, logger: logger, stop: make(chan struct{})}
	c.changed = sync.NewCond(&c.mu)
	return c
}

func (c *orderedLogsCoordinator) add(ctx context.Context, group *orderedLogsGroup) error {
	if len(group.children) == 0 {
		return nil
	}
	if len(group.children) > c.settings.MaxGroupRequests || group.items > c.settings.MaxGroupItems || group.bytes > c.settings.MaxGroupBytes {
		return consumererror.NewPermanent(errors.New("ordered stream group exceeds configured hard limit"))
	}
	inputKeys := make(map[string]struct{}, len(group.children))
	for _, child := range group.children {
		if child.PartitionKey == "" || len(child.PartitionKey) > c.settings.MaxPartitionKeyBytes || child.Position > OrderedPositionEnd || child.Request.LogRecordCount() == 0 || child.QueueID == ([16]byte{}) {
			return consumererror.NewPermanent(errors.New("ordered stream descriptor is invalid"))
		}
		if orderedLogsTailSize(child) > c.settings.MaxRecoveryTailBytes {
			return consumererror.NewPermanent(errors.New("ordered stream fragment exceeds recovery-tail size limit"))
		}
		inputKeys[child.PartitionKey] = struct{}{}
	}
	if len(inputKeys) > c.settings.MaxActivePartitions {
		return consumererror.NewPermanent(errors.New("ordered stream group exceeds active partition limit"))
	}
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return experr.NewShutdownErr(errors.New("ordered stream coordinator is stopped"))
	}
	defer c.mu.Unlock()
	pending := group.children
	for !c.stopped && c.staged+len(pending) > c.settings.MaxStaged && !c.groupAdvancesOpenPartitionLocked(group) {
		// Let a successor through for an already-open partition even after the
		// staging high-water mark. Queue ownership remains charged, so the
		// configured helper queue still bounds the retained requests.
		c.changed.Wait()
	}
	if c.stopped {
		return experr.NewShutdownErr(errors.New("ordered stream coordinator is stopped"))
	}
	for {
		newKeys := make(map[string]struct{})
		for _, child := range pending {
			if c.parts[child.PartitionKey] == nil {
				newKeys[child.PartitionKey] = struct{}{}
			}
		}
		if len(c.parts)+len(newKeys) <= c.settings.MaxActivePartitions || c.groupAdvancesOpenPartitionLocked(group) {
			break
		}
		c.changed.Wait()
		if c.stopped {
			return experr.NewShutdownErr(errors.New("ordered stream coordinator is stopped"))
		}
	}
	for _, child := range pending {
		p := c.parts[child.PartitionKey]
		if p == nil {
			p = &orderedLogsPartition{key: child.PartitionKey}
			c.parts[child.PartitionKey] = p
		}
		t := &orderedLogTask{desc: child, group: group, ctx: ctx, bytes: orderedLogsWire.LogsSize(child.Request)}
		p.tasks = append(p.tasks, t)
		c.staged++
		c.scheduleLocked(p) //nolint:contextcheck // Queue-owned dispatch uses the admitted request context.
	}
	c.scheduleAvailableLocked()
	return nil
}

func (c *orderedLogsCoordinator) groupAdvancesOpenPartitionLocked(group *orderedLogsGroup) bool {
	for _, child := range group.children {
		if partition := c.parts[child.PartitionKey]; partition != nil && partition.streamOpen && len(partition.tasks) > 0 {
			return true
		}
	}
	return false
}

func (c *orderedLogsCoordinator) scheduleLocked(p *orderedLogsPartition) {
	if p.active || p.retrying || c.stopped || c.activeWrites >= c.maxWorkers {
		return
	}
	if p.recoveryPending != nil && !p.recoveryPending.dispatched {
		task := p.recoveryPending
		p.active = true
		p.activeTask = task
		c.activeWrites++
		task.dispatched = true
		if !p.streamOpen || p.attemptCtx == nil {
			c.beginAttemptLocked(p)
		}
		epoch, stream := p.epoch, p.stream
		go c.dispatch(p, task, epoch, stream, true)
		return
	}
	for _, task := range p.tasks {
		if p.recoveryPending != nil && task.desc.QueueID == p.recoveryPending.desc.QueueID {
			continue
		}
		if task.dispatched || task.committed {
			continue
		}
		p.active = true
		p.activeTask = task
		c.activeWrites++
		task.dispatched = true
		if !p.streamOpen || p.attemptCtx == nil {
			c.beginAttemptLocked(p)
		}
		epoch, stream := p.epoch, p.stream
		recovery := task.recoveryTail
		go c.dispatch(p, task, epoch, stream, recovery)
		return
	}
}

func (c *orderedLogsCoordinator) beginAttemptLocked(p *orderedLogsPartition) {
	if p.attemptCancel != nil {
		p.attemptCancel()
	}
	c.nextStream++
	p.stream = c.nextStream
	p.streamOpen = true
	p.attemptCtx, p.attemptCancel = context.WithCancel(context.Background())
}

func stopOrderedTask(task *orderedLogTask) {
	if task == nil {
		return
	}
	if task.timer != nil {
		task.timer.Stop()
	}
	if task.cancel != nil {
		task.cancel()
	}
}

func (c *orderedLogsCoordinator) dispatch(p *orderedLogsPartition, task *orderedLogTask, epoch, stream uint64, recovery bool) {
	var ctx context.Context
	var cancel context.CancelFunc
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(task.ctx, c.timeout)
	} else {
		ctx, cancel = context.WithCancel(task.ctx)
	}
	c.mu.Lock()
	if c.stopped || p.epoch != epoch {
		c.mu.Unlock()
		cancel()
		return
	}
	task.cancel = cancel
	completion := &orderedLogsCompletion{coordinator: c, partition: p, task: task, epoch: epoch, cancel: cancel}
	if c.timeout > 0 {
		completion.timer = time.AfterFunc(c.timeout, func() { completion.Fail(context.DeadlineExceeded) })
		task.timer = completion.timer
	}
	pref.RefLogs(task.desc.Request)
	streamCtx := p.attemptCtx
	c.mu.Unlock()
	defer pref.UnrefLogs(task.desc.Request)
	groupItems := 1
	if task.group != nil {
		groupItems = task.group.items
	}
	dispatch := OrderedLogsDispatch{OrderedLogsDescriptor: task.desc, GroupItems: groupItems, StreamContext: streamCtx, StreamAttempt: stream, Recovery: recovery}
	if err := c.consume(ctx, dispatch, completion); err != nil {
		completion.Fail(err)
	}
}

type orderedLogsCompletion struct {
	coordinator *orderedLogsCoordinator
	partition   *orderedLogsPartition
	task        *orderedLogTask
	epoch       uint64
	releaseOnce sync.Once
	outcomeOnce sync.Once
	timer       *time.Timer
	cancel      context.CancelFunc
}

func (d *orderedLogsCompletion) Release() {
	d.releaseOnce.Do(func() { d.coordinator.release(d.partition, d.task, d.epoch) })
}

func (d *orderedLogsCompletion) Succeed() {
	d.outcomeOnce.Do(func() {
		if d.timer != nil {
			d.timer.Stop()
		}
		d.cancel()
		d.releaseOnce.Do(func() { d.coordinator.release(d.partition, d.task, d.epoch) })
		d.coordinator.succeed(d.partition, d.task, d.epoch)
	})
}

func (d *orderedLogsCompletion) Fail(err error) {
	if err == nil {
		err = errors.New("ordered stream failed without an error")
	}
	d.outcomeOnce.Do(func() {
		if d.timer != nil {
			d.timer.Stop()
		}
		d.cancel()
		d.coordinator.fail(d.partition, d.task, d.epoch, err)
	})
}

func (c *orderedLogsCoordinator) release(p *orderedLogsPartition, task *orderedLogTask, epoch uint64) {
	c.mu.Lock()
	for p.epoch == epoch && !c.stopped && !task.released &&
		(c.releasedRequests >= c.settings.MaxReleasedRequests || c.releasedBytes+task.bytes > c.settings.MaxReleasedBytes) {
		c.changed.Wait()
	}
	if p.epoch == epoch && c.parts[p.key] == p && !c.stopped && !task.released && !task.committed {
		task.released = true
		if p.activeTask == task {
			p.active = false
			p.activeTask = nil
			c.activeWrites--
		}
		c.releasedRequests++
		c.releasedBytes += task.bytes
		if task.desc.Position == OrderedPositionEnd {
			p.streamOpen = false
		}
		c.scheduleLocked(p)
		c.scheduleAvailableLocked()
	}
	c.mu.Unlock()
}

func (c *orderedLogsCoordinator) succeed(p *orderedLogsPartition, task *orderedLogTask, epoch uint64) {
	c.mu.Lock()
	if p.epoch != epoch || c.parts[p.key] != p || c.stopped {
		c.mu.Unlock()
		return
	}
	if task.recoveryTail {
		task.committed = true
		if task.released {
			c.releasedRequests--
			c.releasedBytes -= task.bytes
		}
		if p.recoveryPending == task {
			p.recoveryPending = nil
		}
		c.scheduleLocked(p)
		c.scheduleAvailableLocked()
		c.changed.Broadcast()
		c.mu.Unlock()
		return
	}
	task.committed = true
	retired := c.advanceCommittedPrefixLocked(p, epoch)
	c.changed.Broadcast()
	c.scheduleAvailableLocked()
	c.mu.Unlock()
	finishOrderedTasks(retired, nil)
}

func (c *orderedLogsCoordinator) advanceCommittedPrefixLocked(p *orderedLogsPartition, epoch uint64) []*orderedLogTask {
	count := 0
	var lastCommitted *orderedLogTask
	for count < len(p.tasks) && p.tasks[count].committed {
		lastCommitted = p.tasks[count]
		count++
	}
	if count == 0 {
		return nil
	}
	var nextTail *OrderedLogsDescriptor
	if lastCommitted.desc.Position == OrderedPositionContinue {
		desc := lastCommitted.desc
		nextTail = &desc
	}
	previousBytes := 0
	if p.tail != nil {
		previousBytes = p.tail.tailCharge
	}
	nextTailBytes := 0
	if nextTail != nil {
		nextTailBytes = orderedLogsTailSize(*nextTail)
	}
	if c.tailBytes-previousBytes+nextTailBytes > c.settings.MaxRecoveryTailBytes {
		// Completion callbacks can run on the receiver's ACK reader. Keep the
		// safe prefix queue-owned and resume when another stream frees budget;
		// never block the ACK reader while waiting for capacity.
		p.waitingForTailBudget = true
		return nil
	}
	if c.stopped || p.epoch != epoch {
		return nil
	}
	p.waitingForTailBudget = false
	if nextTail == nil {
		c.clearTailMemoryLocked(p)
	} else {
		c.setTailMemoryLocked(p, lastCommitted)
	}
	retired := make([]*orderedLogTask, 0, count)
	for range count {
		committed := p.tasks[0]
		retired = append(retired, committed)
		if committed.released {
			c.releasedRequests--
			c.releasedBytes -= committed.bytes
		}
		p.tasks = p.tasks[1:]
		c.staged--
	}
	if len(p.tasks) == 0 && !p.streamOpen && !p.active {
		if p.attemptCancel != nil {
			p.attemptCancel()
		}
		delete(c.parts, p.key)
		p.backoff = nil
		p.retryStarted = time.Time{}
	}
	return retired
}

func finishOrderedTasks(tasks []*orderedLogTask, err error) {
	for _, task := range tasks {
		if task.group != nil {
			task.group.finishChild(err)
		}
	}
}

func (c *orderedLogsCoordinator) scheduleRetirementLocked(p *orderedLogsPartition, epoch uint64) {
	if p.retirementPending || c.stopped {
		return
	}
	p.retirementPending = true
	time.AfterFunc(0, func() {
		c.mu.Lock()
		if c.stopped || p.epoch != epoch {
			p.retirementPending = false
			c.mu.Unlock()
			return
		}
		p.retirementPending = false
		retired := c.advanceCommittedPrefixLocked(p, epoch)
		c.changed.Broadcast()
		c.scheduleAvailableLocked()
		c.mu.Unlock()
		finishOrderedTasks(retired, nil)
	})
}

func (c *orderedLogsCoordinator) wakeTailBudgetWaitersLocked() {
	for _, p := range c.parts {
		if p.waitingForTailBudget && !p.retirementPending {
			p.waitingForTailBudget = false
			c.scheduleRetirementLocked(p, p.epoch)
		}
	}
}

func (c *orderedLogsCoordinator) fail(p *orderedLogsPartition, task *orderedLogTask, epoch uint64, err error) {
	c.mu.Lock()
	if p.epoch != epoch || task.committed || c.stopped {
		c.mu.Unlock()
		return
	}
	if consumererror.IsPermanent(err) || !c.retryCfg.Enabled || c.retryExpiredLocked(p) {
		failed := c.terminatePartitionLocked(p)
		c.mu.Unlock()
		finishOrderedTasks(failed, err)
		return
	}
	if p.backoff == nil {
		p.retryStarted = time.Now()
		p.backoff = &backoff.ExponentialBackOff{InitialInterval: c.retryCfg.InitialInterval, RandomizationFactor: c.retryCfg.RandomizationFactor, Multiplier: c.retryCfg.Multiplier, MaxInterval: c.retryCfg.MaxInterval}
	}
	delay := p.backoff.NextBackOff()
	if delay == backoff.Stop {
		failed := c.terminatePartitionLocked(p)
		c.mu.Unlock()
		finishOrderedTasks(failed, err)
		return
	}
	for _, pending := range p.tasks {
		if pending.released {
			c.releasedRequests--
			c.releasedBytes -= pending.bytes
		}
	}
	p.epoch++
	if p.attemptCancel != nil {
		p.attemptCancel()
	}
	p.attemptCtx = nil
	stopOrderedTask(p.recoveryPending)
	if p.recoveryPending != nil && p.recoveryPending.released {
		c.releasedRequests--
		c.releasedBytes -= p.recoveryPending.bytes
	}
	p.streamOpen = true
	if p.active {
		p.active = false
		p.activeTask = nil
		c.activeWrites--
	}
	for _, pending := range p.tasks {
		stopOrderedTask(pending)
		pending.dispatched = false
		pending.released = false
		pending.committed = false
	}
	p.recoveryPending = nil
	if p.tail != nil {
		p.recoveryPending = &orderedLogTask{desc: p.tail.desc, ctx: task.ctx, bytes: p.tail.bytes, recoveryTail: true}
	}
	p.retrying = true
	newEpoch := p.epoch
	time.AfterFunc(delay, func() {
		c.mu.Lock()
		if !c.stopped && p.epoch == newEpoch {
			p.retrying = false
			c.scheduleLocked(p)
			c.scheduleAvailableLocked()
		}
		c.mu.Unlock()
	})
	c.scheduleAvailableLocked()
	if c.logger != nil {
		c.logger.Info("Ordered stream send failed; replaying unretired partition requests", zap.Error(err), zap.Duration("retry_interval", delay))
	}
	c.mu.Unlock()
}

func (c *orderedLogsCoordinator) terminatePartitionLocked(p *orderedLogsPartition) []*orderedLogTask {
	p.epoch++
	if p.attemptCancel != nil {
		p.attemptCancel()
	}
	failed := append([]*orderedLogTask(nil), p.tasks...)
	for _, pending := range failed {
		stopOrderedTask(pending)
		if pending.released {
			c.releasedRequests--
			c.releasedBytes -= pending.bytes
		}
		c.staged--
	}
	stopOrderedTask(p.recoveryPending)
	if p.recoveryPending != nil && p.recoveryPending.released {
		c.releasedRequests--
		c.releasedBytes -= p.recoveryPending.bytes
	}
	if p.active {
		c.activeWrites--
	}
	p.active, p.activeTask = false, nil
	p.tasks, p.recoveryPending = nil, nil
	p.streamOpen = false
	c.clearTailMemoryLocked(p)
	delete(c.parts, p.key)
	c.changed.Broadcast()
	c.scheduleAvailableLocked()
	return failed
}

// scheduleAvailableLocked uses any newly available write permits for other
// partitions whose heads were staged while the worker limit was full.
func (c *orderedLogsCoordinator) scheduleAvailableLocked() {
	for c.activeWrites < c.maxWorkers && !c.stopped {
		before := c.activeWrites
		for _, p := range c.parts {
			c.scheduleLocked(p)
			if c.activeWrites > before {
				break
			}
		}
		if c.activeWrites == before {
			return
		}
	}
}

func (c *orderedLogsCoordinator) retryExpiredLocked(p *orderedLogsPartition) bool {
	return c.retryCfg.MaxElapsedTime > 0 && !p.retryStarted.IsZero() && time.Since(p.retryStarted) >= c.retryCfg.MaxElapsedTime
}

func (c *orderedLogsCoordinator) setTailMemoryLocked(p *orderedLogsPartition, task *orderedLogTask) {
	previousBytes := 0
	if p.tail != nil {
		previousBytes = p.tail.tailCharge
	}
	oldTotalBytes := c.tailBytes
	pref.RefLogs(task.desc.Request)
	if p.tail != nil {
		c.tailBytes -= previousBytes
		pref.UnrefLogs(p.tail.desc.Request)
	}
	p.tail = &orderedLogTask{desc: task.desc, bytes: task.bytes, tailCharge: orderedLogsTailSize(task.desc)}
	c.tailBytes += p.tail.tailCharge
	if c.tailBytes < oldTotalBytes {
		c.wakeTailBudgetWaitersLocked()
	}
}

func (c *orderedLogsCoordinator) clearTailMemoryLocked(p *orderedLogsPartition) {
	if p.tail == nil {
		return
	}
	c.tailBytes -= p.tail.tailCharge
	pref.UnrefLogs(p.tail.desc.Request)
	p.tail = nil
	if c.tailBytes < 0 {
		c.tailBytes = 0
	}
	c.changed.Broadcast()
	c.wakeTailBudgetWaitersLocked()
}

func orderedLogsTailSize(desc OrderedLogsDescriptor) int {
	// Body plus the bounded partition key and fixed queue ID/envelope fields
	// retained beside it for in-memory retry recovery.
	return orderedLogsWire.LogsSize(desc.Request) + len(desc.PartitionKey) + 25
}

func (c *orderedLogsCoordinator) shutdown() {
	c.mu.Lock()
	if !c.stopped {
		c.stopped = true
		close(c.stop)
		c.changed.Broadcast()
	}
	var failed []*orderedLogTask
	for _, p := range c.parts {
		if p.attemptCancel != nil {
			p.attemptCancel()
		}
		for _, task := range p.tasks {
			if task.timer != nil {
				task.timer.Stop()
			}
			if task.cancel != nil {
				task.cancel()
			}
			failed = append(failed, task)
		}
		if p.recoveryPending != nil {
			if p.recoveryPending.timer != nil {
				p.recoveryPending.timer.Stop()
			}
			if p.recoveryPending.cancel != nil {
				p.recoveryPending.cancel()
			}
		}
		if p.tail != nil {
			c.clearTailMemoryLocked(p)
		}
		p.recoveryPending = nil
		p.tasks = nil
	}
	c.parts = make(map[string]*orderedLogsPartition)
	c.staged = 0
	c.releasedRequests = 0
	c.releasedBytes = 0
	c.activeWrites = 0
	c.mu.Unlock()
	for _, task := range failed {
		task.group.finishChild(experr.NewShutdownErr(errors.New("ordered stream coordinator shut down")))
	}
}

// NewLogsRequests creates a logs exporter with atomic descriptor admission and
// per-partition ordered asynchronous completion.
func NewLogsRequests(
	_ context.Context,
	set exporter.Settings,
	converter OrderedLogsConverterFunc,
	pusher OrderedLogsConsumeFunc,
	limits OrderedLogsSettings,
	options ...Option,
) (exporter.Logs, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}
	if converter == nil {
		return nil, errNilLogsConverter
	}
	if pusher == nil {
		return nil, errNilConsumeRequest
	}
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	var coordinator *orderedLogsCoordinator
	baseOptions := append([]Option(nil), options...)
	baseOptions = append(baseOptions, WithAsyncQueue(), WithQueueBatchSettings(NewOrderedLogsQueueBatchSettings()))
	be, baseErr := NewBaseExporter(set, pipeline.SignalLogs, func(ctx context.Context, req request.Request) error {
		group, ok := req.(*orderedLogsGroup)
		if !ok {
			return fmt.Errorf("ordered logs exporter received unexpected request %T", req)
		}
		if err := coordinator.add(ctx, group); err != nil {
			return err
		}
		return nil
	}, baseOptions...)
	if baseErr != nil {
		return nil, baseErr
	}
	maxWorkers := limits.MaxConcurrentWrites
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	coordinator = newOrderedLogsCoordinator(limits, pusher, be.retryCfg, be.timeoutCfg.Timeout, maxWorkers, set.Logger)
	be.asyncShutdown = component.ShutdownFunc(func(context.Context) error {
		coordinator.shutdown()
		return nil
	})
	convert := func(ctx context.Context, ld plog.Logs) (request.Request, error) {
		descriptors, err := converter(ctx, ld)
		if err != nil {
			return nil, err
		}
		group := &orderedLogsGroup{children: append([]OrderedLogsDescriptor(nil), descriptors...)}
		for i := range group.children {
			child := &group.children[i]
			if child.QueueID == ([16]byte{}) {
				if _, err := rand.Read(child.QueueID[:]); err != nil {
					return nil, consumererror.NewPermanent(fmt.Errorf("generate ordered stream queue item ID: %w", err))
				}
			}
			items, bytes := child.Request.LogRecordCount(), orderedLogsWire.LogsSize(child.Request)+len(child.PartitionKey)+25
			if items > math.MaxInt-group.items || bytes > math.MaxInt-group.bytes {
				return nil, consumererror.NewPermanent(errors.New("ordered stream group size overflow"))
			}
			group.items += items
			group.bytes += bytes
		}
		return group, nil
	}
	logsConsumer, err := consumer.NewLogs(newConsumeLogs(convert, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}
	return &logsExporter{BaseExporter: be, Logs: logsConsumer}, nil
}

var _ queue.ReferenceCounter[request.Request] = orderedLogsReferenceCounter{}
