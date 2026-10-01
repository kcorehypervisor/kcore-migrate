// Package proxmox reads a Proxmox VE cluster into the same guest record as
// VMware. The API client lists QEMU and LXC guests and does not download disks.
package proxmox

import (
	"context"
	"errors"
	"fmt"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

// Reader lists guests from the Proxmox API.
type Reader interface {
	Guests(ctx context.Context) ([]inventory.Guest, error)
}

// Source implements inventory.Importer for Proxmox.
type Source struct {
	cfg    inventory.Config
	reader Reader
}

// Open checks the connection settings and returns a source whose Read
// call lists guests through the Proxmox API.
func Open(cfg inventory.Config) (*Source, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("proxmox: --endpoint is required")
	}
	if cfg.Username == "" {
		return nil, errors.New("proxmox: --username is required")
	}
	if cfg.Password == "" {
		return nil, errors.New("proxmox: password is empty; set PROXMOX_PASSWORD or --password-env")
	}
	return &Source{cfg: cfg, reader: apiReader{cfg: cfg}}, nil
}

// WithReader swaps the API client. Tests use this.
func (s *Source) WithReader(r Reader) *Source {
	s.reader = r
	return s
}

func (s *Source) Kind() string { return inventory.SourceProxmox }

func (s *Source) Read(ctx context.Context) (inventory.Snapshot, error) {
	guests, err := s.reader.Guests(ctx)
	if err != nil {
		return inventory.Snapshot{}, fmt.Errorf("proxmox inventory: %w", err)
	}
	return inventory.Snapshot{Source: inventory.SourceProxmox, Guests: guests}, nil
}
