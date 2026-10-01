package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "-h", "--help", "help":
		usage()
	case "inventory":
		fmt.Fprintln(os.Stderr, "kcore-migrate inventory is not implemented yet")
		os.Exit(1)
	case "convert":
		fmt.Fprintln(os.Stderr, "kcore-migrate convert is not implemented yet")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `kcore-migrate — VMware exit tools for kcore

Usage:
  kcore-migrate inventory   Read a vCenter and write Terraform plus a gap report
  kcore-migrate convert     Convert guest disks to qcow2 or raw

inventory talks only to vCenter. It does not copy disks and does not need a kcore cluster.
convert will shell out to virt-v2v or qemu-img. It will not reimplement VMDK conversion.
`)
}
