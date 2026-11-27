package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// grouperPathPattern matches a Grouper full path: one or more colon-delimited
// components, each non-empty and containing no colon or whitespace. This is a
// conservative constraint on system names (the `name` attribute); it is not
// applied to display labels, which may contain spaces. If a deployment uses
// characters outside this set in system names, widen the pattern here.
var grouperPathPattern = regexp.MustCompile(`^[^:\s]+(:[^:\s]+)*$`)

// grouperPathValidators returns the validators applied to a full Grouper path
// attribute (stem or group `name`).
func grouperPathValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthAtLeast(1),
		stringvalidator.RegexMatches(grouperPathPattern,
			"must be a colon-delimited Grouper path with non-empty components and no whitespace, e.g. \"example:apps:myteam\""),
	}
}
