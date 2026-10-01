# kcore-migrate

Tools for leaving VMware onto [kcore](https://kcorehypervisor.com/). This repository is separate from the hypervisor and from `kctl`.

`kctl` administers a cluster that already exists. A VMware exit assessment runs on a jump host that can see vCenter, often before any kcore node is installed. vCenter credentials stay in this tool. They do not sit next to operator client certificates.

## Commands

| Command | What it does | Status |
|---------|----------------|--------|
| `kcore-migrate inventory` | Read-only vCenter inventory. Writes Terraform for `kcore_vm` and a gap report. | Not implemented yet |
| `kcore-migrate convert` | Turns guest disks into `qcow2` or `raw` by shelling out to `virt-v2v` or `qemu-img`. | Not implemented yet |

Apply stays in Terraform. [`terraform-provider-kcore`](https://github.com/kcorehypervisor/terraform-provider-kcore) creates the VMs. `kctl node upload-image` is how a converted disk gets onto a node.

`inventory` does not copy disk bytes. It is safe to run during an assessment. The gap report is the list of things kcore cannot represent yet: snapshots, RDM, vGPU, DRS rules, Fault Tolerance, shared disks, and Windows guests that still need virtio.

`convert` does not reimplement VMDK conversion.

## What the generated files are

`inventory` will write two artifacts from one read of vCenter:

- Terraform HCL that matches the current `kcore_vm` schema.
- A migration plan that names each source disk and the target image path `convert` will fill in.

The provider today registers `kcore_vm` only. Generated NICs reference networks that already exist until a network resource is added. `storage_backend` on `kcore_vm` does not yet include Ceph. The generator must not emit HCL the provider cannot apply.

## Build

```bash
go build -o kcore-migrate ./cmd/kcore-migrate
./kcore-migrate help
```

## License

Apache-2.0. See [LICENSE](LICENSE).
