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
	if req.Plan.Raw.IsNull() {
		return
	}

	// Check if user removed attribute from config or set it to empty
	stateHasValue := !req.StateValue.IsNull() && req.StateValue.ValueString() != ""
	configIsAbsent := req.ConfigValue.IsNull()
	configIsEmpty := !req.ConfigValue.IsNull() && req.ConfigValue.ValueString() == ""

	if stateHasValue && (configIsAbsent || configIsEmpty) {
		resp.Diagnostics.AddError(
			"Cannot Clear Attribute",
			"The "+req.Path.String()+" field cannot be removed once set due to API PATCH semantics. "+
				"You can update it to a different value, but cannot clear it entirely.",
		)
		resp.PlanValue = req.StateValue
	}
}
