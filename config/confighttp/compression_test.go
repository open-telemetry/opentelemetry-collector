// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcompression"
)

func TestHTTPClientCompression(t *testing.T) {
	testBody := []byte("uncompressed_text")
	compressedGzipBody := compressGzip(t, testBody)
	compressedZlibBody := compressZlib(t, testBody)
	compressedDeflateBody := compressZlib(t, testBody)
	compressedSnappyBody := compressSnappy(t, testBody)
	compressedZstdBody := compressZstd(t, testBody)

	tests := []struct {
		name        string
		encoding    configcompression.CompressionType
		reqBody     []byte
		shouldError bool
	}{
		{
			name:        "ValidEmpty",
			encoding:    "",
			reqBody:     testBody,
			shouldError: false,
		},
		{
			name:        "ValidNone",
			encoding:    "none",
			reqBody:     testBody,
			shouldError: false,
		},
		{
			name:        "ValidGzip",
			encoding:    configcompression.Gzip,
			reqBody:     compressedGzipBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidZlib",
			encoding:    configcompression.Zlib,
			reqBody:     compressedZlibBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidDeflate",
			encoding:    configcompression.Deflate,
			reqBody:     compressedDeflateBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidSnappy",
			encoding:    configcompression.Snappy,
			reqBody:     compressedSnappyBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidZstd",
			encoding:    configcompression.Zstd,
			reqBody:     compressedZstdBody.Bytes(),
			shouldError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err, "failed to read request body: %v", err)
				assert.EqualValues(t, tt.reqBody, body)
				w.WriteHeader(200)
			}))
			t.Cleanup(srv.Close)

			reqBody := bytes.NewBuffer(testBody)

			req, err := http.NewRequest(http.MethodGet, srv.URL, reqBody)
			require.NoError(t, err, "failed to create request to test handler")

			clientSettings := HTTPClientSettings{
				Endpoint:    srv.URL,
				Compression: tt.encoding,
			}
			client, err := clientSettings.ToClient(componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
			require.NoError(t, err)
			res, err := client.Do(req)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			_, err = io.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
		})
	}
}

func TestHTTPContentDecompressionHandler(t *testing.T) {
	testBody := []byte("uncompressed_text")
	tests := []struct {
		name     string
		encoding string
		reqBody  *bytes.Buffer
		respCode int
		respBody string
	}{
		{
			name:     "NoCompression",
			encoding: "",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: 200,
		},
		{
			name:     "ValidGzip",
			encoding: "gzip",
			reqBody:  compressGzip(t, testBody),
			respCode: 200,
		},
		{
			name:     "ValidZlib",
			encoding: "zlib",
			reqBody:  compressZlib(t, testBody),
			respCode: 200,
		},
		{
			name:     "InvalidGzip",
			encoding: "gzip",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: 400,
			respBody: "gzip: invalid header\n",
		},
		{
			name:     "InvalidZlib",
			encoding: "zlib",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: 400,
			respBody: "zlib: invalid header\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(httpContentDecompressor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err, "failed to read request body: %v", err)
				assert.EqualValues(t, testBody, string(body))
				w.WriteHeader(200)
			})))
			t.Cleanup(srv.Close)

			req, err := http.NewRequest(http.MethodGet, srv.URL, tt.reqBody)
			require.NoError(t, err, "failed to create request to test handler")
			req.Header.Set("Content-Encoding", tt.encoding)

			client := http.Client{}
			res, err := client.Do(req)
			require.NoError(t, err)

			assert.Equal(t, tt.respCode, res.StatusCode, "test handler returned unexpected status code ")
			if tt.respBody != "" {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
				assert.Equal(t, tt.respBody, string(body))
			}
		})
	}
}

func TestHTTPContentCompressionRequestWithNilBody(t *testing.T) {
	compressedGzipBody := compressGzip(t, []byte{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "failed to read request body: %v", err)
		assert.EqualValues(t, compressedGzipBody.Bytes(), body)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err, "failed to create request to test handler")

	client := http.Client{}
	client.Transport, err = newCompressRoundTripper(http.DefaultTransport, configcompression.Gzip)
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)

	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
}

type copyFailBody struct {
}

func (*copyFailBody) Read(_ []byte) (n int, err error) {
	return 0, fmt.Errorf("read failed")
}

func (*copyFailBody) Close() error {
	return nil
}

func TestHTTPContentCompressionCopyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, &copyFailBody{})
	require.NoError(t, err)

	client := http.Client{}
	client.Transport, err = newCompressRoundTripper(http.DefaultTransport, configcompression.Gzip)
	require.NoError(t, err)
	_, err = client.Do(req)
	require.Error(t, err)
}

type closeFailBody struct {
	*bytes.Buffer
}

func (*closeFailBody) Close() error {
	return fmt.Errorf("close failed")
}

func TestHTTPContentCompressionRequestBodyCloseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest(http.MethodGet, server.URL, &closeFailBody{Buffer: bytes.NewBuffer([]byte("blank"))})
	require.NoError(t, err)

	client := http.Client{}
	client.Transport, err = newCompressRoundTripper(http.DefaultTransport, configcompression.Gzip)
	require.NoError(t, err)
	_, err = client.Do(req)
	require.Error(t, err)
}

func compressGzip(t testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write(body)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	return &buf
}

func compressZlib(t testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, err := zw.Write(body)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return &buf
}

func compressSnappy(t testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	sw := snappy.NewBufferedWriter(&buf)
	_, err := sw.Write(body)
	require.NoError(t, err)
	require.NoError(t, sw.Close())
	return &buf
}

func compressZstd(t testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	zw, _ := zstd.NewWriter(&buf)
	_, err := zw.Write(body)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return &buf
}
