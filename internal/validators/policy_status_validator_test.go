package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPolicyStatusValidator verifies the validator accepts valid policy statuses
// and rejects invalid values. Uses representative test cases as the validator
// checks membership in a 4-value list (active/Active, suspended/Suspended).
func TestPolicyStatusValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: Accepted case variations
		{
			name:      "valid lowercase active",
			value:     types.StringValue("active"),
			expectErr: false,
		},
		{
			name:      "valid uppercase Active",
			value:     types.StringValue("Active"),
			expectErr: false,
		},
		{
			name:      "valid lowercase suspended",
			value:     types.StringValue("suspended"),
			expectErr: false,
		},
		{
			name:      "valid uppercase Suspended",
			value:     types.StringValue("Suspended"),
			expectErr: false,
		},

		// Invalid: Server-managed statuses
		{
			name:      "invalid expired (server-managed)",
			value:     types.StringValue("expired"),
			expectErr: true,
		},
		{
			name:      "invalid validating (server-managed)",
			value:     types.StringValue("validating"),
			expectErr: true,
		},
		{
			name:      "invalid error (server-managed)",
			value:     types.StringValue("error"),
			expectErr: true,
		},

		// Invalid: Other values
		{
			name:      "invalid ACTIVE (all caps)",
			value:     types.StringValue("ACTIVE"),
			expectErr: true,
		},
		{
			name:      "invalid empty string",
			value:     types.StringValue(""),
			expectErr: true,
		},

		// Edge cases: Null/unknown values (skip validation)
		{
			name:      "null value skips validation",
			value:     types.StringNull(),
			expectErr: false,
		},
		{
			name:      "unknown value skips validation",
			value:     types.StringUnknown(),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := PolicyStatus()
			req := validator.StringRequest{
				Path:        path.Root("status"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("PolicyStatus() hasError = %v, expectErr %v, value = %q",
					hasError, tt.expectErr, tt.value.ValueString())
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestPolicyStatusValidator_Description(t *testing.T) {
	v := PolicyStatus()
	ctx := context.Background()

	desc := v.Description(ctx)
	if desc == "" {
		t.Error("Description() returned empty string")
	}

	markdownDesc := v.MarkdownDescription(ctx)
	if markdownDesc == "" {
		t.Error("MarkdownDescription() returned empty string")
	}
}
