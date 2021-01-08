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

package fluentforwardreceiver

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tinylib/msgp/msgp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/testutil/logstest"
)

func setupServer(t *testing.T) (func() net.Conn, *consumertest.LogsSink, *observer.ObservedLogs, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	next := new(consumertest.LogsSink)
	logCore, logObserver := observer.New(zap.DebugLevel)
	logger := zap.New(logCore)

	conf := &Config{
		ListenAddress: "127.0.0.1:0",
	}

	receiver, err := newFluentReceiver(logger, conf, next)
	require.NoError(t, err)
	require.NoError(t, receiver.Start(ctx, nil))

	connect := func() net.Conn {
		conn, err := net.Dial("tcp", receiver.(*fluentReceiver).listener.Addr().String())
		require.Nil(t, err)
		return conn
	}

	go func() {
		<-ctx.Done()
		require.NoError(t, receiver.Shutdown(ctx))
	}()

	return connect, next, logObserver, cancel
}

func waitForConnectionClose(t *testing.T, conn net.Conn) {
	one := make([]byte, 1)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err := conn.Read(one)
	// If this is a timeout, then the connection didn't actually close like
	// expected.
	require.Equal(t, io.EOF, err)
}

// Make sure malformed events don't cause panics.
func TestMessageEventConversionMalformed(t *testing.T) {
	connect, _, observedLogs, cancel := setupServer(t)
	defer cancel()

	eventBytes := parseHexDump("testdata/message-event")

	vulnerableBits := []int{0, 1, 14, 59}

	for _, pos := range vulnerableBits {
		eventBytes[pos]++

		conn := connect()
		n, err := conn.Write(eventBytes)
		require.NoError(t, err)
		require.Len(t, eventBytes, n)

		waitForConnectionClose(t, conn)

		require.Len(t, observedLogs.FilterMessageSnippet("Unexpected").All(), 1)
		_ = observedLogs.TakeAll()
	}
}

func TestMessageEvent(t *testing.T) {
	connect, next, _, cancel := setupServer(t)
	defer cancel()

	eventBytes := parseHexDump("testdata/message-event")

	conn := connect()
	n, err := conn.Write(eventBytes)
	require.NoError(t, err)
	require.Equal(t, len(eventBytes), n)
	require.NoError(t, conn.Close())

	var converted []pdata.Logs
	require.Eventually(t, func() bool {
		converted = next.AllLogs()
		return len(converted) == 1
	}, 5*time.Second, 10*time.Millisecond)

	converted[0].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Attributes().Sort()
	require.EqualValues(t, logstest.Logs(logstest.Log{
		Timestamp: 1593031012000000000,
		Body:      pdata.NewAttributeValueString("..."),
		Attributes: map[string]pdata.AttributeValue{
			"container_id":   pdata.NewAttributeValueString("b00a67eb645849d6ab38ff8beb4aad035cc7e917bf123c3e9057c7e89fc73d2d"),
			"container_name": pdata.NewAttributeValueString("/unruffled_cannon"),
			"fluent.tag":     pdata.NewAttributeValueString("b00a67eb6458"),
			"source":         pdata.NewAttributeValueString("stdout"),
		},
	},
	), converted[0])
}

func TestForwardEvent(t *testing.T) {
	connect, next, _, cancel := setupServer(t)
	defer cancel()

	eventBytes := parseHexDump("testdata/forward-event")

	conn := connect()
	n, err := conn.Write(eventBytes)
	require.NoError(t, err)
	require.Equal(t, len(eventBytes), n)
	require.NoError(t, conn.Close())

	var converted []pdata.Logs
	require.Eventually(t, func() bool {
		converted = next.AllLogs()
		return len(converted) == 1
	}, 5*time.Second, 10*time.Millisecond)

	ls := converted[0].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	ls.At(0).Attributes().Sort()
	ls.At(1).Attributes().Sort()
	require.EqualValues(t, logstest.Logs(
		logstest.Log{
			Timestamp: 1593032377776693638,
			Body:      pdata.NewAttributeValueNull(),
			Attributes: map[string]pdata.AttributeValue{
				"Mem.free":   pdata.NewAttributeValueInt(848908),
				"Mem.total":  pdata.NewAttributeValueInt(7155496),
				"Mem.used":   pdata.NewAttributeValueInt(6306588),
				"Swap.free":  pdata.NewAttributeValueInt(0),
				"Swap.total": pdata.NewAttributeValueInt(0),
				"Swap.used":  pdata.NewAttributeValueInt(0),
				"fluent.tag": pdata.NewAttributeValueString("mem.0"),
			},
		},
		logstest.Log{
			Timestamp: 1593032378756829346,
			Body:      pdata.NewAttributeValueNull(),
			Attributes: map[string]pdata.AttributeValue{
				"Mem.free":   pdata.NewAttributeValueInt(848908),
				"Mem.total":  pdata.NewAttributeValueInt(7155496),
				"Mem.used":   pdata.NewAttributeValueInt(6306588),
				"Swap.free":  pdata.NewAttributeValueInt(0),
				"Swap.total": pdata.NewAttributeValueInt(0),
				"Swap.used":  pdata.NewAttributeValueInt(0),
				"fluent.tag": pdata.NewAttributeValueString("mem.0"),
			},
		},
	), converted[0])
}

func TestEventAcknowledgment(t *testing.T) {
	connect, _, logs, cancel := setupServer(t)
	defer func() { fmt.Printf("%v", logs.All()) }()
	defer cancel()

	const chunkValue = "abcdef01234576789"

	var b []byte

	// Make a message event with the chunk option
	b = msgp.AppendArrayHeader(b, 4)
	b = msgp.AppendString(b, "my-tag")
	b = msgp.AppendInt(b, 5000)
	b = msgp.AppendMapHeader(b, 1)
	b = msgp.AppendString(b, "a")
	b = msgp.AppendFloat64(b, 5.0)
	b = msgp.AppendMapStrStr(b, map[string]string{"chunk": chunkValue})

	conn := connect()
	n, err := conn.Write(b)
	require.NoError(t, err)
	require.Equal(t, len(b), n)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	resp := map[string]interface{}{}
	err = msgp.NewReader(conn).ReadMapStrIntf(resp)
	require.NoError(t, err)

	require.Equal(t, chunkValue, resp["ack"])
}

func TestForwardPackedEvent(t *testing.T) {
	connect, next, _, cancel := setupServer(t)
	defer cancel()

	eventBytes := parseHexDump("testdata/forward-packed")

	conn := connect()
	n, err := conn.Write(eventBytes)
	require.NoError(t, err)
	require.Equal(t, len(eventBytes), n)
	require.NoError(t, conn.Close())

	var converted []pdata.Logs
	require.Eventually(t, func() bool {
		converted = next.AllLogs()
		return len(converted) == 1
	}, 5*time.Second, 10*time.Millisecond)

	ls := converted[0].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < ls.Len(); i++ {
		ls.At(i).Attributes().Sort()
	}
	require.EqualValues(t, logstest.Logs(
		logstest.Log{
			Timestamp: 1593032517024597622,
			Body:      pdata.NewAttributeValueString("starting fluentd worker pid=17 ppid=7 worker=0"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
				"pid":        pdata.NewAttributeValueInt(17),
				"ppid":       pdata.NewAttributeValueInt(7),
				"worker":     pdata.NewAttributeValueInt(0),
			},
		},
		logstest.Log{
			Timestamp: 1593032517028573686,
			Body:      pdata.NewAttributeValueString("delayed_commit_timeout is overwritten by ack_response_timeout"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
			},
		},
		logstest.Log{
			Timestamp: 1593032517028815948,
			Body:      pdata.NewAttributeValueString("following tail of /var/log/kern.log"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
			},
		},
		logstest.Log{
			Timestamp: 1593032517031174229,
			Body:      pdata.NewAttributeValueString("fluentd worker is now running worker=0"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
				"worker":     pdata.NewAttributeValueInt(0),
			},
		},
		logstest.Log{
			Timestamp: 1593032522187382822,
			Body:      pdata.NewAttributeValueString("fluentd worker is now stopping worker=0"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
				"worker":     pdata.NewAttributeValueInt(0),
			},
		},
	), converted[0])
}

func TestForwardPackedCompressedEvent(t *testing.T) {
	connect, next, _, cancel := setupServer(t)
	defer cancel()

	eventBytes := parseHexDump("testdata/forward-packed-compressed")

	conn := connect()
	n, err := conn.Write(eventBytes)
	require.NoError(t, err)
	require.Equal(t, len(eventBytes), n)
	require.NoError(t, conn.Close())

	var converted []pdata.Logs
	require.Eventually(t, func() bool {
		converted = next.AllLogs()
		return len(converted) == 1
	}, 5*time.Second, 10*time.Millisecond)

	ls := converted[0].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < ls.Len(); i++ {
		ls.At(i).Attributes().Sort()
	}
	require.EqualValues(t, logstest.Logs(
		logstest.Log{
			Timestamp: 1593032426012197420,
			Body:      pdata.NewAttributeValueString("starting fluentd worker pid=17 ppid=7 worker=0"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
				"pid":        pdata.NewAttributeValueInt(17),
				"ppid":       pdata.NewAttributeValueInt(7),
				"worker":     pdata.NewAttributeValueInt(0),
			},
		},
		logstest.Log{
			Timestamp: 1593032426013724933,
			Body:      pdata.NewAttributeValueString("delayed_commit_timeout is overwritten by ack_response_timeout"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
			},
		},
		logstest.Log{
			Timestamp: 1593032426020510455,
			Body:      pdata.NewAttributeValueString("following tail of /var/log/kern.log"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
			},
		},
		logstest.Log{
			Timestamp: 1593032426024346580,
			Body:      pdata.NewAttributeValueString("fluentd worker is now running worker=0"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
				"worker":     pdata.NewAttributeValueInt(0),
			},
		},
		logstest.Log{
			Timestamp: 1593032434346935532,
			Body:      pdata.NewAttributeValueString("fluentd worker is now stopping worker=0"),
			Attributes: map[string]pdata.AttributeValue{
				"fluent.tag": pdata.NewAttributeValueString("fluent.info"),
				"worker":     pdata.NewAttributeValueInt(0),
			},
		},
	), converted[0])
}

func TestUnixEndpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	next := new(consumertest.LogsSink)

	tmpdir, err := ioutil.TempDir("", "fluent-socket")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	conf := &Config{
		ListenAddress: "unix://" + filepath.Join(tmpdir, "fluent.sock"),
	}

	receiver, err := newFluentReceiver(zap.NewNop(), conf, next)
	require.NoError(t, err)
	require.NoError(t, receiver.Start(ctx, nil))

	conn, err := net.Dial("unix", receiver.(*fluentReceiver).listener.Addr().String())
	require.NoError(t, err)

	n, err := conn.Write(parseHexDump("testdata/message-event"))
	require.NoError(t, err)
	require.Greater(t, n, 0)

	var converted []pdata.Logs
	require.Eventually(t, func() bool {
		converted = next.AllLogs()
		return len(converted) == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func makeSampleEvent(tag string) []byte {
	var b []byte

	b = msgp.AppendArrayHeader(b, 3)
	b = msgp.AppendString(b, tag)
	b = msgp.AppendInt(b, 5000)
	b = msgp.AppendMapHeader(b, 1)
	b = msgp.AppendString(b, "a")
	b = msgp.AppendFloat64(b, 5.0)
	return b
}

func TestHighVolume(t *testing.T) {
	connect, next, _, cancel := setupServer(t)
	defer cancel()

	const totalRoutines = 8
	const totalMessagesPerRoutine = 1000

	var wg sync.WaitGroup
	for i := 0; i < totalRoutines; i++ {
		wg.Add(1)
		go func(num int) {
			conn := connect()
			for j := 0; j < totalMessagesPerRoutine; j++ {
				eventBytes := makeSampleEvent(fmt.Sprintf("tag-%d-%d", num, j))
				n, err := conn.Write(eventBytes)
				require.NoError(t, err)
				require.Equal(t, len(eventBytes), n)
			}
			require.NoError(t, conn.Close())
			wg.Done()
		}(i)
	}

	wg.Wait()

	var converted []pdata.Logs
	require.Eventually(t, func() bool {
		converted = next.AllLogs()

		var total int
		for i := range converted {
			total += converted[i].LogRecordCount()
		}

		return total == totalRoutines*totalMessagesPerRoutine
	}, 10*time.Second, 100*time.Millisecond)
}
