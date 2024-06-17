// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configrotate

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

func TestRotationEnabledCreate(t *testing.T) {
	tempDir := t.TempDir()
	filename := path.Join(tempDir, "test.log")
	maxMegabytes := 1524
	maxDays := 145
	maxBackups := 155
	localTime := true
	rotateConfig := Config{
		Enabled:      true,
		MaxMegabytes: maxMegabytes,
		MaxDays:      maxDays,
		MaxBackups:   maxBackups,
		LocalTime:    localTime,
	}
	writer, err := rotateConfig.NewWriter(filename)
	assert.NoError(t, err)
	rotate, ok := writer.(*lumberjack.Logger)
	assert.True(t, ok)
	assert.Equal(t, rotate.Filename, filename)
	assert.Equal(t, rotate.MaxSize, maxMegabytes)
	assert.Equal(t, rotate.MaxAge, maxDays)
	assert.Equal(t, rotate.LocalTime, localTime)
}

func TestRotateDisabledCreate(t *testing.T) {
	rotateConfig := Config{
		Enabled: false,
	}
	tempDir := t.TempDir()
	filename := path.Join(tempDir, "test.log")
	writer, err := rotateConfig.NewWriter(filename)
	assert.NoError(t, err)
	file, ok := writer.(*os.File)
	assert.True(t, ok)
	assert.Equal(t, file.Name(), filename)
	assert.NoError(t, file.Close())
}

func TestErrorCreate(t *testing.T) {
	tempdir := t.TempDir()
	tests := []struct {
		name         string
		rotateConfig Config
		filename     string
		error        string
	}{
		{
			name: "invalid filename",
			rotateConfig: Config{
				Enabled: true,
			},
			filename: filepath.Join(tempdir, "invalid/filename"),
			error:    "no such file or directory",
		},
		{
			name: "negative max megabytes",
			rotateConfig: Config{
				Enabled:      true,
				MaxMegabytes: -1,
			},
			filename: filepath.Join(tempdir, "test.log"),
			error:    "invalid MaxMegabytes",
		},
		{
			name: "negative max days",
			rotateConfig: Config{
				Enabled: true,
				MaxDays: -1,
			},
			filename: filepath.Join(tempdir, "test.log"),
			error:    "invalid MaxDays",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := tt.rotateConfig.NewWriter(tt.filename)
			assert.ErrorContains(t, err, tt.error)
			assert.Nil(t, writer)
		})
	}

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	if runtime.GOOS == "windows" {
		time.Sleep(1 * time.Second)
	}
}

func TestCreateWithDefaultConfig(t *testing.T) {
	rotateConfig := NewDefaultRotateConfig()
	tempDir := t.TempDir()
	filename := path.Join(tempDir, "test.log")
	writer, err := rotateConfig.NewWriter(filename)
	assert.NoError(t, err)
	rotate, ok := writer.(*lumberjack.Logger)
	assert.True(t, ok)
	assert.Equal(t, rotate.Filename, filename)

	// Sleep for 1 second to make sure all processes using the files are completed
	// (on Windows fail to delete temp dir otherwise).
	if runtime.GOOS == "windows" {
		time.Sleep(1 * time.Second)
	}
}
