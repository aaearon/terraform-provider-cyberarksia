# Principal Lookup Bug Fix - Solution Summary

**Date**: 2025-10-29
**Status**: ✅ RESOLVED
**Branch**: `003-principal-lookup`

---

## Problem Summary

The principal lookup data source was failing with entity type assertion errors:

```
Phase 2: Entity[0] has unknown type: *models.ArkIdentityEntity
Phase 2: Entity[1] has unknown type: *models.ArkIdentityEntity
...
Principal Not Found
```

**Root Cause**: The ARK SDK's `ListDirectoriesEntities()` returns `[]*ArkIdentityEntity` (slice of pointers to interface), not concrete types. Our code was attempting type assertions directly on the interface pointers without dereferencing them first.

---

## Root Cause Analysis

### SDK Implementation Detail

From `ark-sdk-golang@v1.5.0/pkg/services/identity/directories/ark_identity_directories_service.go`:

```go
entities := make([]*directoriesmodels.ArkIdentityEntity, 0)
// ...
var userEntityIfs directoriesmodels.ArkIdentityEntity = userEntity
entities = append(entities, &userEntityIfs)  // ← Stores POINTER to interface
```

The SDK wraps concrete entity types (`*ArkIdentityUserEntity`, `*ArkIdentityGroupEntity`, `*ArkIdentityRoleEntity`) as **pointers to interface** (`*ArkIdentityEntity`), not as the interface itself.

### Failed Type Assertions

Original code attempted:

```go
for _, entity := range allEntities {
    switch e := entity.(type) {
    case *directoriesmodels.ArkIdentityUserEntity:  // ❌ NEVER MATCHES
        // ...
    }
}
```

This failed because `entity` has type `interface{}` containing `*ArkIdentityEntity`, not the concrete types.

---

## Solution

### Two-Step Dereference Process

1. **First**: Assert to `*ArkIdentityEntity` (pointer to interface)
2. **Second**: Dereference to get interface value
3. **Third**: Type assert to concrete types

```go
for _, entity := range allEntities {
    // Step 1: Assert to pointer-to-interface
    entityPtr, ok := entity.(*directoriesmodels.ArkIdentityEntity)
    if !ok {
        continue
    }

    // Step 2: Dereference to get interface value
    entityValue := *entityPtr

    // Step 3: Type assert to concrete types
    switch e := entityValue.(type) {
    case *directoriesmodels.ArkIdentityUserEntity:  // ✅ NOW MATCHES
        entityName = e.Name
        entityType = e.EntityType
    case *directoriesmodels.ArkIdentityGroupEntity:
        entityName = e.Name
        entityType = e.EntityType
    case *directoriesmodels.ArkIdentityRoleEntity:
        entityName = e.Name
        entityType = e.EntityType
    }
}
```

---

## Files Modified

### `internal/provider/principal_data_source.go`

**Phase 1 Enrichment** (lines 163-184):
- Added two-step dereference before type assertion when matching user by UUID

**Phase 2 Filtering** (lines 205-241):
- Added two-step dereference before type assertion when filtering entities by name
- Added debug logging for type mismatches

**populateDataModel()** (line 332):
- Changed parameter type from `interface{}` to `directoriesmodels.ArkIdentityEntity`

### `internal/provider/principal_data_source_test.go`

Updated test configurations with valid principals:
- **Cloud User**: `tim.schindler@cyberark.cloud.40562` ✅
- **Federated User**: `tim.schindler@cyberiam.com` ✅
- **Group**: `CyberArk Guardians` ✅
- **AD User**: `SchindlerT@cyberiam.tech` ✅
- **Role**: `System Administrator` ✅
- **NotFound**: `nonexistent.user.does.not.exist@invalid.domain.test` ✅

Added `provider "cyberarksia" {}` blocks to all test configurations.

---

## Test Results

### ✅ All Tests Passing

```
=== RUN   TestAccPrincipalDataSource_CloudUser
--- PASS: TestAccPrincipalDataSource_CloudUser (13.16s)
=== RUN   TestAccPrincipalDataSource_FederatedUser
--- PASS: TestAccPrincipalDataSource_FederatedUser (12.63s)
=== RUN   TestAccPrincipalDataSource_Group
--- PASS: TestAccPrincipalDataSource_Group (12.02s)
=== RUN   TestAccPrincipalDataSource_TypeFilter
--- PASS: TestAccPrincipalDataSource_TypeFilter (10.71s)
=== RUN   TestAccPrincipalDataSource_ADUser
--- PASS: TestAccPrincipalDataSource_ADUser (11.49s)
=== RUN   TestAccPrincipalDataSource_NotFound
--- PASS: TestAccPrincipalDataSource_NotFound (3.30s)
=== RUN   TestAccPrincipalDataSource_Role
--- PASS: TestAccPrincipalDataSource_Role (10.99s)
PASS
ok  	github.com/aaearon/terraform-provider-cyberarksia/internal/provider	74.345s
```

**Coverage**:
- ✅ Cloud Directory (CDS) users
- ✅ Federated Directory (FDS) users
- ✅ Active Directory (AdProxy) users
- ✅ Groups
- ✅ Roles
- ✅ Type filtering
- ✅ Error handling (not found)

---

## Key Learnings

1. **SDK Pattern**: ARK SDK uses `[]*InterfaceType` pattern for polymorphic collections
2. **Go Interfaces**: Type assertions on `*Interface` require dereferencing before accessing concrete types
3. **Debug Strategy**: Added type logging helped identify the exact runtime type
4. **POC Value**: The proof-of-concept code in `docs/development/principal-lookup-investigation.md` showed the correct approach

---

## References

- **SDK Source**: `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/identity/directories/ark_identity_directories_service.go`
- **POC Code**: `/tmp/principal-lookup-poc/optimal-hybrid-lookup.go`
- **Investigation**: `docs/development/principal-lookup-investigation.md`
- **Debugging Notes**: `specs/003-principal-lookup/DEBUGGING-NOTES.md`

---

## Status

✅ **COMPLETE** - All acceptance tests passing, ready for commit and PR.
