package configmapprovider

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWatchFile(t *testing.T) {
	// Create a temporary file.
	file, err := ioutil.TempFile("", "file_watcher_test")
	require.NoError(t, err)
	defer func() {
		file.Close()
		os.Remove(file.Name())
	}()

	received := false

	// Write some initial content.
	_, err = file.WriteString("hello")
	require.NoError(t, err)
	require.NoError(t, file.Sync())

	// Setup the watch.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchFile(ctx, file.Name(), testingOnChange(t, &received))
	time.Sleep(time.Second * 2)

	// Update the file and verify we see the updated content.
	_, err = file.WriteString(" world")
	require.NoError(t, err)
	require.NoError(t, file.Sync())

	require.Eventually(t, func() bool {
		return received
	}, time.Second*10, time.Second)

	// Cancel the context.
	cancel()
}

func TestWatchFile_ReloadError(t *testing.T) {
	// Create then delete a temporary file so we have a filename that we know can't be opened.
	file, err := ioutil.TempFile("", "file_watcher_test")
	require.NoError(t, err)
	require.NoError(t, os.Remove(file.Name()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watchFile(ctx, file.Name(), func(event *ChangeEvent) {})
}

func testingOnChange(t *testing.T, r *bool) func(c *ChangeEvent) {
	return func(c *ChangeEvent) {
		require.Nil(t, c.Error)
		*r = true
	}
}
