# kcore-migrate

Tools for leaving VMware or Proxmox onto [kcore](https://kcorehypervisor.com/). This repository is separate from the hypervisor and from `kctl`.

`kctl` administers a cluster that already exists. An exit assessment runs on a jump host that can see the source hypervisor, often before any kcore node is installed.

## Import sources

`kcore-migrate inventory --source` selects an importer. VMware and Proxmox both produce the same guest record. The Terraform writer does not know which hypervisor it came from.

| `--source` | Status |
|------------|--------|
| `vmware` | Lists virtual machines from vCenter through [govmomi](https://github.com/vmware/govmomi). Disk bytes stay on the datastore. |
| `proxmox` | Same shape, for a later Proxmox API client. |

```bash
export VMWARE_PASSWORD=...
kcore-migrate inventory \
  --source vmware \
  --endpoint https://vcenter.example \
  --username migrate@vsphere.local \
  --out ./migrate-out
```

Proxmox uses `PROXMOX_PASSWORD`, or pass `--password-env` for either source.

`inventory` does not copy disk bytes. It writes:

- `guests.tf` for `kcore_vm`
- `gaps.md` for anything that guest record cannot express cleanly (extra disks, source findings)

[`terraform-provider-kcore`](https://github.com/kcorehypervisor/terraform-provider-kcore) applies the HCL. `convert` will shell out to the tools below and is not implemented yet.

The provider today registers `kcore_vm` only. Generated NICs name networks that must already exist. `storage_backend` is `filesystem` until the provider accepts Ceph. `kctl` accepts the finished disk as raw or qcow2.

## Conversion host

Disk conversion runs on a Linux jump host. `nix develop` on Linux includes the open-source tools. The Go toolchain in that shell still builds `kcore-migrate` on Darwin; the converters are Linux-only.

| Piece | Where it comes from | What it does |
|-------|---------------------|--------------|
| `kcore-migrate` | this repository | Reads the source inventory and, later, calls the converters. Inventory does not copy disk bytes. |
| `virt-v2v` | nixpkgs, in the Linux shell | Converts an exported VMware guest (OVA, VMX, or VMDK) to qcow2 and fixes the guest for KVM: virtio disk and NIC, bootloader, VMware tools hooks. |
| `virtio-win` | nixpkgs, linked by `virt-v2v` at `share/virtio-win` | Windows virtio drivers. `virt-v2v` already points at this tree, so the Fedora ISO is not a separate download. |
| `qemu-img` | nixpkgs `qemu-utils`, in the Linux shell | Changes the container format only. Use it for a Proxmox disk that is already qcow2 or raw, or for a flat VMDK whose guest already boots on virtio. |
| `nbdkit` | wrapped inside the nixpkgs `virt-v2v` | Serves the source disk to `virt-v2v` during conversion. |
| VMware VDDK | Broadcom download, installed by the operator | Lets `virt-v2v -i vddk` read disks straight from vCenter. Proprietary. It stays off this repository and off the GitHub release. The `nbdkit` in nixpkgs is built without the VDDK plugin, so this shell converts exported disks. |
| `terraform-provider-kcore` | its own repository | Applies `guests.tf`. |
| `kctl` | the kcore release | Uploads the raw or qcow2 image and creates the VM once the cluster exists. |

Export the guest from vCenter as an OVA or a flat VMDK, or copy the Proxmox disk, then run the conversion on that file. A direct vCenter pull waits on an operator-installed VDDK and an `nbdkit` built with the VDDK plugin.

## Develop

```bash
nix develop
make test
make build
./bin/kcore-migrate version
```

The shell provides Go, git, GNU make, `gh`, and `sha256sum`. On Linux it also provides `virt-v2v` and `qemu-img`.

## GitHub release

`VERSION` is the release version. `make release` tags `v$(VERSION)`, builds archives, and uploads them. Run it from `nix develop` with `gh` authenticated, or with `GH_TOKEN` in the environment or in a gitignored `.env`.

```bash
make release
```

Archives in `dist/`:

- `kcore-migrate-$(VERSION)-linux-amd64.tar.gz`
- `kcore-migrate-$(VERSION)-linux-arm64.tar.gz`
- `kcore-migrate-$(VERSION)-darwin-amd64.tar.gz`
- `kcore-migrate-$(VERSION)-darwin-arm64.tar.gz`
- `SHA256SUMS`

`make release-publish` uploads an existing `dist/` to the tag that already points at `HEAD`. The working tree must be clean. The tag is not moved if it already points at another commit.

## License

Apache-2.0. See [LICENSE](LICENSE).
