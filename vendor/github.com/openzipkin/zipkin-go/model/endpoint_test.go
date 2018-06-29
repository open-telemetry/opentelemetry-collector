package model_test

import (
	"net"
	"testing"

	"github.com/openzipkin/zipkin-go/model"
)

func TestEmptyEndpoint(t *testing.T) {
	var e *model.Endpoint

	if want, have := true, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}

	e = &model.Endpoint{}

	if want, have := true, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}

	e = &model.Endpoint{
		IPv4: net.IPv4zero,
	}

	if want, have := false, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}

	e = &model.Endpoint{
		IPv6: net.IPv6zero,
	}

	if want, have := false, e.Empty(); want != have {
		t.Errorf("Endpoint want %t, have %t", want, have)
	}
}
