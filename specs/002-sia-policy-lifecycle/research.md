# Research: Database Policy Management Implementation

**Date**: 2025-10-28
**Feature**: Database Policy Management - Modular Assignment Pattern
**Objective**: Map ARK SDK v1.5.0 UAP API to Terraform resources, document implementation patterns

---

## 1. ARK SDK UAP API Structure

### ArkUAPSIADBAccessPolicy Hierarchy

The UAP database policy API uses a deeply nested structure with embedded principals and targets:

```go
// Full hierarchy from ark-sdk-golang v1.5.0
type ArkUAPSIADBAccessPolicy struct {
    ArkUAPSIACommonAccessPolicy
}

type ArkUAPSIACommonAccessPolicy struct {
    ArkUAPCommonAccessPolicy
    Conditions ArkUAPSIACommonConditions
    Targets    map[string]ArkUAPSIADBTargets  // Key: "FQDN/IP" only
}

type ArkUAPCommonAccessPolicy struct {
    Metadata                 ArkUAPMetadata
    Principals               []ArkUAPPrincipal
    DelegationClassification string  // "Restricted"|"Unrestricted"
}

type ArkUAPMetadata struct {
    PolicyID          string
    Name              string
    Description       string
    Status            ArkUAPPolicyStatus
    TimeFrame         ArkUAPTimeFrame
    PolicyEntitlement ArkUAPPolicyEntitlement
    CreatedBy         ArkUAPChangeInfo
    UpdatedOn         ArkUAPChangeInfo
    PolicyTags        []string
    TimeZone          string
}

type ArkUAPPolicyStatus struct {
    Status            string  // "Active"|"Suspended"|"Expired"|"Validating"|"Error"
    StatusCode        string
    StatusDescription string
    Link              string
}

type ArkUAPPolicyEntitlement struct {
    TargetCategory string  // "Cloud console"|"VM"|"DB"
    LocationType   string  // "AWS"|"Azure"|"GCP"|"FQDN/IP"
    PolicyType     string  // "Recurring"|"OnDemand"
}

type ArkUAPPrincipal struct {
    ID                  string
    Name                string
    SourceDirectoryName string
    SourceDirectoryID   string
    Type                string  // "USER"|"ROLE"|"GROUP"
}

type ArkUAPSIACommonConditions struct {
    ArkUAPConditions
    IdleTime int  // 1-120 minutes
}

type ArkUAPConditions struct {
    MaxSessionDuration int
    AccessWindow       ArkUAPAccessWindow
}

type ArkUAPAccessWindow struct {
    DaysOfTheWeek []int     // 0=Sunday through 6=Saturday
    FromHour      string    // "HH:MM"
    ToHour        string    // "HH:MM"
}

type ArkUAPSIADBTargets struct {
    Instances []ArkUAPSIADBTargetInstance
}

type ArkUAPSIADBTargetInstance struct {
    InstanceID             int
    InstanceName           string
    InstanceType           string
    AuthenticationMethod   string
    AuthenticationProfile  interface{}  // Type depends on AuthenticationMethod
}
```

### Field Mappings (Terraform → SDK)

**Policy Resource**:
| Terraform Attribute | SDK Field | Type | Notes |
|---------------------|-----------|------|-------|
| `id` | `Metadata.PolicyID` | string (UUID) | Computed |
| `name` | `Metadata.Name` | string | Required, ForceNew |
| `description` | `Metadata.Description` | string | Optional |
| `status` | `Metadata.Status.Status` | string | Required, "Active"\|"Suspended" only |
| `time_frame.from_time` | `Metadata.TimeFrame.FromTime` | string (ISO 8601) | Optional |
| `time_frame.to_time` | `Metadata.TimeFrame.ToTime` | string (ISO 8601) | Optional |
| `policy_tags` | `Metadata.PolicyTags` | []string | Optional, max 20 |
| `time_zone` | `Metadata.TimeZone` | string | Optional, default "GMT" |
| `delegation_classification` | `DelegationClassification` | string | Required, default "Unrestricted" |
| `conditions.max_session_duration` | `Conditions.MaxSessionDuration` | int | Required, 1-24 hours |
| `conditions.idle_time` | `Conditions.IdleTime` | int | Optional, 1-120 minutes, default 10 |
| `conditions.access_window.days_of_the_week` | `Conditions.AccessWindow.DaysOfTheWeek` | []int | Optional |
| `conditions.access_window.from_hour` | `Conditions.AccessWindow.FromHour` | string | Optional |
| `conditions.access_window.to_hour` | `Conditions.AccessWindow.ToHour` | string | Optional |
| N/A (fixed) | `Metadata.PolicyEntitlement.TargetCategory` | string | Always "DB" |
| N/A (fixed) | `Metadata.PolicyEntitlement.LocationType` | string | Always "FQDN/IP" |
| N/A (fixed) | `Metadata.PolicyEntitlement.PolicyType` | string | Always "Recurring" |
| `created_by` | `Metadata.CreatedBy` | object | Computed |
| `updated_on` | `Metadata.UpdatedOn` | object | Computed |

**Principal Assignment Resource**:
| Terraform Attribute | SDK Field | Type | Notes |
|---------------------|-----------|------|-------|
| `id` | N/A (provider-level) | string | Composite: "policy-id:principal-id:principal-type" |
| `policy_id` | N/A (lookup key) | string | ForceNew |
| `principal_id` | `Principals[].ID` | string | ForceNew, max 40 chars |
| `principal_name` | `Principals[].Name` | string | Required, max 512 chars |
| `principal_type` | `Principals[].Type` | string | ForceNew, "USER"\|"GROUP"\|"ROLE" |
| `source_directory_name` | `Principals[].SourceDirectoryName` | string | Required for USER/GROUP |
| `source_directory_id` | `Principals[].SourceDirectoryID` | string | Required for USER/GROUP |

**Database Assignment Resource**:
| Terraform Attribute | SDK Field | Type | Notes |
|---------------------|-----------|------|-------|
| `id` | N/A (provider-level) | string | Composite: "policy-id:database-id" |
| `policy_id` | N/A (lookup key) | string | ForceNew |
| `database_workspace_id` | `Targets["FQDN/IP"].Instances[].InstanceID` | int | ForceNew |
| `authentication_method` | `Targets["FQDN/IP"].Instances[].AuthenticationMethod` | string | Required |
| `*_profile` | `Targets["FQDN/IP"].Instances[].AuthenticationProfile` | object | Required (matches method) |

### Computed vs Configurable Fields

**Computed (API-generated)**:
- `PolicyID` - UUID assigned by API
- `Status.StatusCode`, `Status.StatusDescription`, `Status.Link` - Status metadata
- `CreatedBy` - Creator user and timestamp
- `UpdatedOn` - Last modifier and timestamp
- `PolicyEntitlement` - All sub-fields (TargetCategory, LocationType, PolicyType) - provider-managed

**Configurable (User-provided)**:
- `Name` - Policy name (1-200 chars, unique)
- `Description` - Policy description (max 200 chars)
- `Status.Status` - "Active" or "Suspended"
- `TimeFrame` - Validity period
- `PolicyTags` - Tags (max 20)
- `TimeZone` - Timezone (default "GMT")
- `DelegationClassification` - "Restricted" or "Unrestricted"
- `Conditions` - All condition attributes
- `Principals` - Array (managed via separate assignment resource)
- `Targets` - Map (managed via separate assignment resource)

### UpdatePolicy() API Constraint

**Critical Limitation**: The `UpdatePolicy()` method accepts the full policy structure BUT has a constraint on the `Targets` map:

```go
// API CONSTRAINT: Only ONE workspace type allowed in Targets per UpdatePolicy() call
updatePolicy := &ArkUAPSIADBAccessPolicy{
    ArkUAPSIACommonAccessPolicy: policy.ArkUAPSIACommonAccessPolicy,
    Targets: map[string]ArkUAPSIADBTargets{
        "FQDN/IP": policy.Targets["FQDN/IP"],  // ONLY the modified workspace type
    },
}
```

**Implication**: When updating database assignments, the provider must:
1. Send full policy metadata (name, status, conditions, principals)
2. Send ONLY the modified workspace type in Targets (e.g., "FQDN/IP")
3. NOT send all workspace types (API will reject)

This constraint is already handled in `database_policy_workspace_assignment_resource.go:1110+`.

---

## 2. Read-Modify-Write Pattern

### Algorithm

The read-modify-write pattern preserves UI-managed and other-Terraform-managed elements:

```
1. FETCH: GET policy by ID
   ↓
   Returns complete policy with ALL principals and targets

2. LOCATE: Find our managed element
   ↓
   - Principal assignment: Search Principals[] for our principal_id + principal_type
   - Database assignment: Search Targets["FQDN/IP"].Instances[] for our database_workspace_id

3. MODIFY: Update only our element
   ↓
   - CREATE: Append to array/map
   - UPDATE: Modify in-place
   - DELETE: Remove from array/map
   - PRESERVE: Leave all other elements unchanged

4. VALIDATE: Check API constraints
   ↓
   - Single workspace type in Targets
   - No duplicate principals
   - Valid authentication profiles

5. WRITE: PUT policy back
   ↓
   UpdatePolicy() with modified policy

6. HANDLE CONFLICTS: Retry transient failures
   ↓
   - 409/412: Retry with exponential backoff
   - 404: Resource deleted, remove from state
   - 4xx: Validation error, fail with message
   - 5xx: Server error, retry
```

### Code Example (Principal Assignment)

```go
// Step 1: Fetch
policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
    PolicyID: policyID,
})
if err != nil {
    // Handle 404 → resource deleted
    return
}

// Step 2: Locate
existingIndex := -1
for i, p := range policy.Principals {
    if p.ID == principalID && p.Type == principalType {
        existingIndex = i
        break
    }
}

// Step 3: Modify (CREATE)
if existingIndex == -1 {
    newPrincipal := uapcommonmodels.ArkUAPPrincipal{
        ID:                  data.PrincipalID.ValueString(),
        Name:                data.PrincipalName.ValueString(),
        Type:                data.PrincipalType.ValueString(),
        SourceDirectoryName: data.SourceDirectoryName.ValueString(),
        SourceDirectoryID:   data.SourceDirectoryID.ValueString(),
    }
    policy.Principals = append(policy.Principals, newPrincipal)
}

// Step 5: Write
err = r.providerData.UAPClient.Db().UpdatePolicy(policy)
```

### Code Example (Database Assignment with API Constraint)

```go
// Step 1: Fetch
policy, err := r.providerData.UAPClient.Db().Policy(&req{PolicyID: policyID})

// Step 2: Locate
workspaceType := "FQDN/IP"  // Always for database policies
targets := policy.Targets[workspaceType]
existingIndex := -1
for i, instance := range targets.Instances {
    if instance.InstanceID == databaseID {
        existingIndex = i
        break
    }
}

// Step 3: Modify (CREATE)
if existingIndex == -1 {
    newInstance := uapsiadbmodels.ArkUAPSIADBTargetInstance{
        InstanceID:             databaseID,
        InstanceName:           databaseName,
        InstanceType:           databaseType,
        AuthenticationMethod:   authMethod,
        AuthenticationProfile:  authProfile,
    }
    targets.Instances = append(targets.Instances, newInstance)
    policy.Targets[workspaceType] = targets
}

// Step 4: Validate - API constraint
// CRITICAL: Send ONLY modified workspace type, not all workspace types
updatePolicy := &uapsiadbmodels.ArkUAPSIADBAccessPolicy{
    ArkUAPSIACommonAccessPolicy: policy.ArkUAPSIACommonAccessPolicy,
    Targets: map[string]uapsiadbmodels.ArkUAPSIADBTargets{
        workspaceType: policy.Targets[workspaceType],  // ONLY "FQDN/IP"
    },
}

// Step 5: Write
err = r.providerData.UAPClient.Db().UpdatePolicy(updatePolicy)
```

### Preservation Guarantees

**What is preserved**:
- ✅ UI-managed principals (not in Terraform state)
- ✅ Other Terraform workspace's principals (different state files)
- ✅ UI-managed database assignments
- ✅ Other Terraform workspace's database assignments
- ✅ Policy metadata changes made outside Terraform (detected as drift)

**What is NOT preserved**:
- ❌ Concurrent modifications (last-write-wins, race condition possible)
- ❌ Changes to managed elements made outside Terraform (overwritten on next apply)

---

## 3. Composite ID Strategy

### Principal Assignment: 3-Part Format

**Format**: `policy-id:principal-id:principal-type`

**Example**: `12345678-1234-1234-1234-123456789012:alice@example.com:USER`

**Rationale**: Principal IDs can be duplicated across types:
- User: `admin` (user account)
- Role: `admin` (role name)
- Group: `admin` (group name)

Without the type discriminator, we cannot uniquely identify which principal to manage.

**Parsing Algorithm**:
```go
func parseCompositeID(id string) (policyID, principalID, principalType string, err error) {
    parts := strings.Split(id, ":")
    if len(parts) != 3 {
        return "", "", "", fmt.Errorf("invalid format: expected 'policy-id:principal-id:principal-type', got '%s'", id)
    }

    policyID = parts[0]
    principalID = parts[1]
    principalType = parts[2]

    // Validate non-empty
    if policyID == "" || principalID == "" || principalType == "" {
        return "", "", "", fmt.Errorf("composite ID parts cannot be empty")
    }

    // Validate principal type
    validTypes := []string{"USER", "GROUP", "ROLE"}
    if !slices.Contains(validTypes, principalType) {
        return "", "", "", fmt.Errorf("invalid principal type '%s', must be USER, GROUP, or ROLE", principalType)
    }

    return policyID, principalID, principalType, nil
}
```

### Database Assignment: 2-Part Format

**Format**: `policy-id:database-id`

**Example**: `12345678-1234-1234-1234-123456789012:789`

**Rationale**: Database workspace IDs are globally unique integers, no type discriminator needed.

**Parsing Algorithm** (existing):
```go
func parseCompositeID(id string) (policyID, dbID string, err error) {
    parts := strings.SplitN(id, ":", 2)
    if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
        return "", "", fmt.Errorf("invalid format: expected 'policy-id:database-id', got '%s'", id)
    }
    return parts[0], parts[1], nil
}
```

---

## 4. Policy Status Management

### Status Values

**User-Controllable** (valid for database policies):
- `"Active"` - Policy is enabled, access granted per conditions
- `"Suspended"` - Policy is temporarily disabled, access denied

**Server-Managed** (not exposed in provider):
- `"Expired"` - Policy TimeFrame expired (system-set)
- `"Validating"` - Policy validation in progress (transient)
- `"Error"` - Policy has validation errors (system-set)

### Validation Strategy

**Custom Validator** (`policy_status_validator.go`):
```go
type policyStatusValidator struct{}

func (v policyStatusValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
    if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
        return  // Skip during plan phase
    }

    value := req.ConfigValue.ValueString()
    validStatuses := []string{"Active", "Suspended"}

    if !slices.Contains(validStatuses, value) {
        resp.Diagnostics.AddAttributeError(
            req.Path,
            "Invalid Policy Status",
            fmt.Sprintf("Value %q is not valid. Must be 'Active' or 'Suspended'. "+
                "Note: 'Expired', 'Validating', and 'Error' are server-managed statuses "+
                "and cannot be set by users.", value),
        )
    }
}

func PolicyStatus() validator.String {
    return policyStatusValidator{}
}
```

### Default Behavior

**Schema Definition**:
```go
"status": schema.StringAttribute{
    Required: true,
    Validators: []validator.String{
        validators.PolicyStatus(),
    },
    Description: "Policy status: 'Active' (enabled) or 'Suspended' (disabled)",
}
```

**No default value** - user must explicitly specify status to avoid ambiguity.

---

## 5. ForceNew Attributes

### Policy Resource

**ForceNew** (require resource replacement):
- `name` - Policy identity, changing breaks references
- `location_type` - Breaks compatibility with existing database assignments

**In-Place Update**:
- `description`
- `status`
- `delegation_classification`
- `conditions` (all sub-attributes)
- `policy_tags`
- `time_zone`
- `time_frame`

**Schema Implementation**:
```go
"name": schema.StringAttribute{
    Required: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),  // ForceNew
    },
},
```

### Principal Assignment Resource

**ForceNew** (all identifying attributes):
- `policy_id` - Cannot reassign principal to different policy
- `principal_id` - Cannot change principal identity
- `principal_type` - Cannot change principal type

**No in-place updates** - all other attributes (principal_name, directory info) require recreation.

### Database Assignment Resource

**ForceNew** (existing pattern):
- `policy_id` - Cannot reassign database to different policy
- `database_workspace_id` - Cannot change database identity

**In-Place Update**:
- `authentication_method` - Can change auth method
- `*_profile` - Can update auth profile attributes

---

## 6. Pagination Pattern

### ARK SDK Channel-Based Pagination

The SDK returns a Go channel that emits pages:

```go
// List all policies
policyPages, err := uapAPI.Db().ListPolicies()
if err != nil {
    return err
}

// Iterate pages
for page := range policyPages {
    if page.Error != nil {
        return page.Error
    }

    for _, policy := range page.Items {
        // Process policy
    }
}
```

### Data Source Implementation

**Existing Pattern** (`access_policy_data_source.go:99-220`):

```go
func (d *AccessPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data AccessPolicyDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

    // Get search criteria
    searchID := data.ID.ValueString()
    searchName := data.Name.ValueString()

    // List all policies (paginated)
    policyPages, err := d.uapAPI.Db().ListPolicies()
    if err != nil {
        resp.Diagnostics.AddError("Error Listing Policies", err.Error())
        return
    }

    // Search pages for matching policy
    var foundPolicy *uapsiadbmodels.ArkUAPSIADBAccessPolicy
    for page := range policyPages {
        if page.Error != nil {
            resp.Diagnostics.AddError("Error Reading Policy Page", page.Error.Error())
            return
        }

        for _, policy := range page.Items {
            if (searchID != "" && policy.Metadata.PolicyID == searchID) ||
               (searchName != "" && policy.Metadata.Name == searchName) {
                foundPolicy = &policy
                break
            }
        }

        if foundPolicy != nil {
            break  // Stop paginating once found
        }
    }

    if foundPolicy == nil {
        resp.Diagnostics.AddError("Policy Not Found", "No matching policy")
        return
    }

    // Populate data model
    // ...
}
```

### Performance Considerations

- **Early termination**: Stop paginating once target found
- **Channel cleanup**: SDK handles channel closure automatically
- **Error handling**: Check `page.Error` on each iteration
- **No manual pagination**: SDK abstracts page tokens/cursors

---

## 7. API Error Handling

### ARK SDK Error Patterns

The ARK SDK v1.5.0 returns standard Go `error` interface with no structured error types. Error detection relies on:

1. **Standard Go errors**: `net.Error`, `context.DeadlineExceeded`, `context.Canceled`
2. **Pattern matching**: Case-insensitive string matching on error messages
3. **HTTP status codes**: Embedded in error strings (e.g., "HTTP 404: Not Found")

### Common API Error Responses

**Error Response Structure** (inferred from SDK behavior):
```
HTTP {status_code}: {error_message}
```

**Common Error Scenarios**:

| HTTP Status | Error Pattern | Provider Action |
|-------------|---------------|-----------------|
| 400 | "Bad Request", "Invalid", "Validation" | Map to ValidationError diagnostic |
| 401 | "Unauthorized", "Authentication failed" | Map to AuthenticationError diagnostic |
| 403 | "Forbidden", "Permission denied" | Map to PermissionError diagnostic |
| 404 | "Not Found", "does not exist" | Map to NotFoundError diagnostic |
| 409 | "Conflict", "already exists", "duplicate" | Map to ConflictError diagnostic |
| 429 | "Too many requests", "Rate limit" | Retry with backoff (FR-033) |
| 500 | "Internal server error" | Retry with backoff (FR-033) |
| 502/503 | "Service unavailable", "Bad gateway" | Retry with backoff (FR-033) |

### Error Mapping Strategy

**Provider Implementation** (follows existing `client.MapError()` pattern):

```go
func MapError(err error) diag.Diagnostics {
    if err == nil {
        return nil
    }

    var diags diag.Diagnostics
    errStr := strings.ToLower(err.Error())

    // Standard Go errors
    if errors.Is(err, context.DeadlineExceeded) {
        diags.AddError("Request Timeout", "API request timed out. Try again.")
        return diags
    }

    // Pattern matching (ordered by specificity)
    switch {
    case strings.Contains(errStr, "not found"):
        diags.AddError("Resource Not Found", err.Error())
    case strings.Contains(errStr, "already exists"), strings.Contains(errStr, "duplicate"):
        diags.AddError("Resource Already Exists", err.Error())
    case strings.Contains(errStr, "unauthorized"), strings.Contains(errStr, "authentication"):
        diags.AddError("Authentication Failed", err.Error())
    case strings.Contains(errStr, "forbidden"), strings.Contains(errStr, "permission"):
        diags.AddError("Permission Denied", err.Error())
    case strings.Contains(errStr, "validation"), strings.Contains(errStr, "invalid"):
        diags.AddError("Validation Error", err.Error())
    default:
        diags.AddError("API Error", err.Error())
    }

    return diags
}
```

### Specific Error Messages

**Duplicate Policy Name**:
```
HTTP 409: Policy with name "{name}" already exists
```
**Provider Action**: Map to ConflictError with guidance: "Policy names must be unique. Choose a different name or import the existing policy."

**UAP Service Not Provisioned**:
```
DNS lookup error: no such host uap.{tenant}.cyberark.cloud
```
**Provider Action**: Map to ConfigurationError with guidance: "UAP service not provisioned on tenant. Contact CyberArk support."

**Principal Not Found**:
```
HTTP 404: Principal "{principal_id}" not found in directory "{directory_id}"
```
**Provider Action**: Map to NotFoundError with guidance: "Verify principal exists in identity directory."

**Database Workspace Not Found**:
```
HTTP 404: Database workspace "{workspace_id}" not found
```
**Provider Action**: Map to NotFoundError with guidance: "Ensure database workspace exists before assigning to policy."

**Policy Not Found**:
```
HTTP 404: Policy "{policy_id}" not found
```
**Provider Action**: Map to NotFoundError (standard - no special guidance needed)

**Invalid Composite ID**:
```
Provider-level validation error (before API call)
```
**Provider Action**: Return clear error with format guidance:
```
Invalid composite ID format: expected 'policy-id:principal-id:principal-type', got '{input}'
```

### Retry Logic

**Transient Errors** (per FR-033):
- HTTP 429 (rate limit)
- HTTP 500 (internal server error)
- HTTP 502/503 (service unavailable)
- Network errors (`net.Error`)

**Retry Configuration**:
```go
err := client.RetryWithBackoff(ctx, &client.RetryConfig{
    MaxRetries: 3,
    BaseDelay:  500 * time.Millisecond,
    MaxDelay:   30 * time.Second,
}, func() error {
    return uapAPI.Db().UpdatePolicy(policy)
})
```

**Non-Retryable Errors**:
- 400 (validation errors)
- 401 (authentication failures)
- 403 (permission denied)
- 404 (not found)
- 409 (conflicts)

### Error Handling Best Practices

1. **API-only validation for business rules**: Provider relies on API to validate time_frame, access_window, name length, tag count (FR-034). Reduces code complexity and eliminates drift between provider and API validation rules.
2. **Client-side validation for provider constructs**: Only validate provider-level constructs (composite ID format, enum values with custom validators) to provide immediate feedback (FR-036).
3. **Use MapError() consistently**: All API errors go through centralized mapping to provide clear, actionable error messages (FR-031).
4. **Provide actionable guidance**: Error messages include next steps for users (FR-032).
5. **Never log sensitive data**: Exclude passwords, tokens, secrets from logs.
6. **Retry transient failures**: Automatic retry with exponential backoff for network errors, rate limits, 5xx errors (FR-033).
7. **Fail fast on permanent errors**: No retry for validation (400), auth (401), permission (403), not found (404), or conflict (409) errors.

---

## Summary

**Decisions Made**:

1. ✅ **ARK SDK API Structure** - Fully documented with field mappings
2. ✅ **Read-Modify-Write** - 6-step algorithm with code examples
3. ✅ **Composite IDs** - 3-part for principals, 2-part for databases
4. ✅ **Status Management** - "Active"|"Suspended" only, custom validator
5. ✅ **ForceNew Attributes** - name and location_type for policy resource
6. ✅ **Pagination** - Channel-based pattern, early termination

**Alternatives Considered**:

- 2-part composite ID for principals → Rejected (ambiguous with duplicate IDs)
- Expose all status values → Rejected (causes state conflicts)
- Make location_type updatable → Rejected (breaks target compatibility)
- Manual pagination → Rejected (SDK provides channel abstraction)

**Next Step**: Phase 1 - Generate data-model.md, contracts/, quickstart.md
