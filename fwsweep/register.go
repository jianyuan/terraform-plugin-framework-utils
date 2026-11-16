package fwsweep

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Register adds a test sweeper named name. When the sweeper runs, it calls
// providerData to obtain the provider data, collects the items with fn and
// deletes them with Sweep.
func Register[P any](name string, providerData func() P, fn SweeperFn[P], opts ...Option) {
	o := newOptions(opts)

	resource.AddTestSweepers(name, &resource.Sweeper{
		Name: name,
		F: func(region string) error {
			ctx := context.Background()

			sweepables, err := fn(ctx, providerData())
			if err != nil {
				return fmt.Errorf("failed to collect %q: %w", name, err)
			}

			err = Sweep(ctx, sweepables, opts...)
			if err != nil {
				return fmt.Errorf("failed to sweep %q: %w", name, err)
			}

			return nil
		},
		Dependencies: o.dependencies,
	})
}
