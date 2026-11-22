// Package fwschema provides helpers for building Terraform Plugin Framework
// schemas.
package fwschema

import (
	"context"
	"fmt"
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
	// Default is the attribute's Default, or nil if it has none.
	Default Describer
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
	if d := v.FieldByName("Default"); d.Kind() == reflect.Interface && !d.IsNil() {
		f.Default, _ = d.Interface().(Describer)
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
				f.Description = label + " " + strings.TrimLeft(f.Description, " ")
				return
			}
		}
	}
}

const oneOfPrefix = "value must be one of: "

// Enum appends "Valid values: `a`, `b`." to fields that use a OneOf or
// OneOfCaseInsensitive validator from terraform-plugin-framework-validators,
// including one wrapped in an element validator such as
// listvalidator.ValueStringsAre.
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

const elementPrefix = "element value must satisfy all validations: "

// parseOneOf parses `value must be one of: ["a" "b"]`, also when it is one of
// the validations joined with " + " inside an element validator.
func parseOneOf(desc string) ([]string, bool) {
	if inner, ok := strings.CutPrefix(desc, elementPrefix); ok {
		// shortcut: splits on " + ", so a value containing " + " breaks detection; parse properly if that comes up.
		for part := range strings.SplitSeq(inner, " + ") {
			if values, ok := parseOneOf(part); ok {
				return values, true
			}
		}
		return nil, false
	}
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

// SizeAtLeast appends "Must contain at least N elements." to fields that use a
// SizeAtLeast validator from the listvalidator, setvalidator or mapvalidator
// packages.
func SizeAtLeast() Step {
	return func(ctx context.Context, f *Field) {
		for _, v := range f.Validators {
			// shortcut: parses the validator's description since its minimum is unexported; breaks if upstream rewords it, which the tests pin.
			var kind string
			var n int
			desc := v.Description(ctx)
			if _, err := fmt.Sscanf(desc, "%s must contain at least %d elements", &kind, &n); err != nil ||
				desc != fmt.Sprintf("%s must contain at least %d elements", kind, n) {
				continue
			}
			noun := "elements"
			if n == 1 {
				noun = "element"
			}
			f.Description = endSentence(f.Description) + fmt.Sprintf(" Must contain at least %d %s.", n, noun)
			return
		}
	}
}

// Default appends "Defaults to `x`." to fields with a static Default from the
// framework's default packages, such as stringdefault.StaticString.
func Default() Step {
	return func(ctx context.Context, f *Field) {
		if f.Default == nil {
			return
		}
		// shortcut: parses the default's description since its value is unexported; breaks if upstream rewords it, which the tests pin.
		value, ok := strings.CutPrefix(f.Default.Description(ctx), "value defaults to ")
		if !ok {
			return
		}
		if _, err := strconv.ParseFloat(value, 64); err == nil && strings.Contains(value, ".") {
			value = strings.TrimSuffix(strings.TrimRight(value, "0"), ".") // %f prints 1.500000
		}
		if value == "" {
			value = `""`
		}
		f.Description = endSentence(f.Description) + " Defaults to `" + value + "`."
	}
}

// AlsoRequires appends "Also requires: `a`, `b`." to fields that use an
// AlsoRequires validator from terraform-plugin-framework-validators.
func AlsoRequires() Step {
	return pathStep("Ensure that if an attribute is set, also these are set: ", func(paths []string) string {
		return "Also requires: " + strings.Join(paths, ", ") + "."
	})
}

// ConflictsWith appends "Conflicts with: `a`, `b`." to fields that use a
// ConflictsWith validator from terraform-plugin-framework-validators.
func ConflictsWith() Step {
	return pathStep("Ensure that if an attribute is set, these are not set: ", func(paths []string) string {
		return "Conflicts with: " + strings.Join(paths, ", ") + "."
	})
}

// ExactlyOneOf appends "Exactly one of this attribute or `a` must be set." to
// fields that use an ExactlyOneOf validator from
// terraform-plugin-framework-validators.
func ExactlyOneOf() Step {
	return pathStep("Ensure that one and only one attribute from this collection is set: ", func(paths []string) string {
		return "Exactly one of " + joinOr(append([]string{"this attribute"}, paths...)) + " must be set."
	})
}

// AtLeastOneOf appends "At least one of this attribute or `a` must be set." to
// fields that use an AtLeastOneOf validator from
// terraform-plugin-framework-validators.
func AtLeastOneOf() Step {
	return pathStep("Ensure that at least one attribute from this collection is set: ", func(paths []string) string {
		return "At least one of " + joinOr(append([]string{"this attribute"}, paths...)) + " must be set."
	})
}

// pathStep appends text(paths) to fields with a validator whose description is
// prefix followed by a path expression list such as "[a,<.b]", quoted or not.
// Each path is wrapped in backticks, and sibling paths such as
// path.MatchRelative().AtParent().AtName("b") are shown as `b`.
func pathStep(prefix string, text func(paths []string) string) Step {
	return func(ctx context.Context, f *Field) {
		for _, v := range f.Validators {
			// shortcut: parses the validator's description since its paths are unexported; breaks if upstream rewords it, which the tests pin.
			list, ok := strings.CutPrefix(v.Description(ctx), prefix)
			if !ok {
				continue
			}
			if unquoted, err := strconv.Unquote(list); err == nil {
				list = unquoted
			}
			if !strings.HasPrefix(list, "[") || !strings.HasSuffix(list, "]") || len(list) == 2 {
				continue
			}
			// shortcut: splits on commas, so a map key containing a comma is split too; parse brackets if that comes up.
			paths := strings.Split(list[1:len(list)-1], ",")
			for i, p := range paths {
				paths[i] = "`" + strings.TrimPrefix(p, "<.") + "`"
			}
			f.Description = endSentence(f.Description) + " " + text(paths)
			return
		}
	}
}

// joinOr joins items as "a", "a or b" or "a, b or c".
func joinOr(items []string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
}

// DefaultSteps returns every step in this package, in an order that reads
// well: valid values, size, default, then the path rules. ForceNew runs last
// so an empty description doesn't get a period after "(Force new)".
//
//	resp.Schema = fwschema.Describe(ctx, schema.Schema{...}, fwschema.DefaultSteps()...)
func DefaultSteps() []Step {
	return []Step{
		Enum(),
		SizeAtLeast(),
		Default(),
		AlsoRequires(),
		ConflictsWith(),
		ExactlyOneOf(),
		AtLeastOneOf(),
		ForceNew(),
	}
}
