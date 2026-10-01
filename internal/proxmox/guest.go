package proxmox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

var (
	diskKey  = regexp.MustCompile(`^(scsi|virtio|sata|ide|mp)(\d+)$`)
	bootDisk = regexp.MustCompile(`(?:scsi|virtio|sata|ide|mp)\d+`)
)

func guestFromResource(res resource) inventory.Guest {
	name := res.Name
	if name == "" {
		name = fmt.Sprintf("vm-%d", res.VMID)
	}
	guest := inventory.Guest{
		Name:        name,
		CPU:         res.MaxCPU,
		MemoryBytes: res.MaxMem,
	}
	noteKind(&guest, res)
	return guest
}

func guestFromConfig(res resource, cfg map[string]json.RawMessage) inventory.Guest {
	guest := guestFromResource(res)
	if name := jsonString(cfg["name"]); name != "" {
		guest.Name = name
	}
	if name := jsonString(cfg["hostname"]); name != "" && res.Type == "lxc" {
		guest.Name = name
	}
	cores := jsonInt(cfg["cores"])
	sockets := jsonInt(cfg["sockets"])
	if sockets == 0 {
		sockets = 1
	}
	if cores > 0 {
		guest.CPU = cores * sockets
	}
	if memory := jsonInt(cfg["memory"]); memory > 0 {
		guest.MemoryBytes = int64(memory) * 1024 * 1024
	}
	guest.GuestOS = osName(jsonString(cfg["ostype"]))

	boot := firstBootDisk(jsonString(cfg["boot"]))
	var disks []rankedDisk
	for key, raw := range cfg {
		value := jsonString(raw)
		if value == "" {
			continue
		}
		switch {
		case key == "efidisk0":
			guest.Findings = append(guest.Findings, "efidisk0 omitted from the disk list")
		case key == "tpmstate0":
			guest.Findings = append(guest.Findings, "tpmstate0 omitted from the disk list")
		case strings.HasPrefix(key, "unused"):
			guest.Findings = append(guest.Findings, "unused disk left on storage: "+key)
		case key == "rootfs" || diskKey.MatchString(key):
			disk, ok := parseDisk(key, value)
			if !ok {
				continue
			}
			disks = append(disks, rankedDisk{key: key, boot: key == boot, disk: disk})
		case strings.HasPrefix(key, "net"):
			nic := parseNIC(value)
			guest.NICs = append(guest.NICs, nic)
			if tag := option(value, "tag"); tag != "" {
				guest.Findings = append(guest.Findings, fmt.Sprintf("NIC %s vlan tag %s; the kcore network must already exist", nic.MAC, tag))
			}
		}
	}
	slices.SortStableFunc(disks, func(a, b rankedDisk) int {
		if a.boot != b.boot {
			if a.boot {
				return -1
			}
			return 1
		}
		if rank(a.disk.Bus) != rank(b.disk.Bus) {
			return rank(a.disk.Bus) - rank(b.disk.Bus)
		}
		return strings.Compare(a.key, b.key)
	})
	for _, disk := range disks {
		guest.Disks = append(guest.Disks, disk.disk)
	}
	slices.SortStableFunc(guest.NICs, func(a, b inventory.NIC) int {
		return strings.Compare(a.MAC, b.MAC)
	})
	return guest
}

type rankedDisk struct {
	key  string
	boot bool
	disk inventory.Disk
}

func noteKind(guest *inventory.Guest, res resource) {
	if res.Template != 0 {
		guest.Findings = append(guest.Findings, "Proxmox template; convert a clone")
	}
	if res.Type == "lxc" {
		guest.Findings = append(guest.Findings, "LXC container; kcore imports a QEMU guest")
	}
}

func osName(ostype string) string {
	names := map[string]string{
		"l26":     "Linux",
		"l24":     "Linux 2.4",
		"other":   "other",
		"wxp":     "Windows XP",
		"w2k":     "Windows 2000",
		"w2k3":    "Windows Server 2003",
		"w2k8":    "Windows Server 2008",
		"wvista":  "Windows Vista",
		"win7":    "Windows 7",
		"win8":    "Windows 8",
		"win10":   "Windows 10",
		"win11":   "Windows 11",
		"solaris": "Solaris",
	}
	if name, ok := names[ostype]; ok {
		return name
	}
	return ostype
}

func parseDisk(key, raw string) (inventory.Disk, bool) {
	if strings.Contains(raw, "media=cdrom") || strings.Contains(raw, "cloudinit") {
		return inventory.Disk{}, false
	}
	volume, opts := splitOptions(raw)
	if volume == "" || volume == "none" {
		return inventory.Disk{}, false
	}
	bus := "rootfs"
	if key != "rootfs" {
		bus = diskKey.FindStringSubmatch(key)[1]
	}
	return inventory.Disk{Name: volume, SizeBytes: parseSize(opts["size"]), Bus: bus}, true
}

func parseNIC(raw string) inventory.NIC {
	_, opts := splitOptions(raw)
	if mac := opts["hwaddr"]; mac != "" {
		model := opts["type"]
		if model == "" {
			model = "veth"
		}
		return inventory.NIC{Network: opts["bridge"], Model: model, MAC: mac}
	}
	parts := strings.Split(raw, ",")
	model, mac, _ := strings.Cut(parts[0], "=")
	return inventory.NIC{Network: opts["bridge"], Model: model, MAC: mac}
}

func splitOptions(raw string) (string, map[string]string) {
	opts := map[string]string{}
	volume := ""
	for i, part := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			if i == 0 {
				volume = part
			}
			continue
		}
		opts[key] = value
	}
	return volume, opts
}

func option(raw, key string) string {
	_, opts := splitOptions(raw)
	return opts[key]
}

func firstBootDisk(boot string) string {
	return bootDisk.FindString(boot)
}

func parseSize(raw string) int64 {
	raw = strings.TrimSpace(raw)
	i := 0
	for i < len(raw) && raw[i] >= '0' && raw[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0
	}
	n, err := strconv.ParseInt(raw[:i], 10, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(raw[i:])) {
	case "", "B":
		return n
	case "K":
		return n << 10
	case "M":
		return n << 20
	case "G":
		return n << 30
	case "T":
		return n << 40
	default:
		return 0
	}
}

func rank(bus string) int {
	switch bus {
	case "virtio":
		return 0
	case "scsi":
		return 1
	case "sata":
		return 2
	case "ide":
		return 3
	case "rootfs":
		return 4
	case "mp":
		return 5
	default:
		return 9
	}
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var number json.Number
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&number); err == nil {
		return number.String()
	}
	return ""
}

func jsonInt(raw json.RawMessage) int {
	n, _ := strconv.Atoi(jsonString(raw))
	return n
}

func sortGuests(guests []inventory.Guest) {
	slices.SortStableFunc(guests, func(a, b inventory.Guest) int {
		return strings.Compare(a.Name, b.Name)
	})
}
