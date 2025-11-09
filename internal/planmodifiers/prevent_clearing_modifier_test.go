package planmodifiers_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aaearon/terraform-provider-cyberark-sia/internal/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPreventClearingModifier_PlanModifyString(t *testing.T) {
	tests := []struct {
		name          string
		stateValue    types.String
		planValue     types.String
		expectedError bool
		expectedPlan  types.String
		errorContains string
	}{
		{
			name:          "resource creation - no state value",
			stateValue:    types.StringNull(),
			planValue:     types.StringValue("new-value"),
			expectedError: false,
			expectedPlan:  types.StringValue("new-value"),
		},
		{
			name:          "resource creation - empty state",
			stateValue:    types.StringValue(""),
			planValue:     types.StringValue("new-value"),
			expectedError: false,
			expectedPlan:  types.StringValue("new-value"),
		},
		{
			name:          "update to new value - allowed",
			stateValue:    types.StringValue("old-value"),
			planValue:     types.StringValue("new-value"),
			expectedError: false,
			expectedPlan:  types.StringValue("new-value"),
		},
		{
			name:          "update to empty string - blocked",
			stateValue:    types.StringValue("existing-value"),
			planValue:     types.StringValue(""),
			expectedError: true,
			expectedPlan:  types.StringValue("existing-value"), // Preserved
			errorContains: "cannot be removed once set",
		},
		{
			name:          "update to null - blocked",
			stateValue:    types.StringValue("existing-value"),
			planValue:     types.StringNull(),
			expectedError: true,
			expectedPlan:  types.StringValue("existing-value"), // Preserved
			errorContains: "cannot be removed once set",
		},
		{
			name:          "resource destruction - allowed",
			stateValue:    types.StringValue("existing-value"),
			planValue:     types.StringUnknown(),
			expectedError: false,
			expectedPlan:  types.StringUnknown(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			modifier := planmodifiers.PreventClearing()

			req := planmodifier.StringRequest{
				Path:        path.Root("test_attribute"),
				StateValue:  tt.stateValue,
				PlanValue:   tt.planValue,
				ConfigValue: tt.planValue,
			}

			resp := &planmodifier.StringResponse{
				PlanValue: tt.planValue,
			}

			modifier.PlanModifyString(ctx, req, resp)

			// Check error expectation
			if tt.expectedError {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected error but got none")
				}
				if tt.errorContains != "" {
					found := false
					for _, diag := range resp.Diagnostics.Errors() {
						if diag.Detail() != "" && strings.Contains(diag.Detail(), tt.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error to contain %q, but diagnostics were: %v", tt.errorContains, resp.Diagnostics)
					}
				}
			} else {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected error: %v", resp.Diagnostics)
				}
			}

			// Check plan value
			if !resp.PlanValue.Equal(tt.expectedPlan) {
				t.Errorf("expected plan value %v, got %v", tt.expectedPlan, resp.PlanValue)
			}
		})
	}
}

func TestPreventClearingModifier_Description(t *testing.T) {
	ctx := context.Background()
	modifier := planmodifiers.PreventClearing()

	desc := modifier.Description(ctx)
	if desc == "" {
		t.Error("Description should not be empty")
	}

	mdDesc := modifier.MarkdownDescription(ctx)
	if mdDesc == "" {
		t.Error("MarkdownDescription should not be empty")
	}
}
