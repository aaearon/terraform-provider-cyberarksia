# ARK SDK VM Service API Contract

**Service Package**: `github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/uap/sia/vm`

**Service Struct**: `ArkUAPSIAVMService`

**Purpose**: CRUD operations for CyberArk SIA virtual machine access policies

---

## Service Initialization

```go
import (
    "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm"
    uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
    uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
)

vmService, err := vm.NewArkUAPSIAVMService(ispAuth)
if err != nil {
    return err
}
```

---

## CREATE: AddPolicy

### Signature

```go
AddPolicy(policy *models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)
```

### File Reference

**File**: `ark_uap_sia_vm_service.go:52-74`

### Request

**Input**: `*ArkUAPSIAVMAccessPolicy` with populated fields

**Required Fields**:
- `Metadata.Name` (1-200 chars, unique)
- `Metadata.PolicyEntitlement.LocationType` (AWS/Azure/GCP/FQDN/IP)
- `Metadata.Status.Status` (Active/Suspended)
- `Principals` (array, minimum 1 element)
- `Behavior` (with at least SSH or RDP profile)
- `Targets` (location-specific structure)
- `Conditions` (object, fields optional with defaults)

**Auto-Set Fields**:
- `PolicyEntitlement.TargetCategory` = "VM" (set by SDK)
- `PolicyTags` = `[]` if nil (converted by SDK)

**Example**:
```go
policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{
    Metadata: uapcommonmodels.ArkUAPSIACommonAccessPolicyMetadata{
        Name:        "Production Servers Policy",
        Description: "Access for production web servers",
        TimeZone:    "GMT",
        PolicyEntitlement: uapcommonmodels.ArkUAPPolicyEntitlement{
            LocationType: "FQDN/IP",
            PolicyType:   "Recurring",
        },
        Status: uapcommonmodels.ArkUAPStatus{
            Status: "Active",
        },
    },
    Principals: []uapcommonmodels.ArkUAPPrincipal{
        {
            ID:                  "abc-123-uuid",
            Name:                "admin@example.com",
            Type:                "USER",
            SourceDirectoryName: "CyberArk",
            SourceDirectoryID:   "dir-456",
        },
    },
    Targets: uapsiavmmodels.ArkUAPSIAVMPlatformTargets{
        FQDNIPResource: &uapsiavmmodels.ArkUAPSIAVMFQDNIPResource{
            FQDNRules: []uapsiavmmodels.ArkUAPSIAVMFQDNRule{
                {
                    Operator:            "SUFFIX",
                    ComputernamePattern: "-prod",
                    Domain:              "example.com",
                },
            },
        },
    },
    Behavior: uapsiavmmodels.ArkUAPSSIAVMBehavior{
        SSHProfile: &uapsiavmmodels.ArkUAPSSIAVMSSHProfile{
            Username: "ec2-user",
        },
    },
    Conditions: uapcommonmodels.ArkUAPConditions{
        MaxSessionDuration: 1,  // hours
        IdleTime:           10, // minutes
    },
}

created, err := vmService.AddPolicy(policy)
```

### Response

**Output**: `*ArkUAPSIAVMAccessPolicy` with server-generated fields

**Server-Generated Fields**:
- `Metadata.PolicyID` (UUID)
- `DelegationClassification` ("Restricted" or "Unrestricted")
- `Metadata.CreatedBy` (user + timestamp)
- `Metadata.UpdatedBy` (user + timestamp)

**Behavior**: Returns created policy by calling `Policy()` after creation

### Error Scenarios

| HTTP Code | Error Message | Cause | Terraform Handling |
|-----------|---------------|-------|-------------------|
| 400 Bad Request | Various validation errors | Invalid field values, missing required fields | MapError → Validation error diagnostic |
| 401 Unauthorized | "Invalid credentials" | Invalid/expired OAuth2 token | MapError → Authentication error |
| 409 Conflict | "Policy with name already exists" | Duplicate policy name | MapError → Conflict error with resolution hint |
| 500 Server Error | Internal server error | API backend issue | Retry with backoff, then fail |

---

## READ: Policy

### Signature

```go
Policy(req *ArkUAPGetPolicyRequest) (*models.ArkUAPSIAVMAccessPolicy, error)
```

### File Reference

**File**: `ark_uap_sia_vm_service.go:76-91`

### Request

**Input**: `*ArkUAPGetPolicyRequest`

```go
type ArkUAPGetPolicyRequest struct {
    PolicyID string
}
```

**Example**:
```go
policy, err := vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
    PolicyID: "policy-uuid-123",
})
```

### Response

**Output**: `*ArkUAPSIAVMAccessPolicy` with all fields populated

**Includes**:
- Full metadata (name, description, status, etc.)
- ALL principals (inline + assigned via assignment resources)
- Complete target configuration (FQDN/IP rules or cloud filters)
- Behavior configuration (SSH/RDP profiles)
- Conditions (session duration, idle time, access windows)

**Deserialization**: Converts JSON with snake_case to Go struct

### Error Scenarios

| HTTP Code | Error Message | Cause | Terraform Handling |
|-----------|---------------|-------|-------------------|
| 404 Not Found | "Policy not found" | Policy deleted externally | Remove from state (drift detection) |
| 401 Unauthorized | "Invalid credentials" | Invalid/expired OAuth2 token | MapError → Authentication error |

### Drift Detection Pattern

```go
policy, err := vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{PolicyID: policyID})
if err != nil {
    if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
        // Policy deleted externally - remove from state
        resp.State.RemoveResource(ctx)
        return
    }
    resp.Diagnostics.Append(client.MapError(err, "read VM policy")...)
    return
}
```

---

## UPDATE: UpdatePolicy

### Signature

```go
UpdatePolicy(policy *models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)
```

### File Reference

**File**: `ark_uap_sia_vm_service.go:94-112`

### Request

**Input**: `*ArkUAPSIAVMAccessPolicy` with ALL fields populated

**CRITICAL**: Must use Read-Modify-Write pattern

### Read-Modify-Write Pattern

```go
// Step 1: READ existing policy
existingPolicy, err := vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
    PolicyID: policyID,
})
if err != nil {
    return err
}

// Step 2: IDENTIFY inline principals from Terraform config
inlinePrincipalKeys := make(map[string]bool)
for _, p := range plan.Principals {
    key := fmt.Sprintf("%s:%s", p.PrincipalID, p.PrincipalType)
    inlinePrincipalKeys[key] = true
}

// Step 3: PRESERVE assigned principals (not in inline config)
preservedPrincipals := []uapcommonmodels.ArkUAPPrincipal{}
for _, p := range existingPolicy.Principals {
    key := fmt.Sprintf("%s:%s", p.ID, p.Type)
    if !inlinePrincipalKeys[key] {
        // This principal was added via assignment resource - preserve it
        preservedPrincipals = append(preservedPrincipals, p)
    }
}

// Step 4: BUILD new principals list: inline from plan + preserved assigned
newPrincipals := /* build inline principals from plan.Principals */
newPrincipals = append(newPrincipals, preservedPrincipals...)

// Step 5: MODIFY fields from Terraform plan
existingPolicy.Metadata.Description = plan.Description.ValueString()
existingPolicy.Metadata.TimeZone = plan.TimeZone.ValueString()
existingPolicy.Conditions.MaxSessionDuration = int(plan.MaxSessionDuration.ValueInt64())
existingPolicy.Conditions.IdleTime = int(plan.IdleTime.ValueInt64())
existingPolicy.Principals = newPrincipals

// DO NOT modify computed fields (PolicyID, DelegationClassification, CreatedBy, etc.)

// Step 6: WRITE back entire policy
updated, err := vmService.UpdatePolicy(existingPolicy)
```

### Response

**Output**: `*ArkUAPSIAVMAccessPolicy` with updated values

**Behavior**: Returns updated policy by calling `Policy()` after update

### Error Scenarios

| HTTP Code | Error Message | Cause | Terraform Handling |
|-----------|---------------|-------|-------------------|
| 404 Not Found | "Policy not found" | Policy deleted during update | MapError → Policy deleted diagnostic |
| 400 Bad Request | Various validation errors | Invalid updated values | MapError → Validation error diagnostic |
| 409 Conflict | "Policy with name already exists" | Name changed to duplicate | MapError → Conflict error |
| 401 Unauthorized | "Invalid credentials" | Invalid/expired OAuth2 token | MapError → Authentication error |

### Why Read-Modify-Write Required

1. **SDK Design**: `UpdatePolicy()` replaces entire policy object (no partial updates)
2. **Preserve Unmanaged Fields**:
   - Assigned principals (added via assignment resources)
   - Computed metadata (DelegationClassification, CreatedBy, UpdatedBy)
   - Other policy attributes not managed by current resource
3. **Prevent Data Loss**: Without R-M-W, updating description would delete all assigned principals

---

## DELETE: DeletePolicy

### Signature

```go
DeletePolicy(req *ArkUAPDeletePolicyRequest) error
```

### File Reference

**File**: `ark_uap_sia_vm_service.go:179-182`

### Request

**Input**: `*ArkUAPDeletePolicyRequest`

```go
type ArkUAPDeletePolicyRequest struct {
    PolicyID string
}
```

**Example**:
```go
err := vmService.DeletePolicy(&uapcommonmodels.ArkUAPDeletePolicyRequest{
    PolicyID: "policy-uuid-123",
})
```

### Response

**Output**: `nil` on success, `error` on failure

### Implementation Details

**Use SDK Method Directly** - NO workaround needed

**Why No Workaround Required**:
- VM policies use `BaseDeletePolicy()` from `pkg/services/uap/common/ark_uap_base_service.go`
- Correctly calls `s.client.Delete(context.Background(), fmt.Sprintf(policyURL, policyID), nil)`
- Common HTTP client handles `body == nil` without panic
- Different from SIA workspace/secret DELETE methods which have nil body panic bug

**DO NOT** use `internal/client/delete_workarounds.go` for VM policies

### Error Scenarios

| HTTP Code | Error Message | Cause | Terraform Handling |
|-----------|---------------|-------|-------------------|
| 404 Not Found | "Policy not found" | Already deleted | Treat as success (drift detection) |
| 401 Unauthorized | "Invalid credentials" | Invalid/expired OAuth2 token | MapError → Authentication error |

### Drift Detection Pattern

```go
err := vmService.DeletePolicy(&uapcommonmodels.ArkUAPDeletePolicyRequest{PolicyID: policyID})
if err != nil {
    // 404 = already deleted - treat as success
    if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
        return nil
    }
    resp.Diagnostics.Append(client.MapError(err, "delete VM policy")...)
    return
}
```

---

## LIST: ListPolicies

**Note**: Not required for Terraform provider implementation but available if needed

### Signature

```go
ListPolicies(req *ArkUAPListPoliciesRequest) (*ArkUAPListPoliciesResponse, error)
```

**Use Case**: Could be used for data source implementation in future

---

## Error Handling Patterns

### Retry with Backoff

**All CRUD operations should use retry logic** for transient failures:

```go
import "github.com/aaearon/terraform-provider-cyberarksia/internal/client"

err := client.RetryWithBackoff(ctx, func() error {
    _, err := vmService.AddPolicy(policy)
    return err
})
if err != nil {
    resp.Diagnostics.Append(client.MapError(err, "create VM policy")...)
    return
}
```

**Configuration**:
- Max retries: 3
- Base delay: 1 second
- Max delay: 30 seconds
- Exponential backoff with jitter

### Error Classification

**Use `client.MapError()` for consistent error handling**:

```go
import "github.com/aaearon/terraform-provider-cyberarksia/internal/client"

if err != nil {
    resp.Diagnostics.Append(client.MapError(err, "operation description")...)
    return
}
```

**Error Categories**:
- 401 → Authentication error (suggest token refresh)
- 403 → Permission denied (check IAM policy)
- 404 → Not found (drift detection)
- 409 → Conflict (duplicate resource)
- 429 → Rate limit (automatic retry)
- 500 → Server error (automatic retry)

---

## Serialization/Deserialization

### Camel/Snake Case Conversion

**SDK Behavior**: Automatic conversion between Go struct fields and JSON API

**Pattern**:
- Go struct fields: `snake_case` (e.g., `ssh_profile`)
- JSON API fields: `camelCase` (e.g., `connectAs.ssh`)
- SDK handles conversion automatically in `Serialize()` and `Deserialize()` methods

**File**: `ark_uap_sia_vm_service.go:63,84`

```go
// CREATE/UPDATE - Convert to camelCase
addPolicyJSON := common.ConvertToCamelCase(addPolicySerialized, &policyType)

// READ - Convert to snake_case
policyJSONSnake := common.ConvertToSnakeCase(policyJSON, &respType)
```

**Implication**: Use SDK structs directly, conversion is automatic

### Location Type Requirement

**Serialization Quirk**: `Targets.Serialize()` requires `LocationType` parameter

**File**: `ark_uap_sia_vm_access_policy.go:23`

```go
data["targets"], err = p.Targets.Serialize(p.Metadata.PolicyEntitlement.LocationType)
```

**Implication**: Must set `Metadata.PolicyEntitlement.LocationType` before calling `AddPolicy()` or `UpdatePolicy()`

---

## Testing Recommendations

### Unit Tests (SDK Integration)

**Mock SDK service for unit testing**:

```go
type mockVMService struct {
    AddPolicyFunc func(*models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)
    PolicyFunc    func(*ArkUAPGetPolicyRequest) (*models.ArkUAPSIAVMAccessPolicy, error)
    // ...
}
```

### Acceptance Tests (Live API)

**Test against real CyberArk SIA tenant**:

```bash
export TF_ACC=1
export CYBERARK_USERNAME="service-account@cyberark.cloud.12345"
export CYBERARK_PASSWORD="<password>"
go test ./internal/provider -v -run TestAccVMPolicy
```

**Test Scenarios**:
1. CRUD lifecycle for each location type
2. Principal assignment operations
3. Drift detection (manual deletion via API)
4. ForceNew behavior (name/location_type changes)
5. Validation errors (missing required fields, invalid values)

---

**API Contract Complete**: This document defines the complete interface for ARK SDK VM service integration.
