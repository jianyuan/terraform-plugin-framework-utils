package fwschema_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	for _, desc := range []string{"Size", "Size..", "Size. \n"} {
		in.Attributes[desc] = schema.StringAttribute{
			MarkdownDescription: desc,
			Validators:          []validator.String{stringvalidator.OneOf("a")},
		}
	}
	in.Attributes["question"] = schema.StringAttribute{
		MarkdownDescription: "Which one?",
		Validators:          []validator.String{stringvalidator.OneOf("a")},
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
		{"adds period", out.Attributes["Size"].GetMarkdownDescription(), "Size. Valid values: `a`."},
		{"trims periods", out.Attributes["Size.."].GetMarkdownDescription(), "Size. Valid values: `a`."},
		{"trims whitespace", out.Attributes["Size. \n"].GetMarkdownDescription(), "Size. Valid values: `a`."},
		{"keeps question mark", out.Attributes["question"].GetMarkdownDescription(), "Which one? Valid values: `a`."},
		{"input not mutated", in.Attributes["configured"].GetMarkdownDescription(), ""},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func ExampleSizeAtLeast() {
	ctx := context.Background()
	s := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags to apply",
				Optional:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("owner")),
				},
			},
		},
	}, fwschema.SizeAtLeast(), fwschema.AlsoRequires())

	fmt.Println(s.Attributes["tags"].GetMarkdownDescription())
	// Output:
	// Tags to apply. Must contain at least 1 element. Also requires: `owner`.
}

func TestSizeAtLeastAndAlsoRequires(t *testing.T) {
	ctx := context.Background()
	out := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"set": schema.SetAttribute{
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(2)},
			},
			"map": schema.MapAttribute{
				ElementType: types.StringType,
				Validators:  []validator.Map{mapvalidator.SizeAtLeast(3)},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name.",
				Validators: []validator.String{stringvalidator.AlsoRequires(
					path.MatchRoot("a"),
					path.MatchRelative().AtParent().AtName("b"),
					path.MatchRoot("c").AtAnyListIndex().AtName("d"),
				)},
			},
			"unrelated": schema.StringAttribute{
				MarkdownDescription: "Unrelated.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}, fwschema.SizeAtLeast(), fwschema.AlsoRequires())

	for _, tc := range []struct{ name, got, want string }{
		{"set", out.Attributes["set"].GetMarkdownDescription(), "Must contain at least 2 elements."},
		{"map", out.Attributes["map"].GetMarkdownDescription(), "Must contain at least 3 elements."},
		{"also requires", out.Attributes["name"].GetMarkdownDescription(), "Name. Also requires: `a`, `b`, `c[*].d`."},
		{"unrelated", out.Attributes["unrelated"].GetMarkdownDescription(), "Unrelated."},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func ExampleDefault() {
	ctx := context.Background()
	s := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"retention": schema.Int64Attribute{
				MarkdownDescription: "Days to keep logs.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
				Validators:          []validator.Int64{int64validator.ConflictsWith(path.MatchRoot("archive"))},
			},
		},
	}, fwschema.Default(), fwschema.ConflictsWith())

	fmt.Println(s.Attributes["retention"].GetMarkdownDescription())
	// Output:
	// Days to keep logs. Defaults to `30`. Conflicts with: `archive`.
}

func TestDefaultAndPathSteps(t *testing.T) {
	ctx := context.Background()
	str := func(d defaults.String, v ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Default: d, Validators: v}
	}
	out := fwschema.Describe(ctx, schema.Schema{
		Attributes: map[string]schema.Attribute{
			"string":     str(stringdefault.StaticString("x")),
			"empty":      str(stringdefault.StaticString("")),
			"float":      schema.Float64Attribute{Default: float64default.StaticFloat64(1.5)},
			"whole":      schema.Float64Attribute{Default: float64default.StaticFloat64(2)},
			"bool":       schema.BoolAttribute{Default: booldefault.StaticBool(true)},
			"exactly":    str(nil, stringvalidator.ExactlyOneOf(path.MatchRoot("a"), path.MatchRoot("b"))),
			"exactlyOne": str(nil, stringvalidator.ExactlyOneOf(path.MatchRoot("a"))),
			"atLeast":    str(nil, stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("a"))),
			"elements": schema.ListAttribute{
				ElementType: types.StringType,
				Validators: []validator.List{listvalidator.ValueStringsAre(
					stringvalidator.LengthAtLeast(1),
					stringvalidator.OneOf("x", "y"),
				)},
			},
		},
	}, fwschema.Default(), fwschema.ExactlyOneOf(), fwschema.AtLeastOneOf(), fwschema.Enum())

	for _, tc := range []struct{ name, want string }{
		{"string", "Defaults to `x`."},
		{"empty", "Defaults to `\"\"`."},
		{"float", "Defaults to `1.5`."},
		{"whole", "Defaults to `2`."},
		{"bool", "Defaults to `true`."},
		{"exactly", "Exactly one of this attribute, `a` or `b` must be set."},
		{"exactlyOne", "Exactly one of this attribute or `a` must be set."},
		{"atLeast", "At least one of this attribute or `a` must be set."},
		{"elements", "Valid values: `x`, `y`."},
	} {
		if got := out.Attributes[tc.name].GetMarkdownDescription(); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
