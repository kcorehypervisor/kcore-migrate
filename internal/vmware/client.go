package vmware

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"

	"github.com/kcorehypervisor/kcore-migrate/internal/inventory"
)

type apiReader struct {
	cfg inventory.Config
}

func (r apiReader) Guests(ctx context.Context) ([]inventory.Guest, error) {
	client, err := dial(ctx, r.cfg)
	if err != nil {
		return nil, err
	}
	defer client.Logout(context.WithoutCancel(ctx))
	return listGuests(ctx, client)
}

func dial(ctx context.Context, cfg inventory.Config) (*govmomi.Client, error) {
	endpoint, err := soap.ParseURL(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("vmware: endpoint: %w", err)
	}
	if endpoint == nil {
		return nil, fmt.Errorf("vmware: --endpoint is required")
	}
	endpoint.User = url.UserPassword(cfg.Username, cfg.Password)
	client, err := govmomi.NewClient(ctx, endpoint, cfg.Insecure)
	if err != nil {
		return nil, fmt.Errorf("vmware: connect: %w", err)
	}
	return client, nil
}

func listGuests(ctx context.Context, client *govmomi.Client) ([]inventory.Guest, error) {
	manager := view.NewManager(client.Client)
	container, err := manager.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, fmt.Errorf("vmware: list virtual machines: %w", err)
	}
	defer container.Destroy(context.WithoutCancel(ctx))

	var vms []mo.VirtualMachine
	err = container.Retrieve(ctx, []string{"VirtualMachine"}, []string{"name", "config", "guest"}, &vms)
	if err != nil {
		return nil, fmt.Errorf("vmware: read virtual machine config: %w", err)
	}
	guests := make([]inventory.Guest, 0, len(vms))
	for _, vm := range vms {
		guests = append(guests, guestFromMO(vm))
	}
	return guests, nil
}
