# Azure VM Policy Workaround Implementation Guide

**Status**: Ready for implementation
**Priority**: P2 (unblock Azure support)
**Estimated Time**: 2-3 hours
**Created**: 2025-11-24 (Session 9)

---

## Problem Summary

**SDK Bug**: ARK SDK v1.5.0 constant `WorkspaceTypeAzure = "AZURE"` (uppercase) is used as JSON targets key, but SIA API expects `"Azure"` (mixed case). This causes all Azure VM policy creation requests to fail with HTTP 500 errors.

**GitHub Issue**: Submitted to https://github.com/cyberark/ark-sdk-golang (see `GITHUB-ISSUE-ARK-SDK-AZURE-BUG.md`)

**Impact**: Azure VM policies completely broken. AWS/GCP/FQDN work correctly.

---

## Proof of Bug

### Evidence 1: SDK PoC Test
**File**: `internal/provider/azure_sdk_poc_test.go`

**Result**: Both "Azure" and "AZURE" fail:
- `LocationType="Azure"` → SDK Serialize() rejects ("unsupported workspace type")
- `LocationType="AZURE"` → API rejects with HTTP 500

### Evidence 2: Direct API Test
**Test Date**: 2025-11-24

**Results**:
```bash
# Test 1: targets: {"Azure": {...}}
HTTP 200 - Policy created successfully
Policy ID: c0d56d12-93dd-48e3-8a9e-01ba0df2a779

# Test 2: targets: {"AZURE": {...}}
HTTP 500 - INTERNAL_SERVER_ERROR
```

**Conclusion**: API requires mixed case "Azure", SDK sends uppercase "AZURE"

---

## Workaround Approach: Option 2 (Recommended)

**Pattern**: Follow `internal/client/delete_workarounds.go` pattern

**Strategy**:
1. Detect Azure policies in Create() and Update() methods
2. Bypass SDK's `Serialize()` method for Azure only
3. Manually construct JSON with correct `"Azure"` key
4. Make direct HTTP POST using authenticated context
5. AWS/GCP continue using normal SDK path (unaffected)

---

## Implementation Steps

### Step 1: Create Workaround File

**File**: `internal/client/azure_policy_workaround.go`

```go
package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/cyberark/ark-sdk-golang/pkg/auth"
    uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
)

// AddAzureVMPolicyDirect creates Azure VM policy by bypassing SDK's Serialize()
// and manually constructing JSON with "Azure" (mixed case) targets key.
//
// Bug: SDK v1.5.0 uses "AZURE" but API expects "Azure"
// GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/XXX
// TODO: Remove this workaround when SDK v1.6.0+ fixes case sensitivity
func AddAzureVMPolicyDirect(
    ctx context.Context,
    arkAuth *auth.ArkISPAuth,
    policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy,
) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error) {
    // Get base URL from auth context
    baseURL := arkAuth.GetServiceURL("uap")
    if baseURL == "" {
        return nil, fmt.Errorf("failed to get UAP service URL")
    }

    // Build request body manually
    requestBody := map[string]interface{}{
        "metadata":   serializeMetadata(policy.Metadata),
        "principals": serializePrincipals(policy.Principals),
        "targets": map[string]interface{}{
            "Azure": policy.Targets.AzureResource.Serialize(), // Use "Azure" not "AZURE"
        },
        "behavior":   policy.Behavior.Serialize(),
        "conditions": policy.Conditions.Serialize(),
    }

    if policy.DelegationClassification != "" {
        requestBody["delegationClassification"] = policy.DelegationClassification
    }

    // Marshal to JSON
    jsonData, err := json.Marshal(requestBody)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal policy: %w", err)
    }

    // Create HTTP request
    url := fmt.Sprintf("%s/api/policies", baseURL)
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Add authentication header
    token := arkAuth.GetToken()
    if token == nil {
        return nil, fmt.Errorf("no authentication token available")
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
    req.Header.Set("Content-Type", "application/json")

    // Send request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    // Check response
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
    }

    // Parse response
    var createdPolicy uapsiavmmodels.ArkUAPSIAVMAccessPolicy
    if err := json.NewDecoder(resp.Body).Decode(&createdPolicy); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &createdPolicy, nil
}

// Helper functions to serialize parts of the policy
func serializeMetadata(metadata interface{}) map[string]interface{} {
    // Implement metadata serialization
    // Reference: policy.Metadata.Serialize() pattern
}

func serializePrincipals(principals interface{}) []map[string]interface{} {
    // Implement principals serialization
    // Reference: existing SDK patterns
}

// UpdateAzureVMPolicyDirect updates Azure VM policy bypassing SDK
// (Similar pattern to AddAzureVMPolicyDirect but uses PUT instead of POST)
func UpdateAzureVMPolicyDirect(
    ctx context.Context,
    arkAuth *auth.ArkISPAuth,
    policyID string,
    policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy,
) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error) {
    // Similar implementation using PUT method
    // URL: /api/policies/{policyID}
}
```

---

### Step 2: Modify vm_policy_resource.go Create() Method

**File**: `internal/provider/vm_policy_resource.go`

**Location**: Lines ~738-753 (Create method)

**Current Code**:
```go
// Create policy with retry logic
var created *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
err := client.RetryWithBackoff(ctx, &client.RetryConfig{
    MaxRetries: client.DefaultMaxRetries,
    BaseDelay:  client.BaseDelay,
    MaxDelay:   client.MaxDelay,
}, func() error {
    var createErr error
    created, createErr = vmService.AddPolicy(policy)
    return createErr
})
```

**New Code**:
```go
// Create policy with retry logic
var created *uapsiavmmodels.ArkUAPSIAVMAccessPolicy

// WORKAROUND: Azure VM policies require direct API call due to SDK bug
// GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/XXX
// TODO: Remove when SDK v1.6.0+ fixes WorkspaceTypeAzure case sensitivity
if plan.LocationType.ValueString() == "Azure" || plan.LocationType.ValueString() == "AZURE" {
    // Use workaround for Azure policies
    err := client.RetryWithBackoff(ctx, &client.RetryConfig{
        MaxRetries: client.DefaultMaxRetries,
        BaseDelay:  client.BaseDelay,
        MaxDelay:   client.MaxDelay,
    }, func() error {
        var createErr error
        created, createErr = client.AddAzureVMPolicyDirect(ctx, providerData.AuthContext, policy)
        return createErr
    })
} else {
    // AWS/GCP/FQDN use normal SDK path
    err := client.RetryWithBackoff(ctx, &client.RetryConfig{
        MaxRetries: client.DefaultMaxRetries,
        BaseDelay:  client.BaseDelay,
        MaxDelay:   client.MaxDelay,
    }, func() error {
        var createErr error
        created, createErr = vmService.AddPolicy(policy)
        return createErr
    })
}
```

---

### Step 3: Modify vm_policy_resource.go Update() Method

**File**: `internal/provider/vm_policy_resource.go`

**Location**: Lines ~1011-1026 (Update method, after building updated policy)

**Similar Pattern**:
```go
// Update policy with retry logic
var updated *uapsiavmmodels.ArkUAPSIAVMAccessPolicy

// WORKAROUND: Azure VM policies require direct API call
if plan.LocationType.ValueString() == "Azure" || plan.LocationType.ValueString() == "AZURE" {
    err := client.RetryWithBackoff(ctx, &client.RetryConfig{...}, func() error {
        var updateErr error
        updated, updateErr = client.UpdateAzureVMPolicyDirect(ctx, providerData.AuthContext, policyID, policy)
        return updateErr
    })
} else {
    // AWS/GCP/FQDN use normal SDK path
    err := client.RetryWithBackoff(ctx, &client.RetryConfig{...}, func() error {
        var updateErr error
        updated, updateErr = vmService.UpdatePolicy(policyID, policy)
        return updateErr
    })
}
```

---

### Step 4: Test Azure Policies

**Test File**: `internal/provider/vm_policy_resource_test.go`

**Test**: `TestAccVMPolicy_azureBasic` (already exists, currently failing)

**Run Test**:
```bash
export CYBERARK_USERNAME="timtest@cyberark.cloud.40562"
export CYBERARK_PASSWORD='nvk*phv*hfd3ATR2rfc'
export TF_ACC=1

go test ./internal/provider -v -run TestAccVMPolicy_azureBasic -timeout 10m
```

**Expected Result After Workaround**:
- Test should pass (HTTP 200, policy created)
- Azure policy ID should be returned
- All attributes should match

---

## Reference Files

### Pattern Reference
**File**: `internal/client/delete_workarounds.go`

**Shows**:
- How to make direct HTTP calls
- How to handle authentication
- How to construct URLs
- Error handling patterns
- TODO comment format

### Test Reference
**File**: `internal/provider/azure_sdk_poc_test.go`

**Shows**:
- Complete policy structure for Azure
- Authentication setup
- Expected behavior

### Bug Report
**File**: `specs/001-vm-access-policies/GITHUB-ISSUE-ARK-SDK-AZURE-BUG.md`

**Shows**:
- Complete bug description
- Proof via multiple methods
- GitHub issue format (ready to submit)

---

## Verification Checklist

After implementing the workaround:

- [ ] `internal/client/azure_policy_workaround.go` created
- [ ] `AddAzureVMPolicyDirect()` implemented
- [ ] `UpdateAzureVMPolicyDirect()` implemented (if needed)
- [ ] `vm_policy_resource.go` Create() method modified
- [ ] `vm_policy_resource.go` Update() method modified
- [ ] TODO comments added with GitHub issue link
- [ ] Azure acceptance test passes (`TestAccVMPolicy_azureBasic`)
- [ ] AWS/GCP tests still pass (workaround doesn't affect them)
- [ ] Documentation updated (note workaround is temporary)

---

## Cleanup Plan (Future)

**When SDK v1.6.0+ is released with fix:**

1. Remove `internal/client/azure_policy_workaround.go`
2. Revert Create() and Update() methods to use SDK directly
3. Verify Azure tests still pass with SDK fix
4. Remove all TODO comments referencing the workaround

**Files to grep for cleanup**:
```bash
rg "TODO.*Azure.*workaround" --glob "*.go"
rg "GitHub Issue.*ark-sdk-golang" --glob "*.go"
```

---

## Time Estimate

- **Step 1** (Create workaround file): 1 hour
- **Step 2** (Modify Create method): 15 minutes
- **Step 3** (Modify Update method): 15 minutes
- **Step 4** (Test & verify): 30 minutes
- **Buffer**: 30 minutes

**Total**: 2-3 hours

---

## Success Criteria

✅ Azure VM policies can be created via Terraform
✅ Azure VM policies can be updated via Terraform
✅ `TestAccVMPolicy_azureBasic` passes
✅ AWS/GCP/FQDN tests remain unaffected
✅ Code includes TODO comments for future cleanup
✅ Workaround documented in CLAUDE.md Known TODOs section

---

**Ready to implement when User Story 6 (Update Operations) is complete!**
