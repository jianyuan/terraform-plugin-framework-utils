package fwsweep

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/jianyuan/terraform-plugin-framework-utils/fwdiag"
)

var _ Sweepable = (*sweepResource)(nil)

type sweepResource struct {
	factory      func() resource.Resource
	providerData any
	attributes   map[string]any
}

// NewResource returns a Sweepable that deletes a resource by calling its
// Delete method directly. The resource is configured with providerData and
// given a state built from its schema with the root attributes set.
//
// This only suits resources whose Delete reads nothing but root attributes
// from the state: private state and identity are not populated.
func NewResource(factory func() resource.Resource, providerData any, attributes map[string]any) Sweepable {
	return &sweepResource{
		factory:      factory,
		providerData: providerData,
		attributes:   attributes,
	}
}

func (sr *sweepResource) Delete(ctx context.Context) error {
	res := sr.factory()

	if res, ok := res.(resource.ResourceWithConfigure); ok {
		var configureResp resource.ConfigureResponse
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: sr.providerData}, &configureResp)
		if configureResp.Diagnostics.HasError() {
			return fwdiag.Error(configureResp.Diagnostics)
		}
	}

	var schemaResp resource.SchemaResponse
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		return fwdiag.Error(schemaResp.Diagnostics)
	}

	state := tfsdk.State{
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		Schema: schemaResp.Schema,
	}

	for k, v := range sr.attributes {
		if diags := state.SetAttribute(ctx, path.Root(k), v); diags.HasError() {
			return fwdiag.Error(diags)
		}
	}

	log.Printf("[INFO] Deleting resource: %v", sr.attributes)
	var deleteResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: state}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		return fwdiag.Error(deleteResp.Diagnostics)
	}

	return nil
}
