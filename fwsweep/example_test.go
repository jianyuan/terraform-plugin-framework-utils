package fwsweep_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/jianyuan/terraform-plugin-framework-utils/fwsweep"
)

type apiClient struct{}

type project struct{ ID, Name string }

func (c *apiClient) ListProjects(context.Context) ([]project, error) {
	return []project{{ID: "1", Name: "tf-acc-test-a"}}, nil
}

func newProjectResource() resource.Resource { return &testResource{} }

// Register a sweeper in a _test.go file of the provider package, and call
// resource.TestMain from TestMain so that "go test -sweep=all" runs it.
func ExampleRegister() {
	fwsweep.Register("example_project", func() *apiClient {
		// Build the same client the provider's Configure method passes as
		// provider data, usually from environment variables.
		return &apiClient{}
	}, func(ctx context.Context, client *apiClient) ([]fwsweep.Sweepable, error) {
		projects, err := client.ListProjects(ctx)
		if err != nil {
			return nil, err
		}

		var sweepables []fwsweep.Sweepable
		for _, p := range projects {
			sweepables = append(sweepables, fwsweep.NewResource(newProjectResource, client, map[string]any{
				"id": p.ID,
			}))
		}
		return sweepables, nil
	}, fwsweep.WithDependencies("example_project_member"))
}

func ExampleSweep() {
	var deleted atomic.Int32
	sweepables := []fwsweep.Sweepable{
		sweepableFunc(func(context.Context) error { deleted.Add(1); return nil }),
		sweepableFunc(func(context.Context) error { return fmt.Errorf("project 2: not found") }),
		sweepableFunc(func(context.Context) error { deleted.Add(1); return nil }),
	}

	err := fwsweep.Sweep(context.Background(), sweepables, fwsweep.WithConcurrency(2))
	fmt.Println(deleted.Load(), err)
	// Output:
	// 2 project 2: not found
}
