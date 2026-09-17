package configwatcher

import (
    "context"
    "sync"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/cloudprober/cloudprober/config"
    "github.com/cloudprober/cloudprober/logger"
    "github.com/cloudprober/cloudprober/state"
)

// Watcher watches config files and emits debounced reload events.
type Watcher struct {
    Reload   <-chan struct{}
    stop     context.CancelFunc
    err      error
    mu       sync.Mutex
}

// NewWatcher creates a new watcher for the given ConfigSource.
func NewWatcher(ctx context.Context, cs config.ConfigSource, l *logger.Logger, debounceMs int) (*Watcher, error) {
    w := &Watcher{}

    // Channel that will receive raw events and debounced events.
    rawCh := make(chan struct{}, 1)
    debCh := make(chan struct{})

    // Create fsnotify watcher
    fw, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    // Try to determine files to watch from state.ConfigFilePath(). If set,
    // watch that file. This mirrors how config.ConfigSourceWithFile sets the
    // path in state during GetConfig().
    cfgPath := state.ConfigFilePath()
    if cfgPath != "" {
        if err := fw.Add(cfgPath); err == nil {
            l.Debugf("watching config file: %s", cfgPath)
        } else {
            l.Warningf("failed to watch config file %s: %v", cfgPath, err)
        }
    }

    // Start a goroutine to coalesce raw events into debounced events.
    go func() {
        var timer *time.Timer
        var timerC <-chan time.Time
        for {
            select {
            case <-ctx.Done():
                fw.Close()
                close(debCh)
                return
            case <-rawCh:
                if timer == nil {
                    timer = time.NewTimer(time.Millisecond * time.Duration(debounceMs))
                    timerC = timer.C
                } else {
                    if !timer.Stop() {
                        <-timer.C
                    }
                    timer.Reset(time.Millisecond * time.Duration(debounceMs))
                }
            case <-timerC:
                debCh <- struct{}{}
                timer = nil
                timerC = nil
            }
        }
    }()

    // Start a goroutine to read events from fsnotify and push into rawCh.
    go func() {
        for {
            select {
            case <-ctx.Done():
                fw.Close()
                return
            case ev, ok := <-fw.Events:
                if !ok {
                    return
                }
                _ = ev // we don't filter much here
                select {
                case rawCh <- struct{}{}:
                default:
                }
            case err, ok := <-fw.Errors:
                if !ok {
                    return
                }
                l.Errorf("fsnotify error: %v", err)
            }
        }
    }()

    // Expose debCh as read-only Reload channel
    w.Reload = debCh
    w.stop = func() { fw.Close() }

    return w, nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
    w.mu.Lock()
    defer w.mu.Unlock()
    if w.stop != nil {
        w.stop()
        w.stop = nil
    }
}
