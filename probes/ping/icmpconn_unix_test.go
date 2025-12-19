// Copyright 2020 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package ping

import (
	"bytes"
	"net"
	"syscall"
	"testing"
)

func TestSockaddr(t *testing.T) {
	tests := []struct {
		desc         string
		ipStr        string
		ipVer        int
		expectErr    bool
		expectedAddr []byte
	}{
		{
			desc:         "IPv4 standard: ensures 16-byte slice is converted to 4-byte array",
			ipStr:        "172.70.220.41",
			ipVer:        4,
			expectErr:    false,
			expectedAddr: []byte{172, 70, 220, 41},
		},
		{
			desc:         "IPv6 standard",
			ipStr:        "2001:db8::68",
			ipVer:        6,
			expectErr:    false,
			expectedAddr: []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x68},
		},
		{
			desc:         "IPv4 mapped to IPv6: requesting v6 for v4 IP is allowed (mapped)",
			ipStr:        "1.1.1.1",
			ipVer:        6,
			expectedAddr: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 1, 1, 1, 1},
		},
		{
			desc:      "IPv6 mismatch: requesting v4 for v6 IP",
			ipStr:     "2001:db8::1",
			ipVer:     4,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ip := net.ParseIP(tc.ipStr)
			if ip == nil {
				t.Fatalf("Failed to parse test IP: %s", tc.ipStr)
			}

			sa, err := sockaddr(ip, tc.ipVer)

			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.desc)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error for %s: %v", tc.desc, err)
			}

			var gotAddr []byte

			switch v := sa.(type) {
			case *syscall.SockaddrInet4:
				gotAddr = v.Addr[:]
			case *syscall.SockaddrInet6:
				gotAddr = v.Addr[:]
			default:
				t.Fatalf("Returned sockaddr is not of type Inet4 or Inet6")
			}

			if !bytes.Equal(gotAddr, tc.expectedAddr) {
				t.Errorf("Bind Address Mismatch! Got:  %v, Want: %v", gotAddr, tc.expectedAddr)
			}
		})
	}
}
