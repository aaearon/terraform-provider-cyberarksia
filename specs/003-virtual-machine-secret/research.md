# ARK SDK v1.5.0 VM Secrets Service - Comprehensive Analysis

## Overview

The VM Secrets service in ARK SDK v1.5.0 manages secrets (credentials) used for VM infrastructure provisioning. It differs significantly from the database secrets service in both structure and capabilities.

---

## 1. Service Location & Files

**SDK Base Path**: `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/sia/secrets/vm/`

### Key Service Files
- **Service Implementation**: `ark_sia_secrets_vm_service.go` - Main service with CRUD operations
- **Service Configuration**: `ark_sia_secrets_vm_service_config.go` - Service metadata and registration
- **Models Directory**: `models/` - Data structures for all operations

### Model Files Structure
```
models/
├── ark_sia_vm_secret.go              # ArkSIAVMSecret response structure
├── ark_sia_vm_add_secret.go          # ArkSIAVMAddSecret request structure
├── ark_sia_vm_change_secret.go       # ArkSIAVMChangeSecret request structure
├── ark_sia_vm_delete_secret.go       # ArkSIAVMDeleteSecret request structure
├── ark_sia_vm_get_secret.go          # ArkSIAVMGetSecret request structure
├── ark_sia_vm_secret_info.go         # ArkSIAVMSecretInfo response structure
├── ark_sia_vm_secrets_filter.go      # ArkSIAVMSecretsFilter request structure
└── ark_sia_vm_secrets_stats.go       # ArkSIAVMSecretsStats response structure
```

---

## 2. ArkSIASecretsVMService - Method Signatures

**Package**: `vmsecrets`

### Service Structure
```go
type ArkSIASecretsVMService struct {
    services.ArkService
    *services.ArkBaseService
    ispAuth *auth.ArkISPAuth
    client  *isp.ArkISPServiceClient
}
```

### Constructor
```go
func NewArkSIASecretsVMService(authenticators ...auth.ArkAuth) (*ArkSIASecretsVMService, error)
```

### Public Methods

#### 1. AddSecret
```go
func (s *ArkSIASecretsVMService) AddSecret(
    addSecret *vmsecretsmodels.ArkSIAVMAddSecret
) (*vmsecretsmodels.ArkSIAVMSecret, error)
```
- **HTTP Method**: POST
- **Endpoint**: `/api/secrets`
- **Status Code Expected**: 201 (Created)
- **Request**: Pointer to `ArkSIAVMAddSecret`
- **Response**: Pointer to `ArkSIAVMSecret` or error
- **Key Implementation Detail**: Constructs `secret` field with `tenant_encrypted: false` and handles secret_data based on SecretType

#### 2. ChangeSecret
```go
func (s *ArkSIASecretsVMService) ChangeSecret(
    changeSecret *vmsecretsmodels.ArkSIAVMChangeSecret
) (*vmsecretsmodels.ArkSIAVMSecret, error)
```
- **HTTP Method**: POST (Note: Uses POST, not PATCH, for updates)
- **Endpoint**: `/api/secrets/{secret_id}`
- **Status Code Expected**: 200 (OK)
- **Request**: Pointer to `ArkSIAVMChangeSecret`
- **Response**: Pointer to `ArkSIAVMSecret` or error
- **Key Implementation Detail**: Only sends fields that were changed; supports partial updates for credentials

#### 3. DeleteSecret
```go
func (s *ArkSIASecretsVMService) DeleteSecret(
    deleteSecret *vmsecretsmodels.ArkSIAVMDeleteSecret
) error
```
- **HTTP Method**: DELETE
- **Endpoint**: `/api/secrets/{secret_id}`
- **Status Code Expected**: 204 (No Content)
- **Request**: Pointer to `ArkSIAVMDeleteSecret`
- **Response**: Error only (no data on success)
- **Note**: Does NOT have the nil body panic bug present in database secrets

#### 4. Secret (Get)
```go
func (s *ArkSIASecretsVMService) Secret(
    getSecret *vmsecretsmodels.ArkSIAVMGetSecret
) (*vmsecretsmodels.ArkSIAVMSecret, error)
```
- **HTTP Method**: GET
- **Endpoint**: `/api/secrets/{secret_id}`
- **Status Code Expected**: 200 (OK)
- **Request**: Pointer to `ArkSIAVMGetSecret`
- **Response**: Pointer to `ArkSIAVMSecret` or error

#### 5. ListSecrets
```go
func (s *ArkSIASecretsVMService) ListSecrets() ([]*vmsecretsmodels.ArkSIAVMSecret, error)
```
- **HTTP Method**: GET
- **Endpoint**: `/api/secrets`
- **Status Code Expected**: 200 (OK)
- **Request**: None
- **Response**: Slice of pointers to `ArkSIAVMSecret` or error

#### 6. ListSecretsBy (Filtered)
```go
func (s *ArkSIASecretsVMService) ListSecretsBy(
    filter *vmsecretsmodels.ArkSIAVMSecretsFilter
) ([]*vmsecretsmodels.ArkSIAVMSecret, error)
```
- **HTTP Method**: GET
- **Endpoint**: `/api/secrets`
- **Status Code Expected**: 200 (OK)
- **Request**: Pointer to `ArkSIAVMSecretsFilter`
- **Response**: Slice of pointers to `ArkSIAVMSecret` or error
- **Implementation Detail**: Client-side filtering; fetches all secrets then filters locally on SecretTypes, Name (regex), and IsActive

#### 7. SecretsStats
```go
func (s *ArkSIASecretsVMService) SecretsStats() (*vmsecretsmodels.ArkSIAVMSecretsStats, error)
```
- **HTTP Method**: GET (via ListSecrets internally)
- **Endpoint**: `/api/secrets`
- **Status Code Expected**: 200 (OK)
- **Request**: None
- **Response**: Pointer to `ArkSIAVMSecretsStats` or error
- **Computed Stats**:
  - `SecretsCount` - total count
  - `ActiveSecretsCount` - active count
  - `InactiveSecretsCount` - inactive count
  - `SecretsCountByType` - map of type → count

#### 8. ServiceConfig
```go
func (s *ArkSIASecretsVMService) ServiceConfig() services.ArkServiceConfig
```
- **Returns**: `ServiceConfig` constant (defines service metadata)

---

## 3. Data Structures - Complete Field Mappings

### ArkSIAVMSecret (Response/Full Model)

**File**: `models/ark_sia_vm_secret.go`

**Constants**:
```go
const (
    ProvisionerUser = "ProvisionerUser"  // Secret type for provisioner user credentials
    PCloudAccount   = "PCloudAccount"    // Secret type for Power Cloud account credentials
)
```

**Structure**:
```go
type ArkSIAVMSecret struct {
    // Identifiers
    SecretID   string  `json:"secret_id" mapstructure:"secret_id" flag:"secret-id" desc:"ID of the secret"`
    TenantID   string  `json:"tenant_id,omitempty" mapstructure:"tenant_id,omitempty" flag:"tenant-id" desc:"Tenant ID of the secret"`

    // Secret Container
    Secret     ArkSIAVMSecretData  `json:"secret,omitempty" mapstructure:"secret,omitempty" flag:"secret" desc:"Secret itself"`

    // Metadata
    SecretType string  `json:"secret_type" mapstructure:"secret_type" flag:"secret-type" desc:"Type of the secret" choices:"ProvisionerUser,PCloudAccount"`
    SecretDetails map[string]interface{}  `json:"secret_details" mapstructure:"secret_details" flag:"secret-details" desc:"Secret extra details"`
    SecretName string  `json:"secret_name,omitempty" mapstructure:"secret_name,omitempty" flag:"secret-name" desc:"A friendly name label"`

    // Status
    IsActive   bool    `json:"is_active" mapstructure:"is_active" flag:"is-active" desc:"Whether this secret is active or not and can be retrieved or modified"`
    IsRotatable bool   `json:"is_rotatable" mapstructure:"is_rotatable" flag:"is-rotatable" desc:"Whether this secret can be rotated"`

    // Timestamps
    CreationTime string  `json:"creation_time" mapstructure:"creation_time" flag:"creation-time" desc:"Creation time of the secret"`
    LastModified string  `json:"last_modified" mapstructure:"last_modified" flag:"last-modified" desc:"Last time the secret was modified"`
}
```

**Nested Structure - ArkSIAVMSecretData**:
```go
type ArkSIAVMSecretData struct {
    SecretData      interface{}  `json:"secret_data" mapstructure:"secret_data" flag:"secret-data" desc:"Actual secret data, can be of different types, and is base64 encoded if of SecretBytes, Otherwise Stored in the jit data message as a string Or as a dict of secret data to be encrypted"`
    TenantEncrypted bool         `json:"tenant_encrypted" mapstructure:"tenant_encrypted" flag:"tenant-encrypted" desc:"Whether this secret is encrypted by the tenant key or not"`
}
```

**Key Differences from Database Secrets**:
- Database secrets include `CreatedBy`, `LastUpdatedBy`, `SecretStore`, `SecretLink`, `SecretExposedData`, `Description`, `Purpose`, `Tags`, `IsActive`
- VM secrets are simpler: `TenantID`, `Secret` container, `IsRotatable`, but NO `Description`, `Purpose`, `Tags`
- VM secrets only support 2 types: `ProvisionerUser`, `PCloudAccount`
- Database secrets support 4+ types: `username_password`, `iam_user`, `cyberark_pam`, `atlas_access_keys`

---

### ArkSIAVMAddSecret (Create Request)

**File**: `models/ark_sia_vm_add_secret.go`

```go
type ArkSIAVMAddSecret struct {
    // Basic Metadata
    SecretName   string  `json:"secret_name,omitempty" mapstructure:"secret_name,omitempty" flag:"secret-name" desc:"Optional name of the secret"`
    SecretDetails map[string]interface{}  `json:"secret_details,omitempty" mapstructure:"secret_details,omitempty" flag:"secret-details" desc:"Optional extra details about the secret"`

    // Type & Status
    SecretType   string  `json:"secret_type" mapstructure:"secret_type" flag:"secret-type" validate:"required" choices:"ProvisionerUser,PCloudAccount" desc:"Type of the secret to add, data is picked according to the chosen type"`
    IsDisabled   bool    `json:"is_disabled" mapstructure:"is_disabled" flag:"is-disabled" default:"false" desc:"Whether the secret should be disabled or not"`

    // ProvisionerUser Type Credentials
    ProvisionerUsername  string  `json:"provisioner_username,omitempty" mapstructure:"provisioner_username,omitempty" flag:"provisioner-username" desc:"If provisioner user type is picked, the username"`
    ProvisionerPassword  string  `json:"provisioner_password,omitempty" mapstructure:"provisioner_password,omitempty" flag:"provisioner-password" desc:"If provisioner user type is picked, the password"`

    // PCloudAccount Type Credentials
    PCloudAccountSafe    string  `json:"pcloud_account_safe,omitempty" mapstructure:"pcloud_account_safe,omitempty" flag:"pcloud-account-safe" desc:"If pcloud account type is picked, the account safe"`
    PCloudAccountName    string  `json:"pcloud_account_name,omitempty" mapstructure:"pcloud_account_name,omitempty" flag:"pcloud-account-name" desc:"If pcloud account type is picked, the account name"`
}
```

**SDK Service Implementation Detail** (from `AddSecret` method, lines 67-99):
- `SecretType == "ProvisionerUser"` requires both `ProvisionerUsername` and `ProvisionerPassword`
- `SecretType == "PCloudAccount"` requires both `PCloudAccountSafe` and `PCloudAccountName`
- Any other value returns error: `"invalid secret type: {value}"`
- Request JSON structure includes:
  ```json
  {
    "secret_name": "...",
    "secret_type": "ProvisionerUser|PCloudAccount",
    "secret": {
      "tenant_encrypted": false,
      "secret_data": {
        "username": "...",
        "password": "..."
      }  // OR for PCloudAccount
         // "safe": "...",
         // "account_name": "..."
    },
    "is_active": !IsDisabled,
    "secret_details": {...}
  }
  ```

---

### ArkSIAVMChangeSecret (Update Request)

**File**: `models/ark_sia_vm_change_secret.go`

```go
type ArkSIAVMChangeSecret struct {
    // Required
    SecretID     string  `json:"secret_id" mapstructure:"secret_id" flag:"secret-id" validate:"required" desc:"The secret id to change"`

    // Optional Metadata
    SecretName   string  `json:"secret_name,omitempty" mapstructure:"secret_name,omitempty" flag:"secret-name" desc:"The new name of the secret"`
    SecretDetails map[string]interface{}  `json:"secret_details,omitempty" mapstructure:"secret_details,omitempty" flag:"secret-details" desc:"New secret details to add / change"`

    // Optional Status
    IsDisabled   bool    `json:"is_disabled,omitempty" mapstructure:"is_disabled,omitempty" flag:"is-disabled" default:"false" desc:"Whether to disable the secret"`

    // Optional ProvisionerUser Credentials
    ProvisionerUsername  string  `json:"provisioner_username,omitempty" mapstructure:"provisioner_username,omitempty" flag:"provisioner-username" desc:"If provisioner user type secret, the new username"`
    ProvisionerPassword  string  `json:"provisioner_password,omitempty" mapstructure:"provisioner_password,omitempty" flag:"provisioner-password" desc:"If provisioner user type secret, the new password"`

    // Optional PCloudAccount Credentials
    PCloudAccountSafe    string  `json:"pcloud_account_safe,omitempty" mapstructure:"pcloud_account_safe,omitempty" flag:"pcloud-account-safe" desc:"If pcloud account type secret, the new account safe"`
    PCloudAccountName    string  `json:"pcloud_account_name,omitempty" mapstructure:"pcloud_account_name,omitempty" flag:"pcloud-account-name" desc:"If pcloud account type secret, the new account name"`
}
```

**SDK Service Implementation Detail** (from `ChangeSecret` method, lines 126-176):
- Only non-empty fields are sent in the update request
- For ProvisionerUser: Both username AND password must be provided together to update
- For PCloudAccount: Both safe AND name must be provided together to update
- Request JSON structure includes conditionally:
  ```json
  {
    "is_active": !IsDisabled,  // Always included
    "secret": {                 // Only if credentials provided
      "secret_data": {
        "username": "...",
        "password": "..."
      }
    },
    "secret_name": "...",  // Only if provided
    "secret_details": {...}  // Only if provided
  }
  ```

---

### ArkSIAVMGetSecret (Read/Retrieve Request)

**File**: `models/ark_sia_vm_get_secret.go`

```go
type ArkSIAVMGetSecret struct {
    SecretID  string  `json:"secret_id" mapstructure:"secret_id" flag:"secret-id" validate:"required" desc:"The secret id to get"`
}
```

**Minimal request structure** - only contains the secret ID to retrieve.

---

### ArkSIAVMDeleteSecret (Delete Request)

**File**: `models/ark_sia_vm_delete_secret.go`

```go
type ArkSIAVMDeleteSecret struct {
    SecretID  string  `json:"secret_id" mapstructure:"secret_id" flag:"secret-id" validate:"required" desc:"The secret id to delete"`
}
```

**Minimal request structure** - only contains the secret ID to delete.

---

### ArkSIAVMSecretsFilter (Filter Request)

**File**: `models/ark_sia_vm_secrets_filter.go`

```go
type ArkSIAVMSecretsFilter struct {
    SecretTypes  []string  `json:"secret_types,omitempty" mapstructure:"secret_types,omitempty" flag:"secret-types" desc:"Type of secrets to filter"`
    Name         string    `json:"name,omitempty" mapstructure:"name,omitempty" flag:"name" desc:"Name wildcard to filter with"`
    SecretDetails map[string]interface{}  `json:"secret_details,omitempty" mapstructure:"secret_details,omitempty" flag:"secret-details" desc:"Secret details to filter with"`
    IsActive     bool      `json:"is_active,omitempty" mapstructure:"is_active,omitempty" flag:"is-active" desc:"Filter only active / inactive secrets"`
}
```

**Client-Side Filtering** (from `ListSecretsBy` method):
- `SecretTypes`: Uses first element only (converts []string to single string filter)
- `Name`: Applied as regex pattern matching against SecretName
- `SecretDetails`: Not actually used in filtering (built but never applied)
- `IsActive`: Used to filter secrets where `secret.IsActive == filter.IsActive`

---

### ArkSIAVMSecretInfo (Brief Response)

**File**: `models/ark_sia_vm_secret_info.go`

```go
type ArkSIAVMSecretInfo struct {
    SecretID      string  `json:"secret_id" mapstructure:"secret_id" flag:"secret-id" desc:"ID of the secret"`
    TenantID      string  `json:"tenant_id,omitempty" mapstructure:"tenant_id,omitempty" flag:"tenant-id" desc:"Tenant ID of the secret"`
    SecretType    string  `json:"secret_type" mapstructure:"secret_type" flag:"secret-type" desc:"Type of the secret" choices:"ProvisionerUser,PCloudAccount"`
    SecretName    string  `json:"secret_name,omitempty" mapstructure:"secret_name,omitempty" flag:"secret-name" desc:"A friendly name label"`
    SecretDetails map[string]interface{}  `json:"secret_details" mapstructure:"secret_details" flag:"secret-details" desc:"Secret extra details"`
    IsActive      bool    `json:"is_active" mapstructure:"is_active" flag:"is-active" desc:"Whether this secret is active or not"`
}
```

**Difference from ArkSIAVMSecret**:
- Subset of full secret response
- No `Secret` field (credentials not exposed)
- No `IsRotatable` field
- No timestamp fields (`CreationTime`, `LastModified`)

---

### ArkSIAVMSecretsStats (Statistics Response)

**File**: `models/ark_sia_vm_secrets_stats.go`

```go
type ArkSIAVMSecretsStats struct {
    SecretsCount         int            `json:"secrets_count" mapstructure:"secrets_count"`
    ActiveSecretsCount   int            `json:"active_secrets_count" mapstructure:"active_secrets_count"`
    InactiveSecretsCount int            `json:"inactive_secrets_count" mapstructure:"inactive_secrets_count"`
    SecretsCountByType   map[string]int `json:"secrets_count_by_type" mapstructure:"secrets_count_by_type"`
}
```

**Computed from**: ListSecrets() by iterating and counting.

---

## 4. Comparison: VM Secrets vs Database Secrets

### Service Architecture

| Aspect | Database Secrets | VM Secrets |
|--------|------------------|-----------|
| **Package** | `dbsecrets` | `vmsecrets` |
| **Service Type** | `ArkSIASecretsDBService` | `ArkSIASecretsVMService` |
| **Authenticator Required** | `isp` | `isp` |
| **API Endpoint Base** | `/secrets` (database contexts) | `/api/secrets` |

### Secret Types Supported

| Database Secrets | VM Secrets |
|-----------------|-----------|
| `username_password` | `ProvisionerUser` |
| `iam_user` | `PCloudAccount` |
| `cyberark_pam` | |
| `atlas_access_keys` | |

**Implication**: VM secrets are simpler, supporting only 2 types vs database's 4.

### Data Structure Comparison

#### Database Secret Full Response (ArkSIADBSecretMetadata)
- `SecretID`, `SecretName`, `Description`, `Purpose`, `SecretType`
- `SecretStore` (object), `SecretLink` (map), `SecretExposedData` (map)
- `Tags` (map[string]string)
- `CreatedBy`, `CreationTime`, `LastUpdatedBy`, `LastUpdateTime`
- `IsActive`

#### VM Secret Full Response (ArkSIAVMSecret)
- `SecretID`, `TenantID`, `SecretName`, `SecretType`
- `Secret` (object with `SecretData` and `TenantEncrypted`)
- `SecretDetails` (map[string]interface{})
- `IsActive`, `IsRotatable`
- `CreationTime`, `LastModified`

**Key Differences**:
1. **No Tags**: VM secrets don't support tags
2. **No Description/Purpose**: VM secrets lack semantic metadata fields
3. **Secret Container**: VM uses `Secret` object wrapper with `TenantEncrypted` flag; Database doesn't
4. **Audit Trail**: Database has `CreatedBy`/`LastUpdatedBy`; VM has `IsRotatable` instead
5. **Timestamps**: Database uses `LastUpdateTime`; VM uses `LastModified`
6. **Exposed Data**: Database exposes non-sensitive fields; VM wraps in Secret container

### CRUD Operation Differences

#### Create
| Aspect | Database | VM |
|--------|----------|-----|
| **Request Type** | `ArkSIADBAddSecret` | `ArkSIAVMAddSecret` |
| **Fields** | 10+ fields (description, purpose, store_type, secret-specific) | 8 fields (simpler) |
| **Validation** | SDK validates during creation | SDK validates during creation |
| **Status Code** | 201 Created | 201 Created |

#### Read
| Aspect | Database | VM |
|--------|----------|-----|
| **Method** | `Secret()` returns `ArkSIADBSecretMetadata` | `Secret()` returns `ArkSIAVMSecret` |
| **Exposed Data** | `SecretExposedData` map | Inside `Secret.SecretData` |
| **Metadata** | Full (CreatedBy, Tags, etc.) | Minimal (TenantID, IsRotatable) |

#### Update
| Aspect | Database | VM |
|--------|----------|-----|
| **Method** | `UpdateSecret()` with `ArkSIADBUpdateSecret` | `ChangeSecret()` with `ArkSIAVMChangeSecret` |
| **HTTP Method** | POST | POST |
| **Partial Updates** | Yes (only changed fields) | Yes (only changed fields) |
| **New Field Name** | `NewSecretName` | `SecretName` |
| **Credentials** | `Username`, `Password`, IAM fields | ProvisionerUser or PCloudAccount fields |

#### Delete
| Aspect | Database | VM |
|--------|----------|-----|
| **Method** | `DeleteSecret()` with `ArkSIADBDeleteSecret` | `DeleteSecret()` with `ArkSIAVMDeleteSecret` |
| **HTTP Method** | DELETE | DELETE |
| **Status Code** | 204 No Content | 204 No Content |
| **SDK Bug** | YES - nil body panic (requires workaround) | NO - works correctly |

**Important**: VM secrets DELETE does NOT have the nil body panic bug!

### List Operations

| Aspect | Database | VM |
|--------|----------|-----|
| **List All** | `ListSecrets()` → `ArkSIADBSecretMetadataList` | `ListSecrets()` → `[]*ArkSIAVMSecret` |
| **List With Filter** | `ListSecretsBy(filter)` | `ListSecretsBy(filter)` |
| **Filtering Approach** | Database-side (query parameters) | Client-side (local filtering) |
| **Stats Method** | Calculated from metadata | Calculated by iterating list |

---

## 5. Implementation Pattern Differences for Terraform Resource

### Secret Type-Based Validation

**Database Secrets Pattern** (from database_secret_resource.go):
- Use `ValidateConfig()` method for cross-field validation
- Check authentication_type and validate corresponding fields are set
- Prevent invalid field combinations

**VM Secrets Should Use**:
- Similar `ValidateConfig()` pattern
- For `ProvisionerUser`: Require both `provisioner_username` and `provisioner_password`
- For `PCloudAccount`: Require both `pcloud_account_safe` and `pcloud_account_name`
- Prevent mixing field types

### Credential Handling

**Database Secrets**:
- Three separate auth types: `local`, `domain`, `aws_iam`
- Corresponding field sets: username/password/domain, AWS IAM fields
- Complex validation matrix (3 types × 6 field groups)

**VM Secrets**:
- Two secret types: `ProvisionerUser`, `PCloudAccount`
- Simpler credential structure
- Two separate field sets (not overlapping)

### Update/Change Behavior

**Database Secrets**:
- Uses `UpdateSecret()` - explicit method with dedicated request type
- Can update metadata and credentials independently

**VM Secrets**:
- Uses `ChangeSecret()` - different method name
- Credentials updated together (both username + password, or both safe + name)
- Only sends non-empty fields

### Tag Handling

**Database Secrets**:
- Supports tags: `tags map[string]string`
- Schema includes tags attribute
- Terraform schema: `types.Map` for tags

**VM Secrets**:
- No tag support in SDK
- Don't include in Terraform schema
- No tag validation needed

### Status/Enabled Field

**Database Secrets**:
- `IsActive` boolean in response (read-only)
- No enable/disable in create/update

**VM Secrets**:
- `IsDisabled` in request (`IsActive` in response, inverse mapping)
- Can disable during creation: `is_disabled: true` → `is_active: false`
- Can toggle disabled status during update
- Has `IsRotatable` field (read-only)

---

## 6. Key Implementation Considerations for Terraform Resource

### 1. Secret Type-Specific Fields
- **Schema Design**: Use conditional fields like database_secret_resource.go does
- **ProvisionerUser Fields**:
  - `provisioner_username` (sensitive: false)
  - `provisioner_password` (sensitive: true)
- **PCloudAccount Fields**:
  - `pcloud_account_safe` (sensitive: false)
  - `pcloud_account_name` (sensitive: false)

### 2. Update Behavior
- Unlike database secrets, changing credentials requires BOTH fields
- Partial credential updates (e.g., just password) may not work
- Implementation should match ChangeSecret() behavior: both or nothing

### 3. Response Data Availability
- `CreationTime` and `LastModified` available in response
- `IsRotatable` available but likely read-only
- `SecretDetails` is flexible map - may want to expose as optional JSON or map

### 4. No Delete Workaround Needed
- **VM secrets DELETE is safe** - unlike database secrets
- Can use SDK method directly: `siaAPI.SecretsVM().DeleteSecret(deleteReq)`
- No need for `delete_workarounds.go` adaptation

### 5. Filtering Not Used by Provider
- VM secrets filtering is client-side and incomplete
- For Terraform resource, filtering not needed
- Import would need to search by name manually if needed

### 6. Sensitive Data Handling
```go
// Passwords should be marked Sensitive: true in schema
ProvisionerPassword: schema.StringAttribute{
    Sensitive: true,  // CRITICAL
}

// Never log these fields
tflog.Debug(ctx, "Creating secret",
    map[string]interface{}{
        "name": plan.Name.ValueString(),
        // DON'T log plan.ProvisionerPassword
    },
)
```

### 7. State Management
- SecretDetails (map[string]interface{}) needs careful handling
- During import, SecretDetails from API should be preserved
- Consider whether to expose as JSON string or structured map in schema

---

## 7. Direct Comparison: Database vs VM Secret Creation

### Database Secret Create Request (Terraform → SDK)
```hcl
# Terraform resource
resource "cyberarksia_database_secret" "example" {
  name                  = "db-secret"
  authentication_type   = "local"
  username              = "admin"
  password              = "password123"
  tags = {
    env = "prod"
  }
}
```

**Maps to SDK**:
```go
addSecretReq := &secretsmodels.ArkSIADBAddSecret{
    SecretName: "db-secret",
    SecretType: "username_password",
    Username: "admin",
    Password: "password123",
    Tags: map[string]string{"env": "prod"},
}
```

### VM Secret Create Request (Terraform → SDK) - EXPECTED PATTERN

```hcl
# Terraform resource (expected)
resource "cyberarksia_vm_secret" "example" {
  name                      = "vm-secret"
  secret_type               = "ProvisionerUser"
  provisioner_username      = "provisioner"
  provisioner_password      = "secret_password"
  secret_details = {
    description = "Provisioner for VM management"
  }
}
```

**Maps to SDK**:
```go
addSecretReq := &vmsecretsmodels.ArkSIAVMAddSecret{
    SecretName: "vm-secret",
    SecretType: "ProvisionerUser",
    ProvisionerUsername: "provisioner",
    ProvisionerPassword: "secret_password",
    SecretDetails: map[string]interface{}{
        "description": "Provisioner for VM management",
    },
}
```

---

## 8. JSON Tag Analysis Summary

### Field Name Patterns Across Structures

**Snake Case Consistently Used**:
- All JSON tags use snake_case: `secret_id`, `secret_name`, `secret_type`, `is_active`
- Mapstructure tags mirror JSON tags
- Flag tags convert to CLI flags: `--secret-id`, `--secret-name`

**Omitempty Strategy**:
- Required fields: NO `omitempty`
- Optional fields: Include `omitempty`
- Computed fields: Include `omitempty`

**Example Field Mapping**:
```go
// From ArkSIAVMAddSecret
SecretName string `json:"secret_name,omitempty" mapstructure:"secret_name,omitempty" flag:"secret-name"`

// Maps to:
// - JSON: "secret_name"
// - Mapstructure: "secret_name"
// - CLI Flag: --secret-name
// - Omitted from JSON if empty
```

---

## Summary Table: All VM Secrets Structures

| Structure | Purpose | Used In | Key Fields Count |
|-----------|---------|---------|-----------------|
| `ArkSIAVMSecret` | Full secret response | Read, List | 10 fields |
| `ArkSIAVMAddSecret` | Create request | Create | 8 fields |
| `ArkSIAVMChangeSecret` | Update request | Update | 8 fields |
| `ArkSIAVMGetSecret` | Read request | Read | 1 field (ID only) |
| `ArkSIAVMDeleteSecret` | Delete request | Delete | 1 field (ID only) |
| `ArkSIAVMSecretInfo` | Brief info | List (alternate) | 6 fields |
| `ArkSIAVMSecretsFilter` | Filter criteria | ListBy | 4 fields |
| `ArkSIAVMSecretsStats` | Aggregated stats | Stats | 4 fields |

---

## Critical Implementation Notes

1. **SDK Stability**: VM secrets DELETE is stable (no nil body bug like database secrets)
2. **Method Names**: Use `ChangeSecret()` not `UpdateSecret()` - different from DB secrets
3. **HTTP Method**: Both Create and Update use POST
4. **Inverse Field**: `IsDisabled` in request, `IsActive` in response
5. **Type-Specific Credentials**: Can't mix ProvisionerUser and PCloudAccount fields
6. **No Tags**: Don't include tags schema (not supported by VM secrets SDK)
7. **Simple Types**: Only 2 secret types vs database's 4+ types
8. **No Audit Trail**: `CreatedBy`/`LastUpdatedBy` not available
9. **Client-Side Filtering**: ListSecretsBy does local filtering (doesn't matter for resource)
10. **Timestamp Field Name**: `LastModified` not `LastUpdateTime` (differs from DB)
