<p align="center">
  <img src="https://github.com/kcorehypervisor/kcore/raw/main/assets/kcore-logo.png" alt="kcore" width="180">
</p>

# kcore-migrate

Tools for leaving VMware or Proxmox onto [kcore](https://kcorehypervisor.com/). This repository is separate from the hypervisor and from `kctl`.

The operator guide is also on the website: [Import from VMware or Proxmox](https://kcorehypervisor.com/docs/user/import.html).

`kctl` administers a cluster that already exists, and `kctl migrate` moves a VM between kcore nodes. An exit assessment runs on a jump host that can see the source hypervisor, often before any kcore node is installed.

## Install

On a Linux jump host, `nix develop` in this repository supplies Go and the converters (`virt-v2v`, `qemu-img`):

```bash
git clone git@github.com:kcorehypervisor/kcore-migrate.git
cd kcore-migrate
nix develop
make build
./bin/kcore-migrate version
```

Release archives are on [GitHub Releases](https://github.com/kcorehypervisor/kcore-migrate/releases) for Linux and macOS. Disk conversion still runs on Linux, because `virt-v2v` is Linux-only. Darwin archives build the inventory command.

## Inventory

`kcore-migrate inventory --source` selects an importer. VMware and Proxmox both produce the same guest record. The Terraform writer does not know which hypervisor it came from.

Inventory does not copy disk bytes. The password is read from the environment, not from a flag, so it does not land in shell history. The default variables are `VMWARE_PASSWORD` and `PROXMOX_PASSWORD`. `--password-env` names a different variable.

| `--source` | What it reads |
|------------|----------------|
| `vmware` | Virtual machines from vCenter through [govmomi](https://github.com/vmware/govmomi). Disks stay on the datastore. |
| `proxmox` | QEMU and LXC guests from the Proxmox API. Disks stay on the node. A host without a port is read on port 8006. |

```bash
export VMWARE_PASSWORD=...
kcore-migrate inventory \
  --source vmware \
  --endpoint https://vcenter.example \
  --username migrate@vsphere.local \
  --out ./migrate-out
```

```bash
export PROXMOX_PASSWORD=...
kcore-migrate inventory \
  --source proxmox \
  --endpoint https://pve.example:8006 \
  --username root@pam \
  --out ./migrate-out
```

`--insecure` skips TLS verification.

The output directory contains:

- `guests.tf`, one `kcore_vm` per guest (name, CPU, memory, first disk size, NICs)
- `gaps.md`, for anything that record cannot express cleanly (extra disks, templates, LXC, VLAN tags, raw device mappings, distributed port groups)

[`terraform-provider-kcore`](https://github.com/kcorehypervisor/terraform-provider-kcore) applies the HCL. The provider registers `kcore_vm` only. Generated NICs name networks that must already exist. `storage_backend` is `filesystem` until the provider accepts Ceph. `image_path` stays empty until `convert` produces a qcow2 or raw file. `kctl` accepts only those two formats.

## Convert

`convert` does not read VMDK itself. It shells out to `virt-v2v` or `qemu-img`.

| Input | Tool | Result |
|-------|------|--------|
| `.ova`, `.ovf`, `.vmx`, `.vmdk` | `virt-v2v -o disk` | Guest fixes for KVM, then qcow2 or raw disks in `--out` (`name-sda`, …) |
| `.qcow2`, `.raw`, `.img` | `qemu-img convert` | Container format only. Typical Proxmox disk, or a flat VMDK whose guest already boots on virtio. |

```bash
kcore-migrate convert --input guest.ova --out ./converted
kcore-migrate convert --input vm-100-disk-0.qcow2 --format raw --out ./converted/vm-100.raw
```

`--format` is `qcow2` (the default) or `raw`. `--tool virt-v2v` or `--tool qemu-img` overrides the choice. An OVA, OVF, or VMX cannot use `qemu-img`.

`virt-v2v` from nixpkgs already links the Fedora `virtio-win` driver tree, so Windows virtio drivers are not a separate download. Set `image_path` to the finished file and `image_format` to `qcow2` or `raw` on the generated `kcore_vm`.

Export the guest from vCenter as an OVA or a flat VMDK, or copy the Proxmox disk, then convert that file. A direct vCenter pull needs Broadcom’s proprietary VDDK and an `nbdkit` built with the VDDK plugin. The `nbdkit` in nixpkgs does not include that plugin, and this repository does not ship VDDK.

## Apply

Create the kcore networks named in `guests.tf` before apply. Point `image_path` and `image_format` at the converted disk. Apply with the Terraform provider. Upload the same raw or qcow2 image with `kctl` when the cluster is ready to receive it.

## Conversion host

Disk conversion runs on Linux. The Go toolchain in `nix develop` still builds `kcore-migrate` on Darwin.

| Piece | Where it comes from | What it does |
|-------|---------------------|--------------|
| `kcore-migrate` | this repository | Reads inventory and calls `virt-v2v` or `qemu-img`. |
| `virt-v2v` | nixpkgs, Linux shell | Converts an exported VMware guest and fixes it for KVM. |
| `virtio-win` | nixpkgs, linked by `virt-v2v` | Windows virtio drivers. |
| `qemu-img` | nixpkgs `qemu-utils`, Linux shell | Changes the container format only. |
| `nbdkit` | wrapped inside nixpkgs `virt-v2v` | Serves the source disk during conversion. |
| VMware VDDK | Broadcom download, installed by the operator | Direct vCenter disk read. Proprietary. Not in this repository or the GitHub release. |
| `terraform-provider-kcore` | its own repository | Applies `guests.tf`. |
| `kctl` | the kcore release | Uploads the raw or qcow2 image. |

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
