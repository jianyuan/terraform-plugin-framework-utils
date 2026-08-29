package fwdiag_test

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/jianyuan/terraform-plugin-framework-utils/fwdiag"
)

func ExampleMerge() {
	doSomething := func() (*string, diag.Diagnostics) {
		var diagsInner diag.Diagnostics
		// add in some errors
		diagsInner.AddError("Inner Error", "Something else went wrong")
		return nil, diagsInner
	}

	var diags diag.Diagnostics
	diags.AddWarning("Outer Warning", "Something went wrong")
	_ = fwdiag.Merge(doSomething())(&diags)

	for _, d := range diags {
		fmt.Printf("[%s] %s: %s\n", d.Severity(), d.Summary(), d.Detail())
	}
	// Output:
	// [Warning] Outer Warning: Something went wrong
	// [Error] Inner Error: Something else went wrong
}
