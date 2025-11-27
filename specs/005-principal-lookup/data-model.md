# Data Model: Principal Lookup Data Source

**Feature**: Principal Lookup Data Source
**Date**: 2025-10-29
**Status**: Design Phase

---

## Overview

This document defines the data model for the principal lookup data source, including entity definitions, relationships, field mappings from ARK SDK to Terraform schema, and validation rules.

---

## Entity: Principal

A principal represents a user, group, or role in the CyberArk Identity system.

### Terraform Schema

```hcl
data "cyberarksia_principal" "example" {
  # INPUT ATTRIBUTES
  name = "tim.schindler@cyberark.cloud.40562"  # Required: SystemName (unique identifier)
  type = "USER"                                  # Optional: Filter by USER/GROUP/ROLE

  # COMPUTED ATTRIBUTES (outputs)
  id             = "c2c7bcc6-9560-44e0-8dff-5be221cd37ee"  # Principal UUID
  principal_type = "USER"                                   # USER/GROUP/ROLE
  directory_name = "CyberArk Cloud Directory"              # Localized directory name
  directory_id   = "09B9A9B0-6CE8-465F-AB03-65766D33B05E"  # Directory UUID
  display_name   = "Tim Schindler"                          # Human-readable name
  email          = "tim.schindler@cyberiam.com"            # Email (users only)
  description    = "Cloud administrator"                    # Description (optional)
}
```

### Go Struct

```go
type PrincipalDataSourceModel struct {
    // Input attributes
    Name types.String `tfsdk:"name"`  // Required: SystemName (e.g., "user@domain.com")
    Type types.String `tfsdk:"type"`  // Optional: USER/GROUP/ROLE filter

    // Computed attributes
    ID            types.String `tfsdk:"id"`              // Principal UUID
    PrincipalType types.String `tfsdk:"principal_type"`  // USER/GROUP/ROLE
    DirectoryName types.String `tfsdk:"directory_name"`  // Localized directory name
    DirectoryID   types.String `tfsdk:"directory_id"`    // Directory UUID
    DisplayName   types.String `tfsdk:"display_name"`    // Human-readable name
    Email         types.String `tfsdk:"email"`           // Email (users only, optional)
    Description   types.String `tfsdk:"description"`     // Description (optional)
}
```

### Field Mappings

#### From ARK SDK `ArkIdentityUserEntity` (ListDirectoriesEntities API)

```go
// Source: pkg/services/identity/directories/models/ark_identity_entity.go
type ArkIdentityUserEntity struct {
    ID                       string  // → PrincipalDataSourceModel.ID
    Name                     string  // → PrincipalDataSourceModel.Name (SystemName)
    EntityType               string  // → PrincipalDataSourceModel.PrincipalType (USER)
    DirectoryServiceType     string  // → Used for directory UUID mapping (CDS/FDS/AdProxy)
    ServiceInstanceLocalized string  // → PrincipalDataSourceModel.DirectoryName (CRITICAL)
    DisplayName              string  // → PrincipalDataSourceModel.DisplayName
    Email                    string  // → PrincipalDataSourceModel.Email
    Description              string  // → PrincipalDataSourceModel.Description
}
```

#### From ARK SDK `ArkIdentityGroupEntity` (ListDirectoriesEntities API)

```go
// Source: pkg/services/identity/directories/models/ark_identity_entity.go
type ArkIdentityGroupEntity struct {
    ID                       string  // → PrincipalDataSourceModel.ID
    Name                     string  // → PrincipalDataSourceModel.Name (SystemName)
    EntityType               string  // → PrincipalDataSourceModel.PrincipalType (GROUP)
    DirectoryServiceType     string  // → Used for directory UUID mapping
    ServiceInstanceLocalized string  // → PrincipalDataSourceModel.DirectoryName
    DisplayName              string  // → PrincipalDataSourceModel.DisplayName
    Description              string  // → PrincipalDataSourceModel.Description (optional)
}
```

#### From ARK SDK `ArkIdentityRoleEntity` (ListDirectoriesEntities API)

```go
// Source: pkg/services/identity/directories/models/ark_identity_entity.go
type ArkIdentityRoleEntity struct {
    ID                       string  // → PrincipalDataSourceModel.ID
    Name                     string  // → PrincipalDataSourceModel.Name (SystemName)
    EntityType               string  // → PrincipalDataSourceModel.PrincipalType (ROLE)
    DirectoryServiceType     string  // → Used for directory UUID mapping
    ServiceInstanceLocalized string  // → PrincipalDataSourceModel.DirectoryName
    DisplayName              string  // → PrincipalDataSourceModel.DisplayName
    Description              string  // → PrincipalDataSourceModel.Description (optional)
}
```

#### From ARK SDK `ArkIdentityUser` (UserByName API - Phase 1 only)

```go
// Source: pkg/services/identity/users/ark_identity_users_service.go
type ArkIdentityUser struct {
    UserID      string  // → PrincipalDataSourceModel.ID
    Username    string  // → PrincipalDataSourceModel.Name (SystemName)
    DisplayName string  // → PrincipalDataSourceModel.DisplayName
    Email       string  // → PrincipalDataSourceModel.Email
    // ❌ NO DirectoryServiceType - must get from Phase 2
    // ❌ NO DirectoryServiceUUID - must get from Phase 2
}
```

---

## Entity: Directory

A directory represents an identity source (Cloud Directory, Federated Directory, or Active Directory).

### Directory Type Mapping

| SDK DirectoryServiceType | ServiceInstanceLocalized Example | DirectoryServiceUUID |
|-------------------------|----------------------------------|---------------------|
| `CDS` | "CyberArk Cloud Directory" | "09B9A9B0-6CE8-465F-AB03-65766D33B05E" |
| `FDS` | "Federation with company.com" | "C30B30B1-0B46-49AC-8D99-F6279EED7999" |
| `AdProxy` | "Active Directory (domain.com)" | "76081bc8-a2ba-a183-2a84-ae6180281140" |

### ARK SDK Directory Model

```go
// Source: pkg/services/identity/directories/models/ark_identity_directory.go
type ArkIdentityDirectory struct {
    Directory            string  // "CDS", "FDS", "AdProxy"
    DirectoryServiceUUID string  // UUID for the directory
}
```

### Directory UUID Mapping Function

```go
// Helper function to build directory type → UUID map
func buildDirectoryMap(directories []*directoriesmodels.ArkIdentityDirectory) map[string]string {
    dirMap := make(map[string]string)
    for _, dir := range directories {
        dirMap[dir.Directory] = dir.DirectoryServiceUUID
    }
    return dirMap
}

// Example result:
// {
//   "CDS":     "09B9A9B0-6CE8-465F-AB03-65766D33B05E",
//   "FDS":     "C30B30B1-0B46-49AC-8D99-F6279EED7999",
//   "AdProxy": "76081bc8-a2ba-a183-2a84-ae6180281140"
// }
```

---

## Relationships

### Principal → Directory (Many-to-One)

```
Principal (USER/GROUP/ROLE)
    ├─ Belongs to exactly ONE Directory
    │    └─ Directory.DirectoryServiceUUID
    │
    └─ Identity Fields:
         ├─ ID (Principal UUID) - unique across tenant
         └─ Name (SystemName) - unique across tenant
```

**Key Constraint**: Each principal belongs to exactly ONE directory. Principals are never orphaned.

**Uniqueness**: `SystemName` (Name field) is unique across ALL directories within a tenant. This is why exact SystemName matching is reliable for lookups.

---

## Validation Rules

### Input Validation

#### `name` (Required)
- **Type**: String
- **Required**: Yes
- **Validation**:
  - MUST NOT be empty (enforced by Terraform schema)
  - Case-insensitive matching (implementation detail)
  - No format validation (accepts any string - system validates existence)

#### `type` (Optional)
- **Type**: String
- **Optional**: Yes
- **Valid Values**: `"USER"`, `"GROUP"`, `"ROLE"`
- **Validation**:
  - If provided, MUST be one of the valid values
  - If omitted, searches all principal types
  - Use Terraform `stringvalidator.OneOf()` validator

### Output Validation

#### `id` (Computed)
- **Type**: String (UUID)
- **Required**: Always present when lookup succeeds
- **Format**: UUID (e.g., "c2c7bcc6-9560-44e0-8dff-5be221cd37ee")

#### `principal_type` (Computed)
- **Type**: String
- **Required**: Always present when lookup succeeds
- **Valid Values**: `"USER"`, `"GROUP"`, `"ROLE"`

#### `directory_name` (Computed)
- **Type**: String
- **Required**: Always present when lookup succeeds
- **Format**: Localized human-readable name (e.g., "CyberArk Cloud Directory")
- **Source**: `ServiceInstanceLocalized` field from ARK SDK

#### `directory_id` (Computed)
- **Type**: String (UUID)
- **Required**: Always present when lookup succeeds
- **Format**: UUID (e.g., "09B9A9B0-6CE8-465F-AB03-65766D33B05E")

#### `display_name` (Computed)
- **Type**: String
- **Required**: Always present when lookup succeeds
- **Format**: Human-readable name (e.g., "Tim Schindler")

#### `email` (Computed)
- **Type**: String
- **Optional**: May be null/empty
- **Applies To**: USER principals only (groups/roles don't have email)
- **Handling**: Use `types.StringNull()` if not present

#### `description` (Computed)
- **Type**: String
- **Optional**: May be null/empty
- **Applies To**: All principal types (but often empty)
- **Handling**: Use `types.StringNull()` if not present

---

## State Transitions

Data sources in Terraform are **stateless** and **read-only**:

```
[Terraform Plan/Apply] → Read() → [Query ARK SDK] → [Return Current State]
                                         ↓
                                   No State Stored
```

**Implications**:
- No Create/Update/Delete operations
- Each `terraform plan` or `terraform apply` re-queries the API
- No caching between invocations
- Lookup failures result in plan/apply failure (not stored state)

---

## Error States

### Principal Not Found
```go
resp.Diagnostics.AddError(
    "Principal Not Found",
    fmt.Sprintf("Principal '%s' not found in any directory", principalName),
)
```

### Multiple Matches (Theoretically Impossible)
**Note**: SystemName is unique, so this should never occur. However, if it does:

```go
resp.Diagnostics.AddError(
    "Multiple Principals Found",
    fmt.Sprintf("Found %d principals with name '%s' in different directories: %s",
        count, principalName, directoryList),
)
```

### Authentication Failure
```go
resp.Diagnostics.AddError(
    "Authentication Failed",
    fmt.Sprintf("Failed to authenticate with CyberArk Identity: %v", err),
)
```

### API Connectivity Error
```go
resp.Diagnostics.AddError(
    "API Error",
    fmt.Sprintf("Failed to query CyberArk Identity API: %v", err),
)
```

---

## Performance Considerations

### Lookup Patterns

#### Fast Path (Phase 1 - Users)
```
UserByName() → Match UUID → ListDirectoriesEntities() → Return
  ~100ms        ~500ms           (filter by UUID)
Total: < 1 second
```

#### Fallback Path (Phase 2 - Groups/Roles or User Not Found)
```
ListDirectoriesEntities(search="") → Client-side filter → Return
  ~1s (200-10,000 entities)          ~100ms
Total: < 2 seconds
```

### Scalability

- **Tenant Size**: Tested with 200 entities, designed for up to 10,000
- **API Limits**: SDK PageSize and Limit are both 10,000
- **Caching**: None (stateless data source design)
- **Concurrency**: Each data source instance performs independent lookup

---

## Data Model Summary

```
┌─────────────────────────────────────────────┐
│  Terraform Data Source                      │
│  cyberarksia_principal                      │
│                                             │
│  Input:  name (string), type (string, opt) │
│  Output: id, principal_type, directory_name,│
│          directory_id, display_name, email, │
│          description                        │
└─────────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────┐
│  Principal Entity                           │
│  ├─ ID (UUID)                               │
│  ├─ Name (SystemName) - unique             │
│  ├─ Type (USER/GROUP/ROLE)                 │
│  ├─ DisplayName (human-readable)           │
│  ├─ Email (optional, users only)           │
│  └─ Description (optional)                  │
└─────────────────────────────────────────────┘
                    │
                    │ belongs to
                    ▼
┌─────────────────────────────────────────────┐
│  Directory Entity                           │
│  ├─ DirectoryServiceType (CDS/FDS/AdProxy) │
│  ├─ DirectoryServiceUUID (UUID)            │
│  └─ ServiceInstanceLocalized (display name)│
└─────────────────────────────────────────────┘
```

---

## References

- **Feature Specification**: `specs/003-principal-lookup/spec.md`
- **Investigation Report**: `docs/development/principal-lookup-investigation.md`
- **ARK SDK Source**: `pkg/services/identity/` (users, directories)
- **Existing Data Source**: `internal/provider/access_policy_data_source.go` (pattern reference)
