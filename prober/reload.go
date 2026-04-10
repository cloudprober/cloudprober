package prober

import (
    "fmt"

    configpb "github.com/cloudprober/cloudprober/config/proto"
    probes_configpb "github.com/cloudprober/cloudprober/probes/proto"
    "google.golang.org/protobuf/encoding/prototext"
)

// ApplyProbeDiffs applies minimal probe changes between old and new configs.
func ApplyProbeDiffs(pr *Prober, oldCfg, newCfg *configpb.ProberConfig) error {
    oldMap := map[string]*probes_configpb.ProbeDef{}
    newMap := map[string]*probes_configpb.ProbeDef{}

    if oldCfg != nil {
        for _, p := range oldCfg.GetProbe() {
            oldMap[p.GetName()] = p
        }
    }
    if newCfg != nil {
        for _, p := range newCfg.GetProbe() {
            newMap[p.GetName()] = p
        }
    }

    // Removed probes
    for name := range oldMap {
        if _, ok := newMap[name]; !ok {
            pr.mu.Lock()
            if cancel, ok := pr.probeCancelFunc[name]; ok {
                cancel()
                delete(pr.probeCancelFunc, name)
            }
            delete(pr.Probes, name)
            pr.mu.Unlock()
        }
    }

    // New or modified probes
    for name, newP := range newMap {
        oldP, exists := oldMap[name]
        if !exists {
            if err := pr.addProbe(newP); err != nil {
                return fmt.Errorf("failed to add probe %s: %v", name, err)
            }
            pr.startProbe(name)
            continue
        }

        // Compare
        oldTxt := prototext.Format(oldP)
        newTxt := prototext.Format(newP)
        if oldTxt != newTxt {
            pr.mu.Lock()
            if cancel, ok := pr.probeCancelFunc[name]; ok {
                cancel()
                delete(pr.probeCancelFunc, name)
            }
            delete(pr.Probes, name)
            pr.mu.Unlock()

            if err := pr.addProbe(newP); err != nil {
                return fmt.Errorf("failed to replace probe %s: %v", name, err)
            }
            pr.startProbe(name)
        }
    }

    return nil
}
