// Package inventory is the hypervisor-neutral record kcore-migrate writes
// Terraform from. VMware and Proxmox each implement Importer.
package inventory

import "context"

const (
	SourceVMware  = "vmware"
	SourceProxmox = "proxmox"
)

// Config is what every importer needs to reach its API. Password comes from
// the process environment, not from a flag, so it does not land in shell history.
type Config struct {
	Endpoint string
	Username string
	Password string
	Insecure bool
}

type Disk struct {
	Name      string
	SizeBytes int64
	Bus       string
}

type NIC struct {
	Network string
	Model   string
	MAC     string
}

// Guest is one workload, independent of which hypervisor it came from.
type Guest struct {
	Name        string
	CPU         int
	MemoryBytes int64
	GuestOS     string
	Disks       []Disk
	NICs        []NIC
	// Findings are reasons this guest cannot be represented on kcore yet.
	Findings []string
}

// Snapshot is one read-only collection. It does not include disk bytes.
type Snapshot struct {
	Source string
	Guests []Guest
}

// Importer reads a hypervisor. Implementations must not copy disks.
type Importer interface {
	Kind() string
	Read(ctx context.Context) (Snapshot, error)
}
