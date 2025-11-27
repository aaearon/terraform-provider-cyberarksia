// Package models defines Terraform state models
package models

import (
	"fmt"
	"strings"
	"time"

	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/exp/slices"
)

// PolicyPrincipalAssignmentModel represents the Terraform state for cyberarksia_database_policy_principal_assignment resource
type PolicyPrincipalAssignmentModel struct {
	// Composite ID (provider-level)
	ID types.String `tfsdk:"id"` // Format: "policy-id:principal-id:principal-type"

	// Required (ForceNew)
	PolicyID      types.String `tfsdk:"policy_id"`      // Policy to assign principal to
	PrincipalID   types.String `tfsdk:"principal_id"`   // Max 40 chars
	PrincipalType types.String `tfsdk:"principal_type"` // "USER"|"GROUP"|"ROLE"

	// Required
	PrincipalName types.String `tfsdk:"principal_name"` // Max 512 chars

	// Conditional (required for USER/GROUP, optional for ROLE)
	SourceDirectoryName types.String `tfsdk:"source_directory_name"` // Max 50 chars
	SourceDirectoryID   types.String `tfsdk:"source_directory_id"`

	// Computed
	LastModified types.String `tfsdk:"last_modified"` // Timestamp of last update
}

// ToSDKPrincipal converts Terraform state model to ARK SDK principal struct
func (m *PolicyPrincipalAssignmentModel) ToSDKPrincipal() uapcommonmodels.ArkUAPPrincipal {
	return uapcommonmodels.ArkUAPPrincipal{
		ID:                  m.PrincipalID.ValueString(),
		Name:                m.PrincipalName.ValueString(),
		Type:                m.PrincipalType.ValueString(),
		SourceDirectoryName: m.SourceDirectoryName.ValueString(),
		SourceDirectoryID:   m.SourceDirectoryID.ValueString(),
	}
}

// FromSDKPrincipal populates Terraform state model from ARK SDK principal struct
func (m *PolicyPrincipalAssignmentModel) FromSDKPrincipal(policyID string, principal uapcommonmodels.ArkUAPPrincipal) {
	m.ID = types.StringValue(BuildCompositeID(policyID, principal.ID, principal.Type))
	m.PolicyID = types.StringValue(policyID)
	m.PrincipalID = types.StringValue(principal.ID)
	m.PrincipalType = types.StringValue(principal.Type)
	m.PrincipalName = types.StringValue(principal.Name)
	// Use null for empty strings - API returns null for ROLE principals which Go unmarshals to ""
	m.SourceDirectoryName = stringValueOrNull(principal.SourceDirectoryName)
	m.SourceDirectoryID = stringValueOrNull(principal.SourceDirectoryID)
	m.LastModified = types.StringValue(time.Now().Format(time.RFC3339))
}

// BuildCompositeID builds a 3-part composite ID for principal assignments
// Format: "policy-id:principal-id:principal-type"
// Rationale: Principal IDs can duplicate across types (e.g., user "admin", role "admin")
func BuildCompositeID(policyID, principalID, principalType string) string {
	return fmt.Sprintf("%s:%s:%s", policyID, principalID, principalType)
}

// ParseCompositeID parses a 3-part composite ID into its components
// Returns policyID, principalID, principalType, and error if invalid
func ParseCompositeID(id string) (policyID, principalID, principalType string, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid composite ID format: expected 'policy-id:principal-id:principal-type', got '%s'", id)
	}

	policyID = parts[0]
	principalID = parts[1]
	principalType = parts[2]

	// Validate non-empty
	if policyID == "" || principalID == "" || principalType == "" {
		return "", "", "", fmt.Errorf("composite ID parts cannot be empty")
	}

	// Validate principal type
	validTypes := []string{"USER", "GROUP", "ROLE"}
	if !slices.Contains(validTypes, principalType) {
		return "", "", "", fmt.Errorf("invalid principal type '%s', must be USER, GROUP, or ROLE", principalType)
	}

	return policyID, principalID, principalType, nil
}
