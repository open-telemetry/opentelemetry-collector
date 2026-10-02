// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

var checkpointJournalMagic = []byte{'O', 'C', 'J', 1}

func encodeCheckpointJournal(value []byte, updates []request.QueueItemUpdate) []byte {
	if len(updates) == 0 {
		return value
	}
	var out bytes.Buffer
	out.Write(checkpointJournalMagic)
	_ = binary.Write(&out, binary.BigEndian, uint64(len(value)))
	out.Write(value)
	_ = binary.Write(&out, binary.BigEndian, uint64(len(updates)))
	for _, update := range updates {
		_ = binary.Write(&out, binary.BigEndian, update.Token)
		_ = binary.Write(&out, binary.BigEndian, uint64(len(update.Value)))
		out.Write(update.Value)
	}
	return out.Bytes()
}

func decodeCheckpointJournal(data []byte) ([]byte, []request.QueueItemUpdate, error) {
	if !bytes.HasPrefix(data, checkpointJournalMagic) {
		return data, nil, nil
	}
	r := bytes.NewReader(data[len(checkpointJournalMagic):])
	readValue := func() ([]byte, error) {
		var size uint64
		if err := binary.Read(r, binary.BigEndian, &size); err != nil || size > uint64(r.Len()) {
			return nil, errors.New("invalid checkpoint journal value")
		}
		value := make([]byte, int(size))
		_, err := io.ReadFull(r, value)
		return value, err
	}
	value, err := readValue()
	if err != nil {
		return nil, nil, err
	}
	var count uint64
	if err := binary.Read(r, binary.BigEndian, &count); err != nil || count > uint64(r.Len()/16) {
		return nil, nil, errors.New("invalid checkpoint journal count")
	}
	updates := make([]request.QueueItemUpdate, 0, int(count))
	for range count {
		var token uint64
		if err := binary.Read(r, binary.BigEndian, &token); err != nil {
			return nil, nil, err
		}
		body, err := readValue()
		if err != nil {
			return nil, nil, err
		}
		updates = append(updates, request.QueueItemUpdate{Token: token, Value: body})
	}
	if r.Len() != 0 {
		return nil, nil, errors.New("checkpoint journal has trailing data")
	}
	return value, updates, nil
}
