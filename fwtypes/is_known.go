// Package fwtypes provides helpers for working with Terraform Plugin Framework
// attribute values.
package fwtypes

// IsKnown reports whether value is neither null nor unknown.
func IsKnown(value interface {
	IsNull() bool
	IsUnknown() bool
}) bool {
	return !value.IsNull() && !value.IsUnknown()
}
