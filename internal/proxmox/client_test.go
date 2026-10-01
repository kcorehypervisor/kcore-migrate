package proxmox

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

func TestAPIReader(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.PostForm.Get("username") != "root@pam" || r.PostForm.Get("password") != "secret" {
				http.Error(w, "bad login", http.StatusUnauthorized)
				return
			}
			io.WriteString(w, `{"data":{"ticket":"PVE:ticket"}}`)
		case "/api2/json/cluster/resources":
			if cookie, err := r.Cookie("PVEAuthCookie"); err != nil || cookie.Value != "PVE:ticket" {
				http.Error(w, "no ticket", http.StatusUnauthorized)
				return
			}
			io.WriteString(w, `{"data":[{"vmid":100,"name":"web","node":"pve","type":"qemu","maxcpu":2,"maxmem":2147483648}]}`)
		case "/api2/json/nodes/pve/qemu/100/config":
			io.WriteString(w, `{"data":{"name":"web","cores":2,"memory":2048,"ostype":"l26","scsi0":"local-lvm:vm-100-disk-0,size=32G","net0":"virtio=BC:24:11:AA:BB:CC,bridge=vmbr0"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	src, err := Open(inventory.Config{Endpoint: srv.URL, Username: "root@pam", Password: "secret", Insecure: true})
	if err != nil {
		t.Fatal(err)
	}
	snap, err := src.Read(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != inventory.SourceProxmox || len(snap.Guests) != 1 || snap.Guests[0].Name != "web" || snap.Guests[0].Disks[0].SizeBytes != 32<<30 {
		t.Fatalf("%+v", snap)
	}
}

func TestAPIBaseAddsDefaultPort(t *testing.T) {
	base, err := apiBase("pve.example")
	if err != nil {
		t.Fatal(err)
	}
	if base.String() != "https://pve.example:8006" {
		t.Fatal(base.String())
	}
	parsed, err := url.Parse("https://pve.example:8006/api")
	if err != nil {
		t.Fatal(err)
	}
	base, err = apiBase(parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(base.Host, ":8006") || base.Path != "" {
		t.Fatal(base.String())
	}
}
