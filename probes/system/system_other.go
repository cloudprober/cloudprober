//go:build !linux
// +build !linux

package system

import "fmt"

func diskUsage(path string) (uint64, uint64, error) {
	return 0, 0, fmt.Errorf("disk usage not supported on this OS")
}
