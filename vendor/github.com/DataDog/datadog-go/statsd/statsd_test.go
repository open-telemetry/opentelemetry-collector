// Copyright 2013 Ooyala, Inc.

package statsd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var dogstatsdTests = []struct {
	GlobalNamespace string
	GlobalTags      []string
	Method          string
	Metric          string
	Value           interface{}
	Tags            []string
	Rate            float64
	Expected        string
}{
	{"", nil, "Gauge", "test.gauge", 1.0, nil, 1.0, "test.gauge:1.000000|g"},
	{"", nil, "Gauge", "test.gauge", 1.0, nil, 0.999999, "test.gauge:1.000000|g|@0.999999"},
	{"", nil, "Gauge", "test.gauge", 1.0, []string{"tagA"}, 1.0, "test.gauge:1.000000|g|#tagA"},
	{"", nil, "Gauge", "test.gauge", 1.0, []string{"tagA", "tagB"}, 1.0, "test.gauge:1.000000|g|#tagA,tagB"},
	{"", nil, "Gauge", "test.gauge", 1.0, []string{"tagA"}, 0.999999, "test.gauge:1.000000|g|@0.999999|#tagA"},
	{"", nil, "Count", "test.count", int64(1), []string{"tagA"}, 1.0, "test.count:1|c|#tagA"},
	{"", nil, "Count", "test.count", int64(-1), []string{"tagA"}, 1.0, "test.count:-1|c|#tagA"},
	{"", nil, "Histogram", "test.histogram", 2.3, []string{"tagA"}, 1.0, "test.histogram:2.300000|h|#tagA"},
	{"", nil, "Distribution", "test.distribution", 2.3, []string{"tagA"}, 1.0, "test.distribution:2.300000|d|#tagA"},
	{"", nil, "Set", "test.set", "uuid", []string{"tagA"}, 1.0, "test.set:uuid|s|#tagA"},
	{"flubber.", nil, "Set", "test.set", "uuid", []string{"tagA"}, 1.0, "flubber.test.set:uuid|s|#tagA"},
	{"", []string{"tagC"}, "Set", "test.set", "uuid", []string{"tagA"}, 1.0, "test.set:uuid|s|#tagC,tagA"},
	{"", nil, "Count", "test.count", int64(1), []string{"hello\nworld"}, 1.0, "test.count:1|c|#helloworld"},
}

func assertNotPanics(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()
	f()
}

func TestClientUDP(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client, err := New(addr)
	if err != nil {
		t.Fatal(err)
	}

	clientTest(t, server, client)
}

type statsdWriterWrapper struct {
	io.WriteCloser
}

func (statsdWriterWrapper) SetWriteTimeout(time.Duration) error {
	return nil
}

func TestClientWithConn(t *testing.T) {
	server, conn, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewWithWriter(statsdWriterWrapper{conn})
	if err != nil {
		t.Fatal(err)
	}

	clientTest(t, server, client)
}

func clientTest(t *testing.T, server io.Reader, client *Client) {
	for _, tt := range dogstatsdTests {
		client.Namespace = tt.GlobalNamespace
		client.Tags = tt.GlobalTags
		method := reflect.ValueOf(client).MethodByName(tt.Method)
		e := method.Call([]reflect.Value{
			reflect.ValueOf(tt.Metric),
			reflect.ValueOf(tt.Value),
			reflect.ValueOf(tt.Tags),
			reflect.ValueOf(tt.Rate)})[0]
		errInter := e.Interface()
		if errInter != nil {
			t.Fatal(errInter.(error))
		}

		bytes := make([]byte, 1024)
		n, err := server.Read(bytes)
		if err != nil {
			t.Fatal(err)
		}
		message := bytes[:n]
		if string(message) != tt.Expected {
			t.Errorf("Expected: %s. Actual: %s", tt.Expected, string(message))
		}
	}
}

func TestClientUDS(t *testing.T) {
	dir, err := ioutil.TempDir("", "socket")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	addr := filepath.Join(dir, "dsd.socket")

	udsAddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUnixgram("unixgram", udsAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	addrParts := []string{UnixAddressPrefix, addr}
	client, err := New(strings.Join(addrParts, ""))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range dogstatsdTests {
		client.Namespace = tt.GlobalNamespace
		client.Tags = tt.GlobalTags
		method := reflect.ValueOf(client).MethodByName(tt.Method)
		e := method.Call([]reflect.Value{
			reflect.ValueOf(tt.Metric),
			reflect.ValueOf(tt.Value),
			reflect.ValueOf(tt.Tags),
			reflect.ValueOf(tt.Rate)})[0]
		errInter := e.Interface()
		if errInter != nil {
			t.Fatal(errInter.(error))
		}

		bytes := make([]byte, 1024)
		n, err := server.Read(bytes)
		if err != nil {
			t.Fatal(err)
		}
		message := bytes[:n]
		if string(message) != tt.Expected {
			t.Errorf("Expected: %s. Actual: %s", tt.Expected, string(message))
		}
	}
}

func TestClientUDSClose(t *testing.T) {
	dir, err := ioutil.TempDir("", "socket")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	addr := filepath.Join(dir, "dsd.socket")

	addrParts := []string{UnixAddressPrefix, addr}
	client, err := New(strings.Join(addrParts, ""))
	if err != nil {
		t.Fatal(err)
	}

	assertNotPanics(t, func() { client.Close() })
}

func TestBufferedClient(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	bufferLength := 9
	client, err := NewBuffered(addr, bufferLength)
	if err != nil {
		t.Fatal(err)
	}

	client.Namespace = "foo."
	client.Tags = []string{"dd:2"}

	dur, _ := time.ParseDuration("123us")

	client.Incr("ic", nil, 1)
	client.Decr("dc", nil, 1)
	client.Count("cc", 1, nil, 1)
	client.Gauge("gg", 10, nil, 1)
	client.Histogram("hh", 1, nil, 1)
	client.Distribution("dd", 1, nil, 1)
	client.Timing("tt", dur, nil, 1)
	client.Set("ss", "ss", nil, 1)

	if len(client.commands) != (bufferLength - 1) {
		t.Errorf("Expected client to have buffered %d commands, but found %d\n", (bufferLength - 1), len(client.commands))
	}

	client.Set("ss", "xx", nil, 1)
	client.Lock()
	err = client.flushLocked()
	client.Unlock()
	if err != nil {
		t.Errorf("Error sending: %s", err)
	}

	if len(client.commands) != 0 {
		t.Errorf("Expecting send to flush commands, but found %d\n", len(client.commands))
	}

	buffer := make([]byte, 4096)
	n, err := io.ReadAtLeast(server, buffer, 1)
	result := string(buffer[:n])

	if err != nil {
		t.Error(err)
	}

	expected := []string{
		`foo.ic:1|c|#dd:2`,
		`foo.dc:-1|c|#dd:2`,
		`foo.cc:1|c|#dd:2`,
		`foo.gg:10.000000|g|#dd:2`,
		`foo.hh:1.000000|h|#dd:2`,
		`foo.dd:1.000000|d|#dd:2`,
		`foo.tt:0.123000|ms|#dd:2`,
		`foo.ss:ss|s|#dd:2`,
		`foo.ss:xx|s|#dd:2`,
	}

	for i, res := range strings.Split(result, "\n") {
		if res != expected[i] {
			t.Errorf("Got `%s`, expected `%s`", res, expected[i])
		}
	}

	client.Event(&Event{Title: "title1", Text: "text1", Priority: Normal, AlertType: Success, Tags: []string{"tagg"}})
	client.SimpleEvent("event1", "text1")

	if len(client.commands) != 2 {
		t.Errorf("Expected to find %d commands, but found %d\n", 2, len(client.commands))
	}

	client.Lock()
	err = client.flushLocked()
	client.Unlock()

	if err != nil {
		t.Errorf("Error sending: %s", err)
	}

	if len(client.commands) != 0 {
		t.Errorf("Expecting send to flush commands, but found %d\n", len(client.commands))
	}

	buffer = make([]byte, 1024)
	n, err = io.ReadAtLeast(server, buffer, 1)
	result = string(buffer[:n])

	if err != nil {
		t.Error(err)
	}

	if n == 0 {
		t.Errorf("Read 0 bytes but expected more.")
	}

	expected = []string{
		`_e{6,5}:title1|text1|p:normal|t:success|#dd:2,tagg`,
		`_e{6,5}:event1|text1|#dd:2`,
	}

	for i, res := range strings.Split(result, "\n") {
		if res != expected[i] {
			t.Errorf("Got `%s`, expected `%s`", res, expected[i])
		}
	}

}

func TestBufferedClientBackground(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	bufferLength := 5
	client, err := NewBuffered(addr, bufferLength)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	client.Namespace = "foo."
	client.Tags = []string{"dd:2"}

	client.Count("cc", 1, nil, 1)
	client.Gauge("gg", 10, nil, 1)
	client.Histogram("hh", 1, nil, 1)
	client.Distribution("dd", 1, nil, 1)
	client.Set("ss", "ss", nil, 1)
	client.Set("ss", "xx", nil, 1)

	time.Sleep(client.flushTime * 2)
	client.Lock()
	if len(client.commands) != 0 {
		t.Errorf("Watch goroutine should have flushed commands, but found %d\n", len(client.commands))
	}
	client.Unlock()
}

func TestBufferedClientFlush(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	bufferLength := 5
	client, err := NewBuffered(addr, bufferLength)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	client.Namespace = "foo."
	client.Tags = []string{"dd:2"}

	client.Count("cc", 1, nil, 1)
	client.Gauge("gg", 10, nil, 1)
	client.Histogram("hh", 1, nil, 1)
	client.Distribution("dd", 1, nil, 1)
	client.Set("ss", "ss", nil, 1)
	client.Set("ss", "xx", nil, 1)

	client.Flush()

	client.Lock()
	if len(client.commands) != 0 {
		t.Errorf("Flush should have flushed commands, but found %d\n", len(client.commands))
	}
	client.Unlock()
}

func TestJoinMaxSize(t *testing.T) {
	c := Client{}
	elements := []string{"abc", "abcd", "ab", "xyz", "foobaz", "x", "wwxxyyzz"}
	res, n := c.joinMaxSize(elements, " ", 8)

	if len(res) != len(n) && len(res) != 4 {
		t.Errorf("Was expecting 4 frames to flush but got: %v - %v", n, res)
	}
	if n[0] != 2 {
		t.Errorf("Was expecting 2 elements in first frame but got: %v", n[0])
	}
	if string(res[0]) != "abc abcd" {
		t.Errorf("Join should have returned \"abc abcd\" in frame, but found: %s", res[0])
	}
	if n[1] != 2 {
		t.Errorf("Was expecting 2 elements in second frame but got: %v - %v", n[1], n)
	}
	if string(res[1]) != "ab xyz" {
		t.Errorf("Join should have returned \"ab xyz\" in frame, but found: %s", res[1])
	}
	if n[2] != 2 {
		t.Errorf("Was expecting 2 elements in third frame but got: %v - %v", n[2], n)
	}
	if string(res[2]) != "foobaz x" {
		t.Errorf("Join should have returned \"foobaz x\" in frame, but found: %s", res[2])
	}
	if n[3] != 1 {
		t.Errorf("Was expecting 1 element in fourth frame but got: %v - %v", n[3], n)
	}
	if string(res[3]) != "wwxxyyzz" {
		t.Errorf("Join should have returned \"wwxxyyzz\" in frame, but found: %s", res[3])
	}

	res, n = c.joinMaxSize(elements, " ", 11)

	if len(res) != len(n) && len(res) != 3 {
		t.Errorf("Was expecting 3 frames to flush but got: %v - %v", n, res)
	}
	if n[0] != 3 {
		t.Errorf("Was expecting 3 elements in first frame but got: %v", n[0])
	}
	if string(res[0]) != "abc abcd ab" {
		t.Errorf("Join should have returned \"abc abcd ab\" in frame, but got: %s", res[0])
	}
	if n[1] != 2 {
		t.Errorf("Was expecting 2 elements in second frame but got: %v", n[1])
	}
	if string(res[1]) != "xyz foobaz" {
		t.Errorf("Join should have returned \"xyz foobaz\" in frame, but got: %s", res[1])
	}
	if n[2] != 2 {
		t.Errorf("Was expecting 2 elements in third frame but got: %v", n[2])
	}
	if string(res[2]) != "x wwxxyyzz" {
		t.Errorf("Join should have returned \"x wwxxyyzz\" in frame, but got: %s", res[2])
	}

	res, n = c.joinMaxSize(elements, "    ", 8)

	if len(res) != len(n) && len(res) != 7 {
		t.Errorf("Was expecting 7 frames to flush but got: %v - %v", n, res)
	}
	if n[0] != 1 {
		t.Errorf("Separator is long, expected a single element in frame but got: %d - %v", n[0], res)
	}
	if string(res[0]) != "abc" {
		t.Errorf("Join should have returned \"abc\" in first frame, but got: %s", res)
	}
	if n[1] != 1 {
		t.Errorf("Separator is long, expected a single element in frame but got: %d - %v", n[1], res)
	}
	if string(res[1]) != "abcd" {
		t.Errorf("Join should have returned \"abcd\" in second frame, but got: %s", res[1])
	}
	if n[2] != 1 {
		t.Errorf("Separator is long, expected a single element in third frame but got: %d - %v", n[2], res)
	}
	if string(res[2]) != "ab" {
		t.Errorf("Join should have returned \"ab\" in third frame, but got: %s", res[2])
	}
	if n[3] != 1 {
		t.Errorf("Separator is long, expected a single element in fourth frame but got: %d - %v", n[3], res)
	}
	if string(res[3]) != "xyz" {
		t.Errorf("Join should have returned \"xyz\" in fourth frame, but got: %s", res[3])
	}
	if n[4] != 1 {
		t.Errorf("Separator is long, expected a single element in fifth frame but got: %d - %v", n[4], res)
	}
	if string(res[4]) != "foobaz" {
		t.Errorf("Join should have returned \"foobaz\" in fifth frame, but got: %s", res[4])
	}
	if n[5] != 1 {
		t.Errorf("Separator is long, expected a single element in sixth frame but got: %d - %v", n[5], res)
	}
	if string(res[5]) != "x" {
		t.Errorf("Join should have returned \"x\" in sixth frame, but got: %s", res[5])
	}
	if n[6] != 1 {
		t.Errorf("Separator is long, expected a single element in seventh frame but got: %d - %v", n[6], res)
	}
	if string(res[6]) != "wwxxyyzz" {
		t.Errorf("Join should have returned \"wwxxyyzz\" in seventh frame, but got: %s", res[6])
	}

	res, n = c.joinMaxSize(elements[4:], " ", 6)
	if len(res) != len(n) && len(res) != 3 {
		t.Errorf("Was expecting 3 frames to flush but got: %v - %v", n, res)

	}
	if n[0] != 1 {
		t.Errorf("Element should just fit in frame - expected single element in frame: %d - %v", n[0], res)
	}
	if string(res[0]) != "foobaz" {
		t.Errorf("Join should have returned \"foobaz\" in first frame, but got: %s", res[0])
	}
	if n[1] != 1 {
		t.Errorf("Single element expected in frame, but got. %d - %v", n[1], res)
	}
	if string(res[1]) != "x" {
		t.Errorf("Join should' have returned \"x\" in second frame, but got: %s", res[1])
	}
	if n[2] != 1 {
		t.Errorf("Even though element is greater then max size we still try to send it. %d - %v", n[2], res)
	}
	if string(res[2]) != "wwxxyyzz" {
		t.Errorf("Join should have returned \"wwxxyyzz\" in third frame, but got: %s", res[2])
	}
}

func TestSendMsgUDP(t *testing.T) {
	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client, err := New(addr)
	if err != nil {
		t.Fatal(err)
	}

	err = client.sendMsg(strings.Repeat("x", MaxUDPPayloadSize+1))
	if err == nil {
		t.Error("Expected error to be returned if message size is bigger than MaxUDPPayloadSize")
	}

	message := "test message"

	err = client.sendMsg(message)
	if err != nil {
		t.Errorf("Expected no error to be returned if message size is smaller or equal to MaxUDPPayloadSize, got: %s", err.Error())
	}

	buffer := make([]byte, MaxUDPPayloadSize+1)
	n, err := io.ReadAtLeast(server, buffer, 1)

	if err != nil {
		t.Fatalf("Expected no error to be returned reading the buffer, got: %s", err.Error())
	}

	if n != len(message) {
		t.Fatalf("Failed to read full message from buffer. Got size `%d` expected `%d`", n, MaxUDPPayloadSize)
	}

	if string(buffer[:n]) != message {
		t.Fatalf("The received message did not match what we expect.")
	}

	client, err = NewBuffered(addr, 1)
	if err != nil {
		t.Fatal(err)
	}

	err = client.sendMsg(strings.Repeat("x", MaxUDPPayloadSize+1))
	if err == nil {
		t.Error("Expected error to be returned if message size is bigger than MaxUDPPayloadSize")
	}

	err = client.sendMsg(message)
	if err != nil {
		t.Errorf("Expected no error to be returned if message size is smaller or equal to MaxUDPPayloadSize, got: %s", err.Error())
	}

	client.Lock()
	err = client.flushLocked()
	client.Unlock()

	if err != nil {
		t.Fatalf("Expected no error to be returned flushing the client, got: %s", err.Error())
	}

	buffer = make([]byte, MaxUDPPayloadSize+1)
	n, err = io.ReadAtLeast(server, buffer, 1)

	if err != nil {
		t.Fatalf("Expected no error to be returned reading the buffer, got: %s", err.Error())
	}

	if n != len(message) {
		t.Fatalf("Failed to read full message from buffer. Got size `%d` expected `%d`", n, MaxUDPPayloadSize)
	}

	if string(buffer[:n]) != message {
		t.Fatalf("The received message did not match what we expect.")
	}
}

func TestSendUDSErrors(t *testing.T) {
	dir, err := ioutil.TempDir("", "socket")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	message := "test message"

	addr := filepath.Join(dir, "dsd.socket")
	udsAddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		t.Fatal(err)
	}

	addrParts := []string{UnixAddressPrefix, addr}
	client, err := New(strings.Join(addrParts, ""))
	if err != nil {
		t.Fatal(err)
	}

	// Server not listening yet
	err = client.sendMsg(message)
	if err == nil || !strings.HasSuffix(err.Error(), "no such file or directory") {
		t.Errorf("Expected error \"no such file or directory\", got: %s", err.Error())
	}

	// Start server and send packet
	server, err := net.ListenUnixgram("unixgram", udsAddr)
	if err != nil {
		t.Fatal(err)
	}
	err = client.sendMsg(message)
	if err != nil {
		t.Errorf("Expected no error to be returned when server is listening, got: %s", err.Error())
	}
	bytes := make([]byte, 1024)
	n, err := server.Read(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes[:n]) != message {
		t.Errorf("Expected: %s. Actual: %s", string(message), string(bytes))
	}

	// close server and send packet
	server.Close()
	os.Remove(addr)
	err = client.sendMsg(message)
	if err == nil {
		t.Error("Expected an error, got nil")
	}

	// Restart server and send packet
	server, err = net.ListenUnixgram("unixgram", udsAddr)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	defer server.Close()
	err = client.sendMsg(message)
	if err != nil {
		t.Errorf("Expected no error to be returned when server is listening, got: %s", err.Error())
	}

	bytes = make([]byte, 1024)
	n, err = server.Read(bytes)
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes[:n]) != message {
		t.Errorf("Expected: %s. Actual: %s", string(message), string(bytes))
	}
}

func TestSendUDSIgnoreErrors(t *testing.T) {
	client, err := New("unix:///invalid")
	if err != nil {
		t.Fatal(err)
	}

	// Default mode throws error
	err = client.sendMsg("message")
	if err == nil || !strings.HasSuffix(err.Error(), "no such file or directory") {
		t.Errorf("Expected error \"connect: no such file or directory\", got: %s", err.Error())
	}

	// Skip errors
	client.SkipErrors = true
	err = client.sendMsg("message")
	if err != nil {
		t.Errorf("Expected no error to be returned when in skip errors mode, got: %s", err.Error())
	}
}

func TestNilSafe(t *testing.T) {
	var c *Client
	assertNotPanics(t, func() { c.SetWriteTimeout(0) })
	assertNotPanics(t, func() { c.Flush() })
	assertNotPanics(t, func() { c.Close() })
	assertNotPanics(t, func() { c.Count("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Histogram("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Distribution("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Gauge("", 0, nil, 1) })
	assertNotPanics(t, func() { c.Set("", "", nil, 1) })
	assertNotPanics(t, func() {
		c.send("", "", []byte(""), nil, 1)
	})
	assertNotPanics(t, func() { c.Event(NewEvent("", "")) })
	assertNotPanics(t, func() { c.SimpleEvent("", "") })
	assertNotPanics(t, func() { c.ServiceCheck(NewServiceCheck("", Ok)) })
	assertNotPanics(t, func() { c.SimpleServiceCheck("", Ok) })
}

func TestEvents(t *testing.T) {
	matrix := []struct {
		event   *Event
		encoded string
	}{
		{
			NewEvent("Hello", "Something happened to my event"),
			`_e{5,30}:Hello|Something happened to my event`,
		}, {
			&Event{Title: "hi", Text: "okay", AggregationKey: "foo"},
			`_e{2,4}:hi|okay|k:foo`,
		}, {
			&Event{Title: "hi", Text: "okay", AggregationKey: "foo", AlertType: Info},
			`_e{2,4}:hi|okay|k:foo|t:info`,
		}, {
			&Event{Title: "hi", Text: "w/e", AlertType: Error, Priority: Normal},
			`_e{2,3}:hi|w/e|p:normal|t:error`,
		}, {
			&Event{Title: "hi", Text: "uh", Tags: []string{"host:foo", "app:bar"}},
			`_e{2,2}:hi|uh|#host:foo,app:bar`,
		}, {
			&Event{Title: "hi", Text: "line1\nline2", Tags: []string{"hello\nworld"}},
			`_e{2,12}:hi|line1\nline2|#helloworld`,
		},
	}

	for _, m := range matrix {
		r, err := m.event.Encode()
		if err != nil {
			t.Errorf("Error encoding: %s\n", err)
			continue
		}
		if r != m.encoded {
			t.Errorf("Expected `%s`, got `%s`\n", m.encoded, r)
		}
	}

	e := NewEvent("", "hi")
	if _, err := e.Encode(); err == nil {
		t.Errorf("Expected error on empty Title.")
	}

	e = NewEvent("hi", "")
	if _, err := e.Encode(); err == nil {
		t.Errorf("Expected error on empty Text.")
	}

	e = NewEvent("hello", "world")
	s, err := e.Encode("tag1", "tag2")
	if err != nil {
		t.Error(err)
	}
	expected := "_e{5,5}:hello|world|#tag1,tag2"
	if s != expected {
		t.Errorf("Expected %s, got %s", expected, s)
	}
	if len(e.Tags) != 0 {
		t.Errorf("Modified event in place illegally.")
	}
}

func TestServiceChecks(t *testing.T) {
	matrix := []struct {
		serviceCheck *ServiceCheck
		encoded      string
	}{
		{
			NewServiceCheck("DataCatService", Ok),
			`_sc|DataCatService|0`,
		}, {
			NewServiceCheck("DataCatService", Warn),
			`_sc|DataCatService|1`,
		}, {
			NewServiceCheck("DataCatService", Critical),
			`_sc|DataCatService|2`,
		}, {
			NewServiceCheck("DataCatService", Unknown),
			`_sc|DataCatService|3`,
		}, {
			&ServiceCheck{Name: "DataCatService", Status: Ok, Hostname: "DataStation.Cat"},
			`_sc|DataCatService|0|h:DataStation.Cat`,
		}, {
			&ServiceCheck{Name: "DataCatService", Status: Ok, Hostname: "DataStation.Cat", Message: "Here goes valuable message"},
			`_sc|DataCatService|0|h:DataStation.Cat|m:Here goes valuable message`,
		}, {
			&ServiceCheck{Name: "DataCatService", Status: Ok, Hostname: "DataStation.Cat", Message: "Here are some cyrillic chars: к л м н о п р с т у ф х ц ч ш"},
			`_sc|DataCatService|0|h:DataStation.Cat|m:Here are some cyrillic chars: к л м н о п р с т у ф х ц ч ш`,
		}, {
			&ServiceCheck{Name: "DataCatService", Status: Ok, Hostname: "DataStation.Cat", Message: "Here goes valuable message", Tags: []string{"host:foo", "app:bar"}},
			`_sc|DataCatService|0|h:DataStation.Cat|#host:foo,app:bar|m:Here goes valuable message`,
		}, {
			&ServiceCheck{Name: "DataCatService", Status: Ok, Hostname: "DataStation.Cat", Message: "Here goes \n that should be escaped", Tags: []string{"host:foo", "app:b\nar"}},
			`_sc|DataCatService|0|h:DataStation.Cat|#host:foo,app:bar|m:Here goes \n that should be escaped`,
		}, {
			&ServiceCheck{Name: "DataCatService", Status: Ok, Hostname: "DataStation.Cat", Message: "Here goes m: that should be escaped", Tags: []string{"host:foo", "app:bar"}},
			`_sc|DataCatService|0|h:DataStation.Cat|#host:foo,app:bar|m:Here goes m\: that should be escaped`,
		},
	}

	for _, m := range matrix {
		r, err := m.serviceCheck.Encode()
		if err != nil {
			t.Errorf("Error encoding: %s\n", err)
			continue
		}
		if r != m.encoded {
			t.Errorf("Expected `%s`, got `%s`\n", m.encoded, r)
		}
	}

	sc := NewServiceCheck("", Ok)
	if _, err := sc.Encode(); err == nil {
		t.Errorf("Expected error on empty Name.")
	}

	sc = NewServiceCheck("sc", ServiceCheckStatus(5))
	if _, err := sc.Encode(); err == nil {
		t.Errorf("Expected error on invalid status value.")
	}

	sc = NewServiceCheck("hello", Warn)
	s, err := sc.Encode("tag1", "tag2")
	if err != nil {
		t.Error(err)
	}
	expected := "_sc|hello|1|#tag1,tag2"
	if s != expected {
		t.Errorf("Expected %s, got %s", expected, s)
	}
	if len(sc.Tags) != 0 {
		t.Errorf("Modified serviceCheck in place illegally.")
	}
}

func TestFlushOnClose(t *testing.T) {
	client, err := NewBuffered("localhost:1201", 64)
	if err != nil {
		t.Fatal(err)
	}
	// stop the flushing mechanism so we can test the buffer without interferences
	client.stop <- struct{}{}

	message := "test message"

	err = client.sendMsg(message)
	if err != nil {
		t.Fatal(err)
	}

	if len(client.commands) != 1 {
		t.Errorf("Commands buffer should contain 1 item, got %d", len(client.commands))
	}

	err = client.Close()
	if err != nil {
		t.Fatal(err)
	}

	if len(client.commands) != 0 {
		t.Errorf("Commands buffer should be empty, got %d", len(client.commands))
	}
}

// These benchmarks show that using different format options:
// v1: sprintf-ing together a bunch of intermediate strings is 4-5x faster
// v2: some use of buffer
// v3: removing sprintf from stat generation and pushing stat building into format
func BenchmarkFormatV3(b *testing.B) {
	b.StopTimer()
	c := &Client{}
	c.Namespace = "foo.bar."
	c.Tags = []string{"app:foo", "host:bar"}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c.format("system.cpu.idle", 10, gaugeSuffix, []string{"foo"}, 1)
		c.format("system.cpu.load", 0.1, gaugeSuffix, nil, 0.9)
	}
}

func BenchmarkFormatV1(b *testing.B) {
	b.StopTimer()
	c := &Client{}
	c.Namespace = "foo.bar."
	c.Tags = []string{"app:foo", "host:bar"}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c.formatV1("system.cpu.idle", 10, []string{"foo"}, 1)
		c.formatV1("system.cpu.load", 0.1, nil, 0.9)
	}
}

// V1 formatting function, added to client for tests
func (c *Client) formatV1(name string, value float64, tags []string, rate float64) string {
	valueAsString := fmt.Sprintf("%f|g", value)
	if rate < 1 {
		valueAsString = fmt.Sprintf("%s|@%f", valueAsString, rate)
	}
	if c.Namespace != "" {
		name = fmt.Sprintf("%s%s", c.Namespace, name)
	}

	tags = append(c.Tags, tags...)
	if len(tags) > 0 {
		valueAsString = fmt.Sprintf("%s|#%s", valueAsString, strings.Join(tags, ","))
	}

	return fmt.Sprintf("%s:%s", name, valueAsString)

}

func BenchmarkFormatV2(b *testing.B) {
	b.StopTimer()
	c := &Client{}
	c.Namespace = "foo.bar."
	c.Tags = []string{"app:foo", "host:bar"}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		c.formatV2("system.cpu.idle", 10, []string{"foo"}, 1)
		c.formatV2("system.cpu.load", 0.1, nil, 0.9)
	}
}

// V2 formatting function, added to client for tests
func (c *Client) formatV2(name string, value float64, tags []string, rate float64) string {
	var buf bytes.Buffer
	if c.Namespace != "" {
		buf.WriteString(c.Namespace)
	}
	buf.WriteString(name)
	buf.WriteString(":")
	buf.WriteString(fmt.Sprintf("%f|g", value))
	if rate < 1 {
		buf.WriteString(`|@`)
		buf.WriteString(strconv.FormatFloat(rate, 'f', -1, 64))
	}

	writeTagString(&buf, c.Tags, tags)

	return buf.String()
}
