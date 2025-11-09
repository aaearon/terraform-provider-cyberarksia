# Data Models: Database Policy Management

**Date**: 2025-10-28
**Feature**: Database Policy Management - Modular Assignment Pattern
**Purpose**: Define Terraform state models for all three resources

---

## 1. Database Policy Model

**File**: `internal/models/database_policy.go`

### State Model

```go
package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// DatabasePolicyModel represents the Terraform state for cyberarksia_database_policy resource
type DatabasePolicyModel struct {
	// Identifying attributes
	ID       types.String `tfsdk:"id"`        // Same as PolicyID (UUID)
	PolicyID types.String `tfsdk:"policy_id"` // UUID from API (computed)

	// Required metadata
	Name                       types.String `tfsdk:"name"`                         // ForceNew
	Status                     types.String `tfsdk:"status"`                       // "Active"|"Suspended"
	DelegationClassification   types.String `tfsdk:"delegation_classification"`    // "Restricted"|"Unrestricted"

	// Optional metadata
	Description types.String `tfsdk:"description"`
	TimeZone    types.String `tfsdk:"time_zone"` // Default: "GMT"

	// Optional time frame
	TimeFrame *TimeFrameModel `tfsdk:"time_frame"`

	// Optional tags
	PolicyTags types.List `tfsdk:"policy_tags"` // []string, max 20

	// Conditions (nested block)
	Conditions *ConditionsModel `tfsdk:"conditions"`

	// Computed metadata
	CreatedBy *ChangeInfoModel `tfsdk:"created_by"`
	UpdatedOn *ChangeInfoModel `tfsdk:"updated_on"`
}

// TimeFrameModel represents policy validity period
type TimeFrameModel struct {
	FromTime types.String `tfsdk:"from_time"` // ISO 8601
	ToTime   types.String `tfsdk:"to_time"`   // ISO 8601
}

// ConditionsModel represents policy access conditions
type ConditionsModel struct {
	MaxSessionDuration types.Int64       `tfsdk:"max_session_duration"` // 1-24 hours
	IdleTime           types.Int64       `tfsdk:"idle_time"`            // 1-120 minutes, default 10
	AccessWindow       *AccessWindowModel `tfsdk:"access_window"`
}

// AccessWindowModel represents time-based access restrictions
type AccessWindowModel struct {
	DaysOfTheWeek types.List   `tfsdk:"days_of_the_week"` // []int, 0=Sunday through 6=Saturday
	FromHour      types.String `tfsdk:"from_hour"`        // "HH:MM"
	ToHour        types.String `tfsdk:"to_hour"`          // "HH:MM"
}

// ChangeInfoModel represents user and timestamp for policy changes
type ChangeInfoModel struct {
	User      types.String `tfsdk:"user"`
	Timestamp types.String `tfsdk:"timestamp"` // ISO 8601
}
```

### Validation Rules

**Field Constraints**:
- `name`: 1-200 characters, unique per tenant, ForceNew
- `status`: "Active" or "Suspended" only (custom validator)
- `delegation_classification`: "Restricted" or "Unrestricted", default "Unrestricted"
- `description`: max 200 characters
- `time_zone`: max 50 characters, default "GMT"
- `policy_tags`: max 20 tags
- `conditions.max_session_duration`: 1-24 hours, required
- `conditions.idle_time`: 1-120 minutes, default 10
- `access_window.days_of_the_week`: 0-6 (integers)
- `access_window.from_hour`: "HH:MM" format, must be before to_hour
- `access_window.to_hour`: "HH:MM" format

**Fixed Values** (not in state model, provider-managed):
- `policy_entitlement.target_category`: Always "DB"
- `policy_entitlement.location_type`: Always "FQDN/IP"
- `policy_entitlement.policy_type`: Always "Recurring"

### SDK Mapping

```go
// Terraform State → ARK SDK
func (m *DatabasePolicyModel) ToSDK() *uapsiadbmodels.ArkUAPSIADBAccessPolicy {
	policy := &uapsiadbmodels.ArkUAPSIADBAccessPolicy{
		ArkUAPSIACommonAccessPolicy: uapcommonmodels.ArkUAPSIACommonAccessPolicy{
			ArkUAPCommonAccessPolicy: uapcommonmodels.ArkUAPCommonAccessPolicy{
				Metadata: uapcommonmodels.ArkUAPMetadata{
					PolicyID:    m.PolicyID.ValueString(),
					Name:        m.Name.ValueString(),
					Description: m.Description.ValueString(),
					Status: uapcommonmodels.ArkUAPPolicyStatus{
						Status: m.Status.ValueString(),
					},
					PolicyEntitlement: uapcommonmodels.ArkUAPPolicyEntitlement{
						TargetCategory: "DB",
						LocationType:   "FQDN/IP",
						PolicyType:     "Recurring",
					},
					TimeZone:   m.TimeZone.ValueString(),
					PolicyTags: convertPolicyTags(m.PolicyTags),
				},
				DelegationClassification: m.DelegationClassification.ValueString(),
			},
			Conditions: convertConditions(m.Conditions),
		},
	}

	if m.TimeFrame != nil {
		policy.Metadata.TimeFrame = uapcommonmodels.ArkUAPTimeFrame{
			FromTime: m.TimeFrame.FromTime.ValueString(),
			ToTime:   m.TimeFrame.ToTime.ValueString(),
		}
	}

	return policy
}

// ARK SDK → Terraform State
func (m *DatabasePolicyModel) FromSDK(policy *uapsiadbmodels.ArkUAPSIADBAccessPolicy) {
	m.ID = types.StringValue(policy.Metadata.PolicyID)
	m.PolicyID = types.StringValue(policy.Metadata.PolicyID)
	m.Name = types.StringValue(policy.Metadata.Name)
	m.Description = types.StringValue(policy.Metadata.Description)
	m.Status = types.StringValue(policy.Metadata.Status.Status)
	m.TimeZone = types.StringValue(policy.Metadata.TimeZone)
	m.DelegationClassification = types.StringValue(policy.DelegationClassification)

	// Convert policy tags
	m.PolicyTags = convertPolicyTagsToTerraform(policy.Metadata.PolicyTags)

	// Convert time frame
	if policy.Metadata.TimeFrame.FromTime != "" || policy.Metadata.TimeFrame.ToTime != "" {
		m.TimeFrame = &TimeFrameModel{
			FromTime: types.StringValue(policy.Metadata.TimeFrame.FromTime),
			ToTime:   types.StringValue(policy.Metadata.TimeFrame.ToTime),
		}
	}

	// Convert conditions
	m.Conditions = convertConditionsFromSDK(&policy.Conditions)

	// Computed fields
	m.CreatedBy = &ChangeInfoModel{
		User:      types.StringValue(policy.Metadata.CreatedBy.User),
		Timestamp: types.StringValue(policy.Metadata.CreatedBy.Timestamp),
	}
	m.UpdatedOn = &ChangeInfoModel{
		User:      types.StringValue(policy.Metadata.UpdatedOn.User),
		Timestamp: types.StringValue(policy.Metadata.UpdatedOn.Timestamp),
	}
}
```

---

## 2. Principal Assignment Model

**File**: `internal/models/policy_principal_assignment.go`

### State Model

```go
package models

import "github.com/hashicorp/terraform-plugin-framework/types"

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

	// Conditional (required for USER/GROUP, not for ROLE)
	SourceDirectoryName types.String `tfsdk:"source_directory_name"` // Max 50 chars
	SourceDirectoryID   types.String `tfsdk:"source_directory_id"`

	// Computed
	LastModified types.String `tfsdk:"last_modified"` // Timestamp of last update
}
```

### Validation Rules

**Field Constraints**:
- `policy_id`: UUID format, ForceNew
- `principal_id`: max 40 characters, ForceNew
- `principal_type`: "USER", "GROUP", or "ROLE" only (custom validator), ForceNew
- `principal_name`: max 512 characters, pattern `^[\w.+\-@#]+$`
- `source_directory_name`: max 50 characters, required for USER/GROUP
- `source_directory_id`: required for USER/GROUP

**Conditional Validation**:
```go
// If principal_type == "USER" or "GROUP"
if principalType == "USER" || principalType == "GROUP" {
	if sourceDirectoryName == "" || sourceDirectoryID == "" {
		return error("source_directory_name and source_directory_id are required for USER and GROUP principal types")
	}
}

// If principal_type == "ROLE"
if principalType == "ROLE" {
	// source_directory fields are optional/ignored
}
```

### SDK Mapping

```go
// Terraform State → ARK SDK Principal
func (m *PolicyPrincipalAssignmentModel) ToSDKPrincipal() uapcommonmodels.ArkUAPPrincipal {
	return uapcommonmodels.ArkUAPPrincipal{
		ID:                  m.PrincipalID.ValueString(),
		Name:                m.PrincipalName.ValueString(),
		Type:                m.PrincipalType.ValueString(),
		SourceDirectoryName: m.SourceDirectoryName.ValueString(),
		SourceDirectoryID:   m.SourceDirectoryID.ValueString(),
	}
}

// ARK SDK Principal → Terraform State
func (m *PolicyPrincipalAssignmentModel) FromSDKPrincipal(policyID string, principal uapcommonmodels.ArkUAPPrincipal) {
	m.ID = types.StringValue(buildCompositeID(policyID, principal.ID, principal.Type))
	m.PolicyID = types.StringValue(policyID)
	m.PrincipalID = types.StringValue(principal.ID)
	m.PrincipalType = types.StringValue(principal.Type)
	m.PrincipalName = types.StringValue(principal.Name)
	m.SourceDirectoryName = types.StringValue(principal.SourceDirectoryName)
	m.SourceDirectoryID = types.StringValue(principal.SourceDirectoryID)
	m.LastModified = types.StringValue(time.Now().Format(time.RFC3339))
}

// Build composite ID
func buildCompositeID(policyID, principalID, principalType string) string {
	return fmt.Sprintf("%s:%s:%s", policyID, principalID, principalType)
}

// Parse composite ID
func parseCompositeID(id string) (policyID, principalID, principalType string, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid composite ID format: expected 'policy-id:principal-id:principal-type', got '%s'", id)
	}

	policyID = parts[0]
	principalID = parts[1]
	principalType = parts[2]

	if policyID == "" || principalID == "" || principalType == "" {
		return "", "", "", fmt.Errorf("composite ID parts cannot be empty")
	}

	validTypes := []string{"USER", "GROUP", "ROLE"}
	if !slices.Contains(validTypes, principalType) {
		return "", "", "", fmt.Errorf("invalid principal type '%s', must be USER, GROUP, or ROLE", principalType)
	}

	return policyID, principalID, principalType, nil
}
```

---

## 3. Database Assignment Model

**File**: `internal/models/policy_workspace_assignment.go`

### State Model (Existing)

**No changes needed** - the existing `PolicyDatabaseAssignmentModel` already has the correct structure:

```go
package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// PolicyDatabaseAssignmentModel represents the Terraform state for cyberarksia_database_policy_assignment resource
type PolicyDatabaseAssignmentModel struct {
	// Composite ID (provider-level)
	ID types.String `tfsdk:"id"` // Format: "policy-id:database-id"

	// Required (ForceNew)
	PolicyID            types.String `tfsdk:"policy_id"`
	DatabaseWorkspaceID types.String `tfsdk:"database_workspace_id"`

	// Required
	AuthenticationMethod types.String `tfsdk:"authentication_method"` // "db_auth"|"ldap_auth"|etc.

	// Authentication profiles (mutually exclusive, one required)
	DBAuthProfile          *DBAuthProfileModel          `tfsdk:"db_auth_profile"`
	LDAPAuthProfile        *LDAPAuthProfileModel        `tfsdk:"ldap_auth_profile"`
	OracleAuthProfile      *OracleAuthProfileModel      `tfsdk:"oracle_auth_profile"`
	MongoAuthProfile       *MongoAuthProfileModel       `tfsdk:"mongo_auth_profile"`
	SQLServerAuthProfile   *SQLServerAuthProfileModel   `tfsdk:"sqlserver_auth_profile"`
	RDSIAMUserAuthProfile  *RDSIAMUserAuthProfileModel  `tfsdk:"rds_iam_user_auth_profile"`

	// Computed
	LastModified types.String `tfsdk:"last_modified"`
}

// Authentication profile models (existing, no changes)
// ... (all 6 profile models remain as-is)
```

### Consistency Updates

**Documentation updates only**:
- Update resource name in docs from `cyberarksia_policy_workspace_assignment` to `cyberarksia_database_policy_assignment`
- Clarify that location_type is always "FQDN/IP" for database policies
- Document relationship with new `cyberarksia_database_policy` resource

**No schema changes required**.

---

## Summary

### Files to Create

1. ✅ **internal/models/database_policy.go** (NEW)
   - DatabasePolicyModel
   - TimeFrameModel
   - ConditionsModel
   - AccessWindowModel
   - ChangeInfoModel
   - ToSDK() and FromSDK() methods

2. ✅ **internal/models/policy_principal_assignment.go** (NEW)
   - PolicyPrincipalAssignmentModel
   - ToSDKPrincipal() and FromSDKPrincipal() methods
   - buildCompositeID() and parseCompositeID() helpers

3. ✅ **internal/models/policy_workspace_assignment.go** (EXISTING)
   - No schema changes
   - Documentation updates only

### Model Relationships

```
┌─────────────────────────────────────┐
│  cyberarksia_database_policy        │
│  (Policy metadata + conditions)     │
│  ID: policy_id (UUID)               │
└─────────────┬───────────────────────┘
              │
              │ Referenced by (policy_id)
              │
     ┌────────┴─────────┐
     │                  │
     ▼                  ▼
┌────────────┐    ┌────────────┐
│ Principal  │    │ Database   │
│ Assignment │    │ Assignment │
│ (3-part ID)│    │ (2-part ID)│
└────────────┘    └────────────┘
```

### State Transitions

**Policy Resource**:
- CREATE: None → Active/Suspended
- UPDATE: Active ↔ Suspended (in-place)
- DELETE: Active/Suspended → None (cascade deletes assignments)

**Assignment Resources**:
- CREATE: None → Assigned
- UPDATE: Assigned → Assigned (modify profile)
- DELETE: Assigned → None (remove from policy)

All state models support full CRUD operations with drift detection.
