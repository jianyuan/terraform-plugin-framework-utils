// Package fwschema provides helpers for building Terraform Plugin Framework
// schemas.
package fwschema

import (
	"context"
	"reflect"
	"strconv"
	"strings"
)

// Describer is implemented by plan modifiers and validators.
type Describer interface {
	Description(ctx context.Context) string
}

// Field is the part of an attribute or block that a [Step] reads and edits.
type Field struct {
	// Description is written back to the attribute's Description or
	// MarkdownDescription.
	Description string
	// Markdown reports whether Description is the MarkdownDescription.
	Markdown      bool
	PlanModifiers []Describer
	Validators    []Describer
}

// Step edits one field. Steps run in the order they are passed to [Describe].
type Step func(ctx context.Context, f *Field)

// Describe runs steps on every attribute and block in s, including nested
// ones, and returns the updated schema. s is a resource, data source,
// ephemeral resource or provider schema. The schema's own description is left
// alone.
//
// The steps run once on MarkdownDescription and, if it is set, once on
// Description. An empty MarkdownDescription starts from Description, so
// attributes that only set Description still get a MarkdownDescription.
func Describe[S any](ctx context.Context, s S, steps ...Step) S {
	v := reflect.ValueOf(&s).Elem()
	walk(ctx, v, steps)
	return s
}

// walk updates the Attributes and Blocks maps of an addressable struct.
func walk(ctx context.Context, v reflect.Value, steps []Step) {
	for _, name := range []string{"Attributes", "Blocks"} {
		m := v.FieldByName(name)
		if m.Kind() != reflect.Map || m.IsNil() {
			continue
		}
		// Copy the map so the caller's schema isn't mutated.
		out := reflect.MakeMapWithSize(m.Type(), m.Len())
		iter := m.MapRange()
		for iter.Next() {
			elem := iter.Value()
			if elem.Kind() == reflect.Interface {
				elem = elem.Elem()
			}
			cp := reflect.New(elem.Type()).Elem()
			cp.Set(elem)
			describe(ctx, cp, steps)
			walk(ctx, cp, steps) // SingleNested attributes and blocks
			if nested := cp.FieldByName("NestedObject"); nested.Kind() == reflect.Struct {
				walk(ctx, nested, steps)
			}
			out.SetMapIndex(iter.Key(), cp)
		}
		m.Set(out)
	}
}

func describe(ctx context.Context, v reflect.Value, steps []Step) {
	md, plain := v.FieldByName("MarkdownDescription"), v.FieldByName("Description")
	if md.Kind() != reflect.String || plain.Kind() != reflect.String {
		return
	}
	f := Field{
		PlanModifiers: describers(v.FieldByName("PlanModifiers")),
		Validators:    describers(v.FieldByName("Validators")),
	}
	run := func(desc string, markdown bool) string {
		f.Description, f.Markdown = desc, markdown
		for _, step := range steps {
			step(ctx, &f)
		}
		return strings.TrimSpace(f.Description)
	}
	start := md.String()
	if start == "" {
		start = plain.String()
	}
	md.SetString(run(start, true))
	if plain.String() != "" {
		plain.SetString(run(plain.String(), false))
	}
}

func describers(s reflect.Value) []Describer {
	if s.Kind() != reflect.Slice {
		return nil
	}
	var out []Describer
	for i := range s.Len() {
		if d, ok := s.Index(i).Interface().(Describer); ok {
			out = append(out, d)
		}
	}
	return out
}

// requiresReplace holds the descriptions of the framework's RequiresReplace
// and RequiresReplaceIfConfigured plan modifiers, which have no exported type.
var requiresReplace = map[string]bool{
	"If the value of this attribute changes, Terraform will destroy and recreate the resource.":                   true,
	"If the value of this attribute is configured and changes, Terraform will destroy and recreate the resource.": true,
}

// ForceNew prepends a red "(Force new)" to the MarkdownDescription, and a
// plain "(Force new)" to the Description, of fields that use a RequiresReplace
// or RequiresReplaceIfConfigured plan modifier. RequiresReplaceIf is not
// detected because its description is provider-defined. Use [ForceNewLabel]
// to change the labels.
func ForceNew() Step {
	return ForceNewLabel(`<span style="color:red">(Force new)</span>`, "(Force new)")
}

// ForceNewLabel is [ForceNew] with custom labels for the MarkdownDescription
// and the Description.
func ForceNewLabel(markdown, plain string) Step {
	return func(ctx context.Context, f *Field) {
		for _, m := range f.PlanModifiers {
			if requiresReplace[m.Description(ctx)] {
				label := plain
				if f.Markdown {
					label = markdown
				}
				f.Description = label + " " + f.Description
				return
			}
		}
	}
}

const oneOfPrefix = "value must be one of: "

// Enum appends "Valid values: `a`, `b`." to fields that use a OneOf or
// OneOfCaseInsensitive validator from terraform-plugin-framework-validators.
// The description gets exactly one period before the label, unless it is
// empty or ends with "!" or "?". Use [EnumLabel] to change the label.
func Enum() Step {
	return EnumLabel("Valid values:", "Valid values:")
}

// EnumLabel is [Enum] with custom labels for the MarkdownDescription and the
// Description.
func EnumLabel(markdown, plain string) Step {
	return func(ctx context.Context, f *Field) {
		for _, v := range f.Validators {
			// shortcut: parses the validator's description since its values are unexported; breaks if upstream rewords it, which the tests pin.
			values, ok := parseOneOf(v.Description(ctx))
			if !ok {
				continue
			}
			label := plain
			if f.Markdown {
				label = markdown
			}
			f.Description = endSentence(f.Description) + " " + label + " `" + strings.Join(values, "`, `") + "`."
			return
		}
	}
}

// endSentence trims trailing spaces and periods, then adds one period.
func endSentence(s string) string {
	s = strings.TrimRight(s, ". \t\n")
	if s == "" || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") {
		return s
	}
	return s + "."
}

// parseOneOf parses `value must be one of: ["a" "b"]`.
func parseOneOf(desc string) ([]string, bool) {
	rest, ok := strings.CutPrefix(desc, oneOfPrefix)
	if !ok || !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
		return nil, false
	}
	rest = rest[1 : len(rest)-1]
	var values []string
	for rest = strings.TrimSpace(rest); rest != ""; rest = strings.TrimSpace(rest) {
		quoted, err := strconv.QuotedPrefix(rest)
		if err != nil {
			return nil, false
		}
		value, _ := strconv.Unquote(quoted)
		values = append(values, value)
		rest = rest[len(quoted):]
	}
	return values, len(values) > 0
}
