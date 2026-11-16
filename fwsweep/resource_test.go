package fwsweep_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jianyuan/terraform-plugin-framework-utils/fwsweep"
)

type testResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Project types.String `tfsdk:"project"`
	Name    types.String `tfsdk:"name"`
}

type testResource struct {
	providerData any
	deleted      *testResourceModel
	deleteError  string
}

func (r *testResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "test_resource"
}

func (r *testResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true},
			"project": schema.StringAttribute{Required: true},
			"name":    schema.StringAttribute{Optional: true},
		},
	}
}

func (r *testResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	r.providerData = req.ProviderData
}

func (r *testResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {}
func (r *testResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse)       {}
func (r *testResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}

func (r *testResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if r.deleteError != "" {
		resp.Diagnostics.AddError("Delete failed", r.deleteError)
		return
	}

	var data testResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	r.deleted = &data
}

func TestNewResource(t *testing.T) {
	res := &testResource{}
	err := fwsweep.NewResource(func() resource.Resource { return res }, "provider-data", map[string]any{
		"id":      "123",
		"project": "my-project",
	}).Delete(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.providerData != "provider-data" {
		t.Errorf("expected provider data to be configured, got %v", res.providerData)
	}
	if res.deleted == nil {
		t.Fatal("expected Delete to be called")
	}
	if got := res.deleted.ID.ValueString(); got != "123" {
		t.Errorf("expected id 123, got %q", got)
	}
	if got := res.deleted.Project.ValueString(); got != "my-project" {
		t.Errorf("expected project my-project, got %q", got)
	}
	if !res.deleted.Name.IsNull() {
		t.Errorf("expected name to be null, got %v", res.deleted.Name)
	}
}

func TestNewResource_UnknownAttribute(t *testing.T) {
	res := &testResource{}
	err := fwsweep.NewResource(func() resource.Resource { return res }, nil, map[string]any{
		"missing": "value",
	}).Delete(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
	if res.deleted != nil {
		t.Error("expected Delete not to be called")
	}
}

func TestNewResource_DeleteError(t *testing.T) {
	res := &testResource{deleteError: "boom"}
	err := fwsweep.NewResource(func() resource.Resource { return res }, nil, map[string]any{
		"id": "123",
	}).Delete(t.Context())
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected delete error, got %v", err)
	}
}
