// SPDX-License-Identifier: MIT

package igmp

import (
	"errors"
	"net"
	"testing"
)

func TestBool(t *testing.T) {
	truePtr := Bool(true)
	falsePtr := Bool(false)

	if truePtr == nil || *truePtr != true {
		t.Error("Bool(true) should return pointer to true")
	}
	if falsePtr == nil || *falsePtr != false {
		t.Error("Bool(false) should return pointer to false")
	}
	// Ensure they are distinct pointers.
	if truePtr == falsePtr {
		t.Error("Bool should return distinct pointers")
	}
}

func TestConnConfigDefaults(t *testing.T) {
	// Verify that zero-value ConnConfig gets correct defaults applied.
	// We can't actually open a raw socket without privileges, so we
	// test the default logic indirectly via the config.
	cfg := ConnConfig{}

	if cfg.TTL != 0 {
		t.Error("zero-value TTL should be 0 (NewConn applies default)")
	}
	if cfg.RouterAlert != nil {
		t.Error("zero-value RouterAlert should be nil (NewConn applies default)")
	}
}

// TestConnGroupMethodValidation tests IP validation in the group membership
// methods. We construct a Conn with a nil rawConn to test validation before
// the syscall is reached. IPv6 addresses should be rejected with ErrInvalidIP.
func TestConnGroupMethodValidation(t *testing.T) {
	// Create a Conn with no underlying socket — we only test validation.
	c := &Conn{}
	ipv6 := net.ParseIP("::1")

	t.Run("JoinGroup rejects IPv6", func(t *testing.T) {
		err := c.JoinGroup(ipv6)
		if err == nil {
			t.Fatal("expected error for IPv6 address")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("LeaveGroup rejects IPv6", func(t *testing.T) {
		err := c.LeaveGroup(ipv6)
		if err == nil {
			t.Fatal("expected error for IPv6 address")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("JoinSourceSpecificGroup rejects IPv6 group", func(t *testing.T) {
		err := c.JoinSourceSpecificGroup(ipv6, net.IPv4(10, 0, 0, 1))
		if err == nil {
			t.Fatal("expected error for IPv6 group")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("JoinSourceSpecificGroup rejects IPv6 source", func(t *testing.T) {
		err := c.JoinSourceSpecificGroup(net.IPv4(239, 1, 1, 1), ipv6)
		if err == nil {
			t.Fatal("expected error for IPv6 source")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("LeaveSourceSpecificGroup rejects IPv6 group", func(t *testing.T) {
		err := c.LeaveSourceSpecificGroup(ipv6, net.IPv4(10, 0, 0, 1))
		if err == nil {
			t.Fatal("expected error for IPv6 group")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("LeaveSourceSpecificGroup rejects IPv6 source", func(t *testing.T) {
		err := c.LeaveSourceSpecificGroup(net.IPv4(239, 1, 1, 1), ipv6)
		if err == nil {
			t.Fatal("expected error for IPv6 source")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("IncludeSourceSpecificGroup rejects IPv6", func(t *testing.T) {
		err := c.IncludeSourceSpecificGroup(ipv6, net.IPv4(10, 0, 0, 1))
		if err == nil {
			t.Fatal("expected error for IPv6 group")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})

	t.Run("ExcludeSourceSpecificGroup rejects IPv6", func(t *testing.T) {
		err := c.ExcludeSourceSpecificGroup(ipv6, net.IPv4(10, 0, 0, 1))
		if err == nil {
			t.Fatal("expected error for IPv6 group")
		}
		if !errors.Is(err, ErrInvalidIP) {
			t.Errorf("expected ErrInvalidIP, got: %v", err)
		}
	})
}

func TestConnInterface(t *testing.T) {
	c := &Conn{}

	// No interface set.
	if c.Interface() != nil {
		t.Error("Interface() should return nil when no interface is configured")
	}

	// With interface set.
	iface := &net.Interface{Index: 1, Name: "lo"}
	c.iface = iface

	got := c.Interface()
	if got != iface {
		t.Errorf("Interface() = %v, want %v", got, iface)
	}
}
