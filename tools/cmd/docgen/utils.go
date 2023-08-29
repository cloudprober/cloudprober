package main

import (
	"strings"

	"github.com/cloudprober/cloudprober/logger"
)

func kindToURL(kind string) string {
	if !strings.HasPrefix(kind, "cloudprober.") {
		return ""
	}
	parts := strings.SplitN(kind, ".", 3)
	if len(parts) > 2 {
		return *homeURL + parts[1] + ".html#" + kind
	}
	return ""
}

func arrangeIntoPackages(paths []string, l *logger.Logger) map[string][]string {
	packages := make(map[string][]string)
	for _, path := range paths {
		parts := strings.SplitN(path, ".", 3)
		if len(parts) < 3 {
			l.Warningf("Skipping %s, not enough parts in package", path)
			continue
		}
		if parts[0] != "cloudprober" {
			l.Warningf("Skipping %s, not a cloudprober package", path)
			continue
		}
		packages[parts[1]] = append(packages[parts[1]], path)
	}
	return packages
}
