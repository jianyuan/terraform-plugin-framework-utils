package fwdiag

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// Error converts the error diagnostics into a single error, or returns nil if
// there are none. Warnings are ignored.
func Error(diags diag.Diagnostics) error {
	errList := diags.Errors()
	if len(errList) == 0 {
		return nil
	}

	errs := make([]error, len(errList))
	for i, d := range errList {
		errs[i] = errors.New(Format(d))
	}

	return errors.Join(errs...)
}

// Format renders the diagnostic as a human-readable string, including its
// detail and attribute path when present.
func Format(d diag.Diagnostic) string {
	msg := d.Summary()

	if detail := d.Detail(); detail != "" {
		msg += "\n\n" + detail
	}

	if p, ok := d.(diag.DiagnosticWithPath); ok {
		msg += "\n" + p.Path().String()
	}

	return msg
}
