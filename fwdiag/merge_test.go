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
		diagsInner.AddError("Inner Error", "Something went wrong")
		return nil, diagsInner
	}

	var diags diag.Diagnostics
	diags.AddWarning("Outer Warning", "Something else went wrong")
	_ = fwdiag.Merge(doSomething())(&diags)
	fmt.Println(diags)
	// Output: [{Something else went wrong Outer Warning} {Something went wrong Inner Error}]
}
