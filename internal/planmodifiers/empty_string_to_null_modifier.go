// Package planmodifiers provides custom Terraform plan modifiers for the CyberArk SIA provider.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EmptyStringToNull returns a plan modifier that converts empty strings to null during planning.
// This is useful for API fields that treat "" and null as equivalent and always return null.
func EmptyStringToNull() planmodifier.String {
	return emptyStringToNullModifier{}
}

// emptyStringToNullModifier implements the plan modifier.
type emptyStringToNullModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m emptyStringToNullModifier) Description(_ context.Context) string {
	return "Converts empty string to null. The API treats empty strings and null as equivalent."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m emptyStringToNullModifier) MarkdownDescription(_ context.Context) string {
	return "Converts empty string to null. The API treats empty strings and null as equivalent."
}

// PlanModifyString implements the plan modification logic.
func (m emptyStringToNullModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If config has empty string, plan for null instead
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && req.ConfigValue.ValueString() == "" {
		resp.PlanValue = types.StringNull()
		return
	}

	// Otherwise use the default plan value
	// (which comes from earlier plan modifiers in the chain)
}
