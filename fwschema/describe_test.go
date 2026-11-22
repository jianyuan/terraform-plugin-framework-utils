package fwschema_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/jianyuan/terraform-plugin-framework-utils/fwschema"
)

func ExampleDescribe() {
	ctx := context.Background()
	s := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				MarkdownDescription: "Region to deploy to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.OneOf("us", "eu")},
			},
		},
	}, fwschema.ForceNew(), fwschema.Enum())

	fmt.Println(s.Attributes["region"].GetMarkdownDescription())
	// Output:
	// <span style="color:red">(Force new)</span> Region to deploy to. Valid values: `us`, `eu`.
}

func ExampleForceNewLabel() {
	ctx := context.Background()
	s := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description:   "Name of the bucket.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}, fwschema.ForceNewLabel("**Forces replacement.**", "Forces replacement."))

	fmt.Println(s.Attributes["name"].GetMarkdownDescription())
	fmt.Println(s.Attributes["name"].GetDescription())
	// Output:
	// **Forces replacement.** Name of the bucket.
	// Forces replacement. Name of the bucket.
}

func ExampleEnumLabel() {
	ctx := context.Background()
	s := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"tier": schema.StringAttribute{
				Description: "Pricing tier.",
				Validators:  []validator.String{stringvalidator.OneOf("free", "pro")},
			},
		},
	}, fwschema.EnumLabel("**Allowed:**", "Allowed:"))

	fmt.Println(s.Attributes["tier"].GetMarkdownDescription())
	fmt.Println(s.Attributes["tier"].GetDescription())
	// Output:
	// Pricing tier. **Allowed:** `free`, `pro`.
	// Pricing tier. Allowed: `free`, `pro`.
}

func TestDescribe(t *testing.T) {
	ctx := context.Background()
	in := schema.Schema{
		MarkdownDescription: "Root.",
		Attributes: map[string]schema.Attribute{
			"plain": schema.StringAttribute{Description: "Plain."},
			"configured": schema.Int64Attribute{
				Description:   "Size.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplaceIfConfigured()},
				Validators:    []validator.Int64{int64validator.OneOf(1, 2)},
			},
			"nested": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"kind": schema.StringAttribute{
							Validators: []validator.String{stringvalidator.OneOfCaseInsensitive(`a"b`, "c")},
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"single": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "ID.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
				},
			},
		},
	}
	out := fwschema.Describe(ctx, in, fwschema.ForceNew(), fwschema.Enum())

	configured := out.Attributes["configured"]
	nested := out.Attributes["nested"].(schema.ListNestedAttribute).NestedObject.Attributes["kind"]
	single := out.Blocks["single"].(schema.SingleNestedBlock).Attributes["id"]

	for _, tc := range []struct{ name, got, want string }{
		{"root untouched", out.MarkdownDescription, "Root."},
		{"plain markdown", out.Attributes["plain"].GetMarkdownDescription(), "Plain."},
		{"configured markdown", configured.GetMarkdownDescription(), "<span style=\"color:red\">(Force new)</span> Size. Valid values: `1`, `2`."},
		{"configured plain", configured.GetDescription(), "(Force new) Size. Valid values: `1`, `2`."},
		{"nested", nested.GetMarkdownDescription(), "Valid values: `a\"b`, `c`."},
		{"nested plain stays empty", nested.GetDescription(), ""},
		{"block", single.GetMarkdownDescription(), "<span style=\"color:red\">(Force new)</span> ID."},
		{"input not mutated", in.Attributes["configured"].GetMarkdownDescription(), ""},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
