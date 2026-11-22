# AGENTS.md

Helpers for Terraform Plugin Framework providers, one package per area (`fwacctest`, `fwdiag`, `fwschema`, `fwsweep`, `fwtypes`).

- `make test` runs the tests. Add an `Example` function for new exported APIs; they appear in the README and in `go doc`.
- README.md between the `gomarkdoc` markers is generated. Edit the doc comments and run `make generate`; CI fails if it is out of date.
- `skills/terraform-plugin-framework-utils/SKILL.md` guides agents using this library. Update its table when adding or changing a helper.
- Commit messages and PR titles use Conventional Commits; PR labels are derived from the title.
