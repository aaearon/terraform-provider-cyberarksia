# Research: VM Access Policy Implementation

**Date**: 2025-11-16
**ARK SDK Version**: v1.5.0
**Validation Source**: `/tmp/vm-policies-poc/SPECKIT-VM-POLICIES-REFERENCE.md` (POC-tested + OpenAPI validated)

---

## 1. ARK SDK Model Mappings

### 1.1 Core Policy Structure

**Primary Model**: `ArkUAPSIAVMAccessPolicy`

**Package**: `github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/uap/sia/vm/models`

**File**: `ark_uap_sia_vm_access_policy.go`

```go
type ArkUAPSIAVMAccessPolicy struct {
    sia.ArkUAPSIACommonAccessPolicy `mapstructure:",squash"` // Embedded common policy fields
    Targets                         ArkUAPSIAVMPlatformTargets `json:"targets,omitempty"`
    Behavior                        ArkUAPSSIAVMBehavior       `json:"behavior,omitempty"`
}
```

**Embedded Common Fields** (from `ArkUAPSIACommonAccessPolicy`):
- `Metadata` - Policy metadata (name, description, status, etc.)
- `Principals` - Array of assigned principals (USER/GROUP/ROLE)
- `Conditions` - Time-based access conditions
- `DelegationClassification` - Server-computed (Restricted/Unrestricted)

**Serialization Behavior**:
- Converts to JSON with camelCase field names
- `Targets.Serialize()` requires `LocationType` parameter
- `Behavior.Serialize()` wraps SSH/RDP under `connectAs` object

---

### 1.2 Metadata Model

**Model**: `ArkUAPSIACommonAccessPolicyMetadata` (embedded in common policy)

**Key Fields**:

| Field | Type | Required | Default | Constraints | Terraform Mapping |
|-------|------|----------|---------|-------------|-------------------|
| `PolicyID` | string | Computed | API-generated | UUID | `policy_id` (computed) |
| `Name` | string | Required | N/A | 1-200 chars, unique | `name` (required, ForceNew) |
| `Description` | string | Optional | `""` | Max 200 chars | `description` (optional) |
| `TimeZone` | string | Optional | `"GMT"` | Max 50 chars | `time_zone` (optional, computed) |
| `PolicyTags` | []string | Optional | `[]` | Max 20 items, each max 50 chars | `tags` (optional, computed) |
| `TimeFrame` | object | Optional | unlimited | from/to timestamps | `time_frame` (optional nested block) |
| `CreatedBy` | object | Computed | API-generated | Read-only | `created_by` (computed) |
| `UpdatedBy` | object | Computed | API-generated | Read-only | `updated_by` (computed) |
| `PolicyEntitlement` | object | Required | See below | Nested object | See PolicyEntitlement |
| `Status` | object | Required | See below | Nested object | See Status |

**PolicyEntitlement Subobject**:
- `LocationType` (enum): "AWS", "Azure", "GCP", "FQDN/IP" **[REQUIRED, ForceNew]**
- `TargetCategory` (enum): "VM" (auto-set by SDK) **[Computed]**
- `PolicyType` (enum): "Recurring" (default) or "OnDemand" **[Optional, Computed]**

**Status Subobject**:
- `Status` (enum): "Active", "Suspended" (user-settable); "Expired", "Validating", "Error", "Warning" (server-managed) **[REQUIRED]**

---

### 1.3 Principal Model

**Model**: `ArkUAPPrincipal`

**Package**: `github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/uap/common/models`

```go
type ArkUAPPrincipal struct {
    ID                  string `json:"id"`                    // Required, max 40 chars, UUID format
    Name                string `json:"name"`                  // Required, 1-512 chars, pattern: [\w.\-+]+
    Type                string `json:"type"`                  // Required, enum: USER/GROUP/ROLE
    SourceDirectoryName string `json:"sourceDirectoryName"`   // Required for USER/GROUP, optional for ROLE
    SourceDirectoryID   string `json:"sourceDirectoryId"`     // Required for USER/GROUP, optional for ROLE
}
```

**Conditional Requirements**:
- `SourceDirectoryName` and `SourceDirectoryID` are REQUIRED when `Type` is "USER" or "GROUP"
- Both fields are OPTIONAL when `Type` is "ROLE"

**Validation Rules**:
- Principal uniqueness enforced by API (duplicate principals rejected with 400 error)
- POC Test: `phase3-results.json:5` - Duplicate USER principals fail

---

### 1.4 Conditions Model

**Model**: `ArkUAPSIAConditions` (from common models)

**Fields**:

| Field | Type | Required | Default | Constraints | Terraform Mapping |
|-------|------|----------|---------|-------------|-------------------|
| `MaxSessionDuration` | integer | Optional | 1 | 1-24 hours | `max_session_duration` (optional, computed) |
| `IdleTime` | integer | Optional | 10 | >0, max 120 minutes | `idle_time` (optional, computed) |
| `AccessWindow` | object | Optional | unlimited | See below | `access_window` (optional nested block) |

**AccessWindow Subobject**:
- `DaysOfTheWeek` ([]int, optional): 0-6 (0=Sunday, 6=Saturday), example: `[1,2,3,4,5]` for Monday-Friday
- `FromHour` (string, optional): Time format (e.g., "09:00"), pattern: `\w+`
- `ToHour` (string, optional): Time format (e.g., "17:00"), pattern: `\w+`

**TimeFrame** (in Metadata, not Conditions):
- `FromTime` (string, optional): ISO 8601 date-time (`yyyy-MM-ddTHH:mm:ss`)
- `ToTime` (string, optional): ISO 8601 date-time

**Server Validation**:
- `MaxSessionDuration = 0` rejected with 400 error (POC: phase4-results.json:23)
- `IdleTime = 0` may trigger server default (0 → 10), but spec says >0 required

---

### 1.5 Behavior Models

**Top-Level Model**: `ArkUAPSSIAVMBehavior`

**File**: `ark_uap_sia_vm_behavior.go`

```go
type ArkUAPSSIAVMBehavior struct {
    SSHProfile *ArkUAPSSIAVMSSHProfile `json:"ssh_profile,omitempty"`
    RDPProfile *ArkUAPSSIAVMRDPProfile `json:"rdp_profile,omitempty"`
}
```

**SSH Profile Model**:
```go
type ArkUAPSSIAVMSSHProfile struct {
    Username string `json:"username"` // REQUIRED if SSH block present
}
```

**RDP Profile Model**:
```go
type ArkUAPSSIAVMRDPProfile struct {
    LocalEphemeralUser  *ArkUAPSSIAVMEphemeralUser       `json:"local_ephemeral_user,omitempty"`
    DomainEphemeralUser *ArkUAPSSIAVMDomainEphemeralUser `json:"domain_ephemeral_user,omitempty"`
}
```

**Local Ephemeral User** (for RDP):
```go
type ArkUAPSSIAVMEphemeralUser struct {
    AssignGroups                 []string `json:"assign_groups"`
    EnableEphemeralUserReconnect bool     `json:"enable_ephemeral_user_reconnect"`
}
```

**Domain Ephemeral User** (for RDP):
```go
type ArkUAPSSIAVMDomainEphemeralUser struct {
    ArkUAPSSIAVMEphemeralUser                // Embedded (has AssignGroups, EnableEphemeralUserReconnect)
    AssignDomainGroups            []string `json:"assign_domain_groups"`
}
```

**Serialization Quirk**: SDK transforms `ssh_profile` → `connectAs.ssh` and `rdp_profile` → `connectAs.rdp`

**Validation** (Server-Side):
- At least ONE profile required (SSH or RDP) - POC: phase1-results.json:5
- SSH username non-empty if SSH profile present - POC: phase1-results.json:33
- Both SSH + RDP can coexist - POC: phase1-results.json:27 SUCCESS

**NOTE - Domain Ephemeral User Status**:
- SDK fully supports `DomainEphemeralUser`
- OpenAPI spec only documents `localEphemeralUser` (domainEphemeralUser NOT in spec)
- **Recommendation**: Include in Terraform schema with documentation note "SDK-supported, OpenAPI undocumented"

---

### 1.6 Target Models

#### 1.6.1 Platform Targets Wrapper

**Model**: `ArkUAPSIAVMPlatformTargets`

**File**: `ark_uap_sia_vm_targets.go`

```go
type ArkUAPSIAVMPlatformTargets struct {
    FQDNIPResource *ArkUAPSIAVMFQDNIPResource `json:"fqdn_ip,omitempty"`
    AWSResource    *ArkUAPSIAVMAWSResource    `json:"aws,omitempty"`
    AzureResource  *ArkUAPSIAVMAzureResource  `json:"azure,omitempty"`
    GCPResource    *ArkUAPSIAVMGCPResource    `json:"gcp,omitempty"`
}
```

**CRITICAL CONSTRAINT**: `oneOf` - Exactly ONE location type per policy
- OpenAPI: `InfrastructureVirtualMachineTarget.oneOf`
- Terraform validator: `validators.ExactlyOneOf`

#### 1.6.2 FQDN/IP Location Type

**Model**: `ArkUAPSIAVMFQDNIPResource`

```go
type ArkUAPSIAVMFQDNIPResource struct {
    FQDNRules []ArkUAPSIAVMFQDNRule `json:"fqdn_rules,omitempty"` // OPTIONAL
    IPRules   []ArkUAPSIAVMIPRule   `json:"ip_rules,omitempty"`   // OPTIONAL
}

type ArkUAPSIAVMFQDNRule struct {
    Operator            string `json:"operator"`             // REQUIRED, enum
    ComputernamePattern string `json:"computername_pattern"` // REQUIRED, max 300 chars
    Domain              string `json:"domain,omitempty"`     // OPTIONAL, max 1000 chars
}

type ArkUAPSIAVMIPRule struct {
    Operator    string   `json:"operator"`     // REQUIRED, enum
    IPAddresses []string `json:"ip_addresses"` // REQUIRED, max 1000 items
    LogicalName string   `json:"logical_name"` // REQUIRED, 1-256 chars
}
```

**FQDN Operators** (SDK constants, lines 10-16):
- `EXACTLY` - Exact match
- `WILDCARD` - Wildcard match
- `PREFIX` - Starts with pattern
- `SUFFIX` - Ends with pattern
- `CONTAINS` - Contains pattern

**IP Operators** (SDK constants, lines 19-22):
- `EXACTLY` - Exact IP match
- `WILDCARD` - IP wildcard match

**Server Validation**:
- IP rules without `LogicalName` rejected (POC: phase2-results.json:23,29)
- Invalid operator "DOMAIN" rejected (POC: phase2-results.json:35)

#### 1.6.3 AWS Location Type

**Model**: `ArkUAPSIAVMAWSResource`

```go
type ArkUAPSIAVMAWSResource struct {
    Regions    []string               `json:"regions"`     // OPTIONAL
    Tags       []ArkUAPSIAVMKeyValTag `json:"tags"`        // OPTIONAL
    VPCIDs     []string               `json:"vpc_ids"`     // OPTIONAL (NOT required)
    AccountIDs []string               `json:"account_ids"` // OPTIONAL (NOT required)
}

type ArkUAPSIAVMKeyValTag struct {
    Key   string   `json:"key"`            // REQUIRED
    Value []string `json:"value,omitempty"` // OPTIONAL
}
```

**POC Validation**: phase5-results.json:3 - Empty arrays allowed for VPCIDs/AccountIDs

#### 1.6.4 GCP Location Type

**Model**: `ArkUAPSIAVMGCPResource`

```go
type ArkUAPSIAVMGCPResource struct {
    Regions  []string               `json:"regions"`   // OPTIONAL
    Labels   []ArkUAPSIAVMKeyValTag `json:"labels"`    // OPTIONAL (NOT Tags!)
    VPCIDs   []string               `json:"vpc_ids"`   // OPTIONAL
    Projects []string               `json:"projects"`  // OPTIONAL
}
```

**KEY DIFFERENCE**: GCP uses `Labels`, NOT `Tags` (different from AWS/Azure)

**POC Validation**: phase5-results.json:10 - Empty arrays allowed

#### 1.6.5 Azure Location Type

**Model**: `ArkUAPSIAVMAzureResource`

```go
type ArkUAPSIAVMAzureResource struct {
    Regions        []string               `json:"regions"`         // OPTIONAL
    Tags           []ArkUAPSIAVMKeyValTag `json:"tags"`            // OPTIONAL
    ResourceGroups []string               `json:"resource_groups"` // OPTIONAL
    VNetIDs        []string               `json:"vnet_ids"`        // OPTIONAL
    Subscriptions  []string               `json:"subscriptions"`   // OPTIONAL
}
```

**POC Status**: HTTP 500 errors during testing (phase5-azure-results.json) - may indicate server-side issue

---

## 2. VM Service API Contract

### 2.1 Service Initialization

**Service**: `ArkUAPSIAVMService`

**Package**: `github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/uap/sia/vm`

**Initialization**:
```go
import (
    "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm"
    uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
    uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
)

vmService, err := vm.NewArkUAPSIAVMService(ispAuth)
```

### 2.2 CREATE Operation

**Method Signature**:
```go
AddPolicy(policy *ArkUAPSIAVMAccessPolicy) (*ArkUAPSIAVMAccessPolicy, error)
```

**File**: `ark_uap_sia_vm_service.go:52-74`

**Behavior**:
- Auto-sets `PolicyEntitlement.TargetCategory = "VM"`
- Converts nil `PolicyTags` to empty array `[]`
- Serializes policy to JSON with camelCase conversion
- Returns created policy by calling `Policy()` after creation

**Usage Pattern**:
```go
policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{
    Metadata: /* ... */,
    Principals: principals,  // At least 1 required (schema-enforced)
    Targets: targets,
    Behavior: behavior,
    Conditions: conditions,
}

err := client.RetryWithBackoff(ctx, func() error {
    created, err := vmService.AddPolicy(policy)
    if err != nil {
        return err
    }
    // Store created.Metadata.PolicyID in state
    return nil
})
if err != nil {
    resp.Diagnostics.Append(client.MapError(err, "create VM policy")...)
    return
}
```

**Error Scenarios**:
- 400 Bad Request: Invalid field values, missing required fields
- 401 Unauthorized: Invalid/expired token
- 409 Conflict: Duplicate policy name

### 2.3 READ Operation

**Method Signature**:
```go
Policy(req *ArkUAPGetPolicyRequest) (*ArkUAPSIAVMAccessPolicy, error)
```

**File**: `ark_uap_sia_vm_service.go:76-91`

**Behavior**:
- Fetches policy by ID
- Deserializes JSON with snake_case conversion
- Returns full policy structure including ALL principals (inline + assigned)

**Usage Pattern**:
```go
policy, err := vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
    PolicyID: policyID,
})
if err != nil {
    // 404 = policy deleted externally (drift detection)
    if strings.Contains(err.Error(), "404") {
        resp.State.RemoveResource(ctx)
        return
    }
    resp.Diagnostics.Append(client.MapError(err, "read VM policy")...)
    return
}

// Map SDK model to Terraform state
state.PolicyID = types.StringValue(policy.Metadata.PolicyID)
state.Principals = /* convert policy.Principals to Terraform model */
```

**Error Scenarios**:
- 404 Not Found: Policy deleted (drift detection - remove from state)
- 401 Unauthorized: Invalid/expired token

### 2.4 UPDATE Operation

**Method Signature**:
```go
UpdatePolicy(policy *ArkUAPSIAVMAccessPolicy) (*ArkUAPSIAVMAccessPolicy, error)
```

**File**: `ark_uap_sia_vm_service.go:94-112`

**Behavior**:
- Serializes policy to JSON with camelCase conversion
- Updates entire policy object
- Returns updated policy by calling `Policy()` after update

**CRITICAL - Read-Modify-Write Pattern REQUIRED**:
```go
// 1. READ existing policy
existingPolicy, err := vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
    PolicyID: policyID,
})
if err != nil {
    return err
}

// 2. IDENTIFY inline principals from resource config
inlinePrincipalKeys := make(map[string]bool)
for _, p := range plan.Principals {
    key := fmt.Sprintf("%s:%s", p.PrincipalID, p.PrincipalType)
    inlinePrincipalKeys[key] = true
}

// 3. PRESERVE assigned principals (not in inline config)
preservedPrincipals := []uapcommonmodels.ArkUAPPrincipal{}
for _, p := range existingPolicy.Principals {
    key := fmt.Sprintf("%s:%s", p.ID, p.Type)
    if !inlinePrincipalKeys[key] {
        // This principal was added via assignment resource - preserve it
        preservedPrincipals = append(preservedPrincipals, p)
    }
}

// 4. BUILD new principals list: inline from plan + preserved assigned
newPrincipals := /* inline principals from plan.Principals */
newPrincipals = append(newPrincipals, preservedPrincipals...)

// 5. MODIFY other fields
existingPolicy.Metadata.Description = plan.Description.ValueString()
existingPolicy.Principals = newPrincipals

// 6. WRITE back
err = client.RetryWithBackoff(ctx, func() error {
    updated, err := vmService.UpdatePolicy(existingPolicy)
    return err
})
```

**Why Read-Modify-Write Required**:
- `UpdatePolicy()` replaces entire policy object
- Must preserve unmanaged principals (added via assignment resources)
- Must preserve unmanaged fields (computed metadata, etc.)

**Error Scenarios**:
- 404 Not Found: Policy deleted
- 400 Bad Request: Validation errors
- 409 Conflict: Name conflict if name changed

### 2.5 DELETE Operation

**Method Signature**:
```go
DeletePolicy(req *ArkUAPDeletePolicyRequest) error
```

**File**: `ark_uap_sia_vm_service.go:179-182`

**Implementation**: Use SDK method directly - **NO workaround needed**

**Usage Pattern**:
```go
err := vmService.DeletePolicy(&uapcommonmodels.ArkUAPDeletePolicyRequest{
    PolicyID: policyID,
})
if err != nil {
    // 404 = already deleted (drift detection - treat as success)
    if strings.Contains(err.Error(), "404") {
        return nil
    }
    resp.Diagnostics.Append(client.MapError(err, "delete VM policy")...)
    return
}
```

**Why No Workaround Needed**:
- VM policies use `BaseDeletePolicy()` from `pkg/services/uap/common/ark_uap_base_service.go`
- Calls `s.client.Delete(context.Background(), fmt.Sprintf(policyURL, policyID), nil)` correctly
- Common HTTP client handles `body == nil` without panic
- Different from SIA workspace/secret DELETE methods which have nil body panic bug

**Error Scenarios**:
- 404 Not Found: Already deleted (treat as success)
- 401 Unauthorized: Invalid/expired token

---

## 3. Validation Patterns

### 3.1 Schema-Level Required Fields

Use Terraform `Required: true` for OpenAPI `required` fields:

```go
"name": schema.StringAttribute{
    Required: true,
    Validators: []validator.String{
        stringvalidator.LengthBetween(1, 200),
    },
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),  // ForceNew
    },
},

"status": schema.StringAttribute{
    Required: true,
    Validators: []validator.String{
        stringvalidator.OneOf("Active", "Suspended"),
    },
},

"location_type": schema.StringAttribute{
    Required: true,
    Validators: []validator.String{
        stringvalidator.OneOf("AWS", "Azure", "GCP", "FQDN/IP"),
    },
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),  // Changing location type = new policy
    },
},
```

### 3.2 ExactlyOneOf Pattern for Location Types

**Requirement**: Enforce exactly ONE of: `fqdn_ip_targets`, `aws_targets`, `azure_targets`, `gcp_targets`

**Implementation** (in Schema method):
```go
// Use block-level validation
Blocks: map[string]schema.Block{
    "fqdn_ip_targets": schema.SingleNestedBlock{
        Validators: []validator.Object{
            objectvalidator.ExactlyOneOf(
                path.Root("aws_targets"),
                path.Root("azure_targets"),
                path.Root("gcp_targets"),
            ),
        },
        Attributes: /* ... */,
    },
    "aws_targets": schema.SingleNestedBlock{
        Validators: []validator.Object{
            objectvalidator.ExactlyOneOf(
                path.Root("fqdn_ip_targets"),
                path.Root("azure_targets"),
                path.Root("gcp_targets"),
            ),
        },
        Attributes: /* ... */,
    },
    // Repeat for azure_targets and gcp_targets
}
```

**Alternative** (custom validator):
```go
// In ValidateConfig method
func (r *VMPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var config VMPolicyResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

    locationTypeCount := 0
    if !config.FQDNIPTargets.IsNull() {
        locationTypeCount++
    }
    if !config.AWSTargets.IsNull() {
        locationTypeCount++
    }
    if !config.AzureTargets.IsNull() {
        locationTypeCount++
    }
    if !config.GCPTargets.IsNull() {
        locationTypeCount++
    }

    if locationTypeCount != 1 {
        resp.Diagnostics.AddError(
            "Invalid Location Type Configuration",
            "Exactly one location type must be specified: fqdn_ip_targets, aws_targets, azure_targets, or gcp_targets",
        )
    }
}
```

### 3.3 ListValidator.SizeAtLeast(1) for Principals

**Requirement**: Principals list must have minimum 1 element (schema-enforced)

```go
"principals": schema.ListNestedAttribute{
    Required: true,  // CRITICAL
    Validators: []validator.List{
        listvalidator.SizeAtLeast(1),
    },
    NestedObject: schema.NestedAttributeObject{
        Attributes: map[string]schema.Attribute{
            "principal_id": /* ... */,
            "principal_name": /* ... */,
            "principal_type": /* ... */,
        },
    },
},
```

### 3.4 Conditional Validators for Principals

**Requirement**: `source_directory_name` and `source_directory_id` required when `principal_type` = USER or GROUP

**Implementation Options**:

**Option 1 - Schema-level Optional, Runtime Validation**:
```go
// Schema
"source_directory_name": schema.StringAttribute{
    MarkdownDescription: "Source directory name. **Required** for USER and GROUP types.",
    Optional: true,  // Schema-level optional
},

// In ValidateConfig method
func validatePrincipalDirectoryFields(principals []PrincipalModel) diag.Diagnostics {
    var diags diag.Diagnostics
    for i, p := range principals {
        if p.PrincipalType.ValueString() == "USER" || p.PrincipalType.ValueString() == "GROUP" {
            if p.SourceDirectoryName.IsNull() || p.SourceDirectoryName.ValueString() == "" {
                diags.AddAttributeError(
                    path.Root("principals").AtListIndex(i).AtName("source_directory_name"),
                    "Missing Required Field",
                    fmt.Sprintf("source_directory_name is required for %s principals", p.PrincipalType.ValueString()),
                )
            }
            if p.SourceDirectoryID.IsNull() || p.SourceDirectoryID.ValueString() == "" {
                diags.AddAttributeError(
                    path.Root("principals").AtListIndex(i).AtName("source_directory_id"),
                    "Missing Required Field",
                    fmt.Sprintf("source_directory_id is required for %s principals", p.PrincipalType.ValueString()),
                )
            }
        }
    }
    return diags
}
```

**Option 2 - Custom Attribute Validator**:
```go
// Create custom validator: RequiredForPrincipalTypes
type requiredForPrincipalTypesValidator struct {
    principalTypes []string
}

func (v requiredForPrincipalTypesValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
    // Implementation checks sibling principal_type attribute
    // and validates if source_directory field is provided when required
}
```

### 3.5 Custom Enum Validators

**Extend existing pattern** from `internal/validators/`:

```go
// internal/validators/vm_validators.go

// LocationType validator
func LocationType() validator.String {
    return stringvalidator.OneOf("AWS", "Azure", "GCP", "FQDN/IP")
}

// FQDNOperator validator
func FQDNOperator() validator.String {
    return stringvalidator.OneOf("EXACTLY", "WILDCARD", "PREFIX", "SUFFIX", "CONTAINS")
}

// IPOperator validator
func IPOperator() validator.String {
    return stringvalidator.OneOf("EXACTLY", "WILDCARD")
}

// Status validator (user-settable values only)
func PolicyStatus() validator.String {
    return stringvalidator.OneOf("Active", "Suspended")
}
```

### 3.6 At Least One Connection Profile

**Requirement**: At least one of SSH or RDP behavior must be present

**Implementation** (in ValidateConfig):
```go
func (r *VMPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var config VMPolicyResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

    if config.Behavior.IsNull() {
        resp.Diagnostics.AddError(
            "Missing Required Configuration",
            "behavior block is required",
        )
        return
    }

    var behavior BehaviorModel
    resp.Diagnostics.Append(config.Behavior.As(ctx, &behavior, basetypes.ObjectAsOptions{})...)

    hasSSH := !behavior.SSH.IsNull()
    hasRDP := !behavior.RDP.IsNull()

    if !hasSSH && !hasRDP {
        resp.Diagnostics.AddError(
            "Invalid Behavior Configuration",
            "At least one connection profile (ssh or rdp) must be configured",
        )
    }
}
```

### 3.7 Nested Block Validation Examples

**SSH Username Required if SSH Block Present**:
```go
"ssh": schema.SingleNestedBlock{
    Attributes: map[string]schema.Attribute{
        "username": schema.StringAttribute{
            Required: true,  // Required if parent SSH block exists
            Validators: []validator.String{
                stringvalidator.LengthAtLeast(1),
            },
        },
    },
},
```

**RDP Ephemeral User oneOf**:
```go
// In ValidateConfig for RDP block
if !rdp.LocalEphemeralUser.IsNull() && !rdp.DomainEphemeralUser.IsNull() {
    resp.Diagnostics.AddWarning(
        "Conflicting RDP Configuration",
        "Both local_ephemeral_user and domain_ephemeral_user are specified. Only one will be used (SDK may pick one).",
    )
}

if rdp.LocalEphemeralUser.IsNull() && rdp.DomainEphemeralUser.IsNull() {
    resp.Diagnostics.AddError(
        "Invalid RDP Configuration",
        "At least one of local_ephemeral_user or domain_ephemeral_user must be specified for RDP",
    )
}
```

---

## 4. Principal Preservation Algorithm

### 4.1 Problem Statement

VM policy resources support TWO ways to assign principals:
1. **Inline principals**: Defined in `cyberarksia_vm_policy` resource config (required, min 1)
2. **Assigned principals**: Added via `cyberarksia_vm_policy_principal_assignment` resource (optional)

When updating a policy, must preserve BOTH inline and assigned principals.

### 4.2 Read-Modify-Write Algorithm

```pseudocode
FUNCTION UpdateVMPolicy(plan VMPolicyResourceModel, policyID string):
    # Step 1: READ existing policy from API
    existingPolicy := vmService.Policy({PolicyID: policyID})

    # Step 2: BUILD set of inline principal keys from Terraform config
    inlinePrincipalKeys := new Set<string>
    FOR EACH principal IN plan.Principals:
        key := principal.PrincipalID + ":" + principal.PrincipalType
        inlinePrincipalKeys.add(key)

    # Step 3: IDENTIFY assigned principals (in existing but not in inline set)
    assignedPrincipals := new List<Principal>
    FOR EACH principal IN existingPolicy.Principals:
        key := principal.ID + ":" + principal.Type
        IF NOT inlinePrincipalKeys.contains(key):
            # This principal was added via assignment resource
            assignedPrincipals.add(principal)

    # Step 4: BUILD inline principals from plan
    inlinePrincipals := new List<Principal>
    FOR EACH principal IN plan.Principals:
        inlinePrincipals.add({
            ID: principal.PrincipalID,
            Name: principal.PrincipalName,
            Type: principal.PrincipalType,
            SourceDirectoryName: principal.SourceDirectoryName,
            SourceDirectoryID: principal.SourceDirectoryID,
        })

    # Step 5: MERGE inline + assigned principals
    mergedPrincipals := inlinePrincipals + assignedPrincipals

    # Step 6: UPDATE other fields from plan
    existingPolicy.Metadata.Description := plan.Description
    existingPolicy.Metadata.TimeZone := plan.TimeZone
    existingPolicy.Conditions.MaxSessionDuration := plan.MaxSessionDuration
    # ... etc for all updatable fields

    # Step 7: SET merged principals array
    existingPolicy.Principals := mergedPrincipals

    # Step 8: WRITE back to API
    updatedPolicy := vmService.UpdatePolicy(existingPolicy)

    # Step 9: MAP full result to state (includes ALL principals)
    state.Principals := MAP_ALL_PRINCIPALS(updatedPolicy.Principals)

    RETURN state
```

### 4.3 Duplicate Detection Algorithm

**Used in**: `cyberarksia_vm_policy_principal_assignment` resource Create()

```pseudocode
FUNCTION CreatePrincipalAssignment(policyID, principalID, principalType):
    # Step 1: READ existing policy
    policy := vmService.Policy({PolicyID: policyID})

    # Step 2: CHECK for duplicates across ALL principals (inline + assigned)
    FOR EACH principal IN policy.Principals:
        IF principal.ID == principalID AND principal.Type == principalType:
            ERROR("Principal already assigned to policy (either inline or via assignment)")

    # Step 3: APPEND new principal
    newPrincipal := {
        ID: principalID,
        Name: principalName,
        Type: principalType,
        SourceDirectoryName: directoryName,
        SourceDirectoryID: directoryID,
    }
    policy.Principals.append(newPrincipal)

    # Step 4: UPDATE policy
    updatedPolicy := vmService.UpdatePolicy(policy)

    # Step 5: VERIFY assignment succeeded
    found := FALSE
    FOR EACH principal IN updatedPolicy.Principals:
        IF principal.ID == principalID AND principal.Type == principalType:
            found := TRUE
            BREAK

    IF NOT found:
        ERROR("Principal assignment failed - not found in updated policy")

    RETURN success
```

### 4.4 Edge Cases

**Case 1: Remove all inline principals while keeping assigned principals**
- User removes all `principal` blocks from `cyberarksia_vm_policy` resource
- Terraform will fail plan validation (principals list minimum size 1)
- **Resolution**: Must have at least 1 inline principal (schema requirement)

**Case 2: Delete policy with both inline and assigned principals**
- User runs `terraform destroy` on `cyberarksia_vm_policy` resource
- All principals (inline + assigned) are deleted with policy
- Assignment resources will fail on next apply (404 error)
- **Resolution**: Terraform handles this via dependency ordering (assignments depend on policy)

**Case 3: Update inline principals without affecting assigned principals**
- User changes inline principal from User A to User B in resource config
- Algorithm preserves assigned principals (e.g., User C added via assignment resource)
- Final state: User B (inline) + User C (assigned)

---

## 5. Composite ID Specification

### 5.1 ID Format

**Format**: `policy-id:principal-id:principal-type`

**Why 3 Parts**: Principal IDs can be duplicated across types (same UUID for USER vs GROUP vs ROLE)

**Examples**:
- `a1b2c3d4-5678-90ab-cdef-1234567890ab:e5f6a7b8-9012-34cd-ef56-7890abcdef12:USER`
- `policy-abc123:principal-def456:GROUP`

### 5.2 Parsing Function

**Location**: `internal/provider/helpers/composite_ids.go` (extend existing file)

```go
// ParseVMPolicyPrincipalID parses a composite ID for VM policy principal assignments
// Format: "policy-id:principal-id:principal-type"
func ParseVMPolicyPrincipalID(id string) (policyID, principalID, principalType string, err error) {
    parts := strings.Split(id, ":")
    if len(parts) != 3 {
        return "", "", "", fmt.Errorf(
            "invalid VM policy principal assignment ID format: expected 'policy-id:principal-id:principal-type', got '%s'",
            id,
        )
    }

    policyID = parts[0]
    principalID = parts[1]
    principalType = parts[2]

    // Validate principal type
    if principalType != "USER" && principalType != "GROUP" && principalType != "ROLE" {
        return "", "", "", fmt.Errorf(
            "invalid principal type '%s': must be USER, GROUP, or ROLE",
            principalType,
        )
    }

    return policyID, principalID, principalType, nil
}
```

### 5.3 Building Function

```go
// BuildVMPolicyPrincipalID creates a composite ID for VM policy principal assignments
func BuildVMPolicyPrincipalID(policyID, principalID, principalType string) string {
    return fmt.Sprintf("%s:%s:%s", policyID, principalID, principalType)
}
```

### 5.4 Usage in Resource

```go
// In Create() method
func (r *VMPolicyPrincipalAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data VMPolicyPrincipalAssignmentResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

    // ... perform assignment ...

    // Set composite ID
    compositeID := helpers.BuildVMPolicyPrincipalID(
        data.PolicyID.ValueString(),
        data.PrincipalID.ValueString(),
        data.PrincipalType.ValueString(),
    )
    data.ID = types.StringValue(compositeID)

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// In ImportState() method
func (r *VMPolicyPrincipalAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    policyID, principalID, principalType, err := helpers.ParseVMPolicyPrincipalID(req.ID)
    if err != nil {
        resp.Diagnostics.AddError("Invalid Import ID", err.Error())
        return
    }

    // Set individual attributes from parsed ID
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("policy_id"), policyID)...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_id"), principalID)...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_type"), principalType)...)

    // Read remaining attributes from API
    // ... (call Read method logic)
}
```

---

## 6. Design Decisions Log

### Decision 1: Inline Principals + Assignment Resource Pattern

**Decision**: VM policies REQUIRE at least one principal at creation, implemented via inline `principal` blocks + optional assignment resource.

**Rationale**:
- SDK/API allow 0 principals, but best practice requires minimum 1 for security
- Schema-level enforcement (Required: true + SizeAtLeast(1)) prevents invalid configs at plan time
- Assignment resource enables flexible principal management (add more after creation)
- Follows established database policy pattern for consistency

**Alternatives Considered**:
1. **Make principals optional in schema** - Rejected: Allows insecure policies with zero principals
2. **Use only assignment resources** - Rejected: Requires 2-step creation (policy then assignment), poor UX
3. **Inline-only (no assignment resource)** - Rejected: Less flexible, forces policy updates for principal changes

---

### Decision 2: No Workspace Assignment Resource

**Decision**: VM policies do NOT have a workspace assignment resource. Targets are defined inline in policy resource.

**Rationale**:
- VM policies use rule-based targeting (FQDN/IP patterns, cloud tags) NOT workspace references
- No separate VM workspace resource exists in ARK SDK or API
- OpenAPI schema shows inline target rules, not workspace ID references
- Different architecture from database policies (workspace-based model)

**Evidence**:
- SDK `ArkUAPSIAVMPlatformTargets` contains inline rule definitions
- No `/workspaces/vm` API endpoint
- OpenAPI `InfrastructureVirtualMachineTarget` has no `instanceId` field
- POC testing confirms targets embedded in policy object

**Implication**: Target management happens via policy resource updates (add/remove rules in-place)

---

### Decision 3: Read-Modify-Write for All Updates

**Decision**: Always use Read-Modify-Write pattern for UpdatePolicy() operations.

**Rationale**:
- SDK `UpdatePolicy()` accepts full policy object (not partial updates)
- Must preserve unmanaged principals (added via assignment resources)
- Must preserve computed fields (delegationClassification, createdBy, etc.)
- Prevents data loss when updating policy configuration

**Implementation**:
1. READ existing policy via `Policy()`
2. MODIFY specific fields from Terraform plan
3. PRESERVE principals not in inline config (assigned principals)
4. WRITE back entire policy via `UpdatePolicy()`

**Alternative Rejected**: Partial update pattern - SDK doesn't support it

---

### Decision 4: oneOf for Location Types

**Decision**: Use `ExactlyOneOf` validator to enforce one location type per policy.

**Rationale**:
- OpenAPI schema constraint: `InfrastructureVirtualMachineTarget.oneOf`
- API rejects policies with multiple location types or zero location types
- Terraform best practice: Validate at plan time (fail fast)
- Better UX than server-side validation error

**Implementation**: Custom validation in `ValidateConfig()` method

**Alternative Rejected**: Allow multiple location blocks and let API reject - poor UX, confusing error messages

---

### Decision 5: ForceNew on name and location_type

**Decision**: Changing `name` or `location_type` requires resource replacement (ForceNew).

**Rationale**:
- **Name**: Policy names are unique identifiers per tenant. Changing name creates confusion (is it rename or new policy?). Safer to replace.
- **LocationType**: Changing from FQDN/IP to AWS fundamentally changes target structure. Cannot preserve target rules across location types. Must replace policy.

**Implementation**: Use `stringplanmodifier.RequiresReplace()` on both attributes

**Alternative Rejected**: Allow in-place updates - complex migration logic, risk of data loss

---

### Decision 6: No DELETE Workaround for VM Policies

**Decision**: Use SDK `DeletePolicy()` method directly without custom HTTP client workaround.

**Rationale**:
- VM policies use `BaseDeletePolicy()` which correctly handles nil body
- No panic bug like database workspace/secret DELETE methods
- POC testing confirms SDK DELETE works
- Simpler implementation, fewer workarounds to maintain

**Evidence**:
- File: `pkg/services/uap/common/ark_uap_base_service.go`
- Calls: `s.client.Delete(ctx, route, nil)` safely
- Database policies have different DELETE implementation with nil body bug

**Implication**: Don't use `internal/client/delete_workarounds.go` for VM policies

---

### Decision 7: Include DomainEphemeralUser Despite OpenAPI Gap

**Decision**: Include `domain_ephemeral_user` in RDP behavior schema even though OpenAPI doesn't document it.

**Rationale**:
- SDK fully supports `ArkUAPSSIAVMDomainEphemeralUser` struct
- Provides parity with local ephemeral user + domain group assignments
- May be undocumented feature or OpenAPI spec out of sync with SDK
- Low risk: If API rejects it, users get clear validation error

**Implementation**: Document as "SDK-supported, OpenAPI undocumented" in resource description

**Alternative Rejected**: Exclude domain ephemeral user - limits functionality, users can't use domain-joined RDP if it works

---

### Decision 8: Optional + Computed for Server Defaults

**Decision**: Fields with server defaults use `Optional: true, Computed: true` in schema.

**Rationale**:
- Follows Terraform best practice for fields with API defaults
- Allows users to omit fields (server provides default)
- Allows users to override defaults (optional)
- Terraform tracks actual value in state (computed)

**Examples**:
- `time_zone`: Optional + Computed, default "GMT"
- `max_session_duration`: Optional + Computed, default 1
- `idle_time`: Optional + Computed, default 10

**Alternative Rejected**: Make fields Required - forces users to specify values even when defaults are acceptable

---

## 7. Implementation Checklist

### Phase 1: Core VM Policy Resource

- [ ] Create `internal/models/vm_policy_models.go` - Terraform state models
- [ ] Create `internal/provider/vm_policy_resource.go` - Main resource implementation
- [ ] Implement Schema() with all attributes (metadata, principals, conditions, behavior, targets)
- [ ] Implement Create() with profile building and retry logic
- [ ] Implement Read() with drift detection
- [ ] Implement Update() with Read-Modify-Write principal preservation
- [ ] Implement Delete() using SDK method (no workaround)
- [ ] Implement ImportState() using policy_id
- [ ] Implement ValidateConfig() for location type oneOf and connection profile validation
- [ ] Create `internal/validators/vm_validators.go` - Custom enum validators
- [ ] Create acceptance tests: `internal/provider/vm_policy_resource_test.go`

### Phase 2: Principal Assignment Resource

- [ ] Create `internal/provider/vm_policy_principal_assignment_resource.go`
- [ ] Implement Schema() with composite ID attributes
- [ ] Implement Create() with Read-Modify-Write and duplicate detection
- [ ] Implement Read() to verify assignment exists
- [ ] Implement Update() (ForceNew - all fields require replace)
- [ ] Implement Delete() with Read-Modify-Write to remove principal
- [ ] Implement ImportState() with composite ID parsing
- [ ] Extend `internal/provider/helpers/composite_ids.go` with VM policy functions
- [ ] Create acceptance tests: `internal/provider/vm_policy_principal_assignment_resource_test.go`

### Phase 3: Examples & Documentation

- [ ] Create `examples/resources/cyberarksia_vm_policy/resource.tf` - Basic FQDN/IP + SSH
- [ ] Create `examples/resources/cyberarksia_vm_policy/aws_policy.tf` - AWS cloud example
- [ ] Create `examples/resources/cyberarksia_vm_policy/rdp_policy.tf` - RDP behavior example
- [ ] Create `examples/resources/cyberarksia_vm_policy/complete.tf` - Full-featured example
- [ ] Create `examples/resources/cyberarksia_vm_policy_principal_assignment/resource.tf`
- [ ] Create `examples/testing/crud-test-vm-policy.tf` - CRUD validation template
- [ ] Update `examples/testing/TESTING-GUIDE.md` with VM policy scenarios
- [ ] Run `tfplugindocs generate` to create docs
- [ ] Update `CLAUDE.md` resource table with new resources

### Phase 4: Provider Registration

- [ ] Update `internal/provider/provider.go` - Register new resources
- [ ] Update `main.go` if needed (usually automatic)
- [ ] Build and install locally: `make build && make install`

### Phase 5: Testing & Validation

- [ ] Run acceptance tests: `TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy`
- [ ] Test CRUD lifecycle for each location type (FQDN/IP, AWS, GCP)
- [ ] Test principal assignment CRUD
- [ ] Test drift detection (manual policy deletion)
- [ ] Test ForceNew behavior (name change, location_type change)
- [ ] Test validation errors (zero principals, conflicting location types, invalid operators)
- [ ] Run `make validate` (format, lint, security checks)

---

**Research Phase Complete**: This document provides all necessary information to proceed with implementation planning (data-model.md, contracts/, quickstart.md).
