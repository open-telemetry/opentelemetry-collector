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

package middleware

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testutil"
)

func TestHTTPClientCompression(t *testing.T) {
	testBody := []byte("uncompressed_text")
	compressedGzipBody, _ := compressGzip(testBody)
	compressedZlibBody, _ := compressZlib(testBody)
	compressedDeflateBody, _ := compressZlib(testBody)
	compressedSnappyBody, _ := compressSnappy(testBody)
	compressedZstdBody, _ := compressZstd(testBody)

	tests := []struct {
		name        string
		encoding    CompressionType
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
			encoding:    "gzip",
			reqBody:     compressedGzipBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidZlib",
			encoding:    "zlib",
			reqBody:     compressedZlibBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidDeflate",
			encoding:    "deflate",
			reqBody:     compressedDeflateBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidSnappy",
			encoding:    "snappy",
			reqBody:     compressedSnappyBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidZstd",
			encoding:    "zstd",
			reqBody:     compressedZstdBody.Bytes(),
			shouldError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err, "failed to read request body: %v", err)
				assert.EqualValues(t, tt.reqBody, body)
				w.WriteHeader(200)
			})

			addr := testutil.GetAvailableLocalAddress(t)
			ln, err := net.Listen("tcp", addr)
			require.NoError(t, err, "failed to create listener: %v", err)
			srv := &http.Server{
				Handler: handler,
			}
			go func() {
				_ = srv.Serve(ln)
			}()
			// Wait for the servers to start
			<-time.After(10 * time.Millisecond)

			serverURL := fmt.Sprintf("http://%s", ln.Addr().String())
			reqBody := bytes.NewBuffer(testBody)

			req, err := http.NewRequest("GET", serverURL, reqBody)
			require.NoError(t, err, "failed to create request to test handler")

			client := http.Client{}
			if tt.encoding != CompressionEmpty && tt.encoding != CompressionNone {
				client.Transport = NewCompressRoundTripper(http.DefaultTransport, tt.encoding)
			}
			res, err := client.Do(req)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			_, err = ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
			require.NoError(t, srv.Close())
		})
	}
}

func TestHTTPContentDecompressionHandler(t *testing.T) {
	testBody := []byte("uncompressed_text")
	tests := []struct {
		name        string
		encoding    string
		reqBodyFunc func() (*bytes.Buffer, error)
		respCode    int
		respBody    string
	}{
		{
			name:     "NoCompression",
			encoding: "",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer(testBody), nil
			},
			respCode: 200,
		},
		{
			name:     "ValidGzip",
			encoding: "gzip",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return compressGzip(testBody)
			},
			respCode: 200,
		},
		{
			name:     "ValidZlib",
			encoding: "zlib",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return compressZlib(testBody)
			},
			respCode: 200,
		},
		{
			name:     "InvalidGzip",
			encoding: "gzip",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer(testBody), nil
			},
			respCode: 400,
			respBody: "gzip: invalid header\n",
		},
		{
			name:     "InvalidZlib",
			encoding: "zlib",
			reqBodyFunc: func() (*bytes.Buffer, error) {
				return bytes.NewBuffer(testBody), nil
			},
			respCode: 400,
			respBody: "zlib: invalid header\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err, "failed to read request body: %v", err)
				assert.EqualValues(t, testBody, string(body))
				w.WriteHeader(200)
			})

			addr := testutil.GetAvailableLocalAddress(t)
			ln, err := net.Listen("tcp", addr)
			require.NoError(t, err, "failed to create listener: %v", err)
			srv := &http.Server{
				Handler: HTTPContentDecompressor(handler),
			}
			go func() {
				_ = srv.Serve(ln)
			}()
			// Wait for the servers to start
			<-time.After(10 * time.Millisecond)

			serverURL := fmt.Sprintf("http://%s", ln.Addr().String())
			reqBody, err := tt.reqBodyFunc()
			require.NoError(t, err, "failed to generate request body: %v", err)

			req, err := http.NewRequest("GET", serverURL, reqBody)
			require.NoError(t, err, "failed to create request to test handler")
			req.Header.Set("Content-Encoding", tt.encoding)

			client := http.Client{}
			res, err := client.Do(req)
			require.NoError(t, err)

			assert.Equal(t, tt.respCode, res.StatusCode, "test handler returned unexpected status code ")
			if tt.respBody != "" {
				body, err := ioutil.ReadAll(res.Body)
				require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
				assert.Equal(t, tt.respBody, string(body))
			}
			require.NoError(t, srv.Close())
		})
	}
}

func TestUnmarshalText(t *testing.T) {
	tests := []struct {
		name            string
		compressionName []byte
		shouldError     bool
	}{
		{
			name:            "ValidGzip",
			compressionName: []byte("gzip"),
			shouldError:     false,
		},
		{
			name:            "ValidZlib",
			compressionName: []byte("zlib"),
			shouldError:     false,
		},
		{
			name:            "ValidDeflate",
			compressionName: []byte("deflate"),
			shouldError:     false,
		},
		{
			name:            "ValidSnappy",
			compressionName: []byte("snappy"),
			shouldError:     false,
		},
		{
			name:            "ValidZstd",
			compressionName: []byte("zstd"),
			shouldError:     false,
		},
		{
			name:            "ValidEmpty",
			compressionName: []byte(""),
			shouldError:     false,
		},
		{
			name:            "ValidNone",
			compressionName: []byte("none"),
			shouldError:     false,
		},
		{
			name:            "Invalid",
			compressionName: []byte("ggip"),
			shouldError:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			temp := CompressionNone
			err := temp.UnmarshalText(tt.compressionName)
			if tt.shouldError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, temp, CompressionType(tt.compressionName))
		})
	}
}

func compressGzip(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)
	defer gw.Close()

	_, err := gw.Write(body)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func compressZlib(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)
	defer zw.Close()

	_, err := zw.Write(body)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func compressSnappy(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	sw := snappy.NewBufferedWriter(&buf)
	defer sw.Close()

	_, err := sw.Write(body)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func compressZstd(body []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	zw, _ := zstd.NewWriter(&buf)
	defer zw.Close()

	_, err := zw.Write(body)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}
