package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

type apiReader struct {
	cfg inventory.Config
}

func (r apiReader) Guests(ctx context.Context) ([]inventory.Guest, error) {
	client, err := newClient(r.cfg)
	if err != nil {
		return nil, err
	}
	if err := client.login(ctx); err != nil {
		return nil, err
	}
	resources, err := client.resources(ctx)
	if err != nil {
		return nil, err
	}
	guests := make([]inventory.Guest, 0, len(resources))
	for _, res := range resources {
		guests = append(guests, client.oneGuest(ctx, res))
	}
	sortGuests(guests)
	return guests, nil
}

type client struct {
	base   url.URL
	http   *http.Client
	user   string
	pass   string
	ticket string
}

func newClient(cfg inventory.Config) (*client, error) {
	base, err := apiBase(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &client{
		base: base,
		user: cfg.Username,
		pass: cfg.Password,
		http: &http.Client{Timeout: 60 * time.Second, Transport: transport},
	}, nil
}

func apiBase(endpoint string) (url.URL, error) {
	raw := strings.TrimSpace(endpoint)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return url.URL{}, fmt.Errorf("proxmox: --endpoint must be a host or URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return url.URL{}, fmt.Errorf("proxmox: endpoint scheme must be http or https")
	}
	if parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), "8006")
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return *parsed, nil
}

type resource struct {
	VMID     int    `json:"vmid"`
	Name     string `json:"name"`
	Node     string `json:"node"`
	Type     string `json:"type"`
	MaxCPU   int    `json:"maxcpu"`
	MaxMem   int64  `json:"maxmem"`
	Template int    `json:"template"`
}

func (c *client) login(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", c.user)
	form.Set("password", c.pass)
	var ticket struct {
		Ticket string `json:"ticket"`
	}
	if err := c.call(ctx, http.MethodPost, "/api2/json/access/ticket", form, nil, &ticket); err != nil {
		return err
	}
	if ticket.Ticket == "" {
		return fmt.Errorf("proxmox: login returned an empty ticket")
	}
	c.ticket = ticket.Ticket
	return nil
}

func (c *client) resources(ctx context.Context) ([]resource, error) {
	query := url.Values{}
	query.Set("type", "vm")
	var resources []resource
	if err := c.call(ctx, http.MethodGet, "/api2/json/cluster/resources", nil, query, &resources); err != nil {
		return nil, err
	}
	return resources, nil
}

func (c *client) oneGuest(ctx context.Context, res resource) inventory.Guest {
	if res.Node == "" || res.VMID == 0 || (res.Type != "qemu" && res.Type != "lxc") {
		guest := guestFromResource(res)
		guest.Findings = append(guest.Findings, "cluster resource is missing a node, vmid, or guest type")
		return guest
	}
	kind := "qemu"
	if res.Type == "lxc" {
		kind = "lxc"
	}
	path := fmt.Sprintf("/api2/json/nodes/%s/%s/%d/config", url.PathEscape(res.Node), kind, res.VMID)
	var cfg map[string]json.RawMessage
	if err := c.call(ctx, http.MethodGet, path, nil, nil, &cfg); err != nil {
		guest := guestFromResource(res)
		guest.Findings = append(guest.Findings, "config read failed: "+err.Error())
		return guest
	}
	return guestFromConfig(res, cfg)
}

func (c *client) call(ctx context.Context, method, path string, form url.Values, query url.Values, dest any) error {
	endpoint := c.base
	endpoint.Path = path
	endpoint.RawQuery = query.Encode()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c.ticket != "" {
		req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: c.ticket})
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("proxmox: %s: %w", path, err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("proxmox: %s: %w", path, err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("proxmox: %s: %s", path, strings.TrimSpace(string(payload)))
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("proxmox: %s: %w", path, err)
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, dest); err != nil {
		return fmt.Errorf("proxmox: %s: %w", path, err)
	}
	return nil
}
