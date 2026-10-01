package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
	"github.com/kcorehypervisor/kcore-migrate/internal/proxmox"
	"github.com/kcorehypervisor/kcore-migrate/internal/vmware"
)

func TestOpenSelectsSource(t *testing.T) {
	cfg := inventory.Config{Endpoint: "https://example", Username: "user", Password: "secret"}
	for _, kind := range []string{inventory.SourceVMware, inventory.SourceProxmox} {
		imp, err := Open(kind, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if imp.Kind() != kind {
			t.Fatalf("kind %s, got %s", kind, imp.Kind())
		}
		_, err = imp.Read(context.Background())
		if err == nil {
			t.Fatal("expected unread client")
		}
	}
	if _, err := Open("kvm", cfg); err == nil {
		t.Fatal("unknown source was accepted")
	}
	if _, err := Open(inventory.SourceVMware, inventory.Config{}); err == nil {
		t.Fatal("missing endpoint was accepted")
	}
}

func TestVMwareReaderFeedsSnapshot(t *testing.T) {
	src, err := vmware.Open(inventory.Config{Endpoint: "https://vc", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	src.WithReader(fake{"vm-1"})
	snap, err := src.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != inventory.SourceVMware || len(snap.Guests) != 1 || snap.Guests[0].Name != "vm-1" {
		t.Fatalf("%+v", snap)
	}
}

func TestProxmoxNotReady(t *testing.T) {
	src, err := proxmox.Open(inventory.Config{Endpoint: "https://pve", Username: "root@pam", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = src.Read(context.Background())
	if !errors.Is(err, proxmox.ErrNotReady) {
		t.Fatal(err)
	}
}

type fake struct{ name string }

func (f fake) Guests(context.Context) ([]inventory.Guest, error) {
	return []inventory.Guest{{Name: f.name, CPU: 1, MemoryBytes: 1}}, nil
}
