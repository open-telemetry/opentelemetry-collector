// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/featuregate"
)

func TestHTTPClientCompression(t *testing.T) {
	testBody := []byte("uncompressed_text")
	compressedGzipBody := compressGzip(t, testBody)
	compressedZlibBody := compressZlib(t, testBody)
	compressedDeflateBody := compressZlib(t, testBody)
	compressedSnappyFramedBody := compressSnappyFramed(t, testBody)
	compressedSnappyBody := compressSnappy(t, testBody)
	compressedZstdBody := compressZstd(t, testBody)
	compressedLz4Body := compressLz4(t, testBody)

	const invalidGzipLevel configcompression.Level = 100

	tests := []struct {
		name                string
		encoding            configcompression.Type
		level               configcompression.Level
		framedSnappyEnabled bool
		reqBody             []byte
		shouldError         bool
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
			encoding:    configcompression.TypeGzip,
			level:       gzip.DefaultCompression,
			reqBody:     compressedGzipBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidGzip-DefaultLevel",
			encoding:    configcompression.TypeGzip,
			reqBody:     compressedGzipBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "InvalidGzip",
			encoding:    configcompression.TypeGzip,
			level:       invalidGzipLevel,
			reqBody:     compressedGzipBody.Bytes(),
			shouldError: true,
		},
		{
			name:        "InvalidCompression",
			encoding:    configcompression.Type("invalid"),
			level:       invalidGzipLevel,
			reqBody:     compressedGzipBody.Bytes(),
			shouldError: true,
		},
		{
			name:        "ValidZlib",
			encoding:    configcompression.TypeZlib,
			level:       gzip.DefaultCompression,
			reqBody:     compressedZlibBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidDeflate",
			encoding:    configcompression.TypeDeflate,
			level:       gzip.DefaultCompression,
			reqBody:     compressedDeflateBody.Bytes(),
			shouldError: false,
		},
		{
			name:                "ValidSnappy",
			encoding:            configcompression.TypeSnappy,
			framedSnappyEnabled: true,
			reqBody:             compressedSnappyBody.Bytes(),
			shouldError:         false,
		},
		{
			name:        "InvalidSnappy",
			encoding:    configcompression.TypeSnappy,
			level:       gzip.DefaultCompression,
			reqBody:     compressedSnappyBody.Bytes(),
			shouldError: true,
		},
		{
			name:                "ValidSnappyFramed",
			encoding:            configcompression.TypeSnappyFramed,
			framedSnappyEnabled: true,
			reqBody:             compressedSnappyFramedBody.Bytes(),
			shouldError:         false,
		},
		{
			name:        "InvalidSnappyFramed",
			encoding:    configcompression.TypeSnappyFramed,
			level:       gzip.DefaultCompression,
			reqBody:     compressedSnappyFramedBody.Bytes(),
			shouldError: true,
		},
		{
			name:        "ValidZstd",
			encoding:    configcompression.TypeZstd,
			level:       99,
			reqBody:     compressedZstdBody.Bytes(),
			shouldError: false,
		},
		{
			name:        "ValidLz4",
			encoding:    configcompression.TypeLz4,
			reqBody:     compressedLz4Body.Bytes(),
			shouldError: false,
		},
		{
			name:        "InvalidLz4",
			encoding:    configcompression.TypeLz4,
			level:       gzip.DefaultCompression,
			reqBody:     compressedLz4Body.Bytes(),
			shouldError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(enableFramedSnappy.ID(), tt.framedSnappyEnabled))

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "failed to read request body: %v", err)
				assert.Equal(t, tt.reqBody, body)
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(srv.Close)

			reqBody := bytes.NewBuffer(testBody)

			req, err := http.NewRequest(http.MethodGet, srv.URL, reqBody)
			require.NoError(t, err, "failed to create request to test handler")
			clientSettings := ClientConfig{
				Endpoint:          srv.URL,
				Compression:       tt.encoding,
				CompressionParams: newCompressionParams(tt.level),
			}
			err = clientSettings.Validate()
			if tt.shouldError {
				require.Error(t, err)
				message := fmt.Sprintf("unsupported parameters {Level:%+v} for compression type %q", tt.level, tt.encoding)
				assert.Equal(t, message, err.Error())
				return
			}
			require.NoError(t, err)
			client, err := clientSettings.ToClient(context.Background(), nil, componenttest.NewNopTelemetrySettings())
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

func TestHTTPCustomDecompression(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		assert.NoError(t, err, "failed to read request body: %v", err)
		assert.Equal(t, "decompressed body", string(body))
		w.WriteHeader(http.StatusOK)
	})
	decoders := map[string]func(io.ReadCloser) (io.ReadCloser, error){
		"custom-encoding": func(io.ReadCloser) (io.ReadCloser, error) { //nolint:unparam
			return io.NopCloser(strings.NewReader("decompressed body")), nil
		},
	}
	srv := httptest.NewServer(httpContentDecompressor(handler, defaultMaxRequestBodySize, defaultErrorHandler, defaultCompressionAlgorithms(), decoders))

	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, bytes.NewBuffer([]byte("123decompressed body")))
	require.NoError(t, err, "failed to create request to test handler")
	req.Header.Set("Content-Encoding", "custom-encoding")

	client := srv.Client()
	res, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode, "test handler returned unexpected status code ")
	_, err = io.ReadAll(res.Body)
	require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
}

func TestHTTPContentDecompressionHandler(t *testing.T) {
	testBody := []byte("uncompressed_text")
	noDecoders := map[string]func(io.ReadCloser) (io.ReadCloser, error){}
	tests := []struct {
		name                string
		encoding            string
		reqBody             *bytes.Buffer
		respCode            int
		respBody            string
		framedSnappyEnabled bool
	}{
		{
			name:     "NoCompression",
			encoding: "",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusOK,
		},
		{
			name:     "ValidDeflate",
			encoding: "deflate",
			reqBody:  compressZlib(t, testBody),
			respCode: http.StatusOK,
		},
		{
			name:     "ValidGzip",
			encoding: "gzip",
			reqBody:  compressGzip(t, testBody),
			respCode: http.StatusOK,
		},
		{
			name:     "ValidZlib",
			encoding: "zlib",
			reqBody:  compressZlib(t, testBody),
			respCode: http.StatusOK,
		},
		{
			name:     "ValidZstd",
			encoding: "zstd",
			reqBody:  compressZstd(t, testBody),
			respCode: http.StatusOK,
		},
		{
			name:                "ValidSnappyFramed",
			encoding:            "x-snappy-framed",
			framedSnappyEnabled: true,
			reqBody:             compressSnappyFramed(t, testBody),
			respCode:            http.StatusOK,
		},
		{
			name:     "ValidSnappy",
			encoding: "snappy",
			reqBody:  compressSnappy(t, testBody),
			respCode: http.StatusOK,
		},
		{
			// Should work even without the framed snappy feature gate enabled,
			// since during decompression we're peeking the compression header
			// and identifying which snappy encoding was used.
			name:     "ValidSnappyFramedAsSnappy",
			encoding: "snappy",
			reqBody:  compressSnappyFramed(t, testBody),
			respCode: http.StatusOK,
		},
		{
			name:     "ValidLz4",
			encoding: "lz4",
			reqBody:  compressLz4(t, testBody),
			respCode: http.StatusOK,
		},
		{
			name:     "InvalidDeflate",
			encoding: "deflate",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusBadRequest,
			respBody: "zlib: invalid header\n",
		},
		{
			name:     "InvalidGzip",
			encoding: "gzip",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusBadRequest,
			respBody: "gzip: invalid header\n",
		},
		{
			name:     "InvalidZlib",
			encoding: "zlib",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusBadRequest,
			respBody: "zlib: invalid header\n",
		},
		{
			name:     "InvalidZstd",
			encoding: "zstd",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusBadRequest,
			respBody: "invalid input: magic number mismatch",
		},
		{
			name:                "InvalidSnappyFramed",
			encoding:            "x-snappy-framed",
			framedSnappyEnabled: true,
			reqBody:             bytes.NewBuffer(testBody),
			respCode:            http.StatusBadRequest,
			respBody:            "snappy: corrupt input",
		},
		{
			name:     "InvalidSnappy",
			encoding: "snappy",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusBadRequest,
			respBody: "snappy: corrupt input\n",
		},
		{
			name:     "UnsupportedCompression",
			encoding: "nosuchcompression",
			reqBody:  bytes.NewBuffer(testBody),
			respCode: http.StatusBadRequest,
			respBody: "unsupported Content-Encoding: nosuchcompression\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(enableFramedSnappy.ID(), tt.framedSnappyEnabled))

			srv := httptest.NewServer(httpContentDecompressor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(err.Error()))
					return
				}

				assert.NoError(t, err, "failed to read request body: %v", err)
				assert.EqualValues(t, testBody, string(body))
				w.WriteHeader(http.StatusOK)
			}), defaultMaxRequestBodySize, defaultErrorHandler, defaultCompressionAlgorithms(), noDecoders))
			t.Cleanup(srv.Close)

			req, err := http.NewRequest(http.MethodGet, srv.URL, tt.reqBody)
			require.NoError(t, err, "failed to create request to test handler")
			req.Header.Set("Content-Encoding", tt.encoding)

			client := srv.Client()
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

// TestEmptyCompressionAlgorithmsAllowsUncompressed verifies that when CompressionAlgorithms
// is set to an empty array, requests without Content-Encoding header are accepted.
func TestEmptyCompressionAlgorithmsAllowsUncompressed(t *testing.T) {
	testBody := []byte(`{"message": "test data"}`)

	tests := []struct {
		name                  string
		compressionAlgorithms []string
		contentEncoding       string // empty string means no header
		expectedStatus        int
		expectedError         string
		compressionBypassed   bool // If true, don't compress the body (simulates bypass behavior)
	}{
		// Case 1: Empty array should bypass decompression
		{
			name:                  "EmptyArray_NoContentEncoding_Accepted",
			compressionAlgorithms: []string{},
			contentEncoding:       "",
			expectedStatus:        http.StatusOK,
		},
		{
			name:                  "EmptyArray_Gzip_PassedThrough",
			compressionAlgorithms: []string{},
			contentEncoding:       "gzip",
			expectedStatus:        http.StatusOK,
			compressionBypassed:   true, // Empty array bypasses decompression
		},

		{
			name:                  "EmptyArray_RandomEncoding_Accepted",
			compressionAlgorithms: []string{},
			contentEncoding:       "randomstuff",
			expectedStatus:        http.StatusOK,
		},
		// Case 2: Explicit list with only compressed formats should reject uncompressed
		{
			name:                  "OnlyZstd_NoContentEncoding_Rejected",
			compressionAlgorithms: []string{"zstd"},
			contentEncoding:       "",
			expectedStatus:        http.StatusBadRequest,
			expectedError:         "unsupported Content-Encoding",
		},
		{
			name:                  "OnlyZstd_Zstd_Accepted",
			compressionAlgorithms: []string{"zstd"},
			contentEncoding:       "zstd",
			expectedStatus:        http.StatusOK,
		},
		{
			name:                  "OnlyZstd_GzipContentEncoding_Rejected",
			compressionAlgorithms: []string{"zstd"},
			contentEncoding:       "gzip",
			expectedStatus:        http.StatusBadRequest,
			expectedError:         "unsupported Content-Encoding",
		},
		// Case 3: Explicit list including empty string should accept uncompressed
		{
			name:                  "WithEmptyString_NoContentEncoding_Accepted",
			compressionAlgorithms: []string{"", "gzip", "zstd"},
			contentEncoding:       "",
			expectedStatus:        http.StatusOK,
		},
		{
			name:                  "WithEmptyString_Gzip_Accepted",
			compressionAlgorithms: []string{"", "gzip"},
			contentEncoding:       "gzip",
			expectedStatus:        http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler that echoes back the request body
			// If there's an error reading, it returns 500 which the test will catch
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "failed to read body", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(body)
			})

			// Create ServerConfig with the specified CompressionAlgorithms
			serverConfig := &ServerConfig{
				Endpoint:              "localhost:0",
				CompressionAlgorithms: tt.compressionAlgorithms,
			}

			srv, err := serverConfig.ToServer(
				context.Background(),
				nil,
				componenttest.NewNopTelemetrySettings(),
				handler,
			)
			require.NoError(t, err)

			// Create test server
			testSrv := httptest.NewServer(srv.Handler)
			defer testSrv.Close()

			// Compress the body if needed for the test
			requestBody := testBody
			if !tt.compressionBypassed && tt.expectedStatus == http.StatusOK {
				switch tt.contentEncoding {
				case "gzip":
					requestBody = compressGzip(t, testBody).Bytes()
				case "zstd":
					requestBody = compressZstd(t, testBody).Bytes()
				}
			}

			// Create request
			req, err := http.NewRequest(http.MethodPost, testSrv.URL, bytes.NewReader(requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Set Content-Encoding header if specified
			if tt.contentEncoding != "" {
				req.Header.Set("Content-Encoding", tt.contentEncoding)
			}

			// Send request
			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify response
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code")

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.expectedError != "" {
				assert.Contains(t, string(body), tt.expectedError, "Expected error message not found")
			} else {
				// For successful requests, body should be echoed back
				assert.Equal(t, testBody, body, "Response body should match request body")
			}
		})
	}
}

func TestHTTPContentCompressionRequestWithNilBody(t *testing.T) {
	compressedGzipBody := compressGzip(t, []byte{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "failed to read request body: %v", err)
		assert.Equal(t, compressedGzipBody.Bytes(), body)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, http.NoBody)
	require.NoError(t, err, "failed to create request to test handler")

	client := srv.Client()
	compressionParams := newCompressionParams(gzip.DefaultCompression)
	client.Transport, err = newCompressRoundTripper(http.DefaultTransport, configcompression.TypeGzip, compressionParams)
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)

	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
}

func TestHTTPContentCompressionCopyError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, iotest.ErrReader(errors.New("read failed")))
	require.NoError(t, err)

	client := srv.Client()
	compressionParams := newCompressionParams(gzip.DefaultCompression)
	client.Transport, err = newCompressRoundTripper(http.DefaultTransport, configcompression.TypeGzip, compressionParams)
	require.NoError(t, err)
	_, err = client.Do(req)
	require.Error(t, err)
}

type closeFailBody struct {
	*bytes.Buffer
}

func (*closeFailBody) Close() error {
	return errors.New("close failed")
}

func TestHTTPContentCompressionRequestBodyCloseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, &closeFailBody{Buffer: bytes.NewBuffer([]byte("blank"))})
	require.NoError(t, err)

	client := srv.Client()
	compressionParams := newCompressionParams(gzip.DefaultCompression)
	client.Transport, err = newCompressRoundTripper(http.DefaultTransport, configcompression.TypeGzip, compressionParams)
	require.NoError(t, err)
	_, err = client.Do(req)
	require.Error(t, err)
}

func TestOverrideCompressionList(t *testing.T) {
	// prepare
	configuredDecoders := []string{"none", "zlib"}

	srv := httptest.NewServer(httpContentDecompressor(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), defaultMaxRequestBodySize, defaultErrorHandler, configuredDecoders, nil))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, compressSnappyFramed(t, []byte("123decompressed body")))
	require.NoError(t, err, "failed to create request to test handler")
	req.Header.Set("Content-Encoding", "snappy")

	client := srv.Client()

	// test
	res, err := client.Do(req)
	require.NoError(t, err)

	// verify
	assert.Equal(t, http.StatusBadRequest, res.StatusCode, "test handler returned unexpected status code ")
	_, err = io.ReadAll(res.Body)
	require.NoError(t, res.Body.Close(), "failed to close request body: %v", err)
}

func TestDecompressorAvoidDecompressionBomb(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name                string
		encoding            string
		compress            func(tb testing.TB, payload []byte) *bytes.Buffer
		framedSnappyEnabled bool
	}{
		// None encoding is ignored since it does not
		// enforce the max body size if content encoding header is not set
		{
			name:     "gzip",
			encoding: "gzip",
			compress: compressGzip,
		},
		{
			name:     "zstd",
			encoding: "zstd",
			compress: compressZstd,
		},
		{
			name:     "zlib",
			encoding: "zlib",
			compress: compressZlib,
		},
		{
			name:     "x-snappy-framed",
			encoding: "x-snappy-framed",
			compress: compressSnappyFramed,
		},
		{
			name:                "x-snappy-not-framed",
			encoding:            "x-snappy-framed",
			compress:            compressSnappyFramed,
			framedSnappyEnabled: false,
		},
		{
			name:     "snappy",
			encoding: "snappy",
			compress: compressSnappy,
		},
		{
			name:     "lz4",
			encoding: "lz4",
			compress: compressLz4,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel() // TODO: Re-enable parallel tests once feature gate is removed. We can't parallelize since registry is shared.
			require.NoError(t, featuregate.GlobalRegistry().Set(enableFramedSnappy.ID(), tc.framedSnappyEnabled))

			h := httpContentDecompressor(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					n, err := io.Copy(io.Discard, r.Body)
					assert.Equal(t, int64(1024), n, "Must have only read the limited value of bytes")
					assert.EqualError(t, err, "http: request body too large")
					w.WriteHeader(http.StatusBadRequest)
				}),
				1024,
				defaultErrorHandler,
				defaultCompressionAlgorithms(),
				availableDecoders,
			)

			payload := tc.compress(t, make([]byte, 2*1024)) // 2KB uncompressed payload
			assert.NotEmpty(t, payload.Bytes(), "Must have data available")

			req := httptest.NewRequest(http.MethodPost, "/", payload)
			req.Header.Set("Content-Encoding", tc.encoding)

			resp := httptest.NewRecorder()

			h.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusBadRequest, resp.Code, "Must match the expected code")
			assert.Empty(t, resp.Body.String(), "Must match the returned string")
		})
	}
}

func TestPooledZstdReadCloserReadAfterClose(t *testing.T) {
	h := httpContentDecompressor(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			buf := make([]byte, 1024)
			_, err := r.Body.Read(buf)
			assert.NoError(t, err)
			err = r.Body.Close()
			assert.NoError(t, err)
			_, err = r.Body.Read(buf)
			assert.ErrorIs(t, err, zstd.ErrDecoderClosed)
			w.WriteHeader(http.StatusBadRequest)
		}),
		defaultMaxRequestBodySize,
		defaultErrorHandler,
		defaultCompressionAlgorithms(),
		availableDecoders,
	)

	payload := compressZstd(t, make([]byte, 2*1024)) // 2KB uncompressed payload
	assert.NotEmpty(t, payload.Bytes(), "Must have data available")

	req := httptest.NewRequest(http.MethodPost, "/", payload)
	req.Header.Set("Content-Encoding", "zstd")

	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code, "Must match the expected code")
	assert.Empty(t, resp.Body.String(), "Must match the returned string")
}

func compressGzip(tb testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	gw, _ := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	_, err := gw.Write(body)
	require.NoError(tb, err)
	require.NoError(tb, gw.Close())
	return &buf
}

func compressZlib(tb testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	_, err := zw.Write(body)
	require.NoError(tb, err)
	require.NoError(tb, zw.Close())
	return &buf
}

func compressSnappyFramed(tb testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	sw := snappy.NewBufferedWriter(&buf)
	_, err := sw.Write(body)
	require.NoError(tb, err)
	require.NoError(tb, sw.Close())
	return &buf
}

func compressSnappy(tb testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	compressed := snappy.Encode(nil, body)
	_, err := buf.Write(compressed)
	require.NoError(tb, err)
	return &buf
}

func compressZstd(tb testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	compression := zstd.SpeedFastest
	encoderLevel := zstd.WithEncoderLevel(compression)
	zw, _ := zstd.NewWriter(&buf, encoderLevel)
	_, err := zw.Write(body)
	require.NoError(tb, err)
	require.NoError(tb, zw.Close())
	return &buf
}

func compressLz4(tb testing.TB, body []byte) *bytes.Buffer {
	var buf bytes.Buffer
	lz := lz4.NewWriter(&buf)
	_, err := lz.Write(body)
	require.NoError(tb, err)
	require.NoError(tb, lz.Close())
	return &buf
}
