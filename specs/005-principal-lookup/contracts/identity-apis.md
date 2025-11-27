# ARK SDK Identity APIs Contract

**Feature**: Principal Lookup Data Source
**SDK Version**: v1.5.0
**Package**: `github.com/cyberark/ark-sdk-golang/pkg/services/identity`

---

## API 1: UserByName() - Fast User Lookup

**Purpose**: Look up a user by SystemName (Username field)

**SDK Location**: `pkg/services/identity/users/ark_identity_users_service.go:326`

**API Endpoint**: `Redrock/query` (SQL-like query interface)

**Method Signature**:
```go
func (u *ArkIdentityUsersService) UserByName(user *usersmodels.ArkIdentityUserByName) (*usersmodels.ArkIdentityUser, error)
```

**Input**:
```go
&usersmodels.ArkIdentityUserByName{
    Username: "tim.schindler@cyberark.cloud.40562",  // SystemName
}
```

**Output**:
```go
&usersmodels.ArkIdentityUser{
    UserID:       "c2c7bcc6-9560-44e0-8dff-5be221cd37ee",
    Username:     "tim.schindler@cyberark.cloud.40562",
    DisplayName:  "Tim Schindler",
    Email:        "tim.schindler@cyberiam.com",
    MobileNumber: "",
    LastLogin:    &time.Time{},
}
```

**Characteristics**:
- ✅ Case-insensitive SystemName matching
- ✅ Works for CDS and FDS users
- ✅ Fast (direct query)
- ❌ NO directory information
- ❌ Only users (not groups/roles)

---

## API 2: ListDirectoriesEntities() - Universal Entity Search

**Purpose**: Search for users, groups, and roles across all directories

**SDK Location**: `pkg/services/identity/directories/ark_identity_directories_service.go:138`

**API Endpoint**: `UserMgmt/DirectoryServiceQuery`

**Method Signature**:
```go
func (d *ArkIdentityDirectoriesService) ListDirectoriesEntities(
    request *directoriesmodels.ArkIdentityListDirectoriesEntities,
) (<-chan *directoriesmodels.ArkIdentityEntitiesPage, error)
```

**Input** (Get All):
```go
&directoriesmodels.ArkIdentityListDirectoriesEntities{
    Search:      "",  // CRITICAL: Empty = get all entities
    EntityTypes: []string{"USER", "GROUP", "ROLE"},
    PageSize:    10000,
    Limit:       10000,
}
```

**Output Channel**:
```go
for page := range entitiesChan {
    for _, entity := range page.Items {
        // entity is interface{} - type assert to:
        // - *directoriesmodels.ArkIdentityUserEntity
        // - *directoriesmodels.ArkIdentityGroupEntity
        // - *directoriesmodels.ArkIdentityRoleEntity
    }
}
```

**User Entity**:
```go
&directoriesmodels.ArkIdentityUserEntity{
    ID:                       "c2c7bcc6-9560-44e0-8dff-5be221cd37ee",
    Name:                     "tim.schindler@cyberark.cloud.40562",  // SystemName
    EntityType:               "USER",
    DirectoryServiceType:     "CDS",  // Internal type (for UUID mapping)
    ServiceInstanceLocalized: "CyberArk Cloud Directory",  // Use for source_directory_name
    DisplayName:              "Tim Schindler",
    Email:                    "tim.schindler@cyberiam.com",
    Description:              "",
}
```

**Characteristics**:
- ✅ Returns complete directory information
- ✅ Works for USER, GROUP, ROLE
- ✅ Works across all directory types
- ⚠️ Searches DisplayName only (not SystemName)
- ⚠️ Requires client-side filtering for exact match

---

## API 3: ListDirectories() - Directory UUID Mapping

**Purpose**: Get directory type → UUID mapping

**SDK Location**: `pkg/services/identity/directories/ark_identity_directories_service.go:91`

**API Endpoint**: `Core/GetDirectoryServices`

**Method Signature**:
```go
func (d *ArkIdentityDirectoriesService) ListDirectories(
    request *directoriesmodels.ArkIdentityListDirectories,
) ([]*directoriesmodels.ArkIdentityDirectory, error)
```

**Input**:
```go
&directoriesmodels.ArkIdentityListDirectories{}
```

**Output**:
```go
[]*directoriesmodels.ArkIdentityDirectory{
    {Directory: "CDS", DirectoryServiceUUID: "09B9A9B0-6CE8-465F-AB03-65766D33B05E"},
    {Directory: "FDS", DirectoryServiceUUID: "C30B30B1-0B46-49AC-8D99-F6279EED7999"},
    {Directory: "AdProxy", DirectoryServiceUUID: "76081bc8-a2ba-a183-2a84-ae6180281140"},
}
```

**Characteristics**:
- ✅ Fast (single API call)
- ✅ Required for DirectoryServiceType → DirectoryServiceUUID mapping
- ✅ Tenant-specific UUIDs

---

## Hybrid Lookup Strategy

```
Phase 1: Try UserByName(username)
  ↓ If found → Get directory info by matching UUID in ListDirectoriesEntities()
  ↓ If NOT found OR type filter is GROUP/ROLE → Phase 2

Phase 2: ListDirectoriesEntities(search="")
  ↓ Client-side filter by exact SystemName match (case-insensitive)
  ↓ Return complete principal + directory data
```

---

## Error Handling

All APIs return `(result, error)`. Wrap errors with `internal/client.MapError()` for Terraform diagnostics.

**Common Errors**:
- Authentication failure: 401 Unauthorized
- Network connectivity: Connection timeout
- Principal not found: Empty result (not an error)
- API rate limit: 429 Too Many Requests
