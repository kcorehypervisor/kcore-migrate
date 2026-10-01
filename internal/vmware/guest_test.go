package vmware

import (
	"strings"
	"testing"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestGuestFromMO(t *testing.T) {
	unit0 := int32(0)
	unit1 := int32(1)
	scsi := &types.ParaVirtualSCSIController{}
	scsi.Key = 1000
	scsi.BusNumber = 0
	data := &types.VirtualDisk{}
	data.Key = 2001
	data.ControllerKey = 1000
	data.UnitNumber = &unit1
	data.CapacityInBytes = 20 << 30
	data.Backing = &types.VirtualDiskFlatVer2BackingInfo{
		VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{FileName: "[ds] web/data.vmdk"},
	}
	root := &types.VirtualDisk{}
	root.Key = 2000
	root.ControllerKey = 1000
	root.UnitNumber = &unit0
	root.CapacityInKB = 10 << 20
	root.Backing = &types.VirtualDiskFlatVer2BackingInfo{
		VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{FileName: "[ds] web/root.vmdk"},
	}
	nic := &types.VirtualVmxnet3{}
	nic.MacAddress = "00:11:22:33:44:55"
	nic.Backing = &types.VirtualEthernetCardNetworkBackingInfo{
		VirtualDeviceDeviceBackingInfo: types.VirtualDeviceDeviceBackingInfo{DeviceName: "VM Network"},
	}
	vm := mo.VirtualMachine{
		ManagedEntity: mo.ManagedEntity{Name: "web"},
		Config: &types.VirtualMachineConfigInfo{
			Name:          "web",
			GuestFullName: "Ubuntu Linux (64-bit)",
			Hardware: types.VirtualHardware{
				NumCPU:   4,
				MemoryMB: 2048,
				Device:   []types.BaseVirtualDevice{data, nic, scsi, root},
			},
		},
		Guest: &types.GuestInfo{GuestFullName: "Ubuntu 24.04"},
	}

	got := guestFromMO(vm)
	if got.Name != "web" || got.CPU != 4 || got.MemoryBytes != 2048*1024*1024 || got.GuestOS != "Ubuntu 24.04" {
		t.Fatalf("%+v", got)
	}
	if len(got.Disks) != 2 || got.Disks[0].Name != "[ds] web/root.vmdk" || got.Disks[0].Bus != "pvscsi" || got.Disks[0].SizeBytes != 10<<30 {
		t.Fatalf("disks %+v", got.Disks)
	}
	if got.Disks[1].SizeBytes != 20<<30 {
		t.Fatalf("data disk %+v", got.Disks[1])
	}
	if len(got.NICs) != 1 || got.NICs[0].Network != "VM Network" || got.NICs[0].Model != "vmxnet3" || got.NICs[0].MAC != "00:11:22:33:44:55" {
		t.Fatalf("nic %+v", got.NICs)
	}
	if len(got.Findings) != 0 {
		t.Fatalf("findings %v", got.Findings)
	}
}

func TestGuestFromMOFindings(t *testing.T) {
	if got := guestFromMO(mo.VirtualMachine{ManagedEntity: mo.ManagedEntity{Name: "empty"}}); len(got.Findings) != 1 {
		t.Fatalf("%+v", got)
	}
	disk := &types.VirtualDisk{}
	disk.CapacityInBytes = 1
	disk.Backing = &types.VirtualDiskRawDiskMappingVer1BackingInfo{
		VirtualDeviceFileBackingInfo: types.VirtualDeviceFileBackingInfo{FileName: "[ds] raw.vmdk"},
	}
	nic := &types.VirtualE1000e{}
	nic.MacAddress = "00:50:56:aa:bb:cc"
	nic.Backing = &types.VirtualEthernetCardDistributedVirtualPortBackingInfo{
		Port: types.DistributedVirtualSwitchPortConnection{PortgroupKey: "dvportgroup-1"},
	}
	vm := mo.VirtualMachine{
		Config: &types.VirtualMachineConfigInfo{
			Name:     "db",
			Template: true,
			Hardware: types.VirtualHardware{Device: []types.BaseVirtualDevice{disk, nic}},
		},
	}
	got := guestFromMO(vm)
	text := strings.Join(got.Findings, "\n")
	for _, want := range []string{"template", "raw device mapping", "distributed port group"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if got.NICs[0].Model != "e1000e" || got.NICs[0].Network != "dvportgroup-1" {
		t.Fatalf("%+v", got.NICs)
	}
}
