package proxmox

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGuestFromConfigQEMU(t *testing.T) {
	cfg := map[string]json.RawMessage{
		"name":     json.RawMessage(`"web"`),
		"cores":    json.RawMessage(`2`),
		"sockets":  json.RawMessage(`2`),
		"memory":   json.RawMessage(`2048`),
		"ostype":   json.RawMessage(`"l26"`),
		"boot":     json.RawMessage(`"order=scsi0;net0"`),
		"scsi1":    json.RawMessage(`"local-lvm:vm-100-disk-1,size=8G"`),
		"scsi0":    json.RawMessage(`"local-lvm:vm-100-disk-0,size=32G"`),
		"ide2":     json.RawMessage(`"none,media=cdrom"`),
		"net0":     json.RawMessage(`"virtio=BC:24:11:AA:BB:CC,bridge=vmbr0,tag=10"`),
		"efidisk0": json.RawMessage(`"local-lvm:vm-100-disk-2,size=1M"`),
	}
	got := guestFromConfig(resource{VMID: 100, Name: "other", Type: "qemu"}, cfg)
	if got.Name != "web" || got.CPU != 4 || got.MemoryBytes != 2048*1024*1024 || got.GuestOS != "Linux" {
		t.Fatalf("%+v", got)
	}
	if len(got.Disks) != 2 || got.Disks[0].Name != "local-lvm:vm-100-disk-0" || got.Disks[0].Bus != "scsi" || got.Disks[0].SizeBytes != 32<<30 {
		t.Fatalf("disks %+v", got.Disks)
	}
	if got.Disks[1].SizeBytes != 8<<30 {
		t.Fatalf("second disk %+v", got.Disks[1])
	}
	if len(got.NICs) != 1 || got.NICs[0].Network != "vmbr0" || got.NICs[0].Model != "virtio" || got.NICs[0].MAC != "BC:24:11:AA:BB:CC" {
		t.Fatalf("nic %+v", got.NICs)
	}
	text := strings.Join(got.Findings, "\n")
	for _, want := range []string{"efidisk0", "vlan tag 10"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
}

func TestGuestFromConfigLXCTemplate(t *testing.T) {
	cfg := map[string]json.RawMessage{
		"hostname": json.RawMessage(`"app"`),
		"memory":   json.RawMessage(`512`),
		"cores":    json.RawMessage(`1`),
		"rootfs":   json.RawMessage(`"local:100/vm-100-disk-0.raw,size=8G"`),
		"net0":     json.RawMessage(`"name=eth0,bridge=vmbr0,hwaddr=AA:BB:CC:DD:EE:FF,type=veth"`),
	}
	got := guestFromConfig(resource{VMID: 100, Type: "lxc", Template: 1}, cfg)
	if got.Name != "app" || got.Disks[0].Bus != "rootfs" || got.NICs[0].MAC != "AA:BB:CC:DD:EE:FF" || got.NICs[0].Network != "vmbr0" {
		t.Fatalf("%+v", got)
	}
	text := strings.Join(got.Findings, "\n")
	if !strings.Contains(text, "template") || !strings.Contains(text, "LXC") {
		t.Fatalf("%s", text)
	}
}
