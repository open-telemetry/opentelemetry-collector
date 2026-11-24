// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import (
	"hash/maphash"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/publicsuffix"
)

const (
	// maxCookieBytes is the size of the largest cookie that is stored, and
	// maxCookies the number of distinct cookies stored before the jar is emptied.
	// Both are the minimum capabilities RFC 6265 section 6.1 asks of a user agent.
	maxCookieBytes = 4096
	maxCookies     = 3000
)

// boundedCookieJar is an http.CookieJar whose size a server cannot grow without
// limit. A cookiejar.Jar keeps every cookie it is given, whatever its size and
// however many there are, for as long as the client lives, which for a client
// built by ToClient is the lifetime of the component.
type boundedCookieJar struct {
	seed maphash.Seed

	mu  sync.Mutex
	jar *cookiejar.Jar
	// stored holds a hash of every distinct cookie given to jar.
	// cookiejar.Jar cannot be enumerated, so cookies that it removed on its
	// own (expired or deleted by the server) are still counted.
	stored map[uint64]struct{}
}

func newCookieJar() (*cookiejar.Jar, error) {
	return cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
}

func newBoundedCookieJar() (*boundedCookieJar, error) {
	jar, err := newCookieJar()
	if err != nil {
		return nil, err
	}
	return &boundedCookieJar{
		seed:   maphash.MakeSeed(),
		jar:    jar,
		stored: make(map[uint64]struct{}),
	}, nil
}

// SetCookies stores the cookies of a response to u, except those larger than
// maxCookieBytes. If storing them would leave the jar holding more than
// maxCookies distinct cookies, the jar is emptied first.
func (j *boundedCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	kept := make([]*http.Cookie, 0, min(len(cookies), maxCookies))
	keys := make([]uint64, 0, cap(kept))
	for _, c := range cookies {
		if cookieSize(c) <= maxCookieBytes && len(kept) < maxCookies {
			kept = append(kept, c)
			keys = append(keys, j.key(u, c))
		}
	}
	if len(kept) == 0 {
		return
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	added := 0
	for _, k := range keys {
		if _, ok := j.stored[k]; !ok {
			added++
		}
	}
	// Start over with the cookies of this response, so that a cookie the
	// server sets again with every response, like a sticky session, survives.
	if len(j.stored)+added > maxCookies {
		if fresh, err := newCookieJar(); err == nil {
			j.jar = fresh
			clear(j.stored)
		}
	}
	for _, k := range keys {
		j.stored[k] = struct{}{}
	}
	j.jar.SetCookies(u, kept)
}

func (j *boundedCookieJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.jar.Cookies(u)
}

// cookieSize is the size of c as RFC 6265 measures it: the length of its
// name, value and attributes together.
func cookieSize(c *http.Cookie) int {
	return max(len(c.Raw), len(c.Name)+len(c.Value)+len(c.Domain)+len(c.Path))
}

// key identifies the entry that cookiejar.Jar stores c under when it is set
// by a response to u. It may tell apart cookies that the jar stores as one
// entry, since the jar normalizes the domain and path, and it may also
// count as one cookie two entries that the jar keeps apart (an explicit
// Path=/p and a defaulted path from a response to /p). The jar is therefore
// bounded at roughly twice maxCookies entries.
func (j *boundedCookieJar) key(u *url.URL, c *http.Cookie) uint64 {
	path := c.Path
	if !strings.HasPrefix(path, "/") {
		path = u.Path
	}
	return maphash.Comparable(j.seed, [4]string{u.Host, c.Domain, path, c.Name})
}
