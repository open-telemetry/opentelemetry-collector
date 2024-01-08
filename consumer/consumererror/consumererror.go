// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumererror // import "go.opentelemetry.io/collector/consumer/consumererror"

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/status"
)

func New(errs ...error) error {
	return errors.Join(errs...)
}

type StatusError struct {
	error
	httpStatus *int
	grpcStatus *status.Status
}

func (se *StatusError) Error() string {
	if se.httpStatus != nil {
		return fmt.Sprintf("HTTP Status %d", *se.httpStatus)
	} else {
		return fmt.Sprintf("gRPC Status %s", se.grpcStatus.Code().String())
	}
}

func (se *StatusError) HTTPStatus() int {
	if se.httpStatus != nil {
		return *se.httpStatus
	}

	// TODO Convert gRPC to HTTP
	return 0
}

func (se *StatusError) GRPCStatus() *status.Status {
	if se.grpcStatus != nil {
		return se.grpcStatus
	}

	// TODO Convert HTTP to gRPC
	return &status.Status{}
}

func NewHTTPStatus(err error, code int) error {
	return &StatusError{
		error:      err,
		httpStatus: &code,
	}
}

func NewGRPCStatus(err error, status *status.Status) error {
	return &StatusError{
		error:      err,
		grpcStatus: status,
	}
}
