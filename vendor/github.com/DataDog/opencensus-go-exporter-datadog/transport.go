// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadog.com/).
// Copyright 2018 Datadog, Inc.

package datadog

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// defaultTraceAddr specifies the default address of the Datadog trace agent.
	defaultTraceAddr = "localhost:8126"

	// version specifies the version identifier that will be attached to the
	// HTTP headers. In this case it is prefixed OC for Opencensus.
	version = "OC/0.1.0"
)

// transport holds an HTTP client used to connect to the Datadog agent at the specified URL.
type transport struct {
	client *http.Client
	url    string
}

// newTransport creates a new transport that will connect to the Datadog agent at the given address. If
// addr is empty, it will use the default address, which is "localhost:8126".
func newTransport(addr string) *transport {
	if addr == "" {
		addr = defaultTraceAddr
	}
	httpclient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 1 * time.Second,
	}
	return &transport{
		url:    fmt.Sprintf("http://%s/v0.3/traces", addr),
		client: httpclient,
	}
}

// httpHeaders specifies the set of HTTP headers that will be attached to all HTTP calls
// to the Datadog agent.
var httpHeaders = map[string]string{
	"Datadog-Meta-Lang":             "go",
	"Datadog-Meta-Lang-Version":     strings.TrimPrefix(runtime.Version(), "go"),
	"Datadog-Meta-Lang-Interpreter": runtime.Compiler + "-" + runtime.GOARCH + "-" + runtime.GOOS,
	"Datadog-Meta-Tracer-Version":   version,
	"Content-Type":                  "application/msgpack",
}

// upload sents the given request body to the Datadog agent and assigns the traceCount
// as an HTTP header.
func (t *transport) upload(body *bytes.Buffer, traceCount int) error {
	req, err := http.NewRequest("POST", t.url, body)
	if err != nil {
		return fmt.Errorf("cannot create http request: %v", err)
	}
	for header, value := range httpHeaders {
		req.Header.Set(header, value)
	}
	req.Header.Set("X-Datadog-Trace-Count", strconv.Itoa(traceCount))
	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	response, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if code := response.StatusCode; code >= 400 {
		// error, check the body for context information and
		// return a user friendly error
		msg := make([]byte, 1000)
		n, _ := response.Body.Read(msg)
		txt := http.StatusText(code)
		if n > 0 {
			return fmt.Errorf("%s (Status: %s)", msg[:n], txt)
		}
		return fmt.Errorf("%s", txt)
	}
	return nil
}
