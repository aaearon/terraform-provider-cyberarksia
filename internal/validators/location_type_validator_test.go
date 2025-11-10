package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestLocationTypeValidator verifies the validator enforces "FQDN/IP" as the only
// valid location type. Uses representative test cases as this is a single-value validator.
func TestLocationTypeValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: Only accepted value
		{
			name:      "valid FQDN/IP",
			value:     types.StringValue("FQDN/IP"),
			expectErr: false,
		},

		// Invalid: Case variations not accepted
		{
			name:      "invalid lowercase fqdn/ip",
			value:     types.StringValue("fqdn/ip"),
			expectErr: true,
		},
		{
			name:      "invalid mixed case Fqdn/Ip",
			value:     types.StringValue("Fqdn/Ip"),
			expectErr: true,
		},

		// Invalid: Other common location types (not supported for databases)
		{
			name:      "invalid AWS",
			value:     types.StringValue("AWS"),
			expectErr: true,
		},
		{
			name:      "invalid AZURE",
			value:     types.StringValue("AZURE"),
			expectErr: true,
		},
		{
			name:      "invalid GCP",
			value:     types.StringValue("GCP"),
			expectErr: true,
		},

		// Invalid: Other values
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
			v := LocationType()
			req := validator.StringRequest{
				Path:        path.Root("location_type"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("LocationType() hasError = %v, expectErr %v, value = %q",
					hasError, tt.expectErr, tt.value.ValueString())
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestLocationTypeValidator_Description(t *testing.T) {
	v := LocationType()
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
