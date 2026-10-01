// Package proxmox is the second inventory source. It accepts the same
// connection settings as VMware and returns a snapshot of the same shape.
// The API client is not written yet.
package proxmox

import (
	"context"
	"errors"
	"fmt"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

// ErrNotReady means the Proxmox API client is not connected yet.
var ErrNotReady = errors.New("proxmox: API inventory client is not connected")

// Reader lists guests. A real implementation will use the Proxmox API.
type Reader interface {
	Guests(ctx context.Context) ([]inventory.Guest, error)
}

type pendingReader struct{}

func (pendingReader) Guests(context.Context) ([]inventory.Guest, error) {
	return nil, ErrNotReady
}

// Source implements inventory.Importer for Proxmox.
type Source struct {
	cfg    inventory.Config
	reader Reader
}

// Open checks the connection settings. Read fails with ErrNotReady until a
// Reader is installed with WithReader.
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
	return &Source{cfg: cfg, reader: pendingReader{}}, nil
}

// WithReader swaps the placeholder client.
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
