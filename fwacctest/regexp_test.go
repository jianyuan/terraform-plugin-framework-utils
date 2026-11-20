package fwacctest_test

import (
	"fmt"

	"github.com/jianyuan/terraform-plugin-framework-utils/fwacctest"
)

func ExampleExpectLiteralError() {
	re := fwacctest.ExpectLiteralError("Invalid value (must be > 0)")

	// Terraform wrapped the message across lines.
	fmt.Println(re.MatchString("Error: Invalid value (must\nbe > 0)"))
	fmt.Println(re.MatchString("Error: Invalid value must be > 0"))
	// Output:
	// true
	// false
}
