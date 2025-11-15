# ARK SDK Integration Reference

**ARK SDK Version**: v1.5.0
**Package**: `github.com/cyberark/ark-sdk-golang`
**Last Updated**: 2025-11-14

**Purpose**: Reference guide for using ARK SDK with **implemented** resources

**Scope**: This document covers all currently implemented resources:
- Database workspaces and secrets (database targets, strong accounts)
- VM secrets and target sets (VM credentials, VM target grouping)
- Database access policies and assignments (UAP)
- Certificates and principals

**Related**: For analysis of available but **not yet implemented** SDK services, see [ark-sdk-sia-services-analysis.md](development/ark-sdk-sia-services-analysis.md)

---

## ⚠️ Breaking Changes (Unreleased)

The following breaking changes are planned for the next release. If you're using examples or references in this document, be aware of these changes:

1. **Resource Rename**: `cyberarksia_secret` → `cyberarksia_database_secret`
   - Rationale: Clarifies that it manages database credentials only (not VM secrets)
   - VM credentials use `cyberarksia_virtual_machine_secret`

2. **Resource Rename**: `cyberarksia_database_policy_database_assignment` → `cyberarksia_database_policy_workspace_assignment`
   - Rationale: Consistency with `cyberarksia_database_workspace` naming

3. **AWS IAM Schema**: New required fields when `authentication_type = "aws_iam"`:
   - `aws_account` (string, 12 digits) - AWS account number
   - `aws_username` (string) - IAM username from ARN

**Migration**: See [CHANGELOG.md](/CHANGELOG.md) for complete migration guide with `terraform state mv` commands.

---

## Table of Contents

- [Overview](#overview)
- [Confirmed SDK Packages](#confirmed-sdk-packages)
- [SIA Database Workspace CRUD Operations](#sia-database-workspace-crud-operations)
- [SIA Database Secrets CRUD Operations](#sia-database-secrets-crud-operations)
- [Authentication Pattern](#authentication-pattern)
- [Data Models](#data-models)
- [Error Handling](#error-handling)

---

## Overview

This document serves as a reference for Phase 3+ development, documenting the confirmed ARK SDK packages, methods, and integration patterns for SIA database and secrets management.

---

## Confirmed SDK Packages

### Authentication
```go
import (
	"github.com/cyberark/ark-sdk-golang/pkg/auth"
	authmodels "github.com/cyberark/ark-sdk-golang/pkg/models/auth"
)
```

### SIA Services
```go
import (
	"github.com/cyberark/ark-sdk-golang/pkg/services/sia"
	dbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/db/models"
	dbsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/db/models"
)
```

---

## SIA Database Workspace CRUD Operations

### Add Database
```go
database, err := siaAPI.WorkspacesDB().AddDatabase(
	&dbmodels.ArkSIADBAddDatabase{
		Name:              "MyDatabase",
		ProviderEngine:    "postgres", // String value, not SDK constant
		ReadWriteEndpoint: "myrds.com",
		SecretID:          secret.SecretID, // From secrets operation
	},
)
```

**Engine Types** (string values):
- `"postgres"`, `"mysql"`, `"mariadb"`, `"oracle"`, `"sqlserver"`, `"mongodb"`, `"db2"`, etc.
- See `internal/validators/database_engine_validator.go` for complete list of 60+ supported engines

### Get Database
```go
database, err := siaAPI.WorkspacesDB().Database(
	&dbmodels.ArkSIADBGetDatabase{
		ID: databaseID,  // int ID from creation
	},
)
```

### Update Database
```go
updated, err := siaAPI.WorkspacesDB().UpdateDatabase(
	&dbmodels.ArkSIADBUpdateDatabase{
		ID:   databaseID,
		Name: "UpdatedName",  // Updated field
		// Include other fields as needed
	},
)
```

**Note**: In practice, use `DeleteDatabaseWorkspaceDirect()` workaround to avoid SDK DELETE panic bug.

### Delete Database
```go
// ⚠️ SDK Bug: Use workaround instead
err := client.DeleteDatabaseWorkspaceDirect(ctx, authContext, databaseID)

// ❌ WRONG - Will panic
// err := siaAPI.WorkspacesDB().DeleteDatabase(&dbmodels.ArkSIADBDeleteDatabase{ID: databaseID})
```

---

## SIA Database Secrets CRUD Operations

### Add Secret (Strong Account)
```go
secret, err := siaAPI.SecretsDB().AddSecret(
	&dbsecretsmodels.ArkSIADBAddSecret{
		SecretType: "username_password", // or other types
		Username:   "db_admin",
		Password:   "SecurePassword123!",
	},
)
// Returns: secret.SecretID for use in database registration
```

**Secret Types** (from SDK `choices` annotation):
- `"username_password"` - Username/password authentication
- `"iam_user"` - AWS IAM user credentials (RDS IAM authentication)
- `"cyberark_pam"` - CyberArk PAM vault account reference
- `"atlas_access_keys"` - MongoDB Atlas programmatic API keys

### Get Secret
```go
secret, err := siaAPI.SecretsDB().Secret(
	&dbsecretsmodels.ArkSIADBGetSecret{
		SecretID: secretID,
	},
)
```

**Note**: Response contains **metadata only**, no sensitive credentials per SIA security model.

### Update Secret (Credential Rotation)
```go
updated, err := siaAPI.SecretsDB().UpdateSecret(
	&dbsecretsmodels.ArkSIADBUpdateSecret{
		SecretID: secretID,
		Password: "NewPassword456!",
		// Include other fields as needed
	},
)
```

**Behavior**: SIA updates credentials immediately. New sessions use updated credentials. Existing session handling is SIA's responsibility.

**Note**: In practice, use `DeleteSecretDirect()` workaround to avoid SDK DELETE panic bug.

### Delete Secret
```go
// ⚠️ SDK Bug: Use workaround instead
err := client.DeleteSecretDirect(ctx, authContext, secretID)

// ❌ WRONG - Will panic
// err := siaAPI.SecretsDB().DeleteSecret(&dbsecretsmodels.ArkSIADBDeleteSecret{SecretID: secretID})
```

---

## SIA VM Secrets CRUD Operations

### Add Secret (VM Credentials)
```go
secret, err := siaAPI.SecretsVM().AddSecret(
	&vmsecretsmodels.ArkSIAVMAddSecret{
		SecretType:           "ProvisionerUser", // or "PCloudAccount"
		SecretName:           "vm-admin-creds",
		ProvisionerUsername:  "admin",
		ProvisionerPassword:  "SecureVMPass123!",
	},
)
// Returns: secret.SecretID for use in target set registration
```

**Secret Types**:
- `"ProvisionerUser"` - Username/password for VM provisioning
- `"PCloudAccount"` - Reference to CyberArk PAM vault account (requires Safe + AccountName)

**For PCloudAccount Type**:
```go
secret, err := siaAPI.SecretsVM().AddSecret(
	&vmsecretsmodels.ArkSIAVMAddSecret{
		SecretType:         "PCloudAccount",
		SecretName:         "vault-linked-account",
		PCloudAccountSafe:  "MySafe",      // PAM vault safe name
		PCloudAccountName:  "MyVaultAccount",  // Account name within safe
	},
)
```

**Note**: In Terraform configuration, these map to `pcloud_safe_name` and `pcloud_account_name` attributes.

### Get Secret
```go
secret, err := siaAPI.SecretsVM().Secret(
	&vmsecretsmodels.ArkSIAVMGetSecret{
		SecretID: secretID,
	},
)
```

**Note**: Response contains **metadata only**, no sensitive credentials per SIA security model.

### Update Secret (Credential Rotation)
```go
// ⚠️ SDK Bug: ChangeSecret uses POST instead of PUT
// Use workaround from sdk_workarounds.go
updated, err := client.ChangeVMSecretDirect(
	ctx,
	providerData.AuthContext,
	&vmsecretsmodels.ArkSIAVMChangeSecret{
		SecretID:            secretID,
		ProvisionerPassword: "NewVMPassword456!",
	},
)
```

**Workaround Rationale**: SDK's `ChangeSecret()` method uses POST (create) instead of PUT (update), causing API errors. The workaround calls the correct PUT endpoint directly.

### Delete Secret
```go
// ⚠️ SDK Bug: DELETE panic applies (nil body pointer)
// Use workaround from sdk_workarounds.go
err := client.DeleteVMSecretDirect(ctx, providerData.AuthContext, secretID)

// ❌ WRONG - Will panic
err := siaAPI.SecretsVM().DeleteSecret(&vmsecretsmodels.ArkSIAVMDeleteSecret{SecretID: secretID})
```

### List Secrets
```go
// Basic listing
secrets, err := siaAPI.SecretsVM().ListSecrets()

// Filtering (⚠️ SDK Bug: API filtering broken, use client-side filtering)
allSecrets, err := siaAPI.SecretsVM().ListSecrets()
filtered := []*vmsecretsmodels.ArkSIAVMSecret{}
for _, s := range allSecrets {
	if s.SecretType == "ProvisionerUser" {
		filtered = append(filtered, s)
	}
}
```

**Note**: `ListSecretsBy()` method exists but does not work due to SDK bug (filter parameters not sent to API). Use client-side filtering as shown above.

---

## SIA Target Sets CRUD Operations

### Add Target Set
```go
targetSet, err := siaAPI.WorkspacesTargetSets().AddTargetSet(
	&targetsetmodels.ArkSIAAddTargetSet{
		Name:       "production-webservers",
		SecretType: "ProvisionerUser",
		SecretID:   secret.SecretID,
		Type:       "Domain",  // or "Suffix" or "Target"
	},
)
```

**Matching Types**:
- `"Domain"` - Match all servers in a DNS domain
- `"Suffix"` - Match servers with specific hostname pattern
- `"Target"` - Match specific target hostname(s)

**Key Field Considerations**:
- `Name` is used as the identifier (string ID, not numeric)
- `Name` is **immutable** - changes require destroy + recreate
- `Type` determines the matching strategy
- `ProvisionFormat` (optional) - Custom provision format string

### Get Target Set
```go
targetSet, err := siaAPI.WorkspacesTargetSets().TargetSet(
	&targetsetmodels.ArkSIAGetTargetSet{
		ID: "production-webservers", // SDK field is called ID (not Name), but holds name value
	},
)
```

### Update Target Set
```go
// ⚠️ SDK Bug: omitempty tags cause incomplete serialization
// Use workaround from sdk_workarounds.go
updated, err := client.UpdateTargetSetDirect(
	ctx,
	providerData.AuthContext,
	oldName,  // Current name of target set
	map[string]interface{}{
		"name":       "production-webservers",
		"type":       "Domain",
		"secret_id":  secretID,
		"secret_type": "ProvisionerUser",
		// Include all fields to avoid omitempty issues
	},
)
```

**Workaround Rationale**: SDK's `UpdateTargetSet()` method has `omitempty` tags that cause fields to be dropped during serialization, leading to incomplete updates. The workaround accepts a map to ensure all fields are sent to the API.

**Note**: Target set name changes require delete + recreate in Terraform (ForceNew behavior).

### Delete Target Set
```go
// ⚠️ SDK Bug: DELETE panic applies (nil body pointer)
// Use workaround from sdk_workarounds.go
err := client.DeleteTargetSetDirect(ctx, providerData.AuthContext, targetSetName)

// ❌ WRONG - Will panic
err := siaAPI.WorkspacesTargetSets().DeleteTargetSet(&targetsetmodels.ArkSIADeleteTargetSet{ID: name})
```

### List Target Sets
```go
// Basic listing
targetSets, err := siaAPI.WorkspacesTargetSets().ListTargetSets()

// Filtering by secret type
targetSets, err := siaAPI.WorkspacesTargetSets().ListTargetSetsBy(
	&targetsetmodels.ArkSIATargetSetsFilter{
		SecretType: "ProvisionerUser",
	},
)
```

**Note**: Target set filtering uses client-side regex matching. The SDK fetches all target sets from the API, then filters locally using Go regex. This is a design choice (not a bug like VM secrets' broken filtering), and works correctly for typical use cases.

---

## Authentication Pattern

### Provider Initialization (Already Implemented in Phase 2)
```go
ispAuth := auth.NewArkISPAuth(true) // Enable caching

profile := &authmodels.ArkAuthProfile{
	Username:   fmt.Sprintf("%s@cyberark.cloud.%s", clientID, tenantSubdomain),
	AuthMethod: authmodels.Identity,
	AuthMethodSettings: &authmodels.IdentityArkAuthMethodSettings{
		IdentityURL:             identityURL,
		IdentityTenantSubdomain: tenantSubdomain,
	},
}

secret := &authmodels.ArkSecret{
	Secret: clientSecret,
}

// Note: First parameter is *ArkProfile (optional, nil for default), NOT context.Context
_, err := ispAuth.Authenticate(nil, profile, secret, false, false)
```

### SIA API Client Initialization (Already Implemented in Phase 2)
```go
siaAPI, err := sia.NewArkSIAAPI(ispAuth)
```

---

## Data Models

### Database Target (Phase 3 - VALIDATED)

**ARK SDK Model**: `ArkSIADBAddDatabase` (Create), `ArkSIADBUpdateDatabase` (Update), `ArkSIADBDatabase` (Response)

**Terraform → SDK Field Mappings**:

| Terraform Attribute | SDK Field | Type | Required? | Notes |
|---------------------|-----------|------|-----------|-------|
| `name` | `Name` | string | ✅ Required | Only truly required field per SDK validate tag |
| `network_name` | `NetworkName` | string | Optional | Network segmentation. Defaults to "ON-PREMISE" |
| `database_type` | `ProviderEngine` | string | Optional | e.g., "postgres", "mysql", "mariadb" |
| `address` | `ReadWriteEndpoint` | string | Optional | Hostname/IP/FQDN |
| `port` | `Port` | int | Optional | SDK uses family defaults (PostgreSQL=5432, etc.) |
| `auth_database` | `AuthDatabase` | string | Optional | MongoDB authentication database (default: "admin") |
| `services` | `Services` | []string | Optional | Oracle/SQL Server service list |
| `account` | `Account` | string | Optional | Snowflake/MongoDB Atlas account name |
| `authentication_method` | `ConfiguredAuthMethodType` | string | Optional | Values: "ad_ephemeral_user", "local_ephemeral_user", "rds_iam_authentication", "atlas_ephemeral_user" |
| `secret_id` | `SecretID` | string | Optional | **Required for ZSP/JIT**. References secret for ephemeral access provisioning |
| `enable_certificate_validation` | `EnableCertificateValidation` | bool | Optional | Enforce TLS cert validation (default: true) |
| `certificate_id` | `Certificate` | string | Optional | Certificate ID for TLS/mTLS. Will reference cyberark_sia_certificate resource |
| `cloud_provider` | `Platform` | string | Optional | Values: "AWS", "AZURE", "GCP", "ON-PREMISE", "ATLAS" |
| `region` | `Region` | string | Optional | **Required for RDS IAM auth**. Used in AWS Signature Version 4 token generation |
| `read_only_endpoint` | `ReadOnlyEndpoint` | string | Optional | Read replica endpoint for scaling reads |
| `description` | (Not in SDK) | string | Optional | Provider-only field (not sent to API) |
| `tags` | `Tags` | map[string]string | Optional | Key-value metadata |

**Removed Fields** (Phase 3 Cleanup - Did Not Exist in SDK):
- ❌ `database_version` - No SDK equivalent
- ❌ `aws_account_id` - No SDK equivalent (generic fields only)
- ❌ `azure_tenant_id` - No SDK equivalent
- ❌ `azure_subscription_id` - No SDK equivalent

**Not Yet Exposed** (Available in SDK, Future Enhancement - Active Directory):
- `Domain` - Windows domain name
- `DomainControllerName` - Domain controller hostname
- `DomainControllerNetbios` - Domain controller NetBIOS name
- `DomainControllerUseLDAPS` - Use LDAPS (default: false)
- `DomainControllerEnableCertValidation` - Enforce DC cert validation (default: true)
- `DomainControllerLDAPSCertificate` - Certificate ID for DC TLS

### Strong Account (Secret)

**ARK SDK Model**: `ArkSIADBAddSecret`

**Confirmed Fields**:
- `SecretType` (string) - Authentication method
- `Username` (string) - Account username
- `Password` (string) - Account password (local/domain auth)

**For AWS IAM** (from SDK `ArkSIADBIAMUserSecretData`):
- `AccessKeyID` (string) - Maps to `access_key_id` in JSON
- `SecretAccessKey` (string) - Maps to `secret_access_key` in JSON
- `Account` (string) - AWS account number (12 digits)
- `Username` (string) - IAM username from ARN
- `Region` (string, optional) - AWS region

---

## Error Handling Patterns

### ARK SDK Error Characteristics
- **No Structured Error Types**: SDK v1.5.0 returns standard Go `error` interface
- **No HTTP Status Codes**: Status codes embedded in error strings only
- **No Error Code Constants**: No SDK-provided error categorization

### Our Error Handling Strategy
1. Use `errors.As()` for standard Go errors (`net.Error`, `context` errors)
2. Pattern match error strings for classification
3. Comprehensive fallback for unknown errors
4. See `internal/client/errors.go` for implementation

---

## Retry and Resilience Patterns

### Retryable Operations
- Network errors (`net.Error` with `Temporary()` or `Timeout()`)
- Server errors (5xx)
- Rate limiting (429)
- Context deadline exceeded

### Non-Retryable Operations
- Authentication failures (401)
- Permission errors (403)
- Not found (404)
- Validation errors (400, 422)
- Context canceled (user requested)

### Implementation
See `internal/client/retry.go` for `RetryWithBackoff()` with exponential backoff.

---

## Policy Assignment Deletion Pattern

### Overview
Database policy assignment resources use a unique deletion pattern that differs from standard resource deletion. Instead of calling a Delete API endpoint, they use **Read-Modify-Write with API-enforced constraints**.

### Why This Pattern?
**CyberArk SIA API Constraint**: Policies must have ≥1 principal AND ≥1 target at all times.

**Problem with Client-Side Validation**:
- Race conditions during concurrent deletes (last-write-wins)
- Blocks valid destroy flows (e.g., policy deletion after assignments)
- Duplicates API's constraint logic

**Solution**: Let the API enforce constraints atomically.

### Implementation Pattern

**Affected Resources**:
- `cyberarksia_database_policy_workspace_assignment`
- `cyberarksia_database_policy_principal_assignment`

**Delete() Method Flow** (⚠️ **PSEUDOCODE** - conceptual illustration, not literal function names):
```go
// NOTE: This is PSEUDOCODE for illustration purposes. The actual implementation
// uses resource-specific logic internal to each assignment resource's Delete() method.
// Helper function names like findAssignmentInPolicy() are conceptual examples, not
// actual exported utilities.

// 1. Fetch current policy state
policy, err := siaAPI.Db().Policy(&models.ArkUAPGetPolicyRequest{PolicyID: policyID})
if err != nil {
    // Handle 404 as success (policy already deleted)
    if isNotFound(err) {
        return // Success
    }
    return err
}

// 2. Find assignment in policy (implementation-specific logic)
found := findAssignmentInPolicy(policy, assignmentID)  // Conceptual helper
if !found {
    return // Already deleted - idempotent success
}

// 3. Remove assignment from local array (implementation-specific logic)
newAssignments := removeAssignment(policy.Assignments, assignmentID)  // Conceptual helper

// 4. Update policy (let API validate ≥1 constraint)
policy.Assignments = newAssignments
_, err = siaAPI.Db().UpdatePolicy(policy)

// 5. Translate API constraint errors to helpful messages
if err != nil {
    if isConstraintError(err) {  // Conceptual helper
        return helpful_error_with_resolution_steps(err)  // Conceptual helper
    }
    return err
}
```

### Error Message Translation

**API Error** (cryptic):
```
failed to update database policy - [400]
[{"code":"INVALID_INPUT","message":"List should have at least 1 item after validation, not 0"}]
```

**Translated Error** (clear, actionable):
```
Error: Cannot Remove Last Target Database

Policy abc123 requires at least one database target assignment.
This error occurs because removing this assignment would leave the
policy with no targets.

To resolve: either delete the policy itself, or add another database
target before removing this one.

API Error: [original error details]
```

### Implementation Examples

**Workspace Assignment**: `internal/provider/database_policy_workspace_assignment_resource.go:560-636`
**Principal Assignment**: `internal/provider/database_policy_principal_assignment_resource.go:301-370`

### Benefits of This Approach

✅ **No race conditions** - API handles atomicity, not read-modify-write with separate locks
✅ **No blocking valid flows** - Terraform dependency graph works correctly
✅ **Simpler code** - ~50 lines removed vs. client-side pre-validation
✅ **Better error messages** - Clear guidance when constraints violated
✅ **Avoids DELETE panic bug** - Uses UpdatePolicy(), not DeletePolicy()

### Related Documentation

- **DELETE Panic Bug**: See [ark-sdk-sia-services-analysis.md](development/ark-sdk-sia-services-analysis.md#priority-1-delete-panic-bug) - Policy assignments avoid this bug by using UpdatePolicy
- **CHANGELOG**: [CHANGELOG.md](/CHANGELOG.md) - v0.3.0 (Unreleased) documents this improvement

---

## Helper Utilities Reference

The provider includes centralized helper utilities to simplify common patterns and reduce code duplication.

### Composite ID Parsing (`internal/provider/helpers/composite_ids.go`)

Policy assignment resources use composite IDs to uniquely identify relationships. Helper functions parse and build these IDs.

**Policy-Database Assignment** (2-part ID: `policy-id:database-id`):
```go
import "github.com/aaearon/terraform-provider-cyberarksia/internal/provider/helpers"

// Parse composite ID
policyID, databaseID, err := helpers.ParsePolicyDatabaseID("abc-123:456")
if err != nil {
    // Handle invalid format
}

// Build composite ID
id := helpers.BuildCompositeID("abc-123", "456")
// Returns: "abc-123:456"
```

**Policy-Principal Assignment** (3-part ID: `policy-id:principal-id:principal-type`):
```go
// Parse composite ID
policyID, principalID, principalType, err := helpers.ParsePolicyPrincipalID("abc-123:user-456:User")
if err != nil {
    // Handle invalid format
}

// Build composite ID
id := helpers.BuildCompositeID("abc-123", "user-456", "User")
// Returns: "abc-123:user-456:User"

// Valid principal types: "User", "Group", "Role"
```

**Usage Pattern in Resources**:
```go
// In Read() method
policyID, dbID, err := helpers.ParsePolicyDatabaseID(data.ID.ValueString())
if err != nil {
    resp.Diagnostics.AddError("Invalid ID Format", err.Error())
    return
}

// In ImportState() method
resource.ID = types.StringValue(helpers.BuildCompositeID(policyID, databaseID))
```

### Database ID Conversion (`internal/provider/helpers/id_conversion.go`)

The ARK SDK returns database IDs as strings in JSON responses but requires int64 in URL paths. Helper functions handle this conversion.

**String to Int**:
```go
import "github.com/aaearon/terraform-provider-cyberarksia/internal/provider/helpers"

// Convert string ID from API response
stringID := "12345"
databaseID, ok := helpers.ConvertDatabaseIDToInt(stringID, &diagnostics, path.Root("id"))
if !ok {
    // Diagnostic error already added
    return
}

// Use in SDK calls
database, err := siaAPI.WorkspacesDB().Database(&dbmodels.ArkSIADBGetDatabase{ID: databaseID})
```

**Note**: `ConvertDatabaseIDToInt` returns `(int, bool)` and adds diagnostic errors automatically for invalid input.

**Why This is Needed**:
- API returns: `{"id": "12345"}` (string)
- SDK methods expect: `GetDatabase(id int64)`
- Conversion handles validation and error cases

### Profile Factory (`internal/provider/profile_factory.go`)

Centralized authentication profile building for all 6 authentication methods. Eliminates 410 LOC duplication across resources.

**Usage Pattern**:
```go
import "github.com/aaearon/terraform-provider-cyberarksia/internal/provider"

// In Create() or Update() methods
var diags diag.Diagnostics
profile := provider.BuildAuthenticationProfile(
    ctx,
    data.AuthenticationMethod.ValueString(),
    data,
    &diags,
)
if diags.HasError() {
    resp.Diagnostics.Append(diags...)
    return
}

// Assign to SDK model
addRequest.Profile = profile
```

**Supported Authentication Methods**:
- `db_auth` - Local database ephemeral users (roles)
- `ldap_auth` - Active Directory ephemeral users (groups, roles)
- `oracle_auth` - Oracle Autonomous Database (profiles, cloud service)
- `mongo_auth` - MongoDB ephemeral users (roles, auth source)
- `sqlserver_auth` - SQL Server Active Directory (groups, roles)
- `rds_iam_user_auth` - AWS RDS IAM authentication

**Why Use Profile Factory**:
- ✅ Centralizes validation logic for all auth methods
- ✅ Prevents auth drift bugs (inconsistent validation)
- ✅ Reduces code duplication (410 LOC → 1 function call)
- ✅ Simplifies adding new auth methods in future

---

## SDK Limitations and Workarounds

### 1. No Context Support in Authenticate()
**Limitation**: `Authenticate()` first parameter is `*ArkProfile` (optional), not `context.Context`

**Workaround**: We accept `context.Context` in our wrapper (`NewISPAuth()`) for future-proofing, but cannot pass to SDK. Document limitation in code comments.

**Impact**: Cannot cancel authentication mid-flight via context.

### 2. No Structured Errors
**Limitation**: All errors returned as generic `error` interface with string messages.

**Workaround**:
- Detect standard Go error types (`net.Error`, context errors)
- Pattern match error strings (case-insensitive, ordered by specificity)
- Comprehensive fallback handling

**Impact**: Error classification may be brittle if SDK error messages change.

### 3. Token Caching Handled by SDK
**Good**: `NewArkISPAuth(true)` enables automatic token caching/refresh

**Provider Responsibility**: Ensure `MaxRetries` and `RequestTimeout` wrap SDK calls, not configure SDK's internal HTTP client

### 4. DELETE Panic Bug (Critical)
**Limitation**: DELETE operations pass `nil` body pointer → panic in `http.NewRequestWithContext()`

**Affected Methods**:
- `WorkspacesDB().DeleteDatabase()`
- `SecretsDB().DeleteSecret()`
- `SecretsVM().DeleteSecret()`
- `WorkspacesTargetSets().DeleteTargetSet()`
- Potentially `Db().DeletePolicy()` (if uses BaseDeletePolicy())

**Root Cause**: SDK file `pkg/common/ark_client.go:556-576` - `doRequest()` doesn't handle nil body pointer correctly

**Workaround**: Use direct API calls from `internal/client/sdk_workarounds.go`:
```go
// Database workspaces
err := client.DeleteDatabaseWorkspaceDirect(ctx, authContext, databaseID)

// Database secrets
err := client.DeleteSecretDirect(ctx, authContext, secretID)

// VM secrets
err := client.DeleteVMSecretDirect(ctx, authContext, secretID)

// Target sets
err := client.DeleteTargetSetDirect(ctx, authContext, targetSetName)

// Database policies
err := client.DeleteDatabasePolicyDirect(ctx, authContext, policyID)
```

**Long-term**: Remove workarounds when ARK SDK v1.6.0+ fixes nil body handling

### 5. VM Secrets Filtering Bug
**Limitation**: `ListSecretsBy()` builds filter correctly but passes `nil` to API instead of filter JSON

**Location**: `pkg/services/sia/secrets/vm/ark_sia_secrets_vm_service.go:191-204`

**Impact**: Filtering by secret type, name, or other criteria does not work - returns unfiltered results

**Workaround**: Client-side filtering:
```go
allSecrets, err := siaAPI.SecretsVM().ListSecrets()
filtered := []*vmsecretsmodels.ArkSIAVMSecret{}
for _, s := range allSecrets {
	if s.SecretType == desiredType {
		filtered = append(filtered, s)
	}
}
```

### 6. VM Secrets Update Bug (ChangeSecret)
**Limitation**: `ChangeSecret()` method uses POST instead of PUT, causing API errors

**Workaround**: Use `ChangeVMSecretDirect()` from sdk_workarounds.go:
```go
updated, err := client.ChangeVMSecretDirect(ctx, authContext, changeRequest)
```

**Rationale**: Calls correct PUT endpoint directly

### 7. Target Sets Update Bug (omitempty Serialization)
**Limitation**: `UpdateTargetSet()` method has `omitempty` tags causing fields to be dropped during JSON marshaling

**Impact**: Partial updates fail because required fields are not sent to API

**Workaround**: Use `UpdateTargetSetDirect()` from sdk_workarounds.go:
```go
updated, err := client.UpdateTargetSetDirect(ctx, authContext, updateRequest)
```

**Rationale**: Ensures all fields are serialized and sent to API, bypassing omitempty tags

### 8. Workarounds Reference Summary

All SDK bug workarounds are centralized in `internal/client/sdk_workarounds.go`:

| Workaround Function | Replaces SDK Method | Bug Type | Status |
|---------------------|---------------------|----------|--------|
| `DeleteDatabaseWorkspaceDirect()` | `WorkspacesDB().DeleteDatabase()` | DELETE panic | ✅ Implemented |
| `DeleteSecretDirect()` | `SecretsDB().DeleteSecret()` | DELETE panic | ✅ Implemented |
| `DeleteVMSecretDirect()` | `SecretsVM().DeleteSecret()` | DELETE panic | ✅ Implemented |
| `ChangeVMSecretDirect()` | `SecretsVM().ChangeSecret()` | POST vs PUT | ✅ Implemented |
| `DeleteTargetSetDirect()` | `WorkspacesTargetSets().DeleteTargetSet()` | DELETE panic | ✅ Implemented |
| `UpdateTargetSetDirect()` | `WorkspacesTargetSets().UpdateTargetSet()` | omitempty serialization | ✅ Implemented |
| `DeleteDatabasePolicyDirect()` | `Db().DeletePolicy()` | DELETE panic | ✅ Implemented |

**Note**: Policy assignments avoid DELETE bug by using `UpdatePolicy()` with Read-Modify-Write pattern instead of calling DeletePolicy(). See [Policy Assignment Deletion Pattern](#policy-assignment-deletion-pattern) section.

---

## Testing Strategy

### Acceptance Tests (Primary)
- Test against real SIA API when `TF_ACC=1`
- Use test credentials from environment variables
- Verify CRUD operations end-to-end

### Unit Tests (Selective)
- Complex validators only
- Error classification logic (already tested in `errors_test.go`)
- Retry logic (already tested in `retry_test.go`)

---

## References

- **ARK SDK GitHub**: https://github.com/cyberark/ark-sdk-golang
- **ARK SDK Docs**: https://cyberark.github.io/ark-sdk-golang/
- **Context7 Documentation**: Used for SDK research (see Phase 2 reflection)
- **Terraform Plugin Framework**: https://developer.hashicorp.com/terraform/plugin/framework

---

## Version History

- **2025-10-15 (Phase 2.5)**: Initial version based on Phase 2 research and Context7 examples
- **2025-10-15 (Phase 3 Cleanup)**: Validated database workspace field mappings against ARK SDK v1.5.0, removed non-existent fields, documented actual SDK requirements
- **2025-11-14**:
  - Added Policy Assignment Deletion Pattern section documenting simplified approach (let API validate constraints, translate errors to helpful messages)
  - Added VM Secrets CRUD Operations section with workarounds for DELETE panic, filtering bug, and ChangeSecret POST→PUT bug
  - Added Target Sets CRUD Operations section with workarounds for DELETE panic and omitempty serialization bug
  - Expanded SDK Limitations section with comprehensive workarounds reference (8 SDK bugs documented)
  - Updated scope to reflect VM resources implementation
  - Fixed invalid SDK constants examples (ProviderEngine uses strings, not constants)
