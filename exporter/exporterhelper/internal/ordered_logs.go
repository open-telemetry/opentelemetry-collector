// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
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
	pdatareq "go.opentelemetry.io/collector/pdata/xpdata/request"
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
	// consumption remains single-reader so persistent FIFO order is preserved.
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

func (s OrderedLogsSettings) validate() error {
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
	children        []OrderedLogsDescriptor
	items           int
	bytes           int
	checkpointStore request.QueueCheckpointStore
	persistCtx      context.Context
	queueToken      uint64
	durable         bool
	retired         map[[16]byte]bool

	mu         sync.Mutex
	completion func(error)
	remaining  int
	completed  bool
	firstErr   error
}

var (
	_ request.Request                    = (*orderedLogsGroup)(nil)
	_ request.DeferredQueueCompletion    = (*orderedLogsGroup)(nil)
	_ request.QueueCheckpointStoreSetter = (*orderedLogsGroup)(nil)
)

func (g *orderedLogsGroup) SetQueueItemToken(token uint64) { g.queueToken, g.durable = token, true }
func (g *orderedLogsGroup) QueueRequestsCount() int64      { return int64(len(g.children)) }

func (g *orderedLogsGroup) SetQueueCheckpointStore(store request.QueueCheckpointStore) {
	g.mu.Lock()
	g.checkpointStore = store
	g.mu.Unlock()
}

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

type orderedLogsEncoding struct{}

var (
	orderedLogsMagic = []byte{'O', 'T', 'O', 'G', 3}
	orderedLogsWire  = &plog.ProtoMarshaler{}
)

const orderedLogsCheckpointKey = "ordered-stream-v1"

func (orderedLogsEncoding) Marshal(ctx context.Context, req request.Request) ([]byte, error) {
	g, ok := req.(*orderedLogsGroup)
	if !ok {
		return nil, fmt.Errorf("ordered logs encoding: unexpected request %T", req)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	var out bytes.Buffer
	out.Write(orderedLogsMagic)
	if err := binary.Write(&out, binary.BigEndian, uint32(len(g.children))); err != nil {
		return nil, err
	}
	for _, child := range g.children {
		if child.QueueID == ([16]byte{}) {
			return nil, errors.New("ordered logs encoding: missing queue item ID")
		}
		if len(child.PartitionKey) > math.MaxUint32 {
			return nil, errors.New("ordered logs encoding: partition key too large")
		}
		payload, err := pdatareq.MarshalLogs(ctx, child.Request)
		if err != nil {
			return nil, err
		}
		if len(payload) > math.MaxUint32 {
			return nil, errors.New("ordered logs encoding: child request too large")
		}
		_ = binary.Write(&out, binary.BigEndian, uint32(len(child.PartitionKey)))
		out.WriteString(child.PartitionKey)
		out.WriteByte(byte(child.Position))
		out.Write(child.QueueID[:])
		if g.retired[child.QueueID] {
			out.WriteByte(1)
		} else {
			out.WriteByte(0)
		}
		_ = binary.Write(&out, binary.BigEndian, uint32(len(payload)))
		out.Write(payload)
	}
	return out.Bytes(), nil
}

func (orderedLogsEncoding) Unmarshal(encoded []byte) (context.Context, request.Request, error) {
	if len(encoded) < len(orderedLogsMagic)+4 || !bytes.Equal(encoded[:4], orderedLogsMagic[:4]) || (encoded[4] != 2 && encoded[4] != 3) {
		return nil, nil, errors.New("ordered logs encoding: invalid group header")
	}
	r := bytes.NewReader(encoded[len(orderedLogsMagic):])
	var count uint32
	if err := binary.Read(r, binary.BigEndian, &count); err != nil || count > 1<<20 {
		return nil, nil, errors.New("ordered logs encoding: invalid child count")
	}
	g := &orderedLogsGroup{children: make([]OrderedLogsDescriptor, 0, count), retired: make(map[[16]byte]bool)}
	for range count {
		var keyLen uint32
		if err := binary.Read(r, binary.BigEndian, &keyLen); err != nil || uint64(keyLen) > uint64(r.Len()) {
			return nil, nil, errors.New("ordered logs encoding: invalid partition key length")
		}
		key := make([]byte, keyLen)
		if _, err := io.ReadFull(r, key); err != nil {
			return nil, nil, err
		}
		pos, positionErr := r.ReadByte()
		if positionErr != nil || OrderedPosition(pos) > OrderedPositionEnd {
			return nil, nil, errors.New("ordered logs encoding: invalid stream position")
		}
		var queueID [16]byte
		if _, err := io.ReadFull(r, queueID[:]); err != nil || queueID == ([16]byte{}) {
			return nil, nil, errors.New("ordered logs encoding: invalid queue item ID")
		}
		if encoded[4] >= 3 {
			retired, err := r.ReadByte()
			if err != nil || retired > 1 {
				return nil, nil, errors.New("ordered logs encoding: invalid child retirement flag")
			}
			g.retired[queueID] = retired == 1
		}
		var payloadLen uint32
		if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil || uint64(payloadLen) > uint64(r.Len()) {
			return nil, nil, errors.New("ordered logs encoding: invalid child payload length")
		}
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, nil, err
		}
		ctx, ld, err := pdatareq.UnmarshalLogs(payload)
		if err != nil {
			return nil, nil, err
		}
		if g.persistCtx == nil {
			g.persistCtx = ctx //nolint:fatcontext // Restore the first decoded context; this does not derive a context in the loop.
		}
		g.children = append(g.children, OrderedLogsDescriptor{Request: ld, PartitionKey: string(key), Position: OrderedPosition(pos), QueueID: queueID})
		g.items += ld.LogRecordCount()
		g.bytes += orderedLogsTailSize(OrderedLogsDescriptor{Request: ld, PartitionKey: string(key)})
	}
	if r.Len() != 0 {
		return nil, nil, errors.New("ordered logs encoding: trailing bytes")
	}
	if g.persistCtx == nil {
		g.persistCtx = context.Background()
	}
	return g.persistCtx, g, nil
}

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
		Encoding:         orderedLogsEncoding{},
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
	checkpointRetry      bool
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
	checkpointStore  request.QueueCheckpointStore
	restored         bool
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

func (c *orderedLogsCoordinator) restoreCheckpointLocked(ctx context.Context, store request.QueueCheckpointStore) error {
	if store == nil {
		return nil
	}
	encoded, found, err := store.LoadCheckpoint(ctx, orderedLogsCheckpointKey)
	if err != nil || !found {
		return err
	}
	_, decoded, err := (orderedLogsEncoding{}).Unmarshal(encoded)
	if err != nil {
		return err
	}
	checkpoint := decoded.(*orderedLogsGroup)
	if len(checkpoint.children) > c.settings.MaxActivePartitions {
		return errors.New("checkpoint has more tails than the active-partition limit")
	}
	seen := make(map[string]struct{}, len(checkpoint.children))
	totalBytes := 0
	for _, child := range checkpoint.children {
		if child.PartitionKey == "" || len(child.PartitionKey) > c.settings.MaxPartitionKeyBytes || child.Position != OrderedPositionContinue || child.Request.LogRecordCount() == 0 || child.QueueID == ([16]byte{}) {
			return errors.New("checkpoint contains an invalid recovery tail")
		}
		if _, exists := seen[child.PartitionKey]; exists {
			return errors.New("checkpoint contains duplicate partition tails")
		}
		seen[child.PartitionKey] = struct{}{}
		bytes := orderedLogsTailSize(child)
		if bytes > c.settings.MaxRecoveryTailBytes-totalBytes {
			return errors.New("checkpoint exceeds the recovery-tail byte limit")
		}
		totalBytes += bytes
	}
	for _, child := range checkpoint.children {
		bytes := orderedLogsWire.LogsSize(child.Request)
		pref.RefLogs(child.Request)
		tail := &orderedLogTask{desc: child, bytes: bytes, tailCharge: orderedLogsTailSize(child)}
		partition := &orderedLogsPartition{key: child.PartitionKey, tail: tail, streamOpen: true}
		partition.recoveryPending = &orderedLogTask{desc: child, ctx: ctx, bytes: bytes, tailCharge: tail.tailCharge, recoveryTail: true}
		c.parts[child.PartitionKey] = partition
	}
	c.tailBytes = totalBytes
	return nil
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
	var skipped []*orderedLogTask
	defer func() { c.mu.Unlock(); finishOrderedTasks(skipped, nil) }()
	group.mu.Lock()
	checkpointStore := group.checkpointStore
	group.mu.Unlock()
	if !c.restored {
		if err := c.restoreCheckpointLocked(ctx, checkpointStore); err != nil {
			// Do not let the persistent queue delete this request when recovery
			// state cannot be read. ShutdownErr is the queue's existing signal
			// to leave the dispatched item durable for a later restart.
			return c.fenceRecoveryLocked(fmt.Errorf("restore ordered stream checkpoint: %w", err))
		}
		if group.durable {
			if reader, ok := checkpointStore.(request.QueueItemReader); ok {
				body, err := reader.LoadQueueItem(ctx, group.queueToken)
				if err != nil {
					return c.fenceRecoveryLocked(fmt.Errorf("refresh ordered queue progress: %w", err))
				}
				_, decoded, err := (orderedLogsEncoding{}).Unmarshal(body)
				if err != nil {
					return c.fenceRecoveryLocked(err)
				}
				refreshed := decoded.(*orderedLogsGroup)
				if len(refreshed.children) != len(group.children) {
					return c.fenceRecoveryLocked(errors.New("ordered queue progress changed child count"))
				}
				for i, child := range refreshed.children {
					if child.QueueID != group.children[i].QueueID {
						return c.fenceRecoveryLocked(errors.New("ordered queue progress changed child identity"))
					}
				}
				group.mu.Lock()
				group.retired = refreshed.retired
				group.mu.Unlock()
			}
		}
		c.restored = true
		c.checkpointStore = checkpointStore
	}
	pending := make([]OrderedLogsDescriptor, 0, len(group.children))
	group.mu.Lock()
	for _, child := range group.children {
		if group.retired[child.QueueID] {
			skipped = append(skipped, &orderedLogTask{group: group})
		} else {
			pending = append(pending, child)
		}
	}
	group.mu.Unlock()
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
		c.scheduleLocked(p) //nolint:contextcheck // Queue-owned completion and checkpoint writes must outlive the admission context.
	}
	c.scheduleAvailableLocked()
	return nil
}

// Recovery cannot skip the current FIFO item and continue with its suffix.
// Keep subsequent queue items durable until the exporter is restarted.
func (c *orderedLogsCoordinator) fenceRecoveryLocked(err error) error {
	c.stopped = true
	close(c.stop)
	c.changed.Broadcast()
	return experr.NewShutdownErr(err)
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
		var retired []*orderedLogTask
		if len(p.tasks) > 0 && p.tail != nil && p.tasks[0].desc.QueueID == p.tail.desc.QueueID {
			// A crash can persist the tail checkpoint after its ACK but before
			// the persistent queue deletes that same item. The recovery prelude
			// already replayed it, so retire the duplicate queue handle now.
			p.tasks[0].committed = true
			retired = c.advanceCommittedPrefixLocked(p, epoch)
		}
		c.scheduleLocked(p)
		c.scheduleAvailableLocked()
		c.changed.Broadcast()
		c.mu.Unlock()
		finishOrderedTasks(retired, nil)
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
	if err := c.persistTailSnapshotLocked(p, nextTail, p.tasks[:count]...); err != nil {
		if c.logger != nil {
			c.logger.Warn("Failed to persist ordered stream recovery tail; keeping queue item fenced", zap.Error(err))
		}
		c.scheduleCheckpointRetryLocked(p, epoch, time.Second)
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

func (c *orderedLogsCoordinator) scheduleCheckpointRetryLocked(p *orderedLogsPartition, epoch uint64, delay time.Duration) {
	if p.checkpointRetry || c.stopped {
		return
	}
	p.checkpointRetry = true
	time.AfterFunc(delay, func() {
		c.mu.Lock()
		if c.stopped || p.epoch != epoch {
			p.checkpointRetry = false
			c.mu.Unlock()
			return
		}
		p.checkpointRetry = false
		retired := c.advanceCommittedPrefixLocked(p, epoch)
		c.changed.Broadcast()
		c.scheduleAvailableLocked()
		c.mu.Unlock()
		finishOrderedTasks(retired, nil)
	})
}

func (c *orderedLogsCoordinator) wakeTailBudgetWaitersLocked() {
	for _, p := range c.parts {
		if p.waitingForTailBudget && !p.checkpointRetry {
			p.waitingForTailBudget = false
			c.scheduleCheckpointRetryLocked(p, p.epoch, 0)
		}
	}
}

func (c *orderedLogsCoordinator) persistTailSnapshotLocked(partition *orderedLogsPartition, replacement *OrderedLogsDescriptor, retired ...*orderedLogTask) error {
	groups := make(map[*orderedLogsGroup]struct{})
	for _, task := range retired {
		if task.group == nil {
			continue
		}
		g := task.group
		g.mu.Lock()
		if g.retired == nil {
			g.retired = make(map[[16]byte]bool)
		}
		g.retired[task.desc.QueueID] = true
		g.mu.Unlock()
		groups[g] = struct{}{}
	}
	rollback := func() {
		for _, task := range retired {
			if task.group != nil {
				task.group.mu.Lock()
				delete(task.group.retired, task.desc.QueueID)
				task.group.mu.Unlock()
			}
		}
	}
	if c.checkpointStore == nil {
		return nil
	}
	var updates []request.QueueItemUpdate
	for g := range groups {
		if !g.durable {
			continue
		}
		persistCtx := g.persistCtx
		if persistCtx == nil {
			persistCtx = context.Background()
		}
		value, err := (orderedLogsEncoding{}).Marshal(persistCtx, g)
		if err != nil {
			rollback()
			return err
		}
		updates = append(updates, request.QueueItemUpdate{Token: g.queueToken, Value: value})
	}
	keys := make([]string, 0, len(c.parts))
	tails := make(map[string]OrderedLogsDescriptor, len(c.parts))
	for key, part := range c.parts {
		if part.tail != nil {
			tails[key] = part.tail.desc
		}
	}
	if replacement == nil {
		delete(tails, partition.key)
	} else {
		tails[partition.key] = *replacement
	}
	for key := range tails {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	checkpoint := &orderedLogsGroup{children: make([]OrderedLogsDescriptor, 0, len(keys))}
	for _, key := range keys {
		desc := tails[key]
		checkpoint.children = append(checkpoint.children, desc)
	}
	encoded, err := (orderedLogsEncoding{}).Marshal(context.Background(), checkpoint)
	if err != nil {
		rollback()
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if transaction, ok := c.checkpointStore.(request.QueueCheckpointTransaction); ok {
		err = transaction.SaveCheckpointAndItems(ctx, orderedLogsCheckpointKey, encoded, updates)
	} else if len(updates) > 0 {
		err = errors.New("ordered queue cannot atomically checkpoint child progress")
	} else {
		err = c.checkpointStore.SaveCheckpoint(ctx, orderedLogsCheckpointKey, encoded)
	}
	if err != nil {
		rollback()
	}
	return err
}

func (c *orderedLogsCoordinator) fail(p *orderedLogsPartition, task *orderedLogTask, epoch uint64, err error) {
	c.mu.Lock()
	if p.epoch != epoch || task.committed || c.stopped {
		c.mu.Unlock()
		return
	}
	if consumererror.IsPermanent(err) || !c.retryCfg.Enabled || c.retryExpiredLocked(p) {
		failed, persistErr := c.terminatePartitionLocked(p)
		c.mu.Unlock()
		if persistErr != nil {
			err = experr.NewShutdownErr(persistErr)
		}
		finishOrderedTasks(failed, err)
		return
	}
	if p.backoff == nil {
		p.retryStarted = time.Now()
		p.backoff = &backoff.ExponentialBackOff{InitialInterval: c.retryCfg.InitialInterval, RandomizationFactor: c.retryCfg.RandomizationFactor, Multiplier: c.retryCfg.Multiplier, MaxInterval: c.retryCfg.MaxInterval}
	}
	delay := p.backoff.NextBackOff()
	if delay == backoff.Stop {
		failed, persistErr := c.terminatePartitionLocked(p)
		c.mu.Unlock()
		if persistErr != nil {
			err = experr.NewShutdownErr(persistErr)
		}
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

func (c *orderedLogsCoordinator) terminatePartitionLocked(p *orderedLogsPartition) ([]*orderedLogTask, error) {
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
	persistErr := c.persistTailSnapshotLocked(p, nil, failed...)
	if persistErr != nil && c.logger != nil {
		c.logger.Warn("Failed to persist terminal ordered stream outcome", zap.Error(persistErr))
	}
	c.clearTailMemoryLocked(p)
	delete(c.parts, p.key)
	c.changed.Broadcast()
	c.scheduleAvailableLocked()
	return failed, persistErr
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
	// retained beside it in the recovery checkpoint.
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
			// Keep the durable checkpoint. It is needed if an open stream is
			// restarted after this exporter shuts down.
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
	if err := limits.validate(); err != nil {
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
		coordinator.shutdown() //nolint:contextcheck // Durable cleanup uses a bounded context independent of shutdown cancellation.
		return nil
	})
	convert := func(ctx context.Context, ld plog.Logs) (request.Request, error) {
		descriptors, err := converter(ctx, ld)
		if err != nil {
			return nil, err
		}
		group := &orderedLogsGroup{children: append([]OrderedLogsDescriptor(nil), descriptors...), persistCtx: ctx}
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

var (
	_ queue.Encoding[request.Request]         = orderedLogsEncoding{}
	_ queue.ReferenceCounter[request.Request] = orderedLogsReferenceCounter{}
)
