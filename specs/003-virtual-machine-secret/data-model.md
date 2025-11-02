# Data Model: Virtual Machine Secret Management

**Feature**: Virtual Machine Secret Management
**Branch**: `003-virtual-machine-secret`
**Date**: 2025-11-02
**Source**: [spec.md](./spec.md) | [plan.md](./plan.md)

## Overview

This document defines the data model for VM Secret resources in the Terraform provider. The model supports two credential types: ProvisionerUser (self-contained username/password) and PCloudAccount (references to PAM vault accounts).

## Primary Entity: VirtualMachineSecret

### Entity Definition

A `VirtualMachineSecret` represents a credential for VM/server access managed by CyberArk SIA. Secrets are uniquely identified by a UUID (`secret_id`) and support two distinct authentication patterns.

### Core Attributes

| Attribute | Type | Required | Computed | Mutable | Description |
|-----------|------|----------|----------|---------|-------------|
| `id` | string | No | Yes | No | Terraform resource identifier (equals secret_id) |
| `secret_id` | string | No | Yes | No | SIA-assigned UUID, immutable unique identifier |
| `secret_name` | string | Yes | No | Yes | User-facing label for the secret |
| `secret_type` | string | Yes | No | No | Credential type: "ProvisionerUser" or "PCloudAccount" (ForceNew) |

### Conditional Attributes: ProvisionerUser

Required only when `secret_type = "ProvisionerUser"`:

| Attribute | Type | Required | Sensitive | Description |
|-----------|------|----------|-----------|-------------|
| `provisioner_username` | string | Yes* | No | Username for VM provisioning |
| `provisioner_password` | string | Yes* | Yes | Password for VM provisioning (write-only) |

*Required if and only if `secret_type == "ProvisionerUser"`

### Conditional Attributes: PCloudAccount

Required only when `secret_type = "PCloudAccount"`:

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `pcloud_safe_name` | string | Yes* | PAM vault safe name containing the account |
| `pcloud_account_name` | string | Yes* | PAM account name within the safe |

*Required if and only if `secret_type == "PCloudAccount"`

## Attribute Details

### secret_id (Computed, Immutable)

- **Type**: UUID string (e.g., `"abc-123-def-456"`)
- **Source**: Returned by SIA API on creation
- **Usage**: Unique identifier for all read/update/delete operations
- **Lifecycle**: Assigned on creation, never changes, deleted with resource
- **Terraform**: Used as both `id` and `secret_id` attributes

### secret_name (Required, Mutable)

- **Type**: String
- **Constraints**:
  - Max length: 200 characters
  - Non-empty
  - Duplicates allowed (secret_id is unique identifier)
- **Lifecycle**: Can be updated in-place without ForceNew
- **Use Case**: Human-readable label for identification

### secret_type (Required, Immutable)

- **Type**: Enum string
- **Valid Values**:
  - `"ProvisionerUser"`: Self-contained username/password stored in SIA
  - `"PCloudAccount"`: Reference to existing PAM vault account
- **Constraints**:
  - Must be one of the two valid values
  - Cannot be changed after creation (ForceNew required)
- **Validation**: Schema-level enum validator
- **Plan Modifier**: `stringplanmodifier.RequiresReplace()`

### provisioner_username (Conditional)

- **Type**: String
- **Applicability**: ProvisionerUser secrets only
- **Constraints**:
  - Required when `secret_type == "ProvisionerUser"`
  - Forbidden when `secret_type == "PCloudAccount"`
  - Non-empty
- **Lifecycle**: Can be updated in-place
- **Validation**: Custom validator checks secret_type context

### provisioner_password (Conditional, Sensitive)

- **Type**: String (sensitive)
- **Applicability**: ProvisionerUser secrets only
- **Constraints**:
  - Required when `secret_type == "ProvisionerUser"`
  - Forbidden when `secret_type == "PCloudAccount"`
  - Min length: 8 characters (assumed standard policy)
  - Non-empty
- **Security**:
  - Marked `Sensitive: true` in schema
  - Never returned by API (write-only)
  - Not stored in read responses
- **Lifecycle**: Can be updated (triggers password rotation)
- **State**: Present in state as `<sensitive>` placeholder

### pcloud_safe_name (Conditional)

- **Type**: String
- **Applicability**: PCloudAccount secrets only
- **Constraints**:
  - Required when `secret_type == "PCloudAccount"`
  - Forbidden when `secret_type == "ProvisionerUser"`
  - Must reference existing PAM safe
- **Validation**:
  - Schema-level: non-empty
  - Runtime: API validates safe exists and user has access
- **Lifecycle**: Can be updated to reference different safe

### pcloud_account_name (Conditional)

- **Type**: String
- **Applicability**: PCloudAccount secrets only
- **Constraints**:
  - Required when `secret_type == "PCloudAccount"`
  - Forbidden when `secret_type == "ProvisionerUser"`
  - Must reference existing account in specified safe
- **Validation**:
  - Schema-level: non-empty
  - Runtime: API validates account exists in safe
- **Lifecycle**: Can be updated to reference different account

## Validation Rules

### Schema-Level Validation

1. **secret_type Enum**:
   ```go
   stringvalidator.OneOf("ProvisionerUser", "PCloudAccount")
   ```

2. **secret_name Length**:
   ```go
   stringvalidator.LengthBetween(1, 200)
   ```

3. **provisioner_password Length** (when present):
   ```go
   stringvalidator.LengthAtLeast(8)
   ```

### Cross-Attribute Validation

Implemented in `ValidateConfig()` method:

```go
// ProvisionerUser type requirements
if secret_type == "ProvisionerUser" {
    require: provisioner_username (non-null, non-empty)
    require: provisioner_password (non-null, non-empty)
    forbid:  pcloud_safe_name
    forbid:  pcloud_account_name
}

// PCloudAccount type requirements
if secret_type == "PCloudAccount" {
    require: pcloud_safe_name (non-null, non-empty)
    require: pcloud_account_name (non-null, non-empty)
    forbid:  provisioner_username
    forbid:  provisioner_password
}
```

### API-Level Validation

Performed by SIA API, surfaced as errors:

- PAM safe exists and user has read permission
- PAM account exists in specified safe
- secret_name format acceptable
- Password complexity requirements (if any)

## State Lifecycle

### Create Flow

1. **Terraform Config** → Validate schema and cross-attributes
2. **API Request** → `AddSecret()` with appropriate fields per type
3. **API Response** → Returns `secret_id` + echoed fields (no password)
4. **State Storage** → Store all attributes including `secret_id`

**State After Create**:
```hcl
# ProvisionerUser
id                   = "abc-123-def"
secret_id            = "abc-123-def"
secret_name          = "app-server-admin"
secret_type          = "ProvisionerUser"
provisioner_username = "admin"
provisioner_password = (sensitive value)

# PCloudAccount
id                  = "xyz-789-ghi"
secret_id           = "xyz-789-ghi"
secret_name         = "vault-db-admin"
secret_type         = "PCloudAccount"
pcloud_safe_name    = "Production-Safe"
pcloud_account_name = "db-admin-account"
```

### Read/Refresh Flow

1. **API Request** → `Secret(secret_id)` by UUID
2. **API Response** → Returns metadata, **NO passwords** (write-only)
3. **State Comparison** → Compare with current state:
   - Detect name changes (drift)
   - Passwords never change state (not returned)
   - Username changes detected
   - PAM references detected
4. **Drift Detection** → Propose updates if differences found

**Special Cases**:
- **404 Response**: Secret deleted outside Terraform → Mark for recreation
- **Password in state**: Never updated by Read (write-only field)

### Update Flow

1. **Config Change** → User modifies mutable fields (name, username, password, PAM refs)
2. **Plan** → Detect changes, check ForceNew requirements:
   - `secret_name` change: In-place update
   - `provisioner_username` change: In-place update
   - `provisioner_password` change: In-place update (password rotation)
   - `pcloud_*` change: In-place update
   - `secret_type` change: **ForceNew** (destroy + recreate)
3. **API Request** → `ChangeSecret()` with changed fields only
4. **State Update** → Store new values

### Delete Flow

1. **Config Removal** → Resource removed from configuration
2. **API Request** → `DELETE /api/v1/sia/secrets/vm/{secret_id}`
   - **Note**: DELETE uses workaround, not SDK method (avoids panic bug)
3. **State Removal** → Resource removed from Terraform state

**Idempotency**: 404 errors treated as success (already deleted)

### Import Flow

1. **Import Command** → `terraform import resource_name secret_id`
2. **API Request** → `Secret(secret_id)` by UUID
3. **State Population** → Store retrieved metadata
4. **Password Handling** → Passwords not present (API doesn't return them)
5. **User Action Required** → User must add passwords to config manually

**Post-Import State**:
```hcl
# Imported ProvisionerUser (password absent)
id                   = "abc-123-def"
secret_id            = "abc-123-def"
secret_name          = "app-server-admin"
secret_type          = "ProvisionerUser"
provisioner_username = "admin"
provisioner_password = (not in state - user must add to config)
```

## SDK Mapping

### Terraform Attribute → SDK Field

| Terraform Attribute | SDK Field (Go) | SDK Field (JSON) | Notes |
|---------------------|----------------|------------------|-------|
| `id` | - | - | Internal to Terraform, equals secret_id |
| `secret_id` | `SecretID` | `secret_id` | UUID from API |
| `secret_name` | `SecretName` | `secret_name` | User label |
| `secret_type` | `SecretType` | `secret_type` | "ProvisionerUser" or "PCloudAccount" |
| `provisioner_username` | `ProvisionerUsername` | `provisioner_username` | Optional, for ProvisionerUser |
| `provisioner_password` | `ProvisionerPassword` | `provisioner_password` | Optional, sensitive, write-only |
| `pcloud_safe_name` | `PCloudAccountSafe` | `pcloud_account_safe` | Optional, for PCloudAccount |
| `pcloud_account_name` | `PCloudAccountName` | `pcloud_account_name` | Optional, for PCloudAccount |

### SDK Structures

**Create Request** (`ArkSIAVMAddSecret`):
```go
type ArkSIAVMAddSecret struct {
    SecretName           string `json:"secret_name"`
    SecretType           string `json:"secret_type"`
    ProvisionerUsername  string `json:"provisioner_username,omitempty"`
    ProvisionerPassword  string `json:"provisioner_password,omitempty"`
    PCloudAccountSafe    string `json:"pcloud_account_safe,omitempty"`
    PCloudAccountName    string `json:"pcloud_account_name,omitempty"`
}
```

**Read Response** (`ArkSIAVMSecret`):
```go
type ArkSIAVMSecret struct {
    SecretID             string `json:"secret_id"`
    SecretName           string `json:"secret_name"`
    SecretType           string `json:"secret_type"`
    ProvisionerUsername  string `json:"provisioner_username,omitempty"`
    // No ProvisionerPassword field (write-only)
    PCloudAccountSafe    string `json:"pcloud_account_safe,omitempty"`
    PCloudAccountName    string `json:"pcloud_account_name,omitempty"`
}
```

**Update Request** (`ArkSIAVMChangeSecret`):
```go
type ArkSIAVMChangeSecret struct {
    SecretID             string `json:"secret_id"`
    SecretName           string `json:"secret_name,omitempty"`
    ProvisionerUsername  string `json:"provisioner_username,omitempty"`
    ProvisionerPassword  string `json:"provisioner_password,omitempty"`
    PCloudAccountSafe    string `json:"pcloud_account_safe,omitempty"`
    PCloudAccountName    string `json:"pcloud_account_name,omitempty"`
}
```

## Relationships

### Target Sets (Future)

VM Secrets will be referenced by **Target Sets** (VM workspaces) in a future feature:

```hcl
resource "cyberarksia_target_set" "web_servers" {
  name       = "production-web-servers"
  secret_id  = cyberarksia_virtual_machine_secret.app_server.secret_id
  secret_type = "ProvisionerUser"
}
```

**Relationship**: One-to-many (one secret can be referenced by multiple target sets)

**Dependency Direction**: Target Sets depend on VM Secrets (not the other way)

**Deletion Behavior**: Deleting a VM secret does NOT block if referenced by target sets (per SIA API behavior)

### PAM Vault Accounts (External)

PCloudAccount secrets reference **external PAM vault accounts**:

```hcl
# VM Secret references PAM account (managed outside Terraform)
resource "cyberarksia_virtual_machine_secret" "vault_ref" {
  secret_type         = "PCloudAccount"
  pcloud_safe_name    = "Production-Safe"      # External PAM safe
  pcloud_account_name = "db-admin-account"      # External PAM account
}
```

**Relationship**: External reference (Terraform does not manage PAM accounts)

**Validation**: API validates safe and account exist at create/update time

**Lifecycle**: Changes to PAM account (password rotation) do not affect SIA secret (stores reference, not credentials)

## Design Decisions

### 1. Why Two Secret Types?

**Decision**: Support both ProvisionerUser and PCloudAccount as separate types rather than a single unified model.

**Rationale**:
- Different use cases: Standalone credentials vs. PAM integration
- Mutually exclusive fields prevent confusion
- Matches SIA API design
- Allows for type-specific validation

### 2. Why ForceNew on secret_type?

**Decision**: Changing secret_type requires destroy + recreate (ForceNew).

**Rationale**:
- Fundamental change in credential storage mechanism
- Cannot convert between standalone and PAM reference
- SIA API does not support type conversion
- Clear state transition prevents orphaned credentials

### 3. Why Write-Only Passwords?

**Decision**: Passwords never returned by API, not stored in read responses.

**Rationale**:
- Security best practice (principle of least privilege)
- SIA API design (never returns passwords)
- Consistent with Terraform sensitive data patterns
- Forces explicit password management in config

### 4. Why Allow Duplicate secret_name?

**Decision**: secret_name uniqueness not enforced (secret_id is unique identifier).

**Rationale**:
- SIA API allows duplicate names
- UUID (secret_id) provides true uniqueness
- User freedom in naming conventions
- Matches database secrets behavior

### 5. Why No Tagging/Metadata?

**Decision**: No support for tags, descriptions, or additional metadata.

**Rationale**:
- SDK limitation (VM secrets API doesn't support metadata)
- Keep implementation focused on core functionality
- Future enhancement if SDK adds support

## Testing Considerations

### Unit Test Coverage

- Attribute validation (required fields per type)
- Cross-attribute validation (conditional requirements)
- ForceNew behavior on secret_type change
- Sensitive attribute marking

### Acceptance Test Scenarios

1. **Create ProvisionerUser**: All required fields → secret created
2. **Create PCloudAccount**: PAM reference fields → secret created
3. **Read Idempotency**: No changes → plan shows no diff
4. **Update Name**: secret_name change → in-place update
5. **Update Password**: provisioner_password change → in-place update
6. **Update PAM Ref**: pcloud_account_name change → in-place update
7. **Change Type**: ProvisionerUser → PCloudAccount → ForceNew (destroy + recreate)
8. **Import**: Import by secret_id → state populated (no password)
9. **Delete**: Remove from config → secret deleted
10. **Drift Detection**: Manual name change in SIA → detected in plan

### Edge Case Testing

- Create without required conditional fields → validation error
- Create with wrong secret_type value → enum validation error
- Create PCloudAccount with non-existent safe → API error
- Update secret_type → ForceNew triggered
- Delete already-deleted secret → idempotent success
- Import non-existent secret_id → clear error message

---

**Document Version**: 1.0
**Last Updated**: 2025-11-02
**Status**: Ready for implementation
