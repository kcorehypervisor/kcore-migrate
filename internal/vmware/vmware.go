// Package vmware is the first inventory source. Reader is the vSphere
// client. The default implementation uses govmomi and lists virtual machines
// without copying disk bytes.
package vmware

import (
	"context"
	"errors"
	"fmt"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

// Reader lists guests from vSphere.
type Reader interface {
	Guests(ctx context.Context) ([]inventory.Guest, error)
}

// Source implements inventory.Importer for VMware.
type Source struct {
	cfg    inventory.Config
	reader Reader
}

// Open checks the connection settings and returns a source whose Read
// call lists virtual machines through govmomi.
func Open(cfg inventory.Config) (*Source, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("vmware: --endpoint is required")
	}
	if cfg.Username == "" {
		return nil, errors.New("vmware: --username is required")
	}
	if cfg.Password == "" {
		return nil, errors.New("vmware: password is empty; set VMWARE_PASSWORD or --password-env")
	}
	return &Source{cfg: cfg, reader: apiReader{cfg: cfg}}, nil
}

// WithReader swaps the govmomi client. Tests use this.
func (s *Source) WithReader(r Reader) *Source {
	s.reader = r
	return s
}

func (s *Source) Kind() string { return inventory.SourceVMware }

func (s *Source) Read(ctx context.Context) (inventory.Snapshot, error) {
	guests, err := s.reader.Guests(ctx)
	if err != nil {
		return inventory.Snapshot{}, fmt.Errorf("vmware inventory: %w", err)
	}
	return inventory.Snapshot{Source: inventory.SourceVMware, Guests: guests}, nil
}
