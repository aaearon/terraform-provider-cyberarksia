# Data Model: Target Set Resource

**Resource Name**: `cyberarksia_target_set`
**Date**: 2025-11-08
**Source**: Derived from feature specification and API investigation

## Schema Definition

### Terraform Schema Structure

```go
func (r *TargetSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "Manages a VM/server target set in CyberArk Secure Infrastructure Access (SIA). " +
            "Target sets define logical groupings of virtual machines and servers that share common " +
            "access credentials for Just-In-Time (JIT) privileged access.",

        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed:            true,
                MarkdownDescription: "Identifier of the target set (equals name).",
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Name of the target set. Must be unique across the SIA tenant. " +
                    "Avoid using forward slashes as they cause deletion issues.",
                Validators: []validator.String{
                    stringvalidator.LengthAtLeast(1),
                    validators.NoForwardSlashes(), // Custom validator - warning only
                },
            },
            "type": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Type of matching pattern: `Domain` (matches all servers in a domain), " +
                    "`Suffix` (matches servers with hostname suffix), or `Target` (matches specific hostname).",
                Validators: []validator.String{
                    stringvalidator.OneOf("Domain", "Suffix", "Target"),
                },
            },
            "secret_id": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "ID of the VM secret containing credentials. " +
                    "Reference `cyberarksia_virtual_machine_secret.example.id` for proper dependency ordering.",
            },
            "secret_type": schema.StringAttribute{
                Required:            true,
                MarkdownDescription: "Type of VM secret: `ProvisionerUser` (username/password credentials) or " +
                    "`PCloudAccount` (PAM vault account reference).",
                Validators: []validator.String{
                    stringvalidator.OneOf("ProvisionerUser", "PCloudAccount"),
                },
            },
            "provision_format": schema.StringAttribute{
                Optional:            true,
                Computed:            true,
                Default:             stringdefault.StaticString("<user>-<session-guid>"),
                MarkdownDescription: "Template for ephemeral account names. " +
                    "Placeholders: `<user>` (requesting user), `<session-guid>` (unique session ID). " +
                    "Cannot be removed once set (maintains audit trail consistency).",
                PlanModifiers: []planmodifier.String{
                    planmodifiers.PreventClearing(), // Custom plan modifier
                },
            },
            "description": schema.StringAttribute{
                Optional:            true,
                MarkdownDescription: "Description of the target set.",
            },
            "enable_certificate_validation": schema.BoolAttribute{
                Optional:            true,
                Computed:            true,
                Default:             booldefault.StaticBool(true),
                MarkdownDescription: "Whether to enable TLS/SSL certificate validation for connections to target servers.",
            },
        },
    }
}
```

---

## Attribute Details

### id (Computed, String)

**Purpose**: Unique identifier for the target set

**Behavior**:
- Auto-computed to match `name` attribute (name-as-ID pattern)
- Updated automatically when `name` changes (rename support)
- Used in Terraform state for resource tracking

**API Mapping**: API response includes both `id` and `name` fields with identical values

**Plan Modifier**: `UseStateForUnknown()` - preserves ID during plan phase when name is known

**Validation**: None (computed value)

**Example**: `"prod.example.com"`

---

### name (Required, String)

**Purpose**: Display name and primary identifier for the target set

**Behavior**:
- Must be unique across all target sets in SIA tenant
- Can be changed (rename) without resource recreation
- Serves as URL path component for API calls

**API Mapping**:
- CREATE: `POST /api/targetsets` with `name` in body
- READ: `GET /api/targetsets/{name}` - name in URL path
- UPDATE: `PUT /api/targetsets/{old-name}` - old name in URL, new name in body
- DELETE: `DELETE /api/targetsets/{name}` - name in URL path

**Constraints**:
- Minimum length: 1 character
- Globally unique (API enforces with 400 Bad Request on duplicates)
- Forward slashes accepted by API but cause DELETE failures (403) - validator warns

**Validation**:
- `stringvalidator.LengthAtLeast(1)` - ensure non-empty
- `validators.NoForwardSlashes()` - custom warning validator

**Examples**:
- Domain: `"prod.example.com"`
- Suffix: `"dc1.example.com"`
- Target: `"server01.example.com"`

---

### type (Required, String)

**Purpose**: Defines matching pattern for server selection

**Allowed Values**:
- `"Domain"` - Matches all servers in specified domain (e.g., "example.com" matches any *.example.com)
- `"Suffix"` - Matches servers with specified hostname suffix (e.g., "dc1.example.com" matches *.dc1.example.com)
- `"Target"` - Matches specific server hostname (e.g., "server01.example.com" matches only that server)

**Behavior**:
- Mutable - can be changed without resource recreation
- All 6 bidirectional type changes validated in PoC (Target↔Domain, Target↔Suffix, Domain↔Suffix)

**API Mapping**: `type` field in request/response body

**Validation**: `stringvalidator.OneOf("Domain", "Suffix", "Target")`

**Default**: None (required field, no default value)

**Examples**:
```hcl
type = "Domain"  # Matches all servers in domain
type = "Suffix"  # Matches servers with hostname suffix
type = "Target"  # Matches specific server
```

---

### secret_id (Required, String)

**Purpose**: References the VM secret containing credentials for JIT access

**Format**: UUID string (e.g., `"aec8cf4b-8012-4efb-9aa2-ca14db5f79c0"`)

**Behavior**:
- Mutable - supports credential rotation without resource recreation
- No pre-flight validation (API accepts non-existent UUIDs)
- Terraform dependency graph handles ordering via resource references

**API Mapping**: `secret_id` field in request/response body

**Best Practice**: Reference another resource to create dependency
```hcl
secret_id = cyberarksia_virtual_machine_secret.admin.id
```

**Validation**: None (API performs validation at JIT access time, not configuration time)

**Provider Decision**: Enforce as Required in schema (target set non-functional without credentials)

---

### secret_type (Required, String)

**Purpose**: Specifies the type of VM secret referenced by `secret_id`

**Allowed Values**:
- `"ProvisionerUser"` - Username/password credentials
- `"PCloudAccount"` - Reference to PAM vault account

**Behavior**:
- Mutable - can be changed during credential rotation
- API does not validate type matches actual secret (accepts any value)

**API Mapping**: `secret_type` field in request/response body

**Validation**: `stringvalidator.OneOf("ProvisionerUser", "PCloudAccount")`

**Best Practice**: Match the secret's actual type
```hcl
secret_type = cyberarksia_virtual_machine_secret.admin.secret_type
```

**Provider Decision**: Enforce as Required (API allows omission, but target set non-functional without it)

---

### provision_format (Optional, Computed, String)

**Purpose**: Template for generating ephemeral account names during JIT sessions

**Default**: `"<user>-<session-guid>"`

**Placeholders**:
- `<user>` - Replaced with requesting user's username
- `<session-guid>` - Replaced with unique session identifier

**Behavior**:
- Can be set initially or added later
- Can be updated to new value
- **CANNOT be removed once set** (API PATCH semantics preserve existing value)

**API Mapping**: `provision_format` field in request/response body

**Constraint**: PATCH semantics - sending empty string or omitting field preserves existing value

**Validation**:
- No format validation (API accepts any string including invalid placeholders)
- Custom plan modifier prevents clearing once set

**Plan Modifier**: `planmodifiers.PreventClearing()` - errors at plan time if user attempts to clear

**Error Message** (when clearing attempted):
```
Cannot clear provision_format once set due to API limitations.
You can update it to a different value, but cannot clear it entirely.
```

**Examples**:
```hcl
provision_format = "<user>-<session-guid>"          # Standard
provision_format = "jit-<user>-<session-guid>"      # Custom prefix
provision_format = "<user>-custom-<session-guid>"   # Custom infix
```

---

### description (Optional, String)

**Purpose**: Human-readable description for documentation

**Behavior**:
- Fully mutable
- Not displayed in CyberArk SIA UI (metadata only)
- Useful for Terraform configuration documentation

**API Mapping**: `description` field in request/response body

**Default**: `null` (omitted from API request if not specified)

**Validation**: None

**Examples**:
```hcl
description = "Production environment servers"
description = "Development servers - US West region"
```

---

### enable_certificate_validation (Optional, Computed, Bool)

**Purpose**: Controls TLS/SSL certificate validation for target server connections

**Default**: `true` (secure default)

**Behavior**:
- Mutable - can be toggled without resource recreation
- When `false`, API response omits field (omitempty behavior)

**API Mapping**: `enable_certificate_validation` field in request/response body

**Use Case**: Disable for development/testing environments with self-signed certificates

**Validation**: None (boolean type)

**Examples**:
```hcl
enable_certificate_validation = true   # Validate certificates (default)
enable_certificate_validation = false  # Skip validation (development only)
```

---

## State Transitions

### Initial Creation

**Transition**: None → Exists

**Required Inputs**:
- `name` (unique identifier)
- `type` (matching pattern)
- `secret_id` (credentials reference)
- `secret_type` (credentials type)

**Optional Inputs**:
- `provision_format` (defaults to `"<user>-<session-guid>"`)
- `description` (defaults to null)
- `enable_certificate_validation` (defaults to `true`)

**Computed Outputs**:
- `id` (set to match `name`)

**API Call**: `POST /api/targetsets`

---

### Rename

**Transition**: Name change (e.g., "old-name" → "new-name")

**Behavior**:
- UPDATE operation with old name in URL path, new name in request body
- ID automatically updated to match new name
- No resource recreation (in-place update)
- Old name immediately becomes unavailable (404 on lookup)

**API Call**: `PUT /api/targetsets/{old-name}` with `{"name": "new-name", ...}`

**Validation**: New name must be unique (API returns 400 if duplicate)

---

### Type Change

**Transition**: Matching pattern change (e.g., "Domain" → "Suffix")

**Supported Transitions**:
- Domain ↔ Suffix
- Domain ↔ Target
- Suffix ↔ Target

**Behavior**:
- In-place update (no resource recreation)
- All 6 bidirectional combinations validated in PoC
- No service interruption

**API Call**: `PUT /api/targetsets/{name}` with `{"type": "Suffix", ...}`

---

### Credential Rotation

**Transition**: Change `secret_id` and/or `secret_type`

**Behavior**:
- In-place update (no resource recreation)
- New credentials apply to future JIT sessions
- Active sessions continue using old credentials until expiration

**API Call**: `PUT /api/targetsets/{name}` with `{"secret_id": "new-uuid", "secret_type": "ProvisionerUser", ...}`

---

### Provision Format Updates

**Allowed Transitions**:
- null → "template" (initial set)
- "old-template" → "new-template" (update)

**Blocked Transitions**:
- "template" → null (clearing blocked by plan modifier)
- "template" → "" (empty string - API preserves existing value anyway)

**API Behavior**: PATCH semantics preserve omitted fields

---

### Drift Detection

**External Modification**:
- Detected by READ returning different values than state
- Terraform prompts for import or update

**External Deletion**:
- READ returns 404
- Resource removed from state automatically

**Manual Drift Resolution**:
- `terraform refresh` - updates state to match remote
- `terraform apply` - updates remote to match config

---

## Validation Rules

### Schema-Level Validation

| Field | Validator | Purpose |
|-------|-----------|---------|
| `name` | `stringvalidator.LengthAtLeast(1)` | Ensure non-empty name |
| `name` | `validators.NoForwardSlashes()` | Warn about DELETE issues |
| `type` | `stringvalidator.OneOf(...)` | Enforce enum values |
| `secret_type` | `stringvalidator.OneOf(...)` | Enforce enum values |

### Plan-Level Validation

| Field | Plan Modifier | Purpose |
|-------|---------------|---------|
| `id` | `UseStateForUnknown()` | Preserve ID during plan phase |
| `provision_format` | `PreventClearing()` | Block clearing once set |

### API-Level Validation

| Check | API Behavior | Provider Handling |
|-------|--------------|-------------------|
| Name uniqueness | 400 Bad Request | Propagate error to user |
| Secret existence | No validation | Document best practice, let API validate at runtime |
| Type enum | No validation | Provider enforces via schema |

---

## Data Model Diagram

```
Target Set (cyberarksia_target_set)
├── id (computed: equals name)
├── name (required, unique, mutable)
├── type (required, enum: Domain/Suffix/Target, mutable)
├── secret_id (required, UUID, mutable)
├── secret_type (required, enum: ProvisionerUser/PCloudAccount, mutable)
├── provision_format (optional, computed, default, non-clearable)
├── description (optional, mutable)
└── enable_certificate_validation (optional, computed, default, mutable)

Relationships:
- Target Set --[references]--> VM Secret (by secret_id)
- VM Policy --[references]--> Target Set (by name) [future]
```

---

## Model Struct

```go
type TargetSetModel struct {
    ID                           types.String `tfsdk:"id"`
    Name                         types.String `tfsdk:"name"`
    Type                         types.String `tfsdk:"type"`
    SecretID                     types.String `tfsdk:"secret_id"`
    SecretType                   types.String `tfsdk:"secret_type"`
    ProvisionFormat              types.String `tfsdk:"provision_format"`
    Description                  types.String `tfsdk:"description"`
    EnableCertificateValidation  types.Bool   `tfsdk:"enable_certificate_validation"`
}
```

---

## API Request/Response Mapping

### CREATE Request

```go
addRequest := &targetsets.ArkSIAAddTargetSet{
    Name:                        data.Name.ValueString(),
    Type:                        data.Type.ValueString(),
    SecretID:                    data.SecretID.ValueString(),
    SecretType:                  data.SecretType.ValueString(),
    ProvisionFormat:             data.ProvisionFormat.ValueString(),
    Description:                 data.Description.ValueString(),
    EnableCertificateValidation: data.EnableCertificateValidation.ValueBool(),
}
```

### UPDATE Request

```go
updateRequest := &targetsets.ArkSIAUpdateTargetSet{
    ID:                          state.Name.ValueString(), // OLD name in URL
    Name:                        plan.Name.ValueString(),   // NEW name (may differ)
    Type:                        plan.Type.ValueString(),
    SecretID:                    plan.SecretID.ValueString(),
    SecretType:                  plan.SecretType.ValueString(),
    ProvisionFormat:             plan.ProvisionFormat.ValueString(),
    Description:                 plan.Description.ValueString(),
    EnableCertificateValidation: plan.EnableCertificateValidation.ValueBool(),
}
```

**CRITICAL**: Always include `Name` field in UPDATE to avoid destructive API bug (returns 500 and deletes resource)

### READ Response Mapping

```go
// API returns: {id: "name", name: "name", type: "Domain", ...}
data.ID = types.StringValue(targetSet.Name)
data.Name = types.StringValue(targetSet.Name)
data.Type = types.StringValue(targetSet.Type)
data.SecretID = types.StringValue(targetSet.SecretID)
data.SecretType = types.StringValue(targetSet.SecretType)
data.ProvisionFormat = types.StringValue(targetSet.ProvisionFormat)
data.Description = types.StringValue(targetSet.Description)
data.EnableCertificateValidation = types.BoolValue(targetSet.EnableCertificateValidation)
```

---

## Testing Strategy

### Unit Tests

**Test Coverage**:
- Custom validator (`NoForwardSlashes`) - warning generation
- Custom plan modifier (`PreventClearing`) - error on clear attempt
- Model struct field mappings

**Test Files**:
- `internal/validators/target_set_name_validator_test.go`
- `internal/planmodifiers/prevent_clearing_modifier_test.go`

### Acceptance Tests

**Test File**: `internal/provider/target_set_resource_test.go`

**Test Cases**:

1. **TestAccTargetSet_basic** - Full CRUD lifecycle
   - Create domain-based target set
   - Read and verify all attributes
   - Update description
   - Delete and verify removal

2. **TestAccTargetSet_rename** - Name change handling
   - Create target set with name "test-a"
   - Rename to "test-b"
   - Verify ID updated to "test-b"
   - Verify old name returns 404

3. **TestAccTargetSet_typeChange** - Type mutability
   - Create with type=Domain
   - Update to type=Suffix
   - Verify no resource recreation
   - Update to type=Target
   - Verify no resource recreation

4. **TestAccTargetSet_credentialRotation** - Secret updates
   - Create with secret A
   - Update to secret B
   - Verify secret_id and secret_type changed
   - Verify no resource recreation

5. **TestAccTargetSet_provisionFormat** - Format handling
   - Create without provision_format (uses default)
   - Update to add custom format
   - Verify format stored
   - Attempt to clear (expect plan error)

6. **TestAccTargetSet_import** - Import functionality
   - Manually create target set via API
   - Import into Terraform state
   - Run plan (expect no changes)

7. **TestAccTargetSet_drift** - External modification detection
   - Create target set
   - Manually modify via API
   - Run plan (expect drift detected)
   - Apply to resolve drift

8. **TestAccTargetSet_forwardSlash** - Name validation warning
   - Create with name containing forward slash
   - Verify warning displayed
   - Verify resource created
   - Verify DELETE fails with 403

### CRUD Validation

**Template**: `examples/testing/crud-test-target-set.tf`

**Workflow** (per `TESTING-GUIDE.md`):
1. Copy template to `/tmp/sia-crud-validation-{timestamp}/`
2. Customize with test data (tenant-specific VM secret ID)
3. Run CREATE → verify validation outputs
4. Run READ → verify state matches config
5. Run UPDATE → verify changes applied
6. Run DELETE → verify cleanup successful

---

**Data Model Status**: ✅ COMPLETE
**Next Artifact**: API Contracts (contracts/target-set-api-contract.md)
