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

const separator = "\n"

type Extension interface {
	extension.Extension

	// GetClient will create a client for use by the specified component.
	// Each component can have multiple storages (e.g. one for each signal),
	// which can be identified using storageName parameter.
	// The component can use the client to manage state
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
	}

	c.peekChan = make(chan queue.PeekWithCallback)
	c.writeChan = make(chan queue.WriteOp)
	c.writeResponseChan = make(chan error)
	c.exitChan = make(chan int)
	c.callbackChan = make(chan callback)
	c.peekRequestChan = make(chan struct{})
	c.waitForWriteChan = make(chan struct{})
	m, err := c.retrieveMetaData(c.metaDataFilePath())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	c.metadata = *m
	m, err = c.retrievePeekMetaData(c.peekMetaDataFilePath())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	c.peekMetadata = *m
	c.exitWG.Go(c.readLoop)
	c.exitWG.Go(c.writeLoop)
	return c, nil
}

var _ queue.Client = (*diskAccessClient)(nil)

type diskAccessClient struct {
	writeFile             *os.File
	exitChan              chan int
	peekFile              *os.File
	peekRequestChan       chan struct{}
	waitForWriteChan      chan struct{}
	logger                *zap.Logger
	callbackChan          chan callback
	metadataFile          *os.File
	peekMetadataFile      *os.File
	peekChan              chan queue.PeekWithCallback
	writeChan             chan queue.WriteOp
	writeResponseChan     chan error
	dataPath              string
	name                  string
	peekMetadata          metadata
	metadata              metadata
	exitWG                sync.WaitGroup
	maxBytesPerFile       int64
	syncTimeout           time.Duration
	syncEvery             int64
	metadataTruncateEvery int
	metadataWrites        int
	peekMetadataWrites    int
	exitFlag              atomic.Bool
}

func (d *diskAccessClient) Size() int64 {
	return d.metadata.size.Load()
}

func (d *diskAccessClient) Shutdown(_ context.Context) error {
	close(d.exitChan)

	d.exitFlag.Store(true)
	d.exitWG.Wait()

	close(d.peekChan)

	_ = d.sync()
	_ = d.syncPeek()

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

	if d.peekMetadataFile != nil {
		_ = d.peekMetadataFile.Close()
		d.peekMetadataFile = nil
	}

	return nil
}

func (d *diskAccessClient) Peek() chan queue.PeekWithCallback {
	if d.exitFlag.Load() {
		return nil
	}
	d.peekRequestChan <- struct{}{}
	return d.peekChan
}

func (d *diskAccessClient) Write(op queue.WriteOp) error {
	d.writeChan <- op
	return <-d.writeResponseChan
}

var bufPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

type metadata struct {
	fileNum int64
	pos     int64
	size    *atomic.Int64
}

type callback struct {
	pos     int64
	fileNum int64
	size    int64
}

func (d *diskAccessClient) readOne(callbacks map[int64]int) bool {
	peekData, err := d.peekData()
	if err != nil {
		d.logger.Error("error peeking", zap.Error(err))
		return true
	}
	// caught to the head of the queue.
	if len(peekData.Payload) == 0 {
		return false
	}
	messagePeekFileNum := d.peekMetadata.fileNum
	messagePeekPos := d.peekMetadata.pos
	callbacks[messagePeekFileNum]++
	msg := queue.PeekWithCallback{
		Payload: peekData.Payload,
		ConsumeCallback: func(_ error) {
			d.callbackChan <- callback{
				pos:     messagePeekPos,
				fileNum: messagePeekFileNum,
				size:    peekData.Size,
			}
		},
	}
	select {
	case d.peekChan <- msg:
		if d.peekMetadata.pos+int64(len(peekData.Payload)+16) > d.maxBytesPerFile {
			d.peekMetadata.pos = 0
			d.peekMetadata.fileNum++
		} else {
			d.peekMetadata.pos += int64(len(peekData.Payload) + 16)
		}
	case <-d.exitChan:
	}
	return true
}

func (d *diskAccessClient) persistMetaData() error {
	fileName := d.metaDataFilePath()
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
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(d.metadata.fileNum)))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(d.metadata.pos)))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(d.metadata.size.Load())))

	_, err := d.metadataFile.Write(buf.Bytes())
	if err != nil {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
		return err
	}
	err = d.metadataFile.Sync()
	if err != nil {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
		return err
	}
	d.metadataWrites++

	if d.metadataWrites%d.metadataTruncateEvery == 0 {
		_ = d.metadataFile.Close()
		d.metadataFile = nil
		d.metadataWrites = 0
	}

	return nil
}

func (d *diskAccessClient) peekData() (queue.WriteOp, error) {
	var err error
	if d.peekFile == nil {
		curFileName := d.fileName(d.peekMetadata.fileNum)
		d.peekFile, err = os.OpenFile(curFileName, os.O_RDONLY, 0o600) // #nosec G304
		if err != nil {
			return queue.WriteOp{}, err
		}
		d.logger.Debug("peekData() opened", zap.String("name", d.name), zap.String("filename", curFileName))
	}

	readLen := make([]byte, 8)
	_, err = d.peekFile.ReadAt(readLen, d.peekMetadata.pos)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return queue.WriteOp{}, nil
		}
		_ = d.peekFile.Close()
		d.peekFile = nil
		return queue.WriteOp{}, err
	}
	datalen := binary.BigEndian.Uint64(readLen)
	readSize := make([]byte, 8)
	_, err = d.peekFile.ReadAt(readSize, d.peekMetadata.pos+8)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return queue.WriteOp{}, nil
		}
		_ = d.peekFile.Close()
		d.peekFile = nil
		return queue.WriteOp{}, err
	}
	size := binary.BigEndian.Uint64(readSize)
	readBuf := make([]byte, datalen)
	_, err = d.peekFile.ReadAt(readBuf, d.peekMetadata.pos+16)
	if err != nil {
		_ = d.peekFile.Close()
		d.peekFile = nil
		return queue.WriteOp{}, err
	}

	return queue.WriteOp{
		Payload: readBuf,
		Size:    int64(size),
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
			select {
			case d.waitForWriteChan <- struct{}{}:
			default:
			}
		case <-syncTicker.C:
			if opCount == 0 {
				// avoid sync when there's no activity
				continue
			}
			if opCount == d.syncEvery {
				err := d.sync()
				if err != nil {
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
	peekOps := int64(0)
	callbacks := map[int64]int{}
	defer syncTicker.Stop()
	var p chan struct{}
	for {
		select {
		case <-p:
			p = nil
			if d.readOne(callbacks) {
				peekOps++
			} else {
				p = d.waitForWriteChan
			}
		case <-d.peekRequestChan:
			if d.readOne(callbacks) {
				peekOps++
			} else {
				p = d.waitForWriteChan
			}
		case c := <-d.callbackChan:
			callbacks[c.fileNum]--
			d.metadata.size.Add(-c.size)
			if c.fileNum != d.peekMetadata.fileNum && callbacks[c.fileNum] == 0 {
				f := d.fileName(c.fileNum)
				err := os.Remove(f)
				if err != nil && !os.IsNotExist(err) {
					d.logger.Error(" failed to Remove", zap.String("name", d.name), zap.String("filename", f), zap.Error(err))
				}
			}
		case <-syncTicker.C:
			if peekOps == 0 {
				// avoid sync when there's no activity
				continue
			}
			if peekOps == d.syncEvery {
				err := d.syncPeek()
				if err != nil {
					d.logger.Error("failed to sync", zap.String("name", d.name), zap.Error(err))
				}
				peekOps = 0
			}
		case <-d.exitChan:
			return
		}
	}
}

func (d *diskAccessClient) sync() error {
	if d.writeFile != nil {
		err := d.writeFile.Sync()
		if err != nil {
			_ = d.writeFile.Close()
			d.writeFile = nil
			return err
		}
	}
	if err := d.persistMetaData(); err != nil {
		d.logger.Error("error persisting metadata", zap.Error(err))
	}

	return nil
}

func (d *diskAccessClient) syncPeek() error {
	fileName := d.peekMetaDataFilePath()
	if d.peekMetadataFile == nil {
		f, err := os.OpenFile(fileName, os.O_TRUNC|os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304
		if err != nil {
			return err
		}
		d.peekMetadataFile = f
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	buf.WriteString(separator)
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(d.peekMetadata.fileNum)))
	buf.Write(binary.BigEndian.AppendUint64(nil, uint64(d.peekMetadata.pos)))
	_, err := d.peekMetadataFile.Write(buf.Bytes())
	if err != nil {
		_ = d.peekMetadataFile.Close()
		d.peekMetadataFile = nil
		return err
	}
	err = d.peekMetadataFile.Sync()
	if err != nil {
		_ = d.peekMetadataFile.Close()
		d.peekMetadataFile = nil
		return err
	}
	d.peekMetadataWrites++

	if d.peekMetadataWrites%d.metadataTruncateEvery == 0 {
		_ = d.peekMetadataFile.Close()
		d.peekMetadataFile = nil
		d.peekMetadataWrites = 0
	}

	return nil
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
			_, err = d.writeFile.Seek(d.metadata.pos, 0)
			if err != nil {
				_ = d.writeFile.Close()
				d.writeFile = nil
				return err
			}
		}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(dataLen))
	buf.Write(b)
	binary.BigEndian.PutUint64(b, uint64(op.Size))
	buf.Write(b)
	buf.Write(data)
	_, err := d.writeFile.Write(buf.Bytes())
	bufPool.Put(buf)
	if err != nil {
		_ = d.writeFile.Close()
		d.writeFile = nil
		return err
	}

	d.metadata.pos = d.metadata.pos + dataLen + 16
	d.metadata.size.Add(op.Size)

	// will not wrap-around if maxBytesPerFile + maxMsgSize < Int64Max
	if d.metadata.pos > 0 && d.metadata.pos > d.maxBytesPerFile {
		d.metadata.pos = 0
		d.metadata.fileNum++

		// sync every time we start writing to a new file
		err = d.sync()
		if err != nil {
			d.logger.Error(" failed to sync - %s", zap.String("name", d.name), zap.Error(err))
		}

		if d.writeFile != nil {
			_ = d.writeFile.Close()
			d.writeFile = nil
		}
	}

	return err
}

func (d *diskAccessClient) retrievePeekMetaData(fileName string) (*metadata, error) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0o600) // #nosec G304
	if err != nil {
		return &metadata{}, err
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = f.Seek(-16, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	fileNum := binary.BigEndian.Uint64(buf.Bytes()[0:8])
	pos := binary.BigEndian.Uint64(buf.Bytes()[8:])
	return &metadata{
		fileNum: int64(fileNum),
		pos:     int64(pos),
	}, nil
}

func (d *diskAccessClient) retrieveMetaData(fileName string) (*metadata, error) {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0o600) // #nosec G304
	if err != nil {
		return &metadata{size: &atomic.Int64{}}, err
	}
	defer func() {
		_ = f.Close()
	}()
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	_, err = f.Seek(-24, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	fileNum := binary.BigEndian.Uint64(buf.Bytes()[0:8])
	pos := binary.BigEndian.Uint64(buf.Bytes()[8:16])
	size := binary.BigEndian.Uint64(buf.Bytes()[16:24])
	sizeV := &atomic.Int64{}
	sizeV.Store(int64(size))
	return &metadata{
		fileNum: int64(fileNum),
		pos:     int64(pos),
		size:    sizeV,
	}, nil
}

func (d *diskAccessClient) metaDataFilePath() string {
	return path.Join(d.dataPath, d.name+".diskaccess.meta.dat")
}

func (d *diskAccessClient) peekMetaDataFilePath() string {
	return path.Join(d.dataPath, d.name+".diskaccess.peek.dat")
}

func (d *diskAccessClient) fileName(fileNum int64) string {
	return path.Join(d.dataPath, fmt.Sprintf("%s.diskaccess.%06d.dat", d.name, fileNum))
}
