package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPrincipalTypeValidator verifies the validator accepts valid principal types
// and rejects invalid values. Uses representative test cases as the validator
// checks membership in a 3-value list (USER, GROUP, ROLE).
func TestPrincipalTypeValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: All accepted types
		{
			name:      "valid USER",
			value:     types.StringValue("USER"),
			expectErr: false,
		},
		{
			name:      "valid GROUP",
			value:     types.StringValue("GROUP"),
			expectErr: false,
		},
		{
			name:      "valid ROLE",
			value:     types.StringValue("ROLE"),
			expectErr: false,
		},

		// Invalid: Case sensitivity
		{
			name:      "invalid lowercase user",
			value:     types.StringValue("user"),
			expectErr: true,
		},
		{
			name:      "invalid mixed case Group",
			value:     types.StringValue("Group"),
			expectErr: true,
		},

		// Invalid: Common mistakes
		{
			name:      "invalid ADMIN (not supported)",
			value:     types.StringValue("ADMIN"),
			expectErr: true,
		},
		{
			name:      "invalid SERVICE (not supported)",
			value:     types.StringValue("SERVICE"),
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
			v := PrincipalType()
			req := validator.StringRequest{
				Path:        path.Root("principal_type"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("PrincipalType() hasError = %v, expectErr %v, value = %q",
					hasError, tt.expectErr, tt.value.ValueString())
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestPrincipalTypeValidator_Description(t *testing.T) {
	v := PrincipalType()
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
