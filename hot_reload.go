package cloudprober

import (
    "fmt"
    "log/slog"
    "sync"

    "github.com/cloudprober/cloudprober/logger"
    "github.com/cloudprober/cloudprober/prober"
)

var hotReloadMu sync.Mutex

// reloadConfigAndApply loads the new configuration and applies probe diffs.
func reloadConfigAndApply() error {
    hotReloadMu.Lock()
    defer hotReloadMu.Unlock()

    // Load new config. Access configSource under read-lock to avoid nil deref
    cloudProber.RLock()
    cs := cloudProber.configSource
    cloudProber.RUnlock()

    if cs == nil {
        return fmt.Errorf("config source is nil")
    }

    newCfg, err := cs.GetConfig()
    if err != nil {
        return fmt.Errorf("failed to get config: %w", err)
    }

    pr := GetProber()
    if pr == nil {
        return fmt.Errorf("prober not initialized")
    }

    oldCfg := GetConfig()

    if err := prober.ApplyProbeDiffs(pr, oldCfg, newCfg); err != nil {
        return err
    }

    // Update global config
    cloudProber.Lock()
    cloudProber.config = newCfg
    cloudProber.Unlock()

    l := logger.NewWithAttrs(slog.String("component", "hot-reload"))
    l.Infof("config reload applied successfully")
    return nil
}

