# kcore-migrate

Tools for leaving VMware or Proxmox onto [kcore](https://kcorehypervisor.com/). This repository is separate from the hypervisor and from `kctl`.

`kctl` administers a cluster that already exists. An exit assessment runs on a jump host that can see the source hypervisor, often before any kcore node is installed.

## Import sources

`kcore-migrate inventory --source` selects an importer. VMware and Proxmox both produce the same guest record. The Terraform writer does not know which hypervisor it came from.

| `--source` | Status |
|------------|--------|
| `vmware` | Connection settings are checked. The vSphere client is not connected yet. |
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

[`terraform-provider-kcore`](https://github.com/kcorehypervisor/terraform-provider-kcore) applies the HCL. `convert` will shell out to `virt-v2v` or `qemu-img` and is not implemented yet.

The provider today registers `kcore_vm` only. Generated NICs name networks that must already exist. `storage_backend` is `filesystem` until the provider accepts Ceph.

## Develop

```bash
nix develop
make test
make build
./bin/kcore-migrate version
```

The shell provides Go, git, GNU make, `gh`, and `sha256sum`.

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
