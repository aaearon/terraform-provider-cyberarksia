package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestNoForwardSlashesValidator verifies the validator rejects names containing
// forward slashes. Uses representative test cases to cover the validation logic.
func TestNoForwardSlashesValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: Names without forward slashes
		{
			name:      "valid simple name",
			value:     types.StringValue("my-target-set"),
			expectErr: false,
		},
		{
			name:      "valid with hyphens",
			value:     types.StringValue("prod-web-servers-2024"),
			expectErr: false,
		},
		{
			name:      "valid with underscores",
			value:     types.StringValue("dev_database_servers"),
			expectErr: false,
		},
		{
			name:      "valid with dots",
			value:     types.StringValue("servers.prod.us-east-1"),
			expectErr: false,
		},
		{
			name:      "valid with mixed separators",
			value:     types.StringValue("app-servers_v1.0"),
			expectErr: false,
		},

		// Invalid: Names with forward slashes
		{
			name:      "invalid single forward slash",
			value:     types.StringValue("prod/web-servers"),
			expectErr: true,
		},
		{
			name:      "invalid multiple forward slashes",
			value:     types.StringValue("prod/us-east-1/web-servers"),
			expectErr: true,
		},
		{
			name:      "invalid trailing slash",
			value:     types.StringValue("prod-servers/"),
			expectErr: true,
		},
		{
			name:      "invalid leading slash",
			value:     types.StringValue("/prod-servers"),
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
			v := NoForwardSlashes()
			req := validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("NoForwardSlashes() hasError = %v, expectErr %v, value = %q",
					hasError, tt.expectErr, tt.value.ValueString())
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestNoForwardSlashesValidator_Description(t *testing.T) {
	v := NoForwardSlashes()
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
