package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestUUIDValidator verifies the validator accepts valid UUID formats
// and rejects invalid values. Uses representative test cases to cover
// regex validation without exhaustively testing every hex character combination.
func TestUUIDValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: Standard UUID v4 format
		{
			name:      "valid UUID with dashes",
			value:     types.StringValue("c2c7bcc6-9560-44e0-8dff-5be221cd37ee"),
			expectErr: false,
		},
		{
			name:      "valid UUID with underscores (SIA format)",
			value:     types.StringValue("c2c7bcc6_9560_44e0_8dff_5be221cd37ee"),
			expectErr: false,
		},
		{
			name:      "valid UUID with uppercase",
			value:     types.StringValue("C2C7BCC6-9560-44E0-8DFF-5BE221CD37EE"),
			expectErr: false,
		},
		{
			name:      "valid UUID mixed case",
			value:     types.StringValue("A1B2c3d4-5E6F-7a8B-9C0D-1e2F3a4B5c6D"),
			expectErr: false,
		},
		{
			name:      "valid UUID mixed separators (regex allows this)",
			value:     types.StringValue("c2c7bcc6-9560_44e0-8dff-5be221cd37ee"),
			expectErr: false,
		},

		// Invalid: Malformed
		{
			name:      "invalid missing segment",
			value:     types.StringValue("c2c7bcc6-9560-44e0-5be221cd37ee"),
			expectErr: true,
		},
		{
			name:      "invalid wrong segment length",
			value:     types.StringValue("c2c7bc-9560-44e0-8dff-5be221cd37ee"),
			expectErr: true,
		},
		{
			name:      "invalid characters (non-hex)",
			value:     types.StringValue("g2c7bcc6-9560-44e0-8dff-5be221cd37ee"),
			expectErr: true,
		},
		{
			name:      "invalid no separators",
			value:     types.StringValue("c2c7bcc6956044e08dff5be221cd37ee"),
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
			v := UUID()
			req := validator.StringRequest{
				Path:        path.Root("uuid"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("UUID() hasError = %v, expectErr %v, value = %q",
					hasError, tt.expectErr, tt.value.ValueString())
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestUUIDValidator_Description(t *testing.T) {
	v := UUID()
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
