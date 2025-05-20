// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package limiterhelper

type notification struct {
	c chan struct{}
}

func newNotification() notification {
	return notification{c: make(chan struct{})}
}

func (n *notification) notice() {
	close(n.c)
}

func (n *notification) hasBeen() bool {
	select {
	case <-n.c:
		return true
	default:
		return false
	}
}

func (n *notification) waitFor() {
	<-n.c
}

func (n *notification) channel() <-chan struct{} {
	return n.c
}
