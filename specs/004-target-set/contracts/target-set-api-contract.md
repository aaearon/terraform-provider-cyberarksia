# API Contract: Target Set Resource

**Service**: CyberArk Secure Infrastructure Access (SIA)
**ARK SDK Service**: `siaAPI.WorkspacesTargetSets()`
**API Base Path**: `/api/targetsets`
**Date**: 2025-11-08

## Overview

This document defines the API contract between the Terraform provider and the CyberArk SIA API for target set management. All operations use the ARK SDK except DELETE (workaround required).

---

## Authentication

**Method**: OAuth2 Bearer Token (ISP Authentication)
**SDK Handling**: Automatic token management via ARK SDK
**Token Lifetime**: 15 minutes (automatic refresh by SDK)
**Headers**: Set by ARK SDK automatically

---

## CREATE Operation

### Endpoint

```
POST /api/targetsets
```

### ARK SDK Method

```go
siaAPI := auth.GetClient().SIA()
result, err := siaAPI.WorkspacesTargetSets().AddTargetSet(addRequest)
```

### Request Model

**Note**: API allows `secret_id` and `secret_type` to be omitted (fields have `omitempty` tag), but the provider enforces them as required since target sets are non-functional without credentials.

```go
type ArkSIAAddTargetSet struct {
    Name                        string `json:"name" validate:"required"`
    Type                        string `json:"type" validate:"required,oneof=Domain Suffix Target"`
    SecretID                    string `json:"secret_id,omitempty"`        // API optional, provider enforces
    SecretType                  string `json:"secret_type,omitempty"`      // API optional, provider enforces
    ProvisionFormat             string `json:"provision_format,omitempty"`
    Description                 string `json:"description,omitempty"`
    EnableCertificateValidation bool   `json:"enable_certificate_validation,omitempty"`
}
```

### Request Example

```json
{
  "name": "prod.example.com",
  "type": "Domain",
  "secret_id": "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0",
  "secret_type": "ProvisionerUser",
  "provision_format": "<user>-<session-guid>",
  "description": "Production environment servers",
  "enable_certificate_validation": true
}
```

### Response (Success)

**Status Code**: `201 Created`

**Body**:
```json
{
  "target_set": {
    "id": "prod.example.com",
    "name": "prod.example.com",
    "type": "Domain",
    "secret_id": "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0",
    "secret_type": "ProvisionerUser",
    "provision_format": "<user>-<session-guid>",
    "description": "Production environment servers",
    "enable_certificate_validation": true
  }
}
```

### Response (Error - Duplicate Name)

**Status Code**: `409 Conflict`

**Body**:
```json
{
  "message": "Target set prod.example.com already exists"
}
```

### Validation

- ✅ Name uniqueness enforced (409 Conflict if duplicate)
- ✅ Type must be one of: Domain, Suffix, Target
- ✅ SecretType must be one of: ProvisionerUser, PCloudAccount
- ❌ No validation that secret_id exists (can be completely omitted or non-existent UUID)
- ❌ No validation that secret_type matches actual secret
- ❌ No validation of provision_format placeholders

### Provider Handling

```go
func (r *TargetSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data TargetSetModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    addRequest := &targetsets.ArkSIAAddTargetSet{
        Name:                        data.Name.ValueString(),
        Type:                        data.Type.ValueString(),
        SecretID:                    data.SecretID.ValueString(),
        SecretType:                  data.SecretType.ValueString(),
        ProvisionFormat:             data.ProvisionFormat.ValueString(),
        Description:                 data.Description.ValueString(),
        EnableCertificateValidation: data.EnableCertificateValidation.ValueBool(),
    }

    err := client.RetryWithBackoff(ctx, func() error {
        result, err := siaAPI.WorkspacesTargetSets().AddTargetSet(addRequest)
        if err != nil {
            return err
        }
        // Map response to state
        data.ID = types.StringValue(result.Name)
        return nil
    })

    if err != nil {
        resp.Diagnostics.Append(client.MapError(err, "Failed to create target set")...)
        return
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

---

## READ Operation

### Endpoint

```
GET /api/targetsets/{name}
```

### ARK SDK Method

```go
getRequest := &targetsets.ArkSIAGetTargetSet{
    ID: name,
}
result, err := siaAPI.WorkspacesTargetSets().TargetSet(getRequest)
```

### Request Parameters

- **Path**: `{name}` - Target set name (URL-encoded)

### Response (Success)

**Status Code**: `200 OK`

**Body**:
```json
{
  "target_set": {
    "id": "prod.example.com",
    "name": "prod.example.com",
    "type": "Domain",
    "secret_id": "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0",
    "secret_type": "ProvisionerUser",
    "provision_format": "<user>-<session-guid>",
    "description": "Production environment servers",
    "enable_certificate_validation": true
  }
}
```

### Response (Not Found)

**Status Code**: `404 Not Found`

**Body**:
```json
{
  "message": "Target set prod.example.com not found"
}
```

### Provider Handling

```go
func (r *TargetSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data TargetSetModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    getRequest := &targetsets.ArkSIAGetTargetSet{
        ID: data.Name.ValueString(),
    }

    result, err := siaAPI.WorkspacesTargetSets().TargetSet(getRequest)
    if err != nil {
        if client.IsNotFoundError(err) {
            // Resource deleted outside Terraform - remove from state
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.Append(client.MapError(err, "Failed to read target set")...)
        return
    }

    // Map response to state
    data.ID = types.StringValue(result.Name)
    data.Name = types.StringValue(result.Name)
    data.Type = types.StringValue(result.Type)
    data.SecretID = types.StringValue(result.SecretID)
    data.SecretType = types.StringValue(result.SecretType)
    data.ProvisionFormat = types.StringValue(result.ProvisionFormat)
    data.Description = types.StringValue(result.Description)
    data.EnableCertificateValidation = types.BoolValue(result.EnableCertificateValidation)

    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

---

## UPDATE Operation

### Endpoint

```
PUT /api/targetsets/{current-name}
```

### ARK SDK Method

```go
updateRequest := &targetsets.ArkSIAUpdateTargetSet{
    ID:   currentName, // OLD name (from state)
    Name: newName,     // NEW name (from plan) - may differ for renames
    // ... other fields
}
result, err := siaAPI.WorkspacesTargetSets().UpdateTargetSet(updateRequest)
```

### CRITICAL API BEHAVIOR

**PATCH-Like Semantics**: Server preserves fields NOT included in request body

**Rename Pattern**: Uses old name in URL path, new name in request body
- URL: `/api/targetsets/old-name`
- Body: `{"name": "new-name", ...}`

**DESTRUCTIVE BUG**: Sending UPDATE without `name` field in body causes:
- Returns: `500 Internal Server Error`
- Side Effect: **Target set is DELETED** (not just failed update)

**Provider Mitigation**: ALWAYS include `name` field in UPDATE requests

### Request Model

```go
type ArkSIAUpdateTargetSet struct {
    ID                          string `json:"-"` // Used in URL path, not body
    Name                        string `json:"name" validate:"required"` // MUST include
    Type                        string `json:"type,omitempty"`
    SecretID                    string `json:"secret_id,omitempty"`
    SecretType                  string `json:"secret_type,omitempty"`
    ProvisionFormat             string `json:"provision_format,omitempty"`
    Description                 string `json:"description,omitempty"`
    EnableCertificateValidation bool   `json:"enable_certificate_validation,omitempty"`
}
```

### Request Example (Rename)

**URL**: `PUT /api/targetsets/old-prod.example.com`

**Body**:
```json
{
  "name": "new-prod.example.com",
  "type": "Domain",
  "secret_id": "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0",
  "secret_type": "ProvisionerUser",
  "provision_format": "<user>-<session-guid>",
  "description": "Production environment servers - renamed",
  "enable_certificate_validation": true
}
```

### Request Example (In-Place Update)

**URL**: `PUT /api/targetsets/prod.example.com`

**Body**:
```json
{
  "name": "prod.example.com",
  "type": "Suffix",
  "secret_id": "new-uuid-here",
  "secret_type": "PCloudAccount",
  "description": "Updated description"
}
```

### Response (Success)

**Status Code**: `200 OK`

**Body**: Returns updated target set with all fields

### Response (Error - Missing Name Field)

**Status Code**: `500 Internal Server Error`

**Body**:
```json
{
  "message": "Error occurred while updating target set"
}
```

**Side Effect**: Target set is **DELETED** from system

### Provider Handling

```go
func (r *TargetSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state TargetSetModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    updateRequest := &targetsets.ArkSIAUpdateTargetSet{
        ID:                          state.Name.ValueString(), // OLD name in URL
        Name:                        plan.Name.ValueString(),  // NEW name (ALWAYS include)
        Type:                        plan.Type.ValueString(),
        SecretID:                    plan.SecretID.ValueString(),
        SecretType:                  plan.SecretType.ValueString(),
        ProvisionFormat:             plan.ProvisionFormat.ValueString(),
        Description:                 plan.Description.ValueString(),
        EnableCertificateValidation: plan.EnableCertificateValidation.ValueBool(),
    }

    err := client.RetryWithBackoff(ctx, func() error {
        result, err := siaAPI.WorkspacesTargetSets().UpdateTargetSet(updateRequest)
        if err != nil {
            return err
        }
        // Update ID to match new name (handles rename)
        plan.ID = types.StringValue(result.Name)
        return nil
    })

    if err != nil {
        resp.Diagnostics.Append(client.MapError(err, "Failed to update target set")...)
        return
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

---

## DELETE Operation

### Endpoint

```
DELETE /api/targetsets/{name}
```

### SDK BUG: CANNOT USE SDK METHOD

**ARK SDK v1.5.0 Bug**: `DeleteTargetSet()` passes nil body pointer → panic

**Error**:
```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x8 pc=0x6c7504]

goroutine 1 [running]:
bytes.(*Buffer).Len(...)
net/http.NewRequestWithContext(...)
github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/targetsets.(*ArkSIAWorkspacesTargetSetsService).DeleteTargetSet(...)
```

### WORKAROUND: Direct API Call

**Implementation**: Use `internal/client/delete_workarounds.go`

```go
func DeleteTargetSetDirect(ctx context.Context, auth *ISPAuthContext, name string) error {
    client := isp.FromISPAuth(auth.ISPAuth, "dpa", ".", "", nil)

    endpoint := fmt.Sprintf("/api/targetsets/%s", url.PathEscape(name))
    response, err := client.Delete(ctx, endpoint, map[string]string{})
    if err != nil {
        return fmt.Errorf("failed to delete target set: %w", err)
    }
    defer response.Body.Close()

    if response.StatusCode != 204 {
        body, _ := io.ReadAll(response.Body)
        return fmt.Errorf("unexpected status code %d: %s", response.StatusCode, string(body))
    }

    return nil
}
```

### Response (Success)

**Status Code**: `204 No Content`

**Body**: Empty

### Response (Not Found)

**Status Code**: `404 Not Found`

**Body**:
```json
{
  "message": "Target set prod.example.com not found"
}
```

### Response (Forward Slash Name Issue)

**Status Code**: `403 Forbidden`

**Body**:
```json
{
  "message": "Access denied"
}
```

**Root Cause**: URL path interpretation issue with forward slashes in name

### Provider Handling

```go
func (r *TargetSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var data TargetSetModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Use workaround due to SDK DELETE panic bug
    err := client.DeleteTargetSetDirect(ctx, providerData.AuthContext, data.Name.ValueString())
    if err != nil {
        if client.IsNotFoundError(err) {
            // Already deleted - success
            return
        }
        resp.Diagnostics.Append(client.MapError(err, "Failed to delete target set")...)
        return
    }
}
```

---

## LIST Operation

### Endpoint

```
GET /api/targetsets
```

### ARK SDK Method

```go
result, err := siaAPI.WorkspacesTargetSets().ListTargetSets()
```

### Response (Success)

**Status Code**: `200 OK`

**Body**:
```json
{
  "target_sets": [
    {
      "id": "prod.example.com",
      "name": "prod.example.com",
      "type": "Domain",
      ...
    },
    {
      "id": "staging.example.com",
      "name": "staging.example.com",
      "type": "Domain",
      ...
    }
  ]
}
```

### Usage

Not used by Terraform resource (individual resource CRUD only), but may be useful for data sources or testing.

---

## Error Handling

### Error Classification

| HTTP Status | Error Type | Provider Handling |
|-------------|------------|-------------------|
| 400 | Bad Request | User error - show API message |
| 401 | Unauthorized | Auth error - check credentials |
| 403 | Forbidden | Permission error - check user permissions |
| 404 | Not Found | Resource deleted - remove from state (READ) or treat as success (DELETE) |
| 409 | Conflict | Duplicate name - show API message |
| 500 | Internal Server Error | API error - retry with backoff, show API message |
| 502/503/504 | Service Unavailable | Temporary error - retry with backoff |

### Retry Strategy

**Implementation**: `internal/client/retry.go`

**Configuration**:
- Initial delay: 1 second
- Max delay: 30 seconds
- Max retries: 3
- Backoff multiplier: 2x

**Retryable Errors**:
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- 504 Gateway Timeout
- Network timeout errors

**Non-Retryable Errors**:
- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found
- 409 Conflict

---

## SDK Version Compatibility

**Current SDK**: ARK SDK v1.5.0 (`github.com/cyberark/ark-sdk-golang`)

**Known Issues**:
- DELETE panic bug (requires workaround)
- UPDATE without name field bug (requires field inclusion)

**Future SDK**: v1.6.0+ expected to fix DELETE panic

**Migration Plan**:
1. When SDK v1.6.0+ released, test DELETE method
2. If fixed, remove workaround from `delete_workarounds.go`
3. Update resource to use SDK DELETE method
4. Remove workaround documentation

---

## Testing Endpoints

### Manual Testing (via ARK CLI)

```bash
# Create
ark sia targetsets add \
  --name "test.example.com" \
  --type Domain \
  --secret-id "uuid-here" \
  --secret-type ProvisionerUser

# Read
ark sia targetsets get --name "test.example.com"

# Update
ark sia targetsets update \
  --name "test.example.com" \
  --description "Updated description"

# Delete (WILL PANIC - use direct API call)
# ark sia targetsets delete --name "test.example.com"  # DO NOT USE

# List
ark sia targetsets list
```

### Direct API Testing (curl)

```bash
# Get access token
TOKEN=$(ark auth login --username "$CYBERARK_USERNAME" --password "$CYBERARK_PASSWORD" --json | jq -r '.access_token')

# Create
curl -X POST "https://your-tenant.cyberark.cloud/api/targetsets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test.example.com",
    "type": "Domain",
    "secret_id": "uuid-here",
    "secret_type": "ProvisionerUser"
  }'

# Read
curl -X GET "https://your-tenant.cyberark.cloud/api/targetsets/test.example.com" \
  -H "Authorization: Bearer $TOKEN"

# Update
curl -X PUT "https://your-tenant.cyberark.cloud/api/targetsets/test.example.com" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test.example.com",
    "type": "Suffix",
    "secret_id": "uuid-here",
    "secret_type": "ProvisionerUser",
    "description": "Updated"
  }'

# Delete
curl -X DELETE "https://your-tenant.cyberark.cloud/api/targetsets/test.example.com" \
  -H "Authorization: Bearer $TOKEN"
```

---

**Contract Status**: ✅ COMPLETE
**SDK Version**: v1.5.0 (with workarounds)
**Next Artifact**: Quick Start Guide (quickstart.md)
