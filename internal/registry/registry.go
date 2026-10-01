// Package registry selects an inventory importer by name. Adding a hypervisor
// means a new package plus one case here. The Terraform writer does not change.
package registry

import (
	"fmt"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
	"github.com/kcorehypervisor/kcore-migrate/internal/proxmox"
	"github.com/kcorehypervisor/kcore-migrate/internal/vmware"
)

// Open returns the importer for kind. kind is "vmware" or "proxmox".
func Open(kind string, cfg inventory.Config) (inventory.Importer, error) {
	switch kind {
	case inventory.SourceVMware:
		return vmware.Open(cfg)
	case inventory.SourceProxmox:
		return proxmox.Open(cfg)
	default:
		return nil, fmt.Errorf("unknown import source %q (want %s or %s)", kind, inventory.SourceVMware, inventory.SourceProxmox)
	}
}
