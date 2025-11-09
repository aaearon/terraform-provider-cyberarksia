// Package planmodifiers provides custom Terraform plan modifiers for the CyberArk SIA provider.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IDFollowsName returns a plan modifier that marks ID as unknown when name is changing,
// allowing the ID to be updated during rename operations while preserving it during other updates.
func IDFollowsName() planmodifier.String {
	return idFollowsNameModifier{}
}

// idFollowsNameModifier implements the plan modifier.
type idFollowsNameModifier struct{}

// Description returns a human-readable description of the plan modifier.
func (m idFollowsNameModifier) Description(_ context.Context) string {
	return "ID equals name. When name changes, ID is marked as unknown to allow the rename."
}

// MarkdownDescription returns a markdown description of the plan modifier.
func (m idFollowsNameModifier) MarkdownDescription(_ context.Context) string {
	return "ID equals name. When name changes, ID is marked as unknown to allow the rename."
}

// PlanModifyString implements the plan modification logic.
func (m idFollowsNameModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the resource is being created, no modification needed
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// If the resource is being destroyed, no modification needed
	if req.PlanValue.IsNull() {
		return
	}

	// Get the name attribute from config
	var namePlan string
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &namePlan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var nameState string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("name"), &nameState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If name is changing, mark ID as unknown (it will be recomputed during apply)
	if namePlan != nameState {
		resp.PlanValue = types.StringUnknown()
		return
	}

	// If name is NOT changing, preserve the state ID
	resp.PlanValue = req.StateValue
}
