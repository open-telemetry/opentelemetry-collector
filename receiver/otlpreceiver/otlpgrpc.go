package otlpreceiver

import (
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/grpc/codes"
)

const grpcStatus = "grpc-status"
const grpcMessage = "grpc-message"

var errTooLong = errors.New("response payload is too long to encode into gRPC")

type grpcResponseWriter struct {
	innerRw       http.ResponseWriter
	headerWritten bool
}

func (rw grpcResponseWriter) Header() http.Header {
	return rw.innerRw.Header()
}

func (rw grpcResponseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(int(codes.OK))
	}
	lengthPrefix := make([]byte, 5)
	payloadLen := uint32(len(b))
	if int(payloadLen) != len(b) {
		return 0, errTooLong
	}
	binary.BigEndian.PutUint32(lengthPrefix[1:], payloadLen)
	_, err := rw.innerRw.Write(lengthPrefix)
	if err != nil {
		return 0, err
	}
	return rw.innerRw.Write(b)
}

func (rw *grpcResponseWriter) WriteHeader(statusCode int) {
	if !rw.headerWritten {
		rw.headerWritten = true
		rw.Header()["Date"] = nil
		rw.Header()["Trailer"] = []string{grpcStatus, grpcMessage}
		rw.Header().Set("Content-Type", grpcContentType)
		rw.innerRw.WriteHeader(200)
	}
	rw.Header().Set(grpcStatus, strconv.FormatInt(int64(statusCode), 10))
}

func unwrapProtoFromGrpc(reader io.Reader) (compressed bool, errout error) {
	b := make([]byte, 5)
	_, err := io.ReadFull(reader, b)
	// compressed := b[0] == 1
	return b[0] == 1, err
}
