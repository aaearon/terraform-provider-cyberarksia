// Package planmodifiers provides custom Terraform plan modifiers for the CyberArk SIA provider.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// PreventClearing returns a plan modifier that prevents clearing a string attribute once it has been set.
// This is useful for API fields that use PATCH semantics where omitting a field preserves its value,
// creating perpetual drift if users attempt to clear the value in their configuration.
func PreventClearing() planmodifier.String {
	return preventClearingModifier{}
}

// preventClearingModifier implements the plan modifier.
type preventClearingModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m preventClearingModifier) Description(_ context.Context) string {
	return "Prevents clearing the attribute once set. You can update it to a different value, but cannot clear it entirely."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m preventClearingModifier) MarkdownDescription(_ context.Context) string {
	return "Prevents clearing the attribute once set. You can update it to a different value, but cannot clear it entirely."
}

// PlanModifyString implements the plan modification logic.
func (m preventClearingModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the resource is being created, no validation needed
	if req.StateValue.IsNull() {
		return
	}

	// If the resource is being destroyed, allow it
	if req.PlanValue.IsUnknown() {
		return
	}

	// Check if the user is trying to clear the attribute
	// Because this attribute has a Default, ConfigValue will NEVER be null (framework fills it with default)
	// Instead, we compare PlanValue to StateValue to detect clearing attempts
	stateHasValue := !req.StateValue.IsNull() && req.StateValue.ValueString() != ""

	// User is trying to clear if:
	// 1. State has a non-default value
	// 2. Plan is reverting to the default value
	// 3. ConfigValue is null (user removed attribute from config)
	userRemovedAttribute := req.ConfigValue.IsNull()
	planRevertingToDefault := !req.PlanValue.Equal(req.StateValue) // Plan differs from state

	if stateHasValue && userRemovedAttribute && planRevertingToDefault {
		resp.Diagnostics.AddError(
			"Cannot Clear Attribute",
			"The "+req.Path.String()+" field cannot be removed once set due to API limitations. "+
				"You can update it to a different value, but cannot clear it entirely.",
		)
		// Preserve the state value to prevent drift
		resp.PlanValue = req.StateValue
	}
}
