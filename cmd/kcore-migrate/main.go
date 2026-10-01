package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kcorehypervisor/kcore-migrate/internal/convert"
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
		if err := cmdConvert(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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

func cmdConvert(args []string) error {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	input := fs.String("input", "", "exported OVA, VMX, VMDK, qcow2, or raw disk")
	out := fs.String("out", "", "virt-v2v output directory, or qemu-img output file")
	format := fs.String("format", "qcow2", "output format: qcow2 or raw")
	tool := fs.String("tool", "", "virt-v2v or qemu-img; chosen from the input name when omitted")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cmd, err := convert.Run(context.Background(), convert.Options{
		Input:  *input,
		Output: *out,
		Format: *format,
		Tool:   *tool,
	})
	if err != nil {
		return err
	}
	if cmd.Directory {
		fmt.Printf("converted disks are in %s\n", cmd.Result)
		fmt.Printf("set image_path to the %s file in that directory and image_format = %q\n", *format, *format)
		return nil
	}
	fmt.Printf("wrote %s\n", cmd.Result)
	fmt.Printf("set image_path = %q and image_format = %q\n", cmd.Result, *format)
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `kcore-migrate %s — import a hypervisor estate onto kcore

Usage:
  kcore-migrate inventory --source vmware|proxmox --endpoint URL --username USER
  kcore-migrate convert --input FILE [--out PATH] [--format qcow2|raw] [--tool virt-v2v|qemu-img]
  kcore-migrate version

inventory reads one hypervisor and writes Terraform plus a gap report.
It does not copy disks. --source selects the importer; vmware and proxmox
share the same guest record. The password is read from VMWARE_PASSWORD or
PROXMOX_PASSWORD, or from the variable named by --password-env.

convert shells out to virt-v2v or qemu-img. An OVA, VMX, or VMDK uses virt-v2v.
A qcow2, raw, or img file uses qemu-img. The output is qcow2 or raw.
`, version)
}
