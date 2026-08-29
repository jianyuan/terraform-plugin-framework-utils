package fwdiag

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Merge is a utility function that merges the given diagnostics into the provided
// diagnostics and returns the original value.
func Merge[T any](v T, sourceDiags diag.Diagnostics) func(targetDiags *diag.Diagnostics) T {
	return func(targetDiags *diag.Diagnostics) T {
		targetDiags.Append(sourceDiags...)
		return v
	}
}
