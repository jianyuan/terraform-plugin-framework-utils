package fwdiag_test

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/jianyuan/terraform-plugin-framework-utils/fwdiag"
)

func ExampleError() {
	var diags diag.Diagnostics
	diags.AddWarning("Ignored Warning", "Warnings are not errors")
	diags.AddError("First Error", "Something went wrong")
	diags.AddAttributeError(path.Root("name"), "Second Error", "Invalid name")

	fmt.Println(fwdiag.Error(diags))
	fmt.Println(fwdiag.Error(nil))
	// Output:
	// First Error
	//
	// Something went wrong
	// Second Error
	//
	// Invalid name
	// name
	// <nil>
}
