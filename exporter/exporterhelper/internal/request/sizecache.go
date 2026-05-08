// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"

const uncachedSize = -1

type SizeCache struct {
	size     int
	sizeType SizerType
}

func NewSizeCache() SizeCache {
	return SizeCache{size: uncachedSize}
}

func (sc *SizeCache) Load(sizeType SizerType) (int, bool) {
	return sc.size, sc.size != uncachedSize && sc.sizeType == sizeType
}

func (sc *SizeCache) Store(size int, sizeType SizerType) {
	sc.size = size
	sc.sizeType = sizeType
}
