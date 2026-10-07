// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diskqueueextension // import "go.opentelemetry.io/collector/extension/diskqueueextension"

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/queue"
)

const (
	separator         = "\n"
	recordHeaderSize  = int64(24)
	metadataValueSize = 32
)

var errClientClosed = errors.New("disk queue client is closed")

type Extension interface {
	extension.Extension

	// GetClient will create a client for use by the specified component.
	// Each component can have multiple storages (e.g. one for each signal),
	// which can be identified using storageName parameter.
	GetClient(ctx context.Context, kind component.Kind, id component.ID, storageName string) (queue.Client, error)
}

var _ Extension = (*diskAccessExtension)(nil)

type diskAccessExtension struct {
	cfg    *Config
	logger *zap.Logger
}

func (d *diskAccessExtension) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (d *diskAccessExtension) Shutdown(_ context.Context) error {
	return nil
}

func (d *diskAccessExtension) GetClient(_ context.Context, _ component.Kind, _ component.ID, storageName string) (queue.Client, error) {
	c := &diskAccessClient{
		dataPath:              d.cfg.DataPath,
		name:                  storageName,
		maxBytesPerFile:       d.cfg.MaxBytesPerFile,
		syncTimeout:           d.cfg.SyncTimeout,
		syncEvery:             d.cfg.SyncEvery,
		metadataTruncateEvery: d.cfg.MetadataTruncateEvery,
		exitFlag:              atomic.Bool{},
		logger:                d.logger,
		pendingEntries:        make(map[uint64]queueEntry),
		completed:             make(map[uint64]struct{}),
	}

	c.peekChan = make(chan queue.PeekWithCallback)
	c.writeChan = make(chan queue.WriteOp)
	c.writeResponseChan = make(chan error, 1)
	c.exitChan = make(chan int)
	c.callbackChan = make(chan callback, 128)
	c.peekRequestChan = make(chan struct{})
	c.waitForWriteChan = make(chan struct{})

	m, err := c.retrieveMetadata(c.metadataFilePath())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if m.size == nil {
		m.size = &atomic.Int64{}
	}

	p, err := c.retrieveCompletionMetadata(c.completionMetadataFilePath())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if p.size == nil {
		p.size = &atomic.Int64{}
	}

	// The write metadata stores the cumulative size appended to the queue,
	// while the completion metadata stores the cumulative size acknowledged by
	// consumers. Their difference is the queue size after a restart, even when
	// the two metadata files were synced at different times.
	queueSize := m.totalSize - p.totalSize
	if queueSize < 0 {
		queueSize = 0
	}
	m.size.Store(queueSize)
	p.size.Store(queueSize)

	c.metadata = *m
	c.completionMetadata = *p
	c.peekMetadata = *p
	c.exitWG.Go(c.readLoop)
	c.exitWG.Go(c.writeLoop)
	return c, nil
}

var _ queue.Client = (*diskAccessClient)(nil)

type diskAccessClient struct {
	writeFile              *os.File
	exitChan               chan int
	peekFile               *os.File
	peekFileNum            int64
	peekRequestChan        chan struct{}
	waitForWriteChan       chan struct{}
	logger                 *zap.Logger
	callbackChan           chan callback
	metadataFile           *os.File
	completionMetadataFile *os.File
	peekChan               chan queue.PeekWithCallback
	writeChan              chan queue.WriteOp
	writeResponseChan      chan error
	dataPath               string
	name                   string
	// peekMetadata is the in-memory dispatch/read head. It is deliberately
	// not persisted: after a restart, entries after completionMetadata are replayed.
	peekMetadata metadata
	// completionMetadata is the persisted contiguous completion state.
	completionMetadata metadata
	metadata           metadata
	// pendingEntries contains all peeked entries past the completionMetadata position.
	pendingEntries           map[uint64]queueEntry
	completed                map[uint64]struct{}
	exitWG                   sync.WaitGroup
	maxBytesPerFile          int64
	syncTimeout              time.Duration
	syncEvery                int64
	metadataTruncateEvery    int
	metadataWrites           int
	completionMetadataWrites int
	exitFlag                 atomic.Bool
}

func (d *diskAccessClient) Size() int64 {
	if d.metadata.size == nil {
		return 0
	}
	return d.metadata.size.Load()
}

func (d *diskAccessClient) Shutdown(_ context.Context) error {
	if d.exitFlag.Swap(true) {
		return nil
	}
	close(d.exitChan)
	d.exitWG.Wait()
	close(d.peekChan)

	_ = d.sync()
	_ = d.syncCompletionMetadata()

	if d.writeFile != nil {
		_ = d.writeFile.Close()
		d.writeFile = nil
	}
	if d.peekFile != nil {
		_ = d.peekFile.Close()
		d.peekFile = nil
	}
	if d.metadataFile != nil {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
	}
	if d.completionMetadataFile != nil {
		_ = d.completionMetadataFile.Close()
		d.completionMetadataFile = nil
	}
	return nil
}

func (d *diskAccessClient) Peek() chan queue.PeekWithCallback {
	if d.exitFlag.Load() {
		return nil
	}
	select {
	case d.peekRequestChan <- struct{}{}:
		return d.peekChan
	case <-d.exitChan:
		return nil
	}
}

func (d *diskAccessClient) Write(op queue.WriteOp) error {
	if d.exitFlag.Load() {
		return errClientClosed
	}
	select {
	case d.writeChan <- op:
	case <-d.exitChan:
		return errClientClosed
	}
	select {
	case err := <-d.writeResponseChan:
		return err
	case <-d.exitChan:
		return errClientClosed
	}
}

var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

type metadata struct {
	fileNum      int64
	pos          int64
	logicalIndex uint64
	totalSize    int64
	size         *atomic.Int64
}

type queuePosition struct {
	fileNum int64
	pos     int64
}

type queueEntry struct {
	start        queuePosition
	next         queuePosition
	logicalIndex uint64
	size         int64
}

type queueRecord struct {
	Payload []byte
	Size    int64
	entry   queueEntry
}

type callback struct {
	logicalIndex uint64
}

func (d *diskAccessClient) readOne(_ map[int64]int) bool {
	record, err := d.peekData()
	if err != nil {
		if !os.IsNotExist(err) {
			d.logger.Error("error peeking", zap.Error(err))
			return true
		}
		return false
	}
	if len(record.Payload) == 0 {
		return false
	}

	entry := record.entry
	msg := queue.PeekWithCallback{
		Payload: record.Payload,
		ConsumeCallback: func(_ error) {
			select {
			case d.callbackChan <- callback{logicalIndex: entry.logicalIndex}:
			case <-d.exitChan:
				// A completion that races with shutdown is intentionally not
				// persisted. It will be replayed after restart.
			}
		},
	}
	select {
	case d.peekChan <- msg:
		d.pendingEntries[entry.logicalIndex] = entry
		d.peekMetadata.fileNum = entry.next.fileNum
		d.peekMetadata.pos = entry.next.pos
		d.peekMetadata.logicalIndex = entry.logicalIndex + 1
	case <-d.exitChan:
	}
	return true
}

func (d *diskAccessClient) persistMetadata() error {
	fileName := d.metadataFilePath()
	if d.metadataFile == nil {
		f, err := os.OpenFile(fileName, os.O_TRUNC|os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304
		if err != nil {
			return err
		}
		d.metadataFile = f
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	buf.WriteString(separator)
	d.appendMetadata(buf, d.metadata)

	if _, err := d.metadataFile.Write(buf.Bytes()); err != nil {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
		return err
	}
	if err := d.metadataFile.Sync(); err != nil {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
		return err
	}
	d.metadataWrites++
	if d.metadataTruncateEvery > 0 && d.metadataWrites%d.metadataTruncateEvery == 0 {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
		d.metadataWrites = 0
	}
	return nil
}

func (d *diskAccessClient) peekData() (queueRecord, error) {
	if d.peekFile != nil && d.peekFileNum != d.peekMetadata.fileNum {
		_ = d.peekFile.Close()
		d.peekFile = nil
	}
	if d.peekFile == nil {
		curFileName := d.fileName(d.peekMetadata.fileNum)
		f, err := os.OpenFile(curFileName, os.O_RDONLY, 0o600) // #nosec G304
		if err != nil {
			return queueRecord{}, err
		}
		d.peekFile = f
		d.peekFileNum = d.peekMetadata.fileNum
		d.logger.Debug("peekData() opened", zap.String("name", d.name), zap.String("filename", curFileName))
	}

	header := make([]byte, recordHeaderSize)
	_, err := d.peekFile.ReadAt(header, d.peekMetadata.pos)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return queueRecord{}, nil
		}
		_ = d.peekFile.Close()
		d.peekFile = nil
		return queueRecord{}, err
	}

	logicalIndex := binary.BigEndian.Uint64(header[0:8])
	dataLen := binary.BigEndian.Uint64(header[8:16])
	size := binary.BigEndian.Uint64(header[16:24])
	readBuf := make([]byte, dataLen)
	if _, err = d.peekFile.ReadAt(readBuf, d.peekMetadata.pos+recordHeaderSize); err != nil {
		_ = d.peekFile.Close()
		d.peekFile = nil
		return queueRecord{}, err
	}

	start := queuePosition{fileNum: d.peekMetadata.fileNum, pos: d.peekMetadata.pos}
	next := queuePosition{
		fileNum: d.peekMetadata.fileNum,
		pos:     d.peekMetadata.pos + recordHeaderSize + int64(dataLen),
	}
	if next.pos > d.maxBytesPerFile {
		next.fileNum++
		next.pos = 0
	}
	return queueRecord{
		Payload: readBuf,
		Size:    int64(size),
		entry: queueEntry{
			start:        start,
			next:         next,
			logicalIndex: logicalIndex,
			size:         int64(size),
		},
	}, nil
}

func (d *diskAccessClient) writeLoop() {
	syncTicker := time.NewTicker(d.syncTimeout)
	defer syncTicker.Stop()
	opCount := int64(0)
	for {
		select {
		case msg := <-d.writeChan:
			opCount++
			err := d.write(msg)
			d.writeResponseChan <- err
			if err == nil {
				select {
				case d.waitForWriteChan <- struct{}{}:
				default:
				}
			}
		case <-syncTicker.C:
			if opCount == 0 {
				continue
			}
			if opCount >= d.syncEvery {
				if err := d.sync(); err != nil {
					d.logger.Error("failed to sync", zap.String("name", d.name), zap.Error(err))
				}
				opCount = 0
			}
		case <-d.exitChan:
			return
		}
	}
}

func (d *diskAccessClient) readLoop() {
	syncTicker := time.NewTicker(d.syncTimeout)
	defer syncTicker.Stop()
	completionOps := int64(0)
	var p chan struct{}
	for {
		select {
		case <-p:
			p = nil
			if !d.readOne(nil) {
				p = d.waitForWriteChan
			}
		case <-d.peekRequestChan:
			if !d.readOne(nil) {
				p = d.waitForWriteChan
			}
		case c := <-d.callbackChan:
			if d.complete(c.logicalIndex) {
				completionOps++
				if completionOps >= d.syncEvery {
					if err := d.syncCompletionMetadata(); err != nil {
						d.logger.Error("failed to sync", zap.String("name", d.name), zap.Error(err))
					}
					completionOps = 0
				}
			}
		case <-syncTicker.C:
			if completionOps == 0 {
				continue
			}
			if err := d.syncCompletionMetadata(); err != nil {
				d.logger.Error("failed to sync", zap.String("name", d.name), zap.Error(err))
			}
			completionOps = 0
		case <-d.exitChan:
			return
		}
	}
}

func (d *diskAccessClient) complete(logicalIndex uint64) bool {
	if _, ok := d.pendingEntries[logicalIndex]; !ok {
		return false
	}
	d.completed[logicalIndex] = struct{}{}
	advanced := false
	for {
		current := d.completionMetadata.logicalIndex
		if _, ok := d.completed[current]; !ok {
			break
		}
		entry, ok := d.pendingEntries[current]
		if !ok {
			break
		}
		delete(d.completed, current)
		delete(d.pendingEntries, current)
		d.completionMetadata.fileNum = entry.next.fileNum
		d.completionMetadata.pos = entry.next.pos
		d.completionMetadata.logicalIndex++
		d.completionMetadata.totalSize += entry.size
		d.metadata.size.Add(-entry.size)
		advanced = true
	}
	if advanced {
		d.completionMetadata.size.Store(d.metadata.size.Load())
		d.removeFilesBefore(d.completionMetadata.fileNum)
	}
	return advanced
}

func (d *diskAccessClient) removeFilesBefore(fileNum int64) {
	if d.peekFile != nil && d.peekFileNum < fileNum {
		_ = d.peekFile.Close()
		d.peekFile = nil
	}
	for n := int64(0); n < fileNum; n++ {
		f := d.fileName(n)
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			d.logger.Error("failed to remove", zap.String("name", d.name), zap.String("filename", f), zap.Error(err))
		}
	}
}

func (d *diskAccessClient) sync() error {
	if d.writeFile != nil {
		if err := d.writeFile.Sync(); err != nil {
			_ = d.writeFile.Close()
			d.writeFile = nil
			return err
		}
	}
	return d.persistMetadata()
}

func (d *diskAccessClient) syncCompletionMetadata() error {
	fileName := d.completionMetadataFilePath()
	if d.completionMetadataFile == nil {
		f, err := os.OpenFile(fileName, os.O_TRUNC|os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304
		if err != nil {
			return err
		}
		d.completionMetadataFile = f
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	buf.WriteString(separator)
	d.appendMetadata(buf, d.completionMetadata)
	if _, err := d.completionMetadataFile.Write(buf.Bytes()); err != nil {
		_ = d.completionMetadataFile.Close()
		d.completionMetadataFile = nil
		return err
	}
	if err := d.completionMetadataFile.Sync(); err != nil {
		_ = d.completionMetadataFile.Close()
		d.completionMetadataFile = nil
		return err
	}
	d.completionMetadataWrites++
	if d.metadataTruncateEvery > 0 && d.completionMetadataWrites%d.metadataTruncateEvery == 0 {
		_ = d.completionMetadataFile.Close()
		d.completionMetadataFile = nil
		d.completionMetadataWrites = 0
	}
	return nil
}

func (d *diskAccessClient) appendMetadata(buf *bytes.Buffer, m metadata) {
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(m.fileNum)))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(m.pos)))
	buf.Write(binary.BigEndian.AppendUint64(nil, m.logicalIndex))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(m.totalSize)))
}

func (d *diskAccessClient) write(op queue.WriteOp) error {
	data := op.Payload
	dataLen := int64(len(data))
	if d.writeFile == nil {
		curFileName := d.fileName(d.metadata.fileNum)
		var err error
		d.writeFile, err = os.OpenFile(curFileName, os.O_RDWR|os.O_CREATE, 0o600) // #nosec G304
		if err != nil {
			return err
		}
		d.logger.Debug("writeOne() opened", zap.String("name", d.name), zap.String("filename", curFileName))
		if d.metadata.pos > 0 {
			if _, err = d.writeFile.Seek(d.metadata.pos, 0); err != nil {
				_ = d.writeFile.Close()
				d.writeFile = nil
				return err
			}
		}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(binary.BigEndian.AppendUint64(nil, d.metadata.logicalIndex))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(dataLen)))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(op.Size)))
	buf.Write(data)
	_, err := d.writeFile.Write(buf.Bytes())
	bufPool.Put(buf)
	if err != nil {
		_ = d.writeFile.Close()
		d.writeFile = nil
		return err
	}

	d.metadata.pos += dataLen + recordHeaderSize
	d.metadata.logicalIndex++
	d.metadata.totalSize += op.Size
	d.metadata.size.Add(op.Size)
	if d.metadata.pos > 0 && d.metadata.pos > d.maxBytesPerFile {
		d.metadata.pos = 0
		d.metadata.fileNum++
		err = d.sync()
		if err != nil {
			d.logger.Error("failed to sync", zap.String("name", d.name), zap.Error(err))
		}
		if d.writeFile != nil {
			_ = d.writeFile.Close()
			d.writeFile = nil
		}
	}
	return err
}

func (d *diskAccessClient) retrieveCompletionMetadata(fileName string) (*metadata, error) {
	return d.retrieveMetadataFile(fileName)
}

func (d *diskAccessClient) retrieveMetadata(fileName string) (*metadata, error) {
	return d.retrieveMetadataFile(fileName)
}

func (d *diskAccessClient) retrieveMetadataFile(fileName string) (*metadata, error) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0o600) // #nosec G304
	if err != nil {
		return &metadata{size: &atomic.Int64{}}, err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err = f.Seek(-metadataValueSize, io.SeekEnd); err != nil {
		return nil, err
	}
	data := make([]byte, metadataValueSize)
	if _, err = io.ReadFull(f, data); err != nil {
		return nil, err
	}
	totalSize := int64(binary.BigEndian.Uint64(data[24:32]))
	size := &atomic.Int64{}
	size.Store(totalSize)
	return &metadata{
		fileNum:      int64(binary.BigEndian.Uint64(data[0:8])),
		pos:          int64(binary.BigEndian.Uint64(data[8:16])),
		logicalIndex: binary.BigEndian.Uint64(data[16:24]),
		totalSize:    totalSize,
		size:         size,
	}, nil
}

func (d *diskAccessClient) metadataFilePath() string {
	return path.Join(d.dataPath, d.name+".diskaccess.meta.dat")
}

func (d *diskAccessClient) completionMetadataFilePath() string {
	// This file stores the durable completion head.
	return path.Join(d.dataPath, d.name+".diskaccess.completion.dat")
}

func (d *diskAccessClient) fileName(fileNum int64) string {
	return path.Join(d.dataPath, fmt.Sprintf("%s.diskaccess.%06d.dat", d.name, fileNum))
}
