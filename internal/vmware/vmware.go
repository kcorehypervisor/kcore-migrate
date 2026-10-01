// Package vmware is the first inventory source. The vSphere client is a
// Reader so a later govmomi implementation can replace the placeholder
// without changing the CLI or the Terraform writer.
package vmware

import (
	"context"
	"errors"
	"fmt"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

// ErrNotReady means the vSphere API client is not connected yet.
var ErrNotReady = errors.New("vmware: vSphere inventory client is not connected")

// Reader lists guests. A real implementation will use the vSphere API.
type Reader interface {
	Guests(ctx context.Context) ([]inventory.Guest, error)
}

type pendingReader struct{}

func (pendingReader) Guests(context.Context) ([]inventory.Guest, error) {
	return nil, ErrNotReady
}

// Source implements inventory.Importer for VMware.
type Source struct {
	cfg    inventory.Config
	reader Reader
}

// Open checks the connection settings and returns a source. Read fails with
// ErrNotReady until a Reader is installed with WithReader.
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
	return &Source{cfg: cfg, reader: pendingReader{}}, nil
}

// WithReader swaps the placeholder client. Tests use this; the vSphere
// client will too.
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
