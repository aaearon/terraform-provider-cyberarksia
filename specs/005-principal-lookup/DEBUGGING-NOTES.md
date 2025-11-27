# Principal Data Source - Phase 2 Debugging Notes

**Date**: 2025-10-29
**Status**: Test infrastructure working, Phase 2 entity type handling bug
**Branch**: `003-principal-lookup`

---

## Current Situation

### What's Working ✅

1. **Test Infrastructure**: Fully functional after fixing multiple issues
   - Provider TypeName: `"cyberarksia"` (no underscore - Terraform requirement)
   - Test provider factory: Correctly registered as `"cyberarksia"`
   - Provider schema: `username` and `client_secret` changed to `Optional: true` (allows env vars)
   - Test configs: Include `provider "cyberarksia" {}` blocks
   - Plugin cache: Cleared (`/home/tim/.terraform.d/plugins/`)
   - Authentication: Working with `CYBERARK_USERNAME` and `CYBERARK_CLIENT_SECRET`

2. **Phase 1 (Fast Path)**: Working correctly
   - `UserByName()` API successfully finds user
   - Test user found: `tim.schindler@cyberark.cloud.40562`
   - User ID: `c2c7bcc6-9560-44e0-8dff-5be221cd37ee`
   - Falls through to Phase 2 because directory enrichment fails

3. **Phase 2 (Fallback)**: API call works, filtering broken
   - `ListDirectoriesEntities()` successfully returns 204 entities
   - Entities ARE in the collection
   - **BUG**: Type assertions fail - entities not being processed

---

## The Bug 🐛

### Symptoms
```
Phase 2: Collected 204 entities from ListDirectoriesEntities
Phase 2: Searching for principal 'tim.schindler@cyberark.cloud.40562' among 204 entities
Phase 2: Entity[0] has unknown type: *models.ArkIdentityEntity
Phase 2: Entity[1] has unknown type: *models.ArkIdentityEntity
...
Principal Not Found
```

### Root Cause

The SDK's `ListDirectoriesEntities()` returns entities as `*models.ArkIdentityEntity`, but our code expects:
- `*directoriesmodels.ArkIdentityUserEntity`
- `*directoriesmodels.ArkIdentityGroupEntity`
- `*directoriesmodels.ArkIdentityRoleEntity`

**Result**: Type assertions fail for ALL entities → loop continues without processing → no match found

### SDK Structure (from `ark-sdk-golang@v1.5.0`)

**File**: `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/identity/directories/models/ark_identity_entity.go`

```go
// ArkIdentityEntity is an interface
type ArkIdentityEntity interface {
	GetEntityType() string
}

// ArkIdentityBaseEntity is the struct with all fields
type ArkIdentityBaseEntity struct {
	ArkIdentityEntity        `json:"-" mapstructure:"-"`
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	EntityType               string `json:"entity_type"` // "USER", "GROUP", "ROLE"
	DirectoryServiceType     string `json:"directory_service_type"` // "AdProxy", "CDS", "FDS"
	DisplayName              string `json:"display_name,omitempty"`
	ServiceInstanceLocalized string `json:"service_instance_localized"`
}

// ArkIdentityUserEntity extends base
type ArkIdentityUserEntity struct {
	ArkIdentityBaseEntity
	Email       string `json:"email,omitempty"`
	Description string `json:"description,omitempty"`
}

// ArkIdentityGroupEntity extends base
type ArkIdentityGroupEntity struct {
	ArkIdentityBaseEntity
}

// ArkIdentityRoleEntity extends base
type ArkIdentityRoleEntity struct {
	ArkIdentityBaseEntity
	Description string `json:"description,omitempty"`
}
```

**Runtime Type**: The channel returns entities as `*models.ArkIdentityEntity` (possibly via interface casting)

---

## What We've Tried ❌

1. ✅ **Fixed**: Added provider block to test configs
2. ✅ **Fixed**: Changed schema from Required to Optional
3. ✅ **Fixed**: Cleared plugin cache
4. ✅ **Fixed**: Corrected provider TypeName
5. ❌ **Attempted**: Added case for `*directoriesmodels.ArkIdentityBaseEntity` - didn't match
6. ❌ **Attempted**: Imported `identitymodels "github.com/cyberark/ark-sdk-golang/pkg/models/common/identity"` - type doesn't exist there
7. ❌ **In Progress**: Need to find the correct type assertion for SDK's entity type

---

## The Fix 🔧

### Option 1: Use Reflection/Type Assertion on Interface

Since `ArkIdentityEntity` is an interface, entities might be wrapped. Try:

```go
case interface{ GetEntityType() string }:
	// Get the underlying type
	if baseEntity, ok := entity.(*directoriesmodels.ArkIdentityBaseEntity); ok {
		entityName = baseEntity.Name
		entityType = baseEntity.EntityType
	}
```

### Option 2: JSON Re-marshaling

The entities are coming through as a generic type. Could unmarshal to struct:

```go
// In getDirectoriesAndEntities(), when collecting entities:
for page := range entitiesChan {
	for _, item := range page.Items {
		// item is already typed - use it directly
		allEntities = append(allEntities, item)
	}
}
```

Check what type `page.Items` actually is in the SDK.

### Option 3: Access Fields Via Interface

If the entities implement `ArkIdentityEntity` interface, they should have fields accessible:

```go
// Try getting fields without type assertion
type namedEntity interface {
	GetName() string
	GetEntityType() string
}

if ne, ok := entity.(namedEntity); ok {
	entityName = ne.GetName()
	entityType = ne.GetEntityType()
}
```

### Option 4: Check SDK Documentation

Look at SDK examples for how to handle `ListDirectoriesEntities()` results:
- Check `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/` for example code
- Search for uses of `ListDirectoriesEntities` in SDK tests

---

## Debug Commands

### Check Entity Type at Runtime
```bash
export TF_ACC=1 && export TF_LOG=DEBUG && \
export CYBERARK_USERNAME="timtest@cyberark.cloud.40562" && \
export CYBERARK_CLIENT_SECRET="nvk*phv*hfd3ATR2rfc" && \
go test ./internal/provider/ -v -run TestAccPrincipalDataSource_CloudUser -timeout 10m 2>&1 | grep "Entity\["
```

### Find SDK Type Definition
```bash
rg "type.*ArkIdentityEntity" /home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/ -A 5
```

### Check SDK Examples
```bash
fdfind "example" /home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/ -t f
```

---

## Files Modified

### Working Changes ✅

1. **`internal/provider/provider.go`**:
   - Line 69: `resp.TypeName = "cyberarksia"` (was incorrectly `"cyberark_sia"`)
   - Lines 83, 92: Changed `Required: true` → `Optional: true` for username/client_secret

2. **`internal/provider/provider_test.go`**:
   - Line 17: `"cyberarksia": providerserver.NewProtocol6WithError(...)` (was `"cyberark_sia"`)

3. **`internal/provider/principal_data_source_test.go`**:
   - Line 177: Added `provider "cyberarksia" {}` block to test config
   - All resource names: `cyberarksia_principal` (consistent naming)

### In-Progress Changes 🚧

4. **`internal/provider/principal_data_source.go`**:
   - Added debug logging (lines 308, 201, 222-230)
   - **Line 207-231**: Type switch needs fixing for SDK entity types
   - **Line 15**: May need different import for entity types

---

## Phase 1 Enrichment Bug (Secondary Issue)

Phase 1 finds the user but fails to enrich directory info. This causes fallback to Phase 2.

**Location**: `principal_data_source.go:155-188`

```go
// Phase 1: User found via UserByName, now fetch directory entities to enrich
_, dirMap, allEntities, err := d.getDirectoriesAndEntities(ctx, principalName, principalTypeFilter)
// ...
// Find the matching entity by UUID
found := false
for _, entity := range allEntities {
	if userEntity, ok := entity.(*directoriesmodels.ArkIdentityUserEntity); ok {
		if userEntity.ID == principalID {
			// This type assertion is also failing!
			found = true
			break
		}
	}
}
```

**Same bug**: Type assertion fails here too. Fixing Phase 2 will also fix Phase 1 enrichment.

---

## Test Credentials

**Test User**: `tim.schindler@cyberark.cloud.40562`
**Service Account**: `timtest@cyberark.cloud.40562`
**Credentials Location**: `/home/tim/terraform-provider-cyberarksia/.env`

---

## Next Session TODO

1. **Investigate SDK's actual return type**:
   ```bash
   # Check the ListDirectoriesEntities return type
   rg "func.*ListDirectoriesEntities" /home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/ -A 20
   ```

2. **Try reflection to inspect runtime type**:
   ```go
   // Add temporary debug code
   import "reflect"
   tflog.Debug(ctx, fmt.Sprintf("Entity type: %v, value: %+v", reflect.TypeOf(entity), entity))
   ```

3. **Check if items need unmarshaling**:
   ```go
   // page.Items might need conversion
   for _, item := range page.Items {
       // What is item's actual type?
       tflog.Debug(ctx, fmt.Sprintf("Item type: %T", item))
   }
   ```

4. **Alternative: Use Phase 1 Only**:
   - Since Phase 1 works (finds user), could we just fix the enrichment part?
   - Phase 2 can be optional fallback for groups/roles
   - For MVP, could skip Phase 2 entirely

---

## Success Criteria

Test passes when:
```
Phase 2: Entity[X] name='tim.schindler@cyberark.cloud.40562' type='USER'
Phase 2: Found matching entity
Principal lookup succeeded
```

Then update `populateDataModel()` to handle the correct entity type for setting display_name, email, etc.

---

## Quick Reference

**Key Files**:
- Implementation: `internal/provider/principal_data_source.go:199-246` (Phase 2 filtering)
- Tests: `internal/provider/principal_data_source_test.go`
- SDK Models: `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/identity/directories/models/`

**Test Command**:
```bash
export TF_ACC=1 && export CYBERARK_USERNAME="timtest@cyberark.cloud.40562" && \
export CYBERARK_CLIENT_SECRET="nvk*phv*hfd3ATR2rfc" && \
go test ./internal/provider/ -v -run TestAccPrincipalDataSource_CloudUser -timeout 10m
```
