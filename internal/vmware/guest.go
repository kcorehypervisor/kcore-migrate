package vmware

import (
	"fmt"
	"slices"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

type controller struct {
	name   string
	number int32
}

func guestFromMO(vm mo.VirtualMachine) inventory.Guest {
	guest := inventory.Guest{Name: vm.Name}
	if vm.Config == nil {
		guest.Findings = append(guest.Findings, "vSphere returned no hardware configuration")
		return guest
	}
	if vm.Config.Name != "" {
		guest.Name = vm.Config.Name
	}
	guest.CPU = int(vm.Config.Hardware.NumCPU)
	guest.MemoryBytes = int64(vm.Config.Hardware.MemoryMB) * 1024 * 1024
	guest.GuestOS = vm.Config.GuestFullName
	if guest.GuestOS == "" {
		guest.GuestOS = vm.Config.GuestId
	}
	if vm.Guest != nil && vm.Guest.GuestFullName != "" {
		guest.GuestOS = vm.Guest.GuestFullName
	}
	if vm.Config.Template {
		guest.Findings = append(guest.Findings, "vSphere template; convert a clone, not the template")
	}

	devices := vm.Config.Hardware.Device
	buses := controllerMap(devices)
	var disks []rankedDisk
	for _, dev := range devices {
		switch disk := dev.(type) {
		case *types.VirtualDisk:
			info := buses[disk.ControllerKey]
			unit := int32(0)
			if disk.UnitNumber != nil {
				unit = *disk.UnitNumber
			}
			disks = append(disks, rankedDisk{
				bus:  info.number,
				unit: unit,
				disk: inventory.Disk{
					Name:      diskFile(disk.Backing),
					SizeBytes: diskBytes(disk),
					Bus:       busName(info.name),
				},
			})
			if _, raw := disk.Backing.(*types.VirtualDiskRawDiskMappingVer1BackingInfo); raw {
				guest.Findings = append(guest.Findings, "raw device mapping "+diskFile(disk.Backing))
			}
		default:
			nic, kind, ok := nicFrom(dev)
			if !ok {
				continue
			}
			guest.NICs = append(guest.NICs, nic)
			if kind != "network" {
				guest.Findings = append(guest.Findings, fmt.Sprintf("NIC %s uses %s backing %q; set nic.network to a kcore network name", nic.MAC, kind, nic.Network))
			}
		}
	}
	slices.SortStableFunc(disks, func(a, b rankedDisk) int {
		if a.bus != b.bus {
			return int(a.bus - b.bus)
		}
		return int(a.unit - b.unit)
	})
	for _, disk := range disks {
		guest.Disks = append(guest.Disks, disk.disk)
	}
	return guest
}

type rankedDisk struct {
	bus  int32
	unit int32
	disk inventory.Disk
}

func controllerMap(devices []types.BaseVirtualDevice) map[int32]controller {
	out := map[int32]controller{}
	for _, dev := range devices {
		name, ok := controllerName(dev)
		if !ok {
			continue
		}
		base := dev.GetVirtualDevice()
		out[base.Key] = controller{name: name, number: controllerBusNumber(dev)}
	}
	return out
}

func controllerName(dev types.BaseVirtualDevice) (string, bool) {
	switch dev.(type) {
	case *types.ParaVirtualSCSIController:
		return "pvscsi", true
	case *types.VirtualLsiLogicSASController, *types.VirtualLsiLogicController, *types.VirtualBusLogicController:
		return "scsi", true
	case *types.VirtualAHCIController:
		return "sata", true
	case *types.VirtualNVMEController:
		return "nvme", true
	case *types.VirtualIDEController:
		return "ide", true
	default:
		return "", false
	}
}

func controllerBusNumber(dev types.BaseVirtualDevice) int32 {
	switch c := dev.(type) {
	case *types.ParaVirtualSCSIController:
		return c.BusNumber
	case *types.VirtualLsiLogicSASController:
		return c.BusNumber
	case *types.VirtualLsiLogicController:
		return c.BusNumber
	case *types.VirtualBusLogicController:
		return c.BusNumber
	case *types.VirtualAHCIController:
		return c.BusNumber
	case *types.VirtualNVMEController:
		return c.BusNumber
	case *types.VirtualIDEController:
		return c.BusNumber
	default:
		return 0
	}
}

func busName(name string) string {
	if name == "" {
		return "unknown"
	}
	return name
}

func diskBytes(disk *types.VirtualDisk) int64 {
	if disk.CapacityInBytes > 0 {
		return disk.CapacityInBytes
	}
	return disk.CapacityInKB * 1024
}

func diskFile(backing types.BaseVirtualDeviceBackingInfo) string {
	file, ok := backing.(types.BaseVirtualDeviceFileBackingInfo)
	if !ok || file == nil {
		return ""
	}
	return file.GetVirtualDeviceFileBackingInfo().FileName
}

func nicFrom(dev types.BaseVirtualDevice) (inventory.NIC, string, bool) {
	card, ok := dev.(types.BaseVirtualEthernetCard)
	if !ok {
		return inventory.NIC{}, "", false
	}
	eth := card.GetVirtualEthernetCard()
	network, kind := networkOf(eth.Backing)
	return inventory.NIC{
		Network: network,
		Model:   nicModel(dev),
		MAC:     eth.MacAddress,
	}, kind, true
}

func nicModel(dev types.BaseVirtualDevice) string {
	switch dev.(type) {
	case *types.VirtualVmxnet3:
		return "vmxnet3"
	case *types.VirtualVmxnet2:
		return "vmxnet2"
	case *types.VirtualVmxnet:
		return "vmxnet"
	case *types.VirtualE1000e:
		return "e1000e"
	case *types.VirtualE1000:
		return "e1000"
	case *types.VirtualPCNet32:
		return "pcnet32"
	case *types.VirtualSriovEthernetCard:
		return "sriov"
	default:
		return "ethernet"
	}
}

func networkOf(backing types.BaseVirtualDeviceBackingInfo) (string, string) {
	switch n := backing.(type) {
	case *types.VirtualEthernetCardNetworkBackingInfo:
		return n.DeviceName, "network"
	case *types.VirtualEthernetCardLegacyNetworkBackingInfo:
		return n.DeviceName, "network"
	case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
		if n.Port.PortgroupKey != "" {
			return n.Port.PortgroupKey, "distributed port group"
		}
		return n.Port.PortKey, "distributed port"
	case *types.VirtualEthernetCardOpaqueNetworkBackingInfo:
		return n.OpaqueNetworkId, "opaque network"
	default:
		return "", "unknown"
	}
}
