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
		name          string
		value         types.String
		errorContains string
		expectError   bool
	}{
		{
			name:        "valid name with hyphens",
			value:       types.StringValue("prod-servers"),
			expectError: false,
		},
		{
			name:        "valid name with underscores",
			value:       types.StringValue("prod_servers"),
			expectError: false,
		},
		{
			name:        "valid name with dots",
			value:       types.StringValue("prod.example.com"),
			expectError: false,
		},
		{
			name:          "invalid name with forward slash",
			value:         types.StringValue("prod/servers"),
			expectError:   true,
			errorContains: "Invalid Target Set Name",
		},
		{
			name:          "invalid name with multiple forward slashes",
			value:         types.StringValue("env/region/servers"),
			expectError:   true,
			errorContains: "API limitations",
		},
		{
			name:        "null value - skip validation",
			value:       types.StringNull(),
			expectError: false,
		},
		{
			name:        "unknown value - skip validation",
			value:       types.StringUnknown(),
			expectError: false,
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

			// Check error expectation
			if tt.expectError {
				errors := resp.Diagnostics.Errors()
				if len(errors) == 0 {
					t.Fatalf("expected error but got none")
				}
				if tt.errorContains != "" {
					found := false
					for _, diag := range errors {
						// Check both Detail and Summary for the error text
						if strings.Contains(diag.Detail(), tt.errorContains) || strings.Contains(diag.Summary(), tt.errorContains) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error to contain %q, but diagnostics were: %v", tt.errorContains, resp.Diagnostics)
					}
				}
			} else {
				errors := resp.Diagnostics.Errors()
				if len(errors) > 0 {
					t.Fatalf("unexpected error: %v", resp.Diagnostics)
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
