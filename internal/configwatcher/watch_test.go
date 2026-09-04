package configwatcher

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/cloudprober/cloudprober/logger"
    "github.com/cloudprober/cloudprober/state"
)

func TestWatcherDebounce(t *testing.T) {
    tmpfile, err := os.CreateTemp("", "cfg-*.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    // Set state so watcher will pick up the path
    state.SetConfigFilePath(tmpfile.Name())

    l := logger.NewWithAttrs()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    w, err := NewWatcher(ctx, nil, l, 200)
    if err != nil {
        t.Fatalf("NewWatcher error: %v", err)
    }

    // Write to the file twice quickly and ensure only one debounced reload occurs
    go func() {
        time.Sleep(50 * time.Millisecond)
        os.WriteFile(tmpfile.Name(), []byte("v1"), 0644)
        time.Sleep(20 * time.Millisecond)
        os.WriteFile(tmpfile.Name(), []byte("v2"), 0644)
    }()

    select {
    case <-w.Reload:
        // OK
    case <-time.After(2 * time.Second):
        t.Fatalf("timed out waiting for reload event")
    }

    w.Stop()
}
