## Summary

The SDK constant `WorkspaceTypeAzure = "AZURE"` (uppercase) is used as the JSON targets key in `ArkUAPSIAVMPlatformTargets.Serialize()`, but the SIA API requires `"Azure"` (mixed case). This causes all Azure VM policy creation requests to fail with HTTP 500 errors. AWS and GCP work correctly because their constants match the API expectation.

## Steps to Reproduce

### Method 1: SDK

```go
policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{
    Metadata: uapcommonmodels.ArkUAPMetadata{
        Name: "test-azure",
        PolicyEntitlement: uapcommonmodels.ArkUAPPolicyEntitlement{
            LocationType: "Azure",  // or "AZURE"
            TargetCategory: "VM",
        },
    },
}
policy.Targets.AzureResource = &uapsiavmmodels.ArkUAPSIAVMAzureResource{
    Regions: []string{"eastus"},
}
// ... set principals, behavior, conditions ...

created, err := vmService.AddPolicy(policy)
// Result: HTTP 500 Internal Server Error
```

### Method 2: ARK CLI

```bash
# Already logged in via: ark login

# Create azure-policy.json
cat > azure-policy.json << 'EOF'
{
  "metadata": {
    "name": "test-azure-bug",
    "timezone": "UTC",
    "policyEntitlement": {
      "locationType": "AZURE",
      "targetCategory": "VM",
      "policyType": "Recurring"
    },
    "status": {"status": "Active"}
  },
  "principals": [{
    "id": "a1cfc60d-80e1-489c-8251-c0d7bcb84fc9",
    "name": "user@example.com",
    "type": "USER"
  }],
  "targets": {
    "Azure": {
      "regions": ["eastus"],
      "tags": [],
      "resourceGroups": [],
      "vnetIds": [],
      "subscriptions": []
    }
  },
  "behavior": {"connectAs": {"ssh": {"username": "azureuser"}}},
  "conditions": {
    "max_session_duration": 2,
    "access_window": {"days_of_the_week": [0,1,2,3,4,5,6]},
    "idle_time": 10
  }
}
EOF

ark exec sia vm add-policy --file azure-policy.json

# Result: ERROR: Failed to add policy - [500] - INTERNAL_SERVER_ERROR
```

### Method 3: Direct API

```bash
# Get token
TOKEN=$(curl -s "https://<tenant>.id.cyberark.cloud/oauth2/platformtoken" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials&client_id=<user>&client_secret=<pass>" \
  | jq -r '.access_token')

# Test 1: API with "Azure" (mixed case) - SUCCESS
curl -i "https://<tenant>.uap.cyberark.cloud/api/policies" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"targets": {"Azure": {...}}, ...}'
# Result: HTTP 200

# Test 2: API with "AZURE" (uppercase) - FAILURE
curl -i "https://<tenant>.uap.cyberark.cloud/api/policies" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"targets": {"AZURE": {...}}, ...}'
# Result: HTTP 500
```

## Expected Results

Azure VM policies should be created successfully, identical to AWS and GCP. The SDK should generate JSON with `"Azure"` as the targets key to match the API requirement.

## Actual Results

**SDK generates:**
```json
{"targets": {"AZURE": {...}}}
```

**API expects:**
```json
{"targets": {"Azure": {...}}}
```

**Result:** HTTP 500 Internal Server Error

### Root Cause

**File:** `pkg/services/uap/sia/vm/models/ark_uap_sia_vm_targets.go`

**Lines 12-14:**
```go
const (
    WorkspaceTypeAWS    = "AWS"
    WorkspaceTypeAzure  = "AZURE"  // ❌ Should be "Azure"
    WorkspaceTypeGCP    = "GCP"
)
```

**Lines 409-410:**
```go
if workspace == common.WorkspaceTypeAzure && t.AzureResource != nil {
    data[common.WorkspaceTypeAzure] = t.AzureResource.Serialize()  // Uses "AZURE"
```

The constant is used as both the comparison value AND the JSON key. AWS/GCP work because their constants match API expectations. Azure fails due to case mismatch.

## Reproducible

- [x] Always
- [ ] Sometimes
- [ ] Non-Reproducible

100% reproducible across v1.5.0 and current main branch.

## Version/Tag number

- **SDK:** `github.com/cyberark/ark-sdk-golang v1.5.0`
- **File:** `pkg/services/uap/sia/vm/models/ark_uap_sia_vm_targets.go:12-14, 405-418`
- **API:** CyberArk SIA UAP (production, tested 2025-11-24)

## Environment setup

- **OS:** Linux (Ubuntu 22.04 on WSL2)
- **Go:** 1.25.0
- **Service:** CyberArk SIA - VM Access Policies
- **Context:** Developing Terraform provider using ARK SDK

## Additional Information

### Comparison

| Cloud | SDK Constant | API Expects | Status |
|-------|-------------|-------------|--------|
| AWS | `"AWS"` | `"AWS"` | ✅ Works |
| GCP | `"GCP"` | `"GCP"` | ✅ Works |
| Azure | `"AZURE"` | `"Azure"` | ❌ Broken |

### Proposed Fix (Backward Compatible)

```go
var workspaceAPIKeys = map[string]string{
    WorkspaceTypeAWS:   "AWS",
    WorkspaceTypeAzure: "Azure",  // Map AZURE → Azure
    WorkspaceTypeGCP:   "GCP",
}

func (t *ArkUAPSIAVMPlatformTargets) Serialize(workspace string) (map[string]interface{}, error) {
    data := make(map[string]interface{})
    apiKey := workspaceAPIKeys[workspace]

    if workspace == common.WorkspaceTypeAzure && t.AzureResource != nil {
        data[apiKey] = t.AzureResource.Serialize()  // Uses "Azure"
    }
    // ...
}
```

**Alternative:** Change constant to `WorkspaceTypeAzure = "Azure"` (breaking change)

### Impact

- **Severity:** High - blocks all Azure VM policy automation
- **Affected:** Anyone using SDK/CLI for Azure VM access policies
- **Workaround:** Direct HTTP client bypassing SDK's Serialize()
