package validators_test

import (
	"context"
	"strings"
	"testing"

	"github.com/aaearon/terraform-provider-cyberark-sia/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNoForwardSlashesValidator(t *testing.T) {
	tests := []struct {
		name            string
		value           types.String
		expectWarning   bool
		warningContains string
	}{
		{
			name:          "valid name with hyphens",
			value:         types.StringValue("prod-servers"),
			expectWarning: false,
		},
		{
			name:          "valid name with underscores",
			value:         types.StringValue("prod_servers"),
			expectWarning: false,
		},
		{
			name:          "valid name with dots",
			value:         types.StringValue("prod.example.com"),
			expectWarning: false,
		},
		{
			name:            "invalid name with forward slash",
			value:           types.StringValue("prod/servers"),
			expectWarning:   true,
			warningContains: "forward slashes",
		},
		{
			name:            "invalid name with multiple forward slashes",
			value:           types.StringValue("env/region/servers"),
			expectWarning:   true,
			warningContains: "deletion failures",
		},
		{
			name:          "null value - skip validation",
			value:         types.StringNull(),
			expectWarning: false,
		},
		{
			name:          "unknown value - skip validation",
			value:         types.StringUnknown(),
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			nameValidator := validators.NoForwardSlashes()

			req := validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: tt.value,
			}

			resp := &validator.StringResponse{}

			nameValidator.ValidateString(ctx, req, resp)

			// Check warning expectation
			if tt.expectWarning {
				warnings := resp.Diagnostics.Warnings()
				if len(warnings) == 0 {
					t.Fatalf("expected warning but got none")
				}
				if tt.warningContains != "" {
					found := false
					for _, diag := range warnings {
						// Check both Detail and Summary for the warning text
						if strings.Contains(diag.Detail(), tt.warningContains) || strings.Contains(diag.Summary(), tt.warningContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected warning to contain %q, but diagnostics were: %v", tt.warningContains, resp.Diagnostics)
					}
				}
			} else {
				warnings := resp.Diagnostics.Warnings()
				if len(warnings) > 0 {
					t.Fatalf("unexpected warning: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestNoForwardSlashesValidator_Description(t *testing.T) {
	ctx := context.Background()
	nameValidator := validators.NoForwardSlashes()

	desc := nameValidator.Description(ctx)
	if desc == "" {
		t.Error("Description should not be empty")
	}

	mdDesc := nameValidator.MarkdownDescription(ctx)
	if mdDesc == "" {
		t.Error("MarkdownDescription should not be empty")
	}
}
