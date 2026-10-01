package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
	"github.com/kcorehypervisor/kcore-migrate/internal/plan"
	"github.com/kcorehypervisor/kcore-migrate/internal/registry"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		usage()
	case "version":
		fmt.Println(version)
	case "inventory":
		if err := cmdInventory(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "convert":
		fmt.Fprintln(os.Stderr, "kcore-migrate convert is not implemented yet")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func cmdInventory(args []string) error {
	fs := flag.NewFlagSet("inventory", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	kind := fs.String("source", inventory.SourceVMware, "hypervisor to read: vmware or proxmox")
	endpoint := fs.String("endpoint", "", "API endpoint (vCenter or Proxmox)")
	username := fs.String("username", "", "API user")
	passwordEnv := fs.String("password-env", "", "environment variable that holds the password")
	insecure := fs.Bool("insecure", false, "skip TLS verification")
	out := fs.String("out", "migrate-out", "directory for guests.tf and gaps.md")
	if err := fs.Parse(args); err != nil {
		return err
	}
	envName := *passwordEnv
	if envName == "" {
		if *kind == inventory.SourceProxmox {
			envName = "PROXMOX_PASSWORD"
		} else {
			envName = "VMWARE_PASSWORD"
		}
	}
	imp, err := registry.Open(*kind, inventory.Config{
		Endpoint: *endpoint,
		Username: *username,
		Password: os.Getenv(envName),
		Insecure: *insecure,
	})
	if err != nil {
		return err
	}
	snap, err := imp.Read(context.Background())
	if err != nil {
		return err
	}
	if err := plan.Write(*out, snap); err != nil {
		return err
	}
	fmt.Printf("wrote %s from %s (%d guests)\n", *out, snap.Source, len(snap.Guests))
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `kcore-migrate %s — import a hypervisor estate onto kcore

Usage:
  kcore-migrate inventory --source vmware|proxmox --endpoint URL --username USER
  kcore-migrate convert
  kcore-migrate version

inventory reads one hypervisor and writes Terraform plus a gap report.
It does not copy disks. --source selects the importer; vmware and proxmox
share the same guest record. The password is read from VMWARE_PASSWORD or
PROXMOX_PASSWORD, or from the variable named by --password-env.

convert will shell out to virt-v2v or qemu-img. It is not implemented yet.
`, version)
}
