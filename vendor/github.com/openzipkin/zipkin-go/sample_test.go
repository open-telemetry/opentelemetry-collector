package zipkin_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	zipkin "github.com/openzipkin/zipkin-go"
)

func TestBoundarySampler(t *testing.T) {
	type triple struct {
		id       uint64
		salt     int64
		rate     float64
		hasError bool
	}
	for input, sampled := range map[triple]bool{
		{123, 456, 1.0, false}:    true,
		{123, 456, 999, true}:     true,
		{123, 456, 0.0, false}:    false,
		{123, 456, -42, true}:     false,
		{1229998, 0, 0.01, false}: false,
		{1229999, 0, 0.01, false}: false,
		{1230000, 0, 0.01, false}: true,
		{1230001, 0, 0.01, false}: true,
		{1230098, 0, 0.01, false}: true,
		{1230099, 0, 0.01, false}: true,
		{1230100, 0, 0.01, false}: false,
		{1230101, 0, 0.01, false}: false,
		{1, 9999999, 0.01, false}: false,
		{999, 0, 0.99, false}:     true,
		{9999, 0, 0.99, false}:    false,
	} {
		sampler, err := zipkin.NewBoundarySampler(input.rate, input.salt)
		if want, have := input.hasError, (err != nil); want != have {
			t.Fatalf("%#+v: want error %t, have error %t", input, want, have)
		}
		if input.hasError {
			want := fmt.Errorf("rate should be 0.0 or between 0.0001 and 1: was %f", input.rate)
			if have := err; have == nil || want.Error() != have.Error() {
				t.Fatalf("%#+v: want error %+v, have error %+v", input, want, have)
			}
			continue
		}
		if want, have := sampled, sampler(input.id); want != have {
			t.Errorf("%#+v: want %v, have %v", input, want, have)
		}
	}
}

func TestCountingSampler(t *testing.T) {
	{
		_, have := zipkin.NewCountingSampler(0.009)
		want := fmt.Errorf("rate should be 0.0 or between 0.01 and 1: was %f", 0.009)
		if have == nil || want.Error() != have.Error() {
			t.Errorf("rate 0.009, want error %+v, got %+v", want, have)
		}
	}
	{
		_, have := zipkin.NewCountingSampler(1.001)
		want := fmt.Errorf("rate should be 0.0 or between 0.01 and 1: was %f", 1.001)
		if have == nil || want.Error() != have.Error() {
			t.Errorf("rate 1.001, want error %+v, got %+v", want, have)
		}
	}
	for n := 0; n <= 100; n++ {
		var (
			rate       = float64(n) / 100
			sampler, _ = zipkin.NewCountingSampler(rate)
			found      = 0
		)
		for i := 0; i < 1000; i++ {
			if sampler(1) {
				found++
			}
		}
		if found != n*10 {
			t.Errorf("rate %f, want %d, have %d", rate, n, found)
		}
	}
}

func TestModuleSampler(t *testing.T) {
	rand.Seed(time.Now().Unix())

	for mod := uint64(1); mod <= 100; mod++ {
		var (
			sampler = zipkin.NewModuloSampler(mod)
			want    = uint64(rand.Intn(1000))
			max     = mod * want
			found   = uint64(0)
		)

		for i := uint64(0); i < max; i++ {
			if sampler(i) {
				found++
			}
		}

		if want, have := max/mod, found; want != have {
			t.Errorf("expected %d samples, got %d", want, have)
		}
	}

}
