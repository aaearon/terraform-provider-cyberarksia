package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	dbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/db/models"
)

// TestDatabaseEngineValidator verifies the validator accepts SDK-defined engines
// and rejects invalid values. Uses representative test cases rather than exhaustive
// testing of all 60+ SDK values, as the validator simply checks list membership.
func TestDatabaseEngineValidator(t *testing.T) {
	tests := []struct {
		name      string
		value     types.String
		expectErr bool
	}{
		// Valid: Representative samples from different categories
		{
			name:      "valid generic engine",
			value:     types.StringValue("postgres"),
			expectErr: false,
		},
		{
			name:      "valid AWS RDS variant",
			value:     types.StringValue("postgres-aws-rds"),
			expectErr: false,
		},
		{
			name:      "valid Azure managed variant",
			value:     types.StringValue("mysql-azure-managed"),
			expectErr: false,
		},
		{
			name:      "valid self-hosted variant",
			value:     types.StringValue("mongo-sh"),
			expectErr: false,
		},

		// Invalid: Common mistakes
		{
			name:      "invalid misspelling (postgresql vs postgres)",
			value:     types.StringValue("postgresql"),
			expectErr: true,
		},
		{
			name:      "invalid case sensitivity (POSTGRES)",
			value:     types.StringValue("POSTGRES"),
			expectErr: true,
		},
		{
			name:      "invalid unknown engine",
			value:     types.StringValue("unknown-database"),
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
			v := DatabaseEngine()
			req := validator.StringRequest{
				Path:        path.Root("database_engine"),
				ConfigValue: tt.value,
			}
			resp := &validator.StringResponse{}

			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tt.expectErr {
				t.Errorf("DatabaseEngine() hasError = %v, expectErr %v", hasError, tt.expectErr)
				if hasError {
					t.Logf("Diagnostics: %v", resp.Diagnostics)
				}
			}
		})
	}
}

func TestDatabaseEngineValidator_Description(t *testing.T) {
	v := DatabaseEngine()
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

// TestDatabaseEngineValidator_SDKCoverage verifies that the ARK SDK provides
// a reasonable number of database engine types (50+).
// This ensures our validator stays in sync with SDK updates.
func TestDatabaseEngineValidator_SDKCoverage(t *testing.T) {
	engineCount := len(dbmodels.DatabaseEngineTypes)
	if engineCount < 50 {
		t.Errorf("Expected at least 50 database engine types, got %d", engineCount)
	}

	t.Logf("ARK SDK provides %d database engine types", engineCount)
}
