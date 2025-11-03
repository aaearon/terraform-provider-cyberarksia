# VM Secrets Terraform Resource - Quick Implementation Reference

This is a quick reference for implementing `cyberarksia_vm_secret` resource based on the database_secret_resource.go pattern.

---

## 1. Exact Field Definitions (From SDK Models)

### ArkSIAVMSecret Response Structure
```
SecretID       string
TenantID       string
Secret         {SecretData: interface{}, TenantEncrypted: bool}
SecretType     string (enum: "ProvisionerUser", "PCloudAccount")
SecretDetails  map[string]interface{}
SecretName     string
IsActive       bool
IsRotatable    bool (read-only)
CreationTime   string
LastModified   string
```

### ArkSIAVMAddSecret Request Structure (Create)
```
SecretName               string (optional)
SecretDetails           map[string]interface{} (optional)
SecretType              string (required, enum: "ProvisionerUser", "PCloudAccount")
IsDisabled              bool (default: false)
ProvisionerUsername     string (required if SecretType == "ProvisionerUser")
ProvisionerPassword     string (required if SecretType == "ProvisionerUser")
PCloudAccountSafe       string (required if SecretType == "PCloudAccount")
PCloudAccountName       string (required if SecretType == "PCloudAccount")
```

### ArkSIAVMChangeSecret Request Structure (Update)
```
SecretID                string (required)
SecretName              string (optional)
SecretDetails           map[string]interface{} (optional)
IsDisabled              bool (optional)
ProvisionerUsername     string (optional, but needs password if updating)
ProvisionerPassword     string (optional, but needs username if updating)
PCloudAccountSafe       string (optional, but needs name if updating)
PCloudAccountName       string (optional, but needs safe if updating)
```

---

## 2. Terraform Schema Attributes

### Required Fields
- `name` → `SecretName` (1-255 characters)
- `secret_type` → `SecretType` (enum: "ProvisionerUser", "PCloudAccount")

### Type-Specific Conditional Fields

#### ProvisionerUser Type
- `provisioner_username` → `ProvisionerUsername` (required when secret_type="ProvisionerUser")
- `provisioner_password` → `ProvisionerPassword` (required when secret_type="ProvisionerUser", sensitive: true)

#### PCloudAccount Type
- `pcloud_account_safe` → `PCloudAccountSafe` (required when secret_type="PCloudAccount")
- `pcloud_account_name` → `PCloudAccountName` (required when secret_type="PCloudAccount")

### Optional Fields
- `secret_details` → `SecretDetails` (map[string]interface{}, optional)
- `is_disabled` → `IsDisabled` (boolean, default: false, maps to is_active: !is_disabled)

### Computed Fields
- `id` → `SecretID`
- `created_at` → `CreationTime`
- `last_modified` → `LastModified`
- `is_rotatable` → `IsRotatable` (read-only)

### NOT Supported (Unlike Database Secrets)
- ✗ `tags` - VM secrets don't support tags
- ✗ `description` - Not in VM secrets service
- ✗ `purpose` - Not in VM secrets service

---

## 3. Model Structure (Terraform)

```go
// VMSecretModel represents a VM secret resource in Terraform
type VMSecretModel struct {
    // Computed attributes
    ID           types.String `tfsdk:"id"`
    CreatedAt    types.String `tfsdk:"created_at"`
    LastModified types.String `tfsdk:"last_modified"`
    IsRotatable  types.Bool   `tfsdk:"is_rotatable"`

    // Required attributes
    Name       types.String `tfsdk:"name"`
    SecretType types.String `tfsdk:"secret_type"`

    // ProvisionerUser Type Fields
    ProvisionerUsername  types.String `tfsdk:"provisioner_username"`
    ProvisionerPassword  types.String `tfsdk:"provisioner_password"`

    // PCloudAccount Type Fields
    PCloudAccountSafe    types.String `tfsdk:"pcloud_account_safe"`
    PCloudAccountName    types.String `tfsdk:"pcloud_account_name"`

    // Optional metadata
    SecretDetails types.Map `tfsdk:"secret_details"`
    IsDisabled    types.Bool `tfsdk:"is_disabled"`
}
```

---

## 4. Service Method Calls

### Create (AddSecret)
```go
import vmsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm/models"

addSecretReq := &vmsecretsmodels.ArkSIAVMAddSecret{
    SecretName: plan.Name.ValueString(),
    SecretType: plan.SecretType.ValueString(),
    IsDisabled: plan.IsDisabled.ValueBool(),
    ProvisionerUsername: plan.ProvisionerUsername.ValueString(),
    ProvisionerPassword: plan.ProvisionerPassword.ValueString(),
    PCloudAccountSafe:   plan.PCloudAccountSafe.ValueString(),
    PCloudAccountName:   plan.PCloudAccountName.ValueString(),
    SecretDetails:       secretDetailsMap, // map[string]interface{}
}

secret, err := r.providerData.SIAAPI.SecretsVM().AddSecret(addSecretReq)
```

### Read (Secret/Get)
```go
getSecretReq := &vmsecretsmodels.ArkSIAVMGetSecret{
    SecretID: state.ID.ValueString(),
}

secret, err := r.providerData.SIAAPI.SecretsVM().Secret(getSecretReq)
```

### Update (ChangeSecret)
```go
// CRITICAL: ARK SDK v1.5.0 ChangeSecret has POST→PUT bug on line 153
// SDK uses client.Post() instead of client.Put() causing updates to fail
// MUST use workaround until ARK SDK v1.6.0+

changeSecretReq := &vmsecretsmodels.ArkSIAVMChangeSecret{
    SecretID:             state.ID.ValueString(),
    SecretName:           plan.Name.ValueString(),      // Updatable via PUT (SDK POST bug prevents this)
    IsDisabled:           plan.IsDisabled.ValueBool(),
    ProvisionerUsername:  plan.ProvisionerUsername.ValueString(),
    ProvisionerPassword:  plan.ProvisionerPassword.ValueString(),
    PCloudAccountSafe:    plan.PCloudAccountSafe.ValueString(),
    PCloudAccountName:    plan.PCloudAccountName.ValueString(),
    SecretDetails:        secretDetailsMap,
}

// ✅ CORRECT - Use workaround (PUT to /api/secrets/%s)
secret, err := client.ChangeVMSecretDirect(ctx, r.providerData.AuthContext, changeSecretReq)

// ❌ WRONG - SDK bug causes POST instead of PUT (line 153)
// secret, err := r.providerData.SIAAPI.SecretsVM().ChangeSecret(changeSecretReq)
```

### Delete (DeleteSecret)
```go
// CRITICAL: ARK SDK v1.5.0 DELETE has nil body panic bug - MUST use workaround!
// SDK passes nil body to doRequest() causing panic (same bug as database secrets)
// MUST use workaround until ARK SDK v1.6.0+

// ✅ CORRECT - Use workaround (DELETE to /api/secrets/%s with empty map body)
err := client.DeleteVMSecretDirect(ctx, r.providerData.AuthContext, state.ID.ValueString())

// ❌ WRONG - Will panic (nil body bug)
// deleteSecretReq := &vmsecretsmodels.ArkSIAVMDeleteSecret{
//     SecretID: state.ID.ValueString(),
// }
// err := r.providerData.SIAAPI.SecretsVM().DeleteSecret(deleteSecretReq)
```

---

## 5. Request/Response JSON Examples

### Create Request (ProvisionerUser)
```json
{
  "secret_name": "my-provisioner",
  "secret_type": "ProvisionerUser",
  "secret": {
    "tenant_encrypted": false,
    "secret_data": {
      "username": "provisioner_user",
      "password": "secure_password"
    }
  },
  "is_active": true,
  "secret_details": {
    "description": "VM provisioner account"
  }
}
```

### Create Request (PCloudAccount)
```json
{
  "secret_name": "power-cloud-secret",
  "secret_type": "PCloudAccount",
  "secret": {
    "tenant_encrypted": false,
    "secret_data": {
      "safe": "PowerCloud",
      "account_name": "pcloud_service_acct"
    }
  },
  "is_active": true,
  "secret_details": {}
}
```

### Response (Get/Read)
```json
{
  "secret_id": "12345678-1234-1234-1234-123456789012",
  "tenant_id": "tenant-123",
  "secret_type": "ProvisionerUser",
  "secret_name": "my-provisioner",
  "secret": {
    "secret_data": {
      "username": "provisioner_user",
      "password": "<encrypted>"
    },
    "tenant_encrypted": false
  },
  "secret_details": {
    "description": "VM provisioner account"
  },
  "is_active": true,
  "is_rotatable": false,
  "creation_time": "2025-11-02T15:30:00Z",
  "last_modified": "2025-11-02T15:30:00Z"
}
```

### Update Request (Partial)
```json
{
  "is_active": true,
  "secret_name": "my-provisioner-updated"
}
```

---

## 6. Validation Logic (ValidateConfig)

### Secret Type Validation
```go
func (r *vmSecretResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var config models.VMSecretModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
    if resp.Diagnostics.HasError() {
        return
    }

    secretType := config.SecretType.ValueString()

    switch secretType {
    case "ProvisionerUser":
        // Both fields required
        if config.ProvisionerUsername.IsNull() {
            resp.Diagnostics.AddAttributeError(
                path.Root("provisioner_username"),
                "Missing Required Field",
                "provisioner_username is required when secret_type=ProvisionerUser",
            )
        }
        if config.ProvisionerPassword.IsNull() {
            resp.Diagnostics.AddAttributeError(
                path.Root("provisioner_password"),
                "Missing Required Field",
                "provisioner_password is required when secret_type=ProvisionerUser",
            )
        }

        // Prevent PCloud fields
        if !config.PCloudAccountSafe.IsNull() && !config.PCloudAccountSafe.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                path.Root("pcloud_account_safe"),
                "Invalid Field Combination",
                "pcloud_account_safe cannot be set when secret_type=ProvisionerUser",
            )
        }
        if !config.PCloudAccountName.IsNull() && !config.PCloudAccountName.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                path.Root("pcloud_account_name"),
                "Invalid Field Combination",
                "pcloud_account_name cannot be set when secret_type=ProvisionerUser",
            )
        }

    case "PCloudAccount":
        // Both fields required
        if config.PCloudAccountSafe.IsNull() {
            resp.Diagnostics.AddAttributeError(
                path.Root("pcloud_account_safe"),
                "Missing Required Field",
                "pcloud_account_safe is required when secret_type=PCloudAccount",
            )
        }
        if config.PCloudAccountName.IsNull() {
            resp.Diagnostics.AddAttributeError(
                path.Root("pcloud_account_name"),
                "Missing Required Field",
                "pcloud_account_name is required when secret_type=PCloudAccount",
            )
        }

        // Prevent Provisioner fields
        if !config.ProvisionerUsername.IsNull() && !config.ProvisionerUsername.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                path.Root("provisioner_username"),
                "Invalid Field Combination",
                "provisioner_username cannot be set when secret_type=PCloudAccount",
            )
        }
        if !config.ProvisionerPassword.IsNull() && !config.ProvisionerPassword.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                path.Root("provisioner_password"),
                "Invalid Field Combination",
                "provisioner_password cannot be set when secret_type=PCloudAccount",
            )
        }
    }
}
```

---

## 7. Error Handling

### Use Standard Client Functions
```go
import "github.com/aaearon/terraform-provider-cyberark-sia/internal/client"

// Wrap API calls with retry
err := client.RetryWithBackoff(ctx, &client.RetryConfig{
    MaxRetries: client.DefaultMaxRetries,
    BaseDelay:  client.BaseDelay,
    MaxDelay:   client.MaxDelay,
}, func() error {
    var apiErr error
    secret, apiErr = r.providerData.SIAAPI.SecretsVM().AddSecret(addSecretReq)
    return apiErr
})

// Check for 404 (deleted externally)
if client.IsNotFoundError(err) {
    resp.State.RemoveResource(ctx)
    return
}

// Map SDK errors to Terraform diagnostics
if err != nil {
    resp.Diagnostics.Append(client.MapError(err, "create vm secret"))
    return
}
```

---

## 8. Secret Details Handling

### Convert Terraform types.Map → map[string]interface{}

```go
var secretDetailsMap map[string]interface{}
if !plan.SecretDetails.IsNull() && !plan.SecretDetails.IsUnknown() {
    diag := plan.SecretDetails.ElementsAs(ctx, &secretDetailsMap, false)
    if diag.HasError() {
        resp.Diagnostics.Append(diag...)
        return
    }
}
if secretDetailsMap == nil {
    secretDetailsMap = map[string]interface{}{}
}
```

### Convert map[string]interface{} → Terraform types.Map

```go
if len(secret.SecretDetails) > 0 {
    detailsMap, diag := types.MapValueFrom(ctx, types.StringType, secret.SecretDetails)
    if diag.HasError() {
        resp.Diagnostics.Append(diag...)
        return
    }
    state.SecretDetails = detailsMap
} else {
    state.SecretDetails = types.MapNull(types.StringType)
}
```

---

## 9. Logging (Never Log Sensitive Data!)

```go
// CORRECT: Only log non-sensitive fields
tflog.Info(ctx, "Creating VM secret", map[string]interface{}{
    "name":       plan.Name.ValueString(),
    "secret_type": plan.SecretType.ValueString(),
})

// WRONG: Never log passwords or account names
// tflog.Debug(ctx, "Secret details",
//     map[string]interface{}{
//         "password": plan.ProvisionerPassword.ValueString(),
//     },
// )

// For updates, log only what changed (non-sensitive)
tflog.Info(ctx, "Updating VM secret", map[string]interface{}{
    "id": state.ID.ValueString(),
})
```

---

## 10. Key Differences from Database Secrets (Checklist)

- [ ] Use `SecretsVM()` not `SecretsDB()`
- [ ] Use `ChangeSecret()` not `UpdateSecret()` for updates
- [ ] Import from `vmsecretsmodels` not `secretsmodels`
- [ ] Handle `IsDisabled` field (request) → `IsActive` field (response) inverse mapping
- [ ] Support 2 secret types only: `ProvisionerUser`, `PCloudAccount`
- [ ] No tags support - remove from schema
- [ ] No description/purpose fields - don't add to schema
- [x] **WORKAROUND REQUIRED**: Use `client.DeleteVMSecretDirect()` for DELETE (nil body panic bug)
- [x] **WORKAROUND REQUIRED**: Use `client.ChangeVMSecretDirect()` for UPDATE (POST→PUT bug on line 153)
- [ ] Field name in response is `LastModified` not `LastUpdateTime`
- [ ] No audit trail fields (`CreatedBy`, `LastUpdatedBy`)
- [ ] Mark `provisioner_password` as `Sensitive: true` in schema
- [ ] ValidateConfig() enforces secret_type-specific field combinations
- [ ] Credentials must be updated together (both username + password, or both safe + name)

---

## 11. File Structure

```
terraform-provider-cyberarksia/
├── internal/
│   ├── models/
│   │   └── vm_secret.go              # VMSecretModel struct
│   └── provider/
│       ├── vm_secret_resource.go     # Resource implementation
│       └── vm_secret_resource_test.go  # Acceptance tests
├── examples/
│   └── resources/
│       ├── cyberarksia_vm_secret/
│       │   ├── resource.tf           # Basic example
│       │   └── resource-complete.tf  # Complete example
│       └── testing/
│           └── crud-test-vm-secret.tf  # CRUD testing template
└── docs/
    ├── resources/
    │   └── vm_secret.md              # Generated resource docs
    └── development/
        └── ark-sdk-vm-secrets-service-analysis.md  # This file
```

---

## 12. Testing Template (CRUD Validation)

See `examples/testing/crud-test-vm-secret.tf` for complete template.

```hcl
# Create
resource "cyberarksia_vm_secret" "test_provisioner" {
  name                      = "test-provisioner-${random_id.test.hex}"
  secret_type               = "ProvisionerUser"
  provisioner_username      = "provisioner"
  provisioner_password      = "TempPassword123!"
  secret_details = {
    environment = "test"
  }
}

# Read
output "secret_id" {
  value = cyberarksia_vm_secret.test_provisioner.id
}

# Update
resource "cyberarksia_vm_secret" "test_provisioner_updated" {
  name                      = "test-provisioner-updated-${random_id.test.hex}"
  secret_type               = "ProvisionerUser"
  provisioner_username      = "new_provisioner"
  provisioner_password      = "NewPassword456!"
  secret_details = {
    environment = "updated"
  }
}

# Delete (via destroy)
# terraform destroy -target=cyberarksia_vm_secret.test_provisioner
```

---

## Summary: Quick Checklist for Implementation

1. Create `internal/models/vm_secret.go` with `VMSecretModel` struct
2. Create `internal/provider/vm_secret_resource.go` resource implementation
3. Add model to schema with:
   - Required: `name`, `secret_type`
   - Conditional: `provisioner_username`/`provisioner_password` OR `pcloud_account_safe`/`pcloud_account_name`
   - Optional: `secret_details`, `is_disabled`
   - Computed: `id`, `created_at`, `last_modified`, `is_rotatable`
4. Implement CRUD methods with proper error handling and retry logic
5. Add ValidateConfig() for secret_type validation
6. Mark passwords as `Sensitive: true`
7. Never log sensitive data
8. Use `SecretsVM()` service, not `SecretsDB()`
9. Delete operation doesn't need workaround (SDK is stable)
10. Create acceptance tests with examples in `examples/resources/cyberarksia_vm_secret/`
