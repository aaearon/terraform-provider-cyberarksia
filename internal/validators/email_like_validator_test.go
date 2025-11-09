package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestEmailLikeValidator verifies the validator accepts email-like formats
// and rejects invalid values. Uses representative test cases to cover the
// regex pattern without exhaustively testing every valid character combination.
func TestEmailLikeValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: Representative formats
		{
			name:      "valid standard email",
			value:     types.StringValue("user@example.com"),
			expectErr: false,
		},
		{
			name:      "valid CyberArk cloud directory format",
			value:     types.StringValue("tim@cyberark.cloud.12345"),
			expectErr: false,
		},
		{
			name:      "valid complex format with special chars",
			value:     types.StringValue("user.name+tag-test_123@sub.domain.co.uk"),
			expectErr: false,
		},
		{
			name:      "valid with percent",
			value:     types.StringValue("user%test@example.com"),
			expectErr: false,
		},

		// Invalid: Missing required parts
		{
			name:      "invalid no @ symbol",
			value:     types.StringValue("username.example.com"),
			expectErr: true,
		},
		{
			name:      "invalid missing domain",
			value:     types.StringValue("user@"),
			expectErr: true,
		},
		{
			name:      "invalid missing TLD",
			value:     types.StringValue("user@domain"),
			expectErr: true,
		},
		{
			name:      "invalid missing local part",
			value:     types.StringValue("@domain.com"),
			expectErr: true,
		},

		// Invalid: Malformed
		{
			name:      "invalid multiple @ symbols",
			value:     types.StringValue("user@@example.com"),
			expectErr: true,
		},
		{
			name:      "invalid spaces",
			value:     types.StringValue("user name@example.com"),
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
			v := EmailLike()
			req := validator.StringRequest{
				Path:        path.Root("email"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("EmailLike() hasError = %v, expectErr %v, value = %q",
					hasError, tt.expectErr, tt.value.ValueString())
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestEmailLikeValidator_Description(t *testing.T) {
	v := EmailLike()
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
