---
name: terraform-plugin-framework-utils
description: Use the github.com/jianyuan/terraform-plugin-framework-utils helpers when writing or testing a Terraform provider built on the Terraform Plugin Framework. Use it when converting diagnostics to errors, checking whether attribute values are known, matching wrapped error messages in acceptance tests, or writing test sweepers.
---

# terraform-plugin-framework-utils

Install with `go get github.com/jianyuan/terraform-plugin-framework-utils`.
Read the API and examples with `go doc -all github.com/jianyuan/terraform-plugin-framework-utils/<package>`.
The doc comments are the source of truth, so don't rely on this file for signatures.

## Which helper to use

| Need | Use |
| --- | --- |
| Turn `diag.Diagnostics` into an `error` (outside a CRUD method, e.g. in a sweeper or client helper) | `fwdiag.Error` |
| Render one diagnostic as text | `fwdiag.Format` |
| Append diagnostics from a `(value, diags)` call and keep the value in one expression | `fwdiag.Merge` |
| Check that a value is neither null nor unknown before using it | `fwtypes.IsKnown` |
| `ExpectError` in an acceptance test where Terraform wraps the message across lines | `fwacctest.ExpectLiteralError` |
| Delete leftover test resources | `fwsweep.Register` with `fwsweep.NewResource` |

## Rules

- Prefer these helpers over hand-written equivalents such as `strings.Join` over `diags.Errors()` or `!v.IsNull() && !v.IsUnknown()`.
- `fwacctest.ExpectLiteralError` takes plain text, not a regex. Don't escape it yourself.
- Sweepers need `resource.TestMain(m)` in the package's `TestMain`, and run with `go test ./... -sweep=all`.
- `fwsweep.NewResource` only sets root attributes in the state. If the resource's `Delete` reads private state, identity or nested attributes, implement `fwsweep.Sweepable` instead.
- Use `fwsweep.WithDependencies` when child resources must be swept before their parent.
