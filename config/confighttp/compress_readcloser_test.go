// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

type errorReadCloser struct {
	io.Reader
	err error
}

func (erc errorReadCloser) Close() error {
	return erc.err
}

func TestCompressReadCloser(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		wrapper func(r io.Reader) io.ReadCloser
		content []byte
		errVal  string
	}{
		{
			name: "non mutating wrapper",
			wrapper: func(r io.Reader) io.ReadCloser {
				return errorReadCloser{
					Reader: r,
					err:    nil,
				}
			},
			content: []byte("hello world"),
			errVal:  "",
		},
		{
			name: "failed reader",
			wrapper: func(r io.Reader) io.ReadCloser {
				return errorReadCloser{
					Reader: r,
					err:    errors.New("failed to close reader"),
				}
			},
			errVal: "failed to close reader",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orig := bytes.NewBuffer([]byte("hello world"))

			crc := &compressReadCloser{
				Reader: orig,
				orig:   tc.wrapper(orig),
			}

			require.NoError(t, iotest.TestReader(crc, orig.Bytes()), "Must be able to read original content")

			err := crc.Close()
			if tc.errVal != "" {
				require.EqualError(t, err, tc.errVal, "Must match the expected error message")
			} else {
				require.NoError(t, err, "Must not error when closing reader")
			}
		})
	}
}

func TestPanicRecoverReadCloser(t *testing.T) {
	t.Parallel()

	t.Run("read_panic_recovered", func(t *testing.T) {
		t.Parallel()

		pr := &panicRecoverReadCloser{
			inner: &panickingReadCloser{panicOnRead: true},
		}
		buf := make([]byte, 10)
		n, err := pr.Read(buf)
		require.Equal(t, 0, n)
		require.Error(t, err)
		require.Contains(t, err.Error(), "decompression panic")
	})

	t.Run("close_panic_recovered", func(t *testing.T) {
		t.Parallel()

		pr := &panicRecoverReadCloser{
			inner: &panickingReadCloser{panicOnClose: true},
		}
		err := pr.Close()
		require.Error(t, err)
		require.Contains(t, err.Error(), "decompression close panic")
	})

	t.Run("normal_operation", func(t *testing.T) {
		t.Parallel()

		data := []byte("hello world")
		pr := &panicRecoverReadCloser{
			inner: io.NopCloser(bytes.NewReader(data)),
		}
		got, err := io.ReadAll(pr)
		require.NoError(t, err)
		require.Equal(t, data, got)
		require.NoError(t, pr.Close())
	})
}

// panickingReadCloser is a test helper that panics on Read and/or Close.
type panickingReadCloser struct {
	panicOnRead  bool
	panicOnClose bool
}

func (p *panickingReadCloser) Read([]byte) (int, error) {
	if p.panicOnRead {
		panic("bad decompression data")
	}
	return 0, io.EOF
}

func (p *panickingReadCloser) Close() error {
	if p.panicOnClose {
		panic("bad close")
	}
	return nil
}
