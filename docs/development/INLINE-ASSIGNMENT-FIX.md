# Inline Assignment Implementation Fix

**Date**: 2025-10-28
**Status**: Implementation 95% complete, model fix required
**Issue**: Model mismatch causing Terraform validation error

## Problem Summary

The inline assignment feature for `cyberarksia_database_policy` resource was implemented but has a model mismatch error:

```
Error: Value Conversion Error
mismatch between struct and object: Struct defines fields not found in object: id, last_modified, and policy_id.
```

**Root Cause**: The implementation reuses `PolicyDatabaseAssignmentModel` and `PolicyPrincipalAssignmentModel` from the assignment resources. These models have extra Terraform resource management fields (`id`, `last_modified`, `policy_id`) that don't exist in the policy resource's inline schema blocks.

## What Was Completed

### 1. Schema Implementation ✅
- **File**: `internal/provider/database_policy_resource.go` (lines 127-272)
- Added `target_databases` ListNestedBlock with all 6 authentication methods
- Added `principals` ListNestedBlock with all principal fields
- Schema is correct and compiles

### 2. Validation ✅
- **File**: `internal/provider/database_policy_resource.go` (lines 390-481)
- `ValidateConfig` method enforces at least 1 database + 1 principal
- Authentication method profile validation

### 3. CRUD Logic ✅
- **Create**: Lines 486-602 - Builds policy with inline assignments
- **Update**: Lines 648-773 - Updates policy with inline assignments
- **Read**: Needs to populate inline blocks from API response
- Helper: `buildInstanceTarget()` (lines 857-979) - Converts inline blocks to SDK structs

### 4. Test Configuration ✅
- **File**: `/tmp/sia-crud-validation-20251028-133244/main.tf`
- Demonstrates hybrid pattern:
  - Service account: Inline in policy (meets API requirement)
  - Test user: Via separate assignment resource
  - Database: Inline in policy
- Includes `lifecycle { ignore_changes = [principals] }` for hybrid pattern

### 5. Documentation ✅
- **File**: `/tmp/sia-crud-validation-20251028-133244/INLINE-ASSIGNMENTS-README.md`
- Explains both patterns (inline vs hybrid with assignment resources)
- Complete usage examples

### 6. Provider Binary ✅
- Builds successfully: `go build -v`
- Binary location: `~/terraform-provider-cyberarksia/terraform-provider-cyberarksia`
- **Must copy manually** to:
  - `~/.terraform.d/plugins/terraform.local/local/cyberarksia/0.1.0/linux_amd64/terraform-provider-cyberarksia_v0.1.0`
  - `/tmp/sia-crud-validation-20251028-133244/.terraform/providers/terraform.local/local/cyberarksia/0.1.0/linux_amd64/terraform-provider-cyberarksia`

## Required Fix: Create Simplified Inline Models

### Current Problem (database_policy.go:40-41)

```go
type DatabasePolicyModel struct {
    // ... other fields ...

    // PROBLEM: These reuse assignment resource models with extra fields
    TargetDatabases []PolicyDatabaseAssignmentModel  `tfsdk:"target_databases"`
    Principals      []PolicyPrincipalAssignmentModel `tfsdk:"principals"`
}
```

`PolicyDatabaseAssignmentModel` has:
- ✅ `database_workspace_id` - needed
- ✅ `authentication_method` - needed
- ✅ `db_auth_profile` - needed
- ❌ `id` - **NOT in schema, causes error**
- ❌ `policy_id` - **NOT in schema, causes error**
- ❌ `last_modified` - **NOT in schema, causes error**

### Solution: Add New Inline-Only Models

**File to modify**: `internal/models/database_policy.go`

Add after line 73 (after `ChangeInfoModel`):

```go
// ============================================================================
// INLINE ASSIGNMENT MODELS (for policy resource inline blocks)
// ============================================================================
// These models match the schema blocks in the policy resource and map directly
// to SDK structs without Terraform resource management fields.
// ============================================================================

// InlineDatabaseAssignmentModel represents a database assignment within a policy resource
// Maps to: ArkUAPSIADBInstanceTarget (SDK)
type InlineDatabaseAssignmentModel struct {
	DatabaseWorkspaceID  types.String `tfsdk:"database_workspace_id"`
	AuthenticationMethod types.String `tfsdk:"authentication_method"`

	// Authentication profiles (one will be set based on authentication_method)
	// Reuse existing profile models from policy_workspace_assignment.go
	DBAuthProfile         *DBAuthProfileModel         `tfsdk:"db_auth_profile"`
	LDAPAuthProfile       *LDAPAuthProfileModel       `tfsdk:"ldap_auth_profile"`
	OracleAuthProfile     *OracleAuthProfileModel     `tfsdk:"oracle_auth_profile"`
	MongoAuthProfile      *MongoAuthProfileModel      `tfsdk:"mongo_auth_profile"`
	SQLServerAuthProfile  *SQLServerAuthProfileModel  `tfsdk:"sqlserver_auth_profile"`
	RDSIAMUserAuthProfile *RDSIAMUserAuthProfileModel `tfsdk:"rds_iam_user_auth_profile"`
}

// InlinePrincipalModel represents a principal assignment within a policy resource
// Maps to: ArkUAPPrincipal (SDK)
type InlinePrincipalModel struct {
	PrincipalID         types.String `tfsdk:"principal_id"`
	PrincipalType       types.String `tfsdk:"principal_type"`
	PrincipalName       types.String `tfsdk:"principal_name"`
	SourceDirectoryName types.String `tfsdk:"source_directory_name"`
	SourceDirectoryID   types.String `tfsdk:"source_directory_id"`
}
```

**Then update DatabasePolicyModel (lines 40-41)**:

```go
type DatabasePolicyModel struct {
    // ... existing fields ...

    // Inline assignments (optional at schema level, required via ValidateConfig)
    TargetDatabases []InlineDatabaseAssignmentModel `tfsdk:"target_databases"` // CHANGED
    Principals      []InlinePrincipalModel          `tfsdk:"principals"`       // CHANGED

    // ... rest of model ...
}
```

### Update database_policy_resource.go

The `buildInstanceTarget()` helper (lines 857-979) needs to accept `InlineDatabaseAssignmentModel` instead of the full assignment model. The logic is already correct, just update the parameter type.

**No other changes needed** - the Create/Update logic already converts these to SDK structs correctly.

## SDK Structure Reference

The inline models map to these SDK structures:

**Target Database** → `ArkUAPSIADBInstanceTarget`:
```go
type ArkUAPSIADBInstanceTarget struct {
    InstanceName         string  // Fetched from database workspace
    InstanceType         string  // Fetched from database workspace
    InstanceID           string  // = DatabaseWorkspaceID
    AuthenticationMethod string  // = AuthenticationMethod

    // One profile based on authentication method
    DBAuthProfile         *ArkUAPSIADBDBAuthProfile
    LDAPAuthProfile       *ArkUAPSIADBLDAPAuthProfile
    // ... etc
}
```

**Principal** → `ArkUAPPrincipal`:
```go
type ArkUAPPrincipal struct {
    ID                  string  // = PrincipalID
    Name                string  // = PrincipalName
    Type                string  // = PrincipalType
    SourceDirectoryName string  // = SourceDirectoryName (optional)
    SourceDirectoryID   string  // = SourceDirectoryID (optional)
}
```

## Testing After Fix

### 1. Rebuild Provider

```bash
cd ~/terraform-provider-cyberarksia
go build -v

# Copy to both locations
cp terraform-provider-cyberarksia ~/.terraform.d/plugins/terraform.local/local/cyberarksia/0.1.0/linux_amd64/terraform-provider-cyberarksia_v0.1.0

cp terraform-provider-cyberarksia /tmp/sia-crud-validation-20251028-133244/.terraform/providers/terraform.local/local/cyberarksia/0.1.0/linux_amd64/terraform-provider-cyberarksia
```

### 2. Test Configuration

```bash
cd /tmp/sia-crud-validation-20251028-133244
rm -f .terraform.lock.hcl
terraform init
terraform plan
```

**Expected**: Plan should succeed with no schema errors

### 3. Full CRUD Test

```bash
# CREATE - Policy with inline service account + database
# PLUS separate assignment resource for test user
terraform apply -target=cyberarksia_database_policy.test_policy -auto-approve
terraform apply -target=cyberarksia_database_policy_principal_assignment.test_user -auto-approve

# READ - Verify no drift
terraform plan  # Should show: No changes

# UPDATE - Modify inline assignments
# Edit main.tf to change roles or add principal
terraform apply

# DELETE
terraform destroy -auto-approve
```

## Key Design Points

### Why Separate Models?

1. **Assignment Resource Models** (`PolicyDatabaseAssignmentModel`):
   - Have resource management fields: `id`, `policy_id`, `last_modified`
   - Used by: `cyberarksia_database_policy_workspace_assignment` resource
   - Purpose: Manage individual assignments as separate resources

2. **Inline Models** (`InlineDatabaseAssignmentModel`):
   - Only have API data fields: `database_workspace_id`, `authentication_method`, profiles
   - Used by: `cyberarksia_database_policy` resource inline blocks
   - Purpose: Embed assignments directly in policy resource

Both map to the same SDK structs, but inline models don't need Terraform resource IDs.

### API Behavior

The SIA API uses PUT for entire policies:
- **No separate endpoints** for adding/removing principals or targets
- **Both patterns** (inline and assignment resources) do:
  1. GET policy (fetch full state)
  2. Modify Principals or Targets arrays
  3. PUT policy (replace entire policy)

The difference is **who manages the state**:
- **Inline blocks**: Policy resource owns assignments, full drift detection
- **Assignment resources**: Separate resources own assignments, use `lifecycle { ignore_changes }` on policy

### Hybrid Pattern

The test configuration demonstrates the hybrid pattern:
- Minimal initial assignments inline (meets API requirement)
- Additional assignments via assignment resources
- `lifecycle { ignore_changes = [principals] }` prevents drift detection

This allows:
- ✅ Meeting API requirement (at least 1 database + 1 principal)
- ✅ Flexible distributed management via assignment resources
- ✅ No conflicts between policy resource and assignment resources

## Files to Modify

1. ✅ `internal/models/database_policy.go` - Add inline models (after line 73)
2. ✅ `internal/models/database_policy.go` - Update `DatabasePolicyModel` fields (lines 40-41)
3. ⚠️ `internal/provider/database_policy_resource.go` - Update `buildInstanceTarget()` parameter type

## Validation Checklist

After implementing the fix:

- [ ] `go build -v` - succeeds
- [ ] Copy provider binary to both locations
- [ ] `terraform init` - succeeds
- [ ] `terraform plan` - NO schema errors
- [ ] `terraform apply` - policy created with inline assignments
- [ ] `terraform plan` - No changes detected
- [ ] Policy visible in SIA UI with correct principals and targets
- [ ] Assignment resource can add additional principals (hybrid pattern)
- [ ] `terraform destroy` - cleanup successful

## References

- **ARK SDK**: `github.com/cyberark/ark-sdk-golang@v1.5.0`
- **SDK Structs**:
  - `pkg/services/uap/sia/db/models/ArkUAPSIADBInstanceTarget`
  - `pkg/services/uap/common/models/ArkUAPPrincipal`
- **Test Config**: `/tmp/sia-crud-validation-20251028-133244/`
- **Documentation**: `/tmp/sia-crud-validation-20251028-133244/INLINE-ASSIGNMENTS-README.md`
