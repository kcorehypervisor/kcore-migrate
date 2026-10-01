package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

func TestWriteFromEitherSource(t *testing.T) {
	for _, src := range []string{inventory.SourceVMware, inventory.SourceProxmox} {
		t.Run(src, func(t *testing.T) {
			dir := t.TempDir()
			err := Write(dir, inventory.Snapshot{
				Source: src,
				Guests: []inventory.Guest{{
					Name:        "Web App",
					CPU:         2,
					MemoryBytes: 2048,
					GuestOS:     "windows",
					Disks:       []inventory.Disk{{Name: "scsi0", SizeBytes: 100}, {Name: "scsi1", SizeBytes: 50}},
					NICs:        []inventory.NIC{{Network: "lan", Model: "virtio", MAC: "aa:bb:cc:dd:ee:ff"}},
					Findings:    []string{"snapshot present"},
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			hcl, err := os.ReadFile(filepath.Join(dir, "guests.tf"))
			if err != nil {
				t.Fatal(err)
			}
			text := string(hcl)
			for _, want := range []string{src, `resource "kcore_vm" "web_app"`, `name          = "Web App"`, `cpu           = 2`, `network = "lan"`} {
				if !strings.Contains(text, want) {
					t.Fatalf("guests.tf missing %q\n%s", want, text)
				}
			}
			gap, err := os.ReadFile(filepath.Join(dir, "gaps.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(gap), "snapshot present") || !strings.Contains(string(gap), "2 disks") {
				t.Fatalf("gaps.md: %s", gap)
			}
		})
	}
}

func TestIdent(t *testing.T) {
	if got := ident("01 First"); got != "vm_01_first" {
		t.Fatal(got)
	}
	seen := map[string]int{}
	if uniqueIdent("a", seen) != "a" || uniqueIdent("a", seen) != "a_2" {
		t.Fatal(seen)
	}
}
