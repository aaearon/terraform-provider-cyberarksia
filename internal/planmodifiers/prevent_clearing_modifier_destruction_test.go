package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestPreventClearingModifier_ResourceDestruction specifically tests that
// resource destruction is allowed (regression test for the bug where
// Plan.Raw.IsNull() wasn't checked, blocking resource destruction)
func TestPreventClearingModifier_ResourceDestruction(t *testing.T) {
	ctx := context.Background()
	modifier := planmodifiers.PreventClearing()

	// During resource destruction:
	// - State has a value
	// - Plan.Raw is null (entire resource being destroyed)
	// - PlanValue can be null or unknown

	req := planmodifier.StringRequest{
		Path:        path.Root("provision_format"),
		StateValue:  types.StringValue("existing-format"),
		PlanValue:   types.StringNull(), // Field value is null during destruction
		ConfigValue: types.StringNull(),
		Plan: tfsdk.Plan{
			// Key: During resource destruction, the entire plan is null
			Raw: tftypes.NewValue(tftypes.Object{}, nil),
		},
		State: tfsdk.State{
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"provision_format": tftypes.String,
				},
			}, map[string]tftypes.Value{
				"provision_format": tftypes.NewValue(tftypes.String, "existing-format"),
			}),
		},
	}

	resp := &planmodifier.StringResponse{
		PlanValue: types.StringNull(),
	}

	// This should NOT produce an error - resource destruction must be allowed
	modifier.PlanModifyString(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("resource destruction should be allowed, but got error: %v", resp.Diagnostics)
	}

	// Plan value should remain null (not preserved from state)
	if !resp.PlanValue.IsNull() {
		t.Errorf("expected plan value to remain null during destruction, got %v", resp.PlanValue)
	}
}

// TestPreventClearingModifier_FieldClearingBlocked tests that clearing a field
// (without destroying the entire resource) is correctly blocked
func TestPreventClearingModifier_FieldClearingBlocked(t *testing.T) {
	ctx := context.Background()
	modifier := planmodifiers.PreventClearing()

	// During field clearing (user sets field to null in config):
	// - State has a value
	// - Plan.Raw is NOT null (resource still exists, just field is being cleared)
	// - PlanValue is null

	req := planmodifier.StringRequest{
		Path:        path.Root("provision_format"),
		StateValue:  types.StringValue("existing-format"),
		PlanValue:   types.StringNull(), // Field is being cleared
		ConfigValue: types.StringNull(),
		Plan: tfsdk.Plan{
			// Key: Plan is NOT null - resource still exists, just field is being cleared
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"provision_format": tftypes.String,
				},
			}, map[string]tftypes.Value{
				"provision_format": tftypes.NewValue(tftypes.String, nil), // Field is null
			}),
		},
		State: tfsdk.State{
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"provision_format": tftypes.String,
				},
			}, map[string]tftypes.Value{
				"provision_format": tftypes.NewValue(tftypes.String, "existing-format"),
			}),
		},
	}

	resp := &planmodifier.StringResponse{
		PlanValue: types.StringNull(),
	}

	// This SHOULD produce an error - field clearing must be blocked
	modifier.PlanModifyString(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when trying to clear field, but got none")
	}

	// Plan value should be preserved from state
	if !resp.PlanValue.Equal(types.StringValue("existing-format")) {
		t.Errorf("expected plan value to be preserved as 'existing-format', got %v", resp.PlanValue)
	}
}

// TestPreventClearingModifier_UpdateAllowed tests that updating to a new value is allowed
func TestPreventClearingModifier_UpdateAllowed(t *testing.T) {
	ctx := context.Background()
	modifier := planmodifiers.PreventClearing()

	req := planmodifier.StringRequest{
		Path:        path.Root("provision_format"),
		StateValue:  types.StringValue("old-format"),
		PlanValue:   types.StringValue("new-format"),
		ConfigValue: types.StringValue("new-format"),
		Plan: tfsdk.Plan{
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"provision_format": tftypes.String,
				},
			}, map[string]tftypes.Value{
				"provision_format": tftypes.NewValue(tftypes.String, "new-format"),
			}),
		},
		State: tfsdk.State{
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					"provision_format": tftypes.String,
				},
			}, map[string]tftypes.Value{
				"provision_format": tftypes.NewValue(tftypes.String, "old-format"),
			}),
		},
	}

	resp := &planmodifier.StringResponse{
		PlanValue: types.StringValue("new-format"),
	}

	modifier.PlanModifyString(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("updating to new value should be allowed, but got error: %v", resp.Diagnostics)
	}

	// Plan value should be the new value
	if !resp.PlanValue.Equal(types.StringValue("new-format")) {
		t.Errorf("expected plan value to be 'new-format', got %v", resp.PlanValue)
	}
}
