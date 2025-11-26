// Package client provides CyberArk SIA API client wrappers
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/cyberark/ark-sdk-golang/pkg/common"
	"github.com/cyberark/ark-sdk-golang/pkg/common/isp"
	vmsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm/models"
	uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ARK SDK v1.5.0 BUG WORKAROUNDS
//
// This file contains direct HTTP client workarounds for multiple bugs in ARK SDK v1.5.0:
//
// 1. DELETE OPERATIONS - Nil body pointer panic
//    Root Cause: pkg/common/ark_client.go:556-576
//      - Line 556: var bodyBytes *bytes.Buffer (defaults to nil)
//      - Line 557-575: Only initializes bodyBytes if body != nil
//      - Line 576: Passes nil pointer to http.NewRequestWithContext()
//      - Result: Panic when http.Request calls bodyBytes.Len()
//    Affected SDK Methods:
//      - DeleteDatabase() - pkg/services/sia/workspaces/db/ark_sia_workspaces_db_service.go:188
//      - DeleteSecret() - pkg/services/sia/secrets/db/ark_sia_secrets_db_service.go:343
//      - DeletePolicy() - pkg/services/sia/policies/ark_sia_policies_service.go
//    Workaround: Pass empty map map[string]string{} instead of nil
//
// 2. UPDATE OPERATIONS - Wrong HTTP method (POST instead of PUT)
//    Root Cause: pkg/services/sia/secrets/vm/ark_sia_secrets_vm_service.go:153
//      - ChangeSecret() uses client.Post() instead of client.Put()
//      - Result: 403 "Invalid key=value pair" auth signature mismatch
//    Affected SDK Methods:
//      - ChangeSecret() - VM secret updates fail completely
//    Workaround: Make PUT request directly with correct endpoint
//
// 3. AZURE VM POLICY SERIALIZATION - Wrong JSON key casing
//    Root Cause: pkg/common/constants.go
//      - WorkspaceTypeAzure = "AZURE" (uppercase) is used as JSON targets key
//      - SIA API expects "Azure" (mixed case) for targets key
//      - Result: HTTP 500 INTERNAL_SERVER_ERROR on policy creation
//    Affected SDK Methods:
//      - AddPolicy() - Azure VM policies fail completely
//      - UpdatePolicy() - Azure VM policy updates fail completely
//    Workaround: Bypass Serialize() and manually construct JSON with correct "Azure" key
//    GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/32
//
// 4. TARGET SET OPERATIONS - DELETE panic + UPDATE omitempty issues
//    Root Cause: Same nil body panic as #1, plus omitempty serialization drops required fields
//    Affected SDK Methods:
//      - DeleteTargetSet() - Nil body panic
//      - UpdateTargetSet() - omitempty drops provision_format when empty string intended
//    Workaround: Direct HTTP calls with proper body handling
//
// Pattern: All workarounds bypass SDK methods and make direct HTTP calls using ISP client
//
// REMOVAL CRITERIA (for LLM maintainers):
// - Check ARK SDK changelog for v1.6.0+ release
// - Verify fix by testing: DeleteDatabase(), DeleteSecret(), DeletePolicy() with nil body
// - Verify Azure fix by creating Azure VM policy via SDK AddPolicy()
// - Once verified, delete this file and update:
//   - CLAUDE.md (remove workaround references)
//   - docs/development/vm-policy-implementation.md (update CRUD section)
//   - All resources using these helpers (switch to SDK methods)
//
// TODO: Remove this file when ARK SDK v1.6.0+ fixes these bugs

const (
	// Database workspace DELETE endpoint (from SDK source)
	databaseWorkspaceDeleteURL = "/api/adb/resources/%d"

	// Secret DELETE endpoint (from SDK source)
	secretDeleteURL = "/api/adb/secretsmgmt/secrets/%s" //nolint:gosec // URL path template, not a credential

	// VM Secret DELETE endpoint (from SDK source)
	vmSecretDeleteURL = "/api/secrets/%s" //nolint:gosec // URL path template, not a credential

	// Policy DELETE endpoint (from SDK source)
	policyDeleteURL = "/api/policies/%s"

	// Target Set DELETE endpoint (from SDK source)
	targetSetDeleteURL = "/api/targetsets/%s"

	// Target Set UPDATE endpoint (from SDK source)
	targetSetUpdateURL = "/api/targetsets/%s"

	// VM Policy CREATE endpoint (from SDK source)
	vmPolicyCreateURL = "/api/policies"

	// VM Policy UPDATE endpoint (from SDK source)
	vmPolicyUpdateURL = "/api/policies/%s"
)

// DeleteDatabaseWorkspaceDirect bypasses SDK's buggy DeleteDatabase() method
// and makes HTTP DELETE request directly with empty body workaround.
//
// This function replicates the SDK's delete logic but passes map[string]string{}
// instead of nil to avoid the panic.
//
// API Endpoint: DELETE /api/adb/resources/{id}
// Success Response: HTTP 204 No Content
// Error Responses:
//   - 404 Not Found: Resource already deleted (treated as success)
//   - 409 Conflict: Resource in use (e.g., has active connections)
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - databaseID: Database workspace ID (integer)
//
// Returns:
//   - error: nil on success (including 404), error on failure
func DeleteDatabaseWorkspaceDirect(ctx context.Context, authCtx *ISPAuthContext, databaseID int) error {
	tflog.Debug(ctx, "Executing DELETE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "database_workspace",
		"database_id":   databaseID,
		"workaround":    "empty_map_body",
	})

	// Create temporary ISP service client (same pattern as CertificatesClient)
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"dpa", // Service name (constructs https://{subdomain}.dpa.{domain})
		".",   // Separator
		"",    // Base path
		nil,   // No refresh callback needed for one-time operation
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for DELETE workaround", map[string]interface{}{
			"database_id": databaseID,
			"error":       err.Error(),
		})
		return fmt.Errorf("failed to create ISP client for DELETE: %w", err)
	}

	// Construct endpoint URL
	endpoint := fmt.Sprintf(databaseWorkspaceDeleteURL, databaseID)

	// Execute DELETE with empty map workaround (NOT nil!)
	// This prevents the SDK panic by ensuring bodyBytes is initialized
	response, err := client.Delete(ctx, endpoint, map[string]string{})
	if err != nil {
		tflog.Error(ctx, "DELETE workaround request failed", map[string]interface{}{
			"database_id": databaseID,
			"error":       err.Error(),
		})
		return fmt.Errorf("failed to delete database workspace %d: %w", databaseID, err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "DELETE workaround response received", map[string]interface{}{
		"database_id": databaseID,
		"status_code": response.StatusCode,
	})

	// Handle HTTP status codes (same as SDK's DeleteDatabase logic)
	if response.StatusCode == http.StatusNotFound {
		tflog.Debug(ctx, "Database workspace already deleted (404)", map[string]interface{}{
			"database_id": databaseID,
		})
		// Resource already deleted - treat as success
		return nil
	}

	if response.StatusCode != http.StatusNoContent {
		tflog.Error(ctx, "DELETE workaround failed with unexpected status", map[string]interface{}{
			"database_id": databaseID,
			"status_code": response.StatusCode,
		})
		return fmt.Errorf("failed to delete database workspace %d - [%d] - [%s]",
			databaseID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	tflog.Debug(ctx, "DELETE workaround successful", map[string]interface{}{
		"database_id": databaseID,
	})

	return nil
}

// DeleteSecretDirect bypasses SDK's buggy DeleteSecret() method
// and makes HTTP DELETE request directly with empty body workaround.
//
// This function replicates the SDK's delete logic but passes map[string]string{}
// instead of nil to avoid the panic.
//
// API Endpoint: DELETE /api/adb/secretsmgmt/secrets/{id}
// Success Response: HTTP 204 No Content
// Error Responses:
//   - 404 Not Found: Secret already deleted (treated as success)
//   - 409 Conflict: Secret in use (e.g., referenced by database workspace)
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - secretID: Secret ID (UUID string)
//
// Returns:
//   - error: nil on success (including 404), error on failure
func DeleteSecretDirect(ctx context.Context, authCtx *ISPAuthContext, secretID string) error {
	tflog.Debug(ctx, "Executing DELETE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "secret",
		"secret_id":     secretID,
		"workaround":    "empty_map_body",
	})

	// Create temporary ISP service client (same pattern as CertificatesClient)
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"dpa", // Service name (constructs https://{subdomain}.dpa.{domain})
		".",   // Separator
		"",    // Base path
		nil,   // No refresh callback needed for one-time operation
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for DELETE workaround", map[string]interface{}{
			"secret_id": secretID,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to create ISP client for DELETE: %w", err)
	}

	// Construct endpoint URL
	endpoint := fmt.Sprintf(secretDeleteURL, secretID)

	// Execute DELETE with empty map workaround (NOT nil!)
	// This prevents the SDK panic by ensuring bodyBytes is initialized
	response, err := client.Delete(ctx, endpoint, map[string]string{})
	if err != nil {
		tflog.Error(ctx, "DELETE workaround request failed", map[string]interface{}{
			"secret_id": secretID,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to delete secret %s: %w", secretID, err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "DELETE workaround response received", map[string]interface{}{
		"secret_id":   secretID,
		"status_code": response.StatusCode,
	})

	// Handle HTTP status codes (same as SDK's DeleteSecret logic)
	if response.StatusCode == http.StatusNotFound {
		tflog.Debug(ctx, "Secret already deleted (404)", map[string]interface{}{
			"secret_id": secretID,
		})
		// Secret already deleted - treat as success
		return nil
	}

	if response.StatusCode != http.StatusNoContent {
		tflog.Error(ctx, "DELETE workaround failed with unexpected status", map[string]interface{}{
			"secret_id":   secretID,
			"status_code": response.StatusCode,
		})
		return fmt.Errorf("failed to delete secret %s - [%d] - [%s]",
			secretID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	tflog.Debug(ctx, "DELETE workaround successful", map[string]interface{}{
		"secret_id": secretID,
	})

	return nil
}

// DeleteVMSecretDirect bypasses SDK's buggy DeleteSecret() method for VM secrets
// and makes HTTP DELETE request directly with empty body workaround.
//
// This function replicates the SDK's delete logic but passes map[string]string{}
// instead of nil to avoid the panic.
//
// API Endpoint: DELETE /api/secrets/{id}
// Success Response: HTTP 204 No Content
// Error Responses:
//   - 404 Not Found: VM secret already deleted (treated as success)
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - secretID: VM Secret ID (UUID string)
//
// Returns:
//   - error: nil on success (including 404), error on failure
func DeleteVMSecretDirect(ctx context.Context, authCtx *ISPAuthContext, secretID string) error {
	tflog.Debug(ctx, "Executing DELETE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "vm_secret",
		"secret_id":     secretID,
		"workaround":    "empty_map_body",
	})

	// Create temporary ISP service client (same pattern as database secrets)
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"dpa", // Service name (constructs https://{subdomain}.dpa.{domain})
		".",   // Separator
		"",    // Base path
		nil,   // No refresh callback needed for one-time operation
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for DELETE workaround", map[string]interface{}{
			"secret_id": secretID,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to create ISP client for DELETE: %w", err)
	}

	// Construct endpoint URL
	endpoint := fmt.Sprintf(vmSecretDeleteURL, secretID)

	// Execute DELETE with empty map workaround (NOT nil!)
	// This prevents the SDK panic by ensuring bodyBytes is initialized
	response, err := client.Delete(ctx, endpoint, map[string]string{})
	if err != nil {
		tflog.Error(ctx, "DELETE workaround request failed", map[string]interface{}{
			"secret_id": secretID,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to delete VM secret %s: %w", secretID, err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "DELETE workaround response received", map[string]interface{}{
		"secret_id":   secretID,
		"status_code": response.StatusCode,
	})

	// Handle HTTP status codes (same as SDK's DeleteSecret logic)
	if response.StatusCode == http.StatusNotFound {
		tflog.Debug(ctx, "VM secret already deleted (404)", map[string]interface{}{
			"secret_id": secretID,
		})
		// VM secret already deleted - treat as success
		return nil
	}

	if response.StatusCode != http.StatusNoContent {
		tflog.Error(ctx, "DELETE workaround failed with unexpected status", map[string]interface{}{
			"secret_id":   secretID,
			"status_code": response.StatusCode,
		})
		return fmt.Errorf("failed to delete VM secret %s - [%d] - [%s]",
			secretID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	tflog.Debug(ctx, "DELETE workaround successful", map[string]interface{}{
		"secret_id": secretID,
	})

	return nil
}

// DeleteDatabasePolicyDirect bypasses SDK's buggy DeletePolicy() method
// and makes HTTP DELETE request directly with empty body workaround.
//
// This function replicates the SDK's delete logic but passes map[string]string{}
// instead of nil to avoid the panic.
//
// API Endpoint: DELETE /api/policies/{id}
// Success Response: HTTP 204 No Content
// Error Responses:
//   - 404 Not Found: Policy already deleted (treated as success)
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - policyID: Policy ID (UUID string)
//
// Returns:
//   - error: nil on success (including 404), error on failure
func DeleteDatabasePolicyDirect(ctx context.Context, authCtx *ISPAuthContext, policyID string) error {
	tflog.Debug(ctx, "Executing DELETE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "database_policy",
		"policy_id":     policyID,
		"workaround":    "empty_map_body",
	})

	// Create temporary ISP service client for UAP (policies use different service than SIA)
	// UAP policies use "uap" service: https://{subdomain}.uap.{domain}
	// SIA resources (database_workspace, secret) use "dpa" service: https://{subdomain}.dpa.{domain}
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"uap", // Service name for UAP policies (NOT "dpa")
		".",   // Separator
		"",    // Base path
		nil,   // No refresh callback needed for one-time operation
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for DELETE workaround", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to create ISP client for DELETE: %w", err)
	}

	// Construct endpoint URL
	endpoint := fmt.Sprintf(policyDeleteURL, policyID)

	// Execute DELETE with empty map workaround (NOT nil!)
	// This prevents the SDK panic by ensuring bodyBytes is initialized
	response, err := client.Delete(ctx, endpoint, map[string]string{})
	if err != nil {
		tflog.Error(ctx, "DELETE workaround request failed", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return fmt.Errorf("failed to delete policy %s: %w", policyID, err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "DELETE workaround response received", map[string]interface{}{
		"policy_id":   policyID,
		"status_code": response.StatusCode,
	})

	// Handle HTTP status codes
	// UAP DELETE returns 200 OK with empty body (differs from database/secret DELETE which return 204)
	if response.StatusCode == http.StatusNotFound {
		tflog.Debug(ctx, "Policy already deleted (404)", map[string]interface{}{
			"policy_id": policyID,
		})
		// Policy already deleted - treat as success
		return nil
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		tflog.Error(ctx, "DELETE workaround failed with unexpected status", map[string]interface{}{
			"policy_id":   policyID,
			"status_code": response.StatusCode,
		})
		return fmt.Errorf("failed to delete policy %s - [%d] - [%s]",
			policyID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	tflog.Debug(ctx, "DELETE workaround successful", map[string]interface{}{
		"policy_id": policyID,
	})

	return nil
}

// ChangeVMSecretDirect updates a VM secret using direct HTTP PUT
// Workaround for ARK SDK v1.5.0 bug where ChangeSecret() uses POST instead of PUT
//
// The SDK's ChangeSecret() method incorrectly uses client.Post() on line 153,
// causing 403 "Invalid key=value pair" errors. This function makes the correct
// PUT request directly, reusing the SDK's existing request structure.
//
// API Endpoint: PUT /api/secrets/{id}
// Success Response: HTTP 200 OK with empty body {}
// Error Responses:
//   - 403 Forbidden: Wrong HTTP method or auth issue
//   - 404 Not Found: Secret doesn't exist
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - changeSecretReq: SDK request struct with conditional field logic already applied
//
// Returns:
//   - *vmsecretsmodels.ArkSIAVMSecret: Updated secret (minimal response from API)
//   - error: nil on success, error on failure
//
// TODO: Remove this workaround when ARK SDK v1.6.0+ is released with the fix
func ChangeVMSecretDirect(
	ctx context.Context,
	authCtx *ISPAuthContext,
	changeSecretReq *vmsecretsmodels.ArkSIAVMChangeSecret,
) (*vmsecretsmodels.ArkSIAVMSecret, error) {
	tflog.Debug(ctx, "Executing ChangeSecret workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "vm_secret",
		"secret_id":     changeSecretReq.SecretID,
		"workaround":    "PUT_instead_of_POST",
	})

	// Create ISP client for DPA service with token refresh callback
	// This ensures long-running sessions can auto-refresh expired tokens (15min expiry)
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"dpa", // Service name (constructs https://{subdomain}.dpa.{domain})
		".",   // Separator
		"",    // Base path
		func(ac *common.ArkClient) error {
			return isp.RefreshClient(ac, authCtx.ISPAuth)
		},
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for ChangeSecret workaround", map[string]interface{}{
			"secret_id": changeSecretReq.SecretID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to create ISP client: %w", err)
	}

	// Build request body matching SIA UI's successful request format
	// CRITICAL: Must match UI request structure for name updates to work
	// Based on actual UI network traffic analysis

	// Extract secret_type from SecretDetails (passed by provider)
	secretType := "ProvisionerUser" // Safe default
	if changeSecretReq.SecretDetails != nil {
		if st, ok := changeSecretReq.SecretDetails["secret_type"].(string); ok && st != "" {
			secretType = st
		}
	}

	changeSecretJSON := map[string]interface{}{
		"is_active":   !changeSecretReq.IsDisabled,
		"secret_name": changeSecretReq.SecretName,
		"secret_type": secretType,
	}

	// Preserve existing secret_details from the current secret
	// CRITICAL: account_domain and other fields must match existing secret values
	// UI evidence shows these fields are required and must not be hardcoded
	secretDetails := map[string]interface{}{}
	if changeSecretReq.SecretDetails != nil {
		// Copy all fields from preserved secret_details
		for k, v := range changeSecretReq.SecretDetails {
			if k != "secret_type" { // Skip secret_type (already in top-level)
				secretDetails[k] = v
			}
		}
	}

	// Ensure required fields exist with defaults only if missing
	if _, exists := secretDetails["certFileName"]; !exists {
		secretDetails["certFileName"] = ""
	}
	if _, exists := secretDetails["domain"]; !exists {
		secretDetails["domain"] = ""
	}
	if _, exists := secretDetails["domains"]; !exists {
		secretDetails["domains"] = []string{}
	}
	if _, exists := secretDetails["account_domain"]; !exists {
		// Default only if not present (should always be present from current secret)
		tflog.Warn(ctx, "account_domain missing from secret_details, using default 'local'")
		secretDetails["account_domain"] = "local"
	}
	if _, exists := secretDetails["ephemeral_domain_user_data"]; !exists {
		secretDetails["ephemeral_domain_user_data"] = map[string]interface{}{}
	}

	changeSecretJSON["secret_details"] = secretDetails

	// Only include 'secret' field when credentials are being updated
	// UI does NOT send this field when only updating metadata (name, is_active, etc.)
	if changeSecretReq.ProvisionerUsername != "" && changeSecretReq.ProvisionerPassword != "" {
		changeSecretJSON["secret"] = map[string]interface{}{
			"secret_data": map[string]interface{}{
				"username": changeSecretReq.ProvisionerUsername,
				"password": changeSecretReq.ProvisionerPassword,
			},
			"tenant_encrypted": false,
		}
	} else if changeSecretReq.PCloudAccountSafe != "" && changeSecretReq.PCloudAccountName != "" {
		changeSecretJSON["secret"] = map[string]interface{}{
			"secret_data": map[string]interface{}{
				"safe":         changeSecretReq.PCloudAccountSafe,
				"account_name": changeSecretReq.PCloudAccountName,
			},
			"tenant_encrypted": false,
		}
	}

	// Use same endpoint as SIA UI (NOT public/v1 - that was incorrect)
	// Confirmed by actual UI network traffic
	endpoint := fmt.Sprintf("/api/secrets/%s", changeSecretReq.SecretID)

	// Make PUT request (FIX: SDK uses Post() here, we use Put())
	response, err := client.Put(ctx, endpoint, changeSecretJSON)
	if err != nil {
		tflog.Error(ctx, "PUT workaround request failed", map[string]interface{}{
			"secret_id": changeSecretReq.SecretID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("PUT request failed: %w", err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "PUT workaround response received", map[string]interface{}{
		"secret_id":   changeSecretReq.SecretID,
		"status_code": response.StatusCode,
	})

	// Check response status (SDK expects 200 OK)
	if response.StatusCode != http.StatusOK {
		tflog.Error(ctx, "PUT workaround failed with unexpected status", map[string]interface{}{
			"secret_id":   changeSecretReq.SecretID,
			"status_code": response.StatusCode,
		})
		return nil, fmt.Errorf("failed to change secret - [%d] - [%s]",
			response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	// API returns empty body {} per Swagger spec
	// SDK expects non-nil ArkSIAVMSecret return value
	// Return minimal valid secret (Terraform will refresh via Read() if needed)
	tflog.Debug(ctx, "ChangeSecret workaround successful", map[string]interface{}{
		"secret_id": changeSecretReq.SecretID,
	})

	return &vmsecretsmodels.ArkSIAVMSecret{
		SecretID:   changeSecretReq.SecretID,
		SecretName: changeSecretReq.SecretName,
	}, nil
}

// DeleteTargetSetDirect bypasses SDK's buggy DeleteTargetSet() method
// and makes HTTP DELETE request directly with empty body workaround.
//
// This function replicates the SDK's delete logic but passes map[string]string{}
// instead of nil to avoid the panic.
//
// API Endpoint: DELETE /api/targetsets/{name}
// Success Response: HTTP 204 No Content
// Error Responses:
//   - 404 Not Found: Target set already deleted (treated as success)
//   - 403 Forbidden: Name contains forward slashes (URL path interpretation issue)
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - name: Target set name (string identifier)
//
// Returns:
//   - error: nil on success (including 404), error on failure
func DeleteTargetSetDirect(ctx context.Context, authCtx *ISPAuthContext, name string) error {
	tflog.Debug(ctx, "Executing DELETE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "target_set",
		"name":          name,
		"workaround":    "empty_map_body",
	})

	// Create temporary ISP service client with token refresh callback
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"dpa", // Service name (constructs https://{subdomain}.dpa.{domain})
		".",   // Separator
		"",    // Base path
		func(ac *common.ArkClient) error {
			return isp.RefreshClient(ac, authCtx.ISPAuth)
		},
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for DELETE workaround", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to create ISP client for DELETE: %w", err)
	}

	// Construct endpoint URL (SDK already handles URL encoding)
	endpoint := fmt.Sprintf(targetSetDeleteURL, name)

	// Execute DELETE with empty map workaround (NOT nil!)
	// This prevents the SDK panic by ensuring bodyBytes is initialized
	response, err := client.Delete(ctx, endpoint, map[string]string{})
	if err != nil {
		tflog.Error(ctx, "DELETE workaround request failed", map[string]interface{}{
			"name":  name,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to delete target set %s: %w", name, err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "DELETE workaround response received", map[string]interface{}{
		"name":        name,
		"status_code": response.StatusCode,
	})

	// Handle HTTP status codes (same as SDK's DeleteTargetSet logic)
	if response.StatusCode == http.StatusNotFound {
		tflog.Debug(ctx, "Target set already deleted (404)", map[string]interface{}{
			"name": name,
		})
		// Resource already deleted - treat as success
		return nil
	}

	if response.StatusCode != http.StatusNoContent {
		tflog.Error(ctx, "DELETE workaround failed with unexpected status", map[string]interface{}{
			"name":        name,
			"status_code": response.StatusCode,
		})
		return fmt.Errorf("failed to delete target set %s - [%d] - [%s]",
			name, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	tflog.Debug(ctx, "DELETE workaround successful", map[string]interface{}{
		"name": name,
	})

	return nil
}

// UpdateTargetSetDirect bypasses SDK's ArkSIAUpdateTargetSet struct which has omitempty tags
// that prevent sending empty strings and false booleans (description="" and
// enable_certificate_validation=false). Uses map[string]interface{} to explicitly
// include zero values in the request body.
//
// API Endpoint: PUT /api/targetsets/{name}
// Success Response: HTTP 200 OK with updated target set
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - oldName: Current target set name (used in URL path)
//   - req: Update request with all fields (including zero values)
//
// Returns:
//   - map[string]interface{}: Updated target set
//   - error: nil on success, error on failure
func UpdateTargetSetDirect(ctx context.Context, authCtx *ISPAuthContext, oldName string, req map[string]interface{}) (map[string]interface{}, error) {
	tflog.Debug(ctx, "Executing UPDATE workaround (SDK omitempty bypass)", map[string]interface{}{
		"resource_type": "target_set",
		"old_name":      oldName,
		"workaround":    "PUT_method_and_explicit_zero_values",
	})

	// Create temporary ISP service client with token refresh callback
	// This ensures long-running sessions can auto-refresh expired tokens (15min expiry)
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"dpa",
		".",
		"",
		func(ac *common.ArkClient) error {
			return isp.RefreshClient(ac, authCtx.ISPAuth)
		},
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for UPDATE workaround", map[string]interface{}{
			"old_name": oldName,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("failed to create ISP client for UPDATE: %w", err)
	}

	// Construct endpoint URL (SDK already handles URL encoding)
	endpoint := fmt.Sprintf(targetSetUpdateURL, oldName)

	// Execute PUT with map[string]interface{} to preserve zero values
	response, err := client.Put(ctx, endpoint, req)
	if err != nil {
		tflog.Error(ctx, "UPDATE workaround request failed", map[string]interface{}{
			"old_name": oldName,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("failed to update target set %s: %w", oldName, err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "UPDATE workaround response received", map[string]interface{}{
		"old_name":    oldName,
		"status_code": response.StatusCode,
	})

	// Handle HTTP status codes
	if response.StatusCode != http.StatusOK {
		tflog.Error(ctx, "UPDATE workaround failed with unexpected status", map[string]interface{}{
			"old_name":    oldName,
			"status_code": response.StatusCode,
		})
		return nil, fmt.Errorf("failed to update target set %s - [%d] - [%s]",
			oldName, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	// Parse response body - API wraps response in {"target_set": {...}}
	var wrapper struct {
		TargetSet map[string]interface{} `json:"target_set"`
	}
	if err := json.NewDecoder(response.Body).Decode(&wrapper); err != nil {
		tflog.Error(ctx, "Failed to decode UPDATE response", map[string]interface{}{
			"old_name": oldName,
			"error":    err.Error(),
		})
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tflog.Debug(ctx, "UPDATE workaround successful", map[string]interface{}{
		"old_name": oldName,
		"new_name": wrapper.TargetSet["name"],
	})

	return wrapper.TargetSet, nil
}

// CreateAzureVMPolicyDirect creates an Azure VM policy by bypassing SDK's Serialize()
// and manually constructing JSON with "Azure" (mixed case) targets key.
//
// Bug: ARK SDK v1.5.0 uses uppercase "AZURE" constant (WorkspaceTypeAzure) as JSON key,
// but SIA API expects mixed case "Azure". This causes HTTP 500 errors on policy creation.
//
// The workaround:
// 1. Builds policy using SDK models (validation, defaults)
// 2. Marshals to JSON
// 3. Unmarshals to map[string]interface{}
// 4. Fixes targets key: targets.AZURE → targets.Azure
// 5. Makes direct POST request with corrected JSON
//
// API Endpoint: POST /api/policies
// Success Response: HTTP 200/201 with created policy
// Error Responses:
//   - 500 Internal Server Error: Wrong targets key casing (AZURE vs Azure)
//   - 400 Bad Request: Invalid policy structure
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - policy: SDK policy model (built by provider with correct structure)
//
// Returns:
//   - *uapsiavmmodels.ArkUAPSIAVMAccessPolicy: Created policy with ID
//   - error: nil on success, error on failure
//
// GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/32
// TODO: Remove this workaround when ARK SDK v1.6.0+ fixes WorkspaceTypeAzure case sensitivity
func CreateAzureVMPolicyDirect(
	ctx context.Context,
	authCtx *ISPAuthContext,
	policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy,
) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error) {
	tflog.Debug(ctx, "Executing Azure VM policy CREATE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "vm_policy",
		"policy_name":   policy.Metadata.Name,
		"workaround":    "Azure_targets_key_casing_fix",
	})

	// Create ISP client for UAP service with token refresh callback
	// VM policies use "uap" service: https://{subdomain}.uap.{domain}
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"uap", // Service name for UAP policies
		".",   // Separator
		"",    // Base path
		func(ac *common.ArkClient) error {
			return isp.RefreshClient(ac, authCtx.ISPAuth)
		},
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for Azure VM policy CREATE workaround", map[string]interface{}{
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to create ISP client: %w", err)
	}

	// Build policy JSON - can't use policy.Serialize() because it rejects "Azure" location type
	// Instead: Marshal → ConvertToCamelCase → Fix Azure key

	// Step 1: Marshal to get basic JSON structure
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		tflog.Error(ctx, "Failed to marshal policy", map[string]interface{}{
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}

	var policyMapSnakeCase map[string]interface{}
	if err := json.Unmarshal(policyJSON, &policyMapSnakeCase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	// Step 2: Apply ConvertToCamelCase like the SDK does
	policyType := uapsiavmmodels.ArkUAPSIAVMAccessPolicy{}
	reflectType := reflect.TypeOf(policyType)
	camelCaseMap := common.ConvertToCamelCase(policyMapSnakeCase, &reflectType)

	policyMap, ok := camelCaseMap.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert camelCase result to map")
	}

	// Step 3: Fix Azure key and ensure Azure targets have camelCase fields
	// - ConvertToCamelCase converts azure_resource → azureResource, but we need "Azure"
	// - Also need to ensure Azure targets have camelCase fields (resourceGroups, vnetIds)
	if targets, ok := policyMap["targets"].(map[string]interface{}); ok {
		// Get the properly serialized Azure targets with camelCase fields
		if policy.Targets.AzureResource != nil {
			azureTargetsSerialized := policy.Targets.AzureResource.Serialize()
			delete(targets, "azureResource")          // Remove the camelCased version
			targets["Azure"] = azureTargetsSerialized // Add with correct capitalization

			tflog.Debug(ctx, "Fixed Azure targets: replaced azureResource with Azure (camelCase fields)", map[string]interface{}{
				"policy_name": policy.Metadata.Name,
			})
		}
	}

	// Step 4: Fix behavior structure - API expects connectAs wrapper
	// SDK: behavior.sshProfile → API: behavior.connectAs.ssh
	tflog.Debug(ctx, "Before behavior fix", map[string]interface{}{
		"behavior": fmt.Sprintf("%+v", policyMap["behavior"]),
	})

	if behavior, ok := policyMap["behavior"].(map[string]interface{}); ok {
		tflog.Debug(ctx, "Behavior is a map, checking for profiles", map[string]interface{}{
			"has_sshProfile": behavior["sshProfile"] != nil,
			"has_rdpProfile": behavior["rdpProfile"] != nil,
		})

		connectAs := make(map[string]interface{})

		if sshProfile, exists := behavior["sshProfile"]; exists {
			connectAs["ssh"] = sshProfile
			delete(behavior, "sshProfile")
			tflog.Debug(ctx, "Moved sshProfile to connectAs.ssh", nil)
		}

		if rdpProfile, exists := behavior["rdpProfile"]; exists {
			connectAs["rdp"] = rdpProfile
			delete(behavior, "rdpProfile")
			tflog.Debug(ctx, "Moved rdpProfile to connectAs.rdp", nil)
		}

		if len(connectAs) > 0 {
			behavior["connectAs"] = connectAs
			tflog.Debug(ctx, "Fixed behavior structure: wrapped profiles in connectAs", map[string]interface{}{
				"connectAs": fmt.Sprintf("%+v", connectAs),
			})
		}
	} else {
		tflog.Error(ctx, "Behavior is not a map!", map[string]interface{}{
			"type": fmt.Sprintf("%T", policyMap["behavior"]),
		})
	}

	// Step 5: Clean up metadata - remove server-managed empty objects and fix policyTags
	// Note: Keys are now camelCase after ConvertToCamelCase()
	if metadata, ok := policyMap["metadata"].(map[string]interface{}); ok {
		// Remove empty objects that the SDK emits but the API doesn't expect
		for _, key := range []string{"createdBy", "updatedOn", "timeFrame"} {
			if val, exists := metadata[key]; exists {
				// Check if it's an empty map
				if mapVal, isMap := val.(map[string]interface{}); isMap && len(mapVal) == 0 {
					delete(metadata, key)
					tflog.Debug(ctx, "Removed empty metadata field", map[string]interface{}{
						"field": key,
					})
				}
			}
		}

		// Remove duplicate policy_tags (snake_case version) and fix policyTags
		delete(metadata, "policy_tags")
		if metadata["policyTags"] == nil {
			metadata["policyTags"] = []string{}
			tflog.Debug(ctx, "Fixed policyTags from null to empty array", nil)
		}
	}

	// Step 6: Make POST request with corrected JSON
	// Log the full JSON being sent for debugging
	if jsonBytes, err := json.MarshalIndent(policyMap, "", "  "); err == nil {
		tflog.Debug(ctx, "Sending Azure VM policy JSON to API", map[string]interface{}{
			"policy_name": policy.Metadata.Name,
			"json":        string(jsonBytes),
		})
	}

	response, err := client.Post(ctx, vmPolicyCreateURL, policyMap)
	if err != nil {
		tflog.Error(ctx, "Azure VM policy CREATE workaround request failed", map[string]interface{}{
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("POST request failed: %w", err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "Azure VM policy CREATE workaround response received", map[string]interface{}{
		"policy_name": policy.Metadata.Name,
		"status_code": response.StatusCode,
	})

	// Check response status (API returns 200 or 201 for successful creation)
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		tflog.Error(ctx, "Azure VM policy CREATE workaround failed with unexpected status", map[string]interface{}{
			"policy_name": policy.Metadata.Name,
			"status_code": response.StatusCode,
		})
		return nil, fmt.Errorf("failed to create Azure VM policy - [%d] - [%s]",
			response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	// Parse CREATE response - API only returns {"policyId": "xxx"}, not the full policy
	// We need to make a GET request to fetch the full policy (like SDK's AddPolicy does)
	var createResponse struct {
		PolicyID string `json:"policyId"`
	}
	if err := json.NewDecoder(response.Body).Decode(&createResponse); err != nil {
		tflog.Error(ctx, "Failed to decode CREATE response", map[string]interface{}{
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if createResponse.PolicyID == "" {
		return nil, fmt.Errorf("CREATE response missing policyId")
	}

	tflog.Debug(ctx, "Azure VM policy created, fetching full policy", map[string]interface{}{
		"policy_name": policy.Metadata.Name,
		"policy_id":   createResponse.PolicyID,
	})

	// Step 7: GET the full policy to return complete data
	getEndpoint := fmt.Sprintf(vmPolicyUpdateURL, createResponse.PolicyID) // Same endpoint for GET and PUT
	getResponse, err := client.Get(ctx, getEndpoint, nil)
	if err != nil {
		tflog.Error(ctx, "Failed to fetch created Azure VM policy", map[string]interface{}{
			"policy_id":   createResponse.PolicyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to fetch created policy: %w", err)
	}
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch created Azure VM policy %s - [%d] - [%s]",
			createResponse.PolicyID, getResponse.StatusCode, common.SerializeResponseToJSON(getResponse.Body))
	}

	// Parse GET response - convert camelCase to snake_case for SDK mapstructure
	var responseMap map[string]interface{}
	if err := json.NewDecoder(getResponse.Body).Decode(&responseMap); err != nil {
		tflog.Error(ctx, "Failed to decode GET response", map[string]interface{}{
			"policy_id":   createResponse.PolicyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to decode GET response: %w", err)
	}

	tflog.Debug(ctx, "GET response map (camelCase)", map[string]interface{}{
		"response": fmt.Sprintf("%+v", responseMap),
	})

	// Fix Azure-related fields before conversion - SDK expects "AZURE" not "Azure"
	// 1. Fix targets key
	if targets, ok := responseMap["targets"].(map[string]interface{}); ok {
		if azureTargets, exists := targets["Azure"]; exists {
			delete(targets, "Azure")
			targets["AZURE"] = azureTargets
			tflog.Debug(ctx, "Fixed GET response: Azure → AZURE targets key", nil)
		}
	}
	// 2. Fix locationType in metadata.policyEntitlement
	if metadata, ok := responseMap["metadata"].(map[string]interface{}); ok {
		if entitlement, ok := metadata["policyEntitlement"].(map[string]interface{}); ok {
			if locationType, ok := entitlement["locationType"].(string); ok && locationType == "Azure" {
				entitlement["locationType"] = "AZURE"
				tflog.Debug(ctx, "Fixed GET response: Azure → AZURE locationType", nil)
			}
		}
	}

	// Convert from camelCase (API response) to snake_case (SDK mapstructure tags)
	respType := reflect.TypeOf(uapsiavmmodels.ArkUAPSIAVMAccessPolicy{})
	responseMapSnake, ok := common.ConvertToSnakeCase(responseMap, &respType).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert GET response to snake_case")
	}

	var createdPolicy uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	if err := createdPolicy.Deserialize(responseMapSnake); err != nil {
		tflog.Error(ctx, "Failed to deserialize GET response", map[string]interface{}{
			"policy_id":   createResponse.PolicyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to deserialize GET response: %w", err)
	}

	tflog.Debug(ctx, "Azure VM policy CREATE workaround successful", map[string]interface{}{
		"policy_name": policy.Metadata.Name,
		"policy_id":   createdPolicy.Metadata.PolicyID,
	})

	return &createdPolicy, nil
}

// ReadAzureVMPolicyDirect reads an Azure VM policy by fixing the API response
// before deserializing (SDK's Deserialize() expects "AZURE" but API returns "Azure").
//
// Bug: ARK SDK v1.5.0 expects "AZURE" (uppercase) in both targets key and locationType,
// but SIA API returns "Azure" (mixed case). This causes "unsupported workspace type" errors.
//
// The workaround:
// 1. Makes GET request to fetch policy
// 2. Fixes targets key: Azure → AZURE
// 3. Fixes locationType: Azure → AZURE
// 4. Converts to snake_case for SDK mapstructure
// 5. Deserializes into SDK struct
//
// GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/32
func ReadAzureVMPolicyDirect(
	ctx context.Context,
	authCtx *ISPAuthContext,
	policyID string,
) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error) {
	tflog.Debug(ctx, "Executing Azure VM policy READ workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "vm_policy",
		"policy_id":     policyID,
		"workaround":    "Azure_locationType_fix",
	})

	// Create ISP client for UAP service with token refresh callback
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"uap", // Service name for UAP policies
		".",   // Separator
		"",    // Base path
		func(ac *common.ArkClient) error {
			return isp.RefreshClient(ac, authCtx.ISPAuth)
		},
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for Azure VM policy READ workaround", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to create ISP client: %w", err)
	}

	// GET the policy
	getEndpoint := fmt.Sprintf(vmPolicyUpdateURL, policyID)
	getResponse, err := client.Get(ctx, getEndpoint, nil)
	if err != nil {
		tflog.Error(ctx, "Azure VM policy READ workaround request failed", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("GET request failed: %w", err)
	}
	defer getResponse.Body.Close()

	if getResponse.StatusCode == http.StatusNotFound {
		// Policy doesn't exist - return nil without error for drift detection
		return nil, nil
	}

	if getResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to read Azure VM policy %s - [%d] - [%s]",
			policyID, getResponse.StatusCode, common.SerializeResponseToJSON(getResponse.Body))
	}

	// Parse GET response
	var responseMap map[string]interface{}
	if err := json.NewDecoder(getResponse.Body).Decode(&responseMap); err != nil {
		tflog.Error(ctx, "Failed to decode READ response", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Fix Azure-related fields before conversion - SDK expects "AZURE" not "Azure"
	// 1. Fix targets key
	if targets, ok := responseMap["targets"].(map[string]interface{}); ok {
		if azureTargets, exists := targets["Azure"]; exists {
			delete(targets, "Azure")
			targets["AZURE"] = azureTargets
			tflog.Debug(ctx, "Fixed READ response: Azure → AZURE targets key", nil)
		}
	}
	// 2. Fix locationType in metadata.policyEntitlement
	if metadata, ok := responseMap["metadata"].(map[string]interface{}); ok {
		if entitlement, ok := metadata["policyEntitlement"].(map[string]interface{}); ok {
			if locationType, ok := entitlement["locationType"].(string); ok && locationType == "Azure" {
				entitlement["locationType"] = "AZURE"
				tflog.Debug(ctx, "Fixed READ response: Azure → AZURE locationType", nil)
			}
		}
	}

	// Convert from camelCase (API response) to snake_case (SDK mapstructure tags)
	respType := reflect.TypeOf(uapsiavmmodels.ArkUAPSIAVMAccessPolicy{})
	responseMapSnake, ok := common.ConvertToSnakeCase(responseMap, &respType).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert READ response to snake_case")
	}

	var policy uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	if err := policy.Deserialize(responseMapSnake); err != nil {
		tflog.Error(ctx, "Failed to deserialize READ response", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return nil, fmt.Errorf("failed to deserialize response: %w", err)
	}

	tflog.Debug(ctx, "Azure VM policy READ workaround successful", map[string]interface{}{
		"policy_id":   policyID,
		"policy_name": policy.Metadata.Name,
	})

	return &policy, nil
}

// UpdateAzureVMPolicyDirect updates an Azure VM policy by bypassing SDK's Serialize()
// and manually constructing JSON with "Azure" (mixed case) targets key.
//
// Bug: ARK SDK v1.5.0 uses uppercase "AZURE" constant (WorkspaceTypeAzure) as JSON key,
// but SIA API expects mixed case "Azure". This causes HTTP 500 errors on policy updates.
//
// The workaround:
// 1. Builds policy using SDK models (validation, defaults)
// 2. Marshals to JSON
// 3. Unmarshals to map[string]interface{}
// 4. Fixes targets key: targets.AZURE → targets.Azure
// 5. Makes direct PUT request with corrected JSON
//
// API Endpoint: PUT /api/policies/{id}
// Success Response: HTTP 200 with updated policy
// Error Responses:
//   - 500 Internal Server Error: Wrong targets key casing (AZURE vs Azure)
//   - 404 Not Found: Policy doesn't exist
//   - 400 Bad Request: Invalid policy structure
//
// Parameters:
//   - ctx: Context for request cancellation
//   - authCtx: ISPAuthContext for authentication
//   - policyID: Policy ID (UUID string)
//   - policy: SDK policy model (built by provider with correct structure)
//
// Returns:
//   - *uapsiavmmodels.ArkUAPSIAVMAccessPolicy: Updated policy
//   - error: nil on success, error on failure
//
// GitHub Issue: https://github.com/cyberark/ark-sdk-golang/issues/32
// TODO: Remove this workaround when ARK SDK v1.6.0+ fixes WorkspaceTypeAzure case sensitivity
func UpdateAzureVMPolicyDirect(
	ctx context.Context,
	authCtx *ISPAuthContext,
	policyID string,
	policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy,
) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error) {
	tflog.Debug(ctx, "Executing Azure VM policy UPDATE workaround (ARK SDK bug bypass)", map[string]interface{}{
		"resource_type": "vm_policy",
		"policy_id":     policyID,
		"policy_name":   policy.Metadata.Name,
		"workaround":    "Azure_targets_key_casing_fix",
	})

	// Create ISP client for UAP service with token refresh callback
	// VM policies use "uap" service: https://{subdomain}.uap.{domain}
	client, err := isp.FromISPAuth(
		authCtx.ISPAuth,
		"uap", // Service name for UAP policies
		".",   // Separator
		"",    // Base path
		func(ac *common.ArkClient) error {
			return isp.RefreshClient(ac, authCtx.ISPAuth)
		},
	)
	if err != nil {
		tflog.Error(ctx, "Failed to create ISP client for Azure VM policy UPDATE workaround", map[string]interface{}{
			"policy_id":   policyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to create ISP client: %w", err)
	}

	// Build policy JSON - can't use policy.Serialize() because it may fail with Azure policies
	// that were read from API (workspace type "AZURE" vs SDK expectation)
	// Instead: Marshal → ConvertToCamelCase → Fix Azure key (same as CreateAzureVMPolicyDirect)

	// Step 1: Marshal to get basic JSON structure
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		tflog.Error(ctx, "Failed to marshal policy", map[string]interface{}{
			"policy_id":   policyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("failed to marshal policy: %w", err)
	}

	var policyMapSnakeCase map[string]interface{}
	if err := json.Unmarshal(policyJSON, &policyMapSnakeCase); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	// Step 2: Apply ConvertToCamelCase like the SDK does
	policyType := uapsiavmmodels.ArkUAPSIAVMAccessPolicy{}
	reflectType := reflect.TypeOf(policyType)
	camelCaseMap := common.ConvertToCamelCase(policyMapSnakeCase, &reflectType)

	policyMap, ok := camelCaseMap.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert camelCase result to map")
	}

	// Step 3: Fix Azure key and ensure Azure targets have camelCase fields
	// - ConvertToCamelCase converts azure_resource → azureResource, but we need "Azure"
	// - Also need to ensure Azure targets have camelCase fields (resourceGroups, vnetIds)
	// - Also need to ensure null arrays become empty arrays (API requires this)
	if targets, ok := policyMap["targets"].(map[string]interface{}); ok {
		// Get the properly serialized Azure targets with camelCase fields
		if policy.Targets.AzureResource != nil {
			azureTargetsSerialized := policy.Targets.AzureResource.Serialize()
			delete(targets, "azureResource") // Remove the camelCased version

			// Fix null arrays → empty arrays in Azure targets (API requires non-null arrays)
			// Serialize() returns map[string]interface{} where nil slices become JSON null
			// We need to ensure all array fields are non-null for the API
			// Force all array fields to empty array if they would marshal to JSON null
			for _, key := range []string{"regions", "resourceGroups", "subscriptions", "vnetIds", "tags"} {
				val := azureTargetsSerialized[key]
				needsFix := false

				if val == nil {
					needsFix = true
				} else {
					// Use reflection to check for nil slice
					rv := reflect.ValueOf(val)
					if rv.Kind() == reflect.Slice && rv.IsNil() {
						needsFix = true
					}
				}

				if needsFix {
					azureTargetsSerialized[key] = []interface{}{}
					tflog.Debug(ctx, "Fixed Azure target null → empty array", map[string]interface{}{
						"field": key,
					})
				}
			}

			targets["Azure"] = azureTargetsSerialized // Add with correct capitalization

			tflog.Debug(ctx, "Fixed Azure targets: replaced azureResource with Azure (camelCase fields)", map[string]interface{}{
				"policy_id":   policyID,
				"policy_name": policy.Metadata.Name,
			})
		}
	}

	// Step 4: Fix behavior structure - API expects connectAs wrapper
	// SDK: behavior.sshProfile → API: behavior.connectAs.ssh
	tflog.Debug(ctx, "Before behavior fix", map[string]interface{}{
		"behavior": fmt.Sprintf("%+v", policyMap["behavior"]),
	})

	if behavior, ok := policyMap["behavior"].(map[string]interface{}); ok {
		tflog.Debug(ctx, "Behavior is a map, checking for profiles", map[string]interface{}{
			"has_sshProfile": behavior["sshProfile"] != nil,
			"has_rdpProfile": behavior["rdpProfile"] != nil,
		})

		connectAs := make(map[string]interface{})

		if sshProfile, exists := behavior["sshProfile"]; exists {
			connectAs["ssh"] = sshProfile
			delete(behavior, "sshProfile")
			tflog.Debug(ctx, "Moved sshProfile to connectAs.ssh", nil)
		}

		if rdpProfile, exists := behavior["rdpProfile"]; exists {
			connectAs["rdp"] = rdpProfile
			delete(behavior, "rdpProfile")
			tflog.Debug(ctx, "Moved rdpProfile to connectAs.rdp", nil)
		}

		if len(connectAs) > 0 {
			behavior["connectAs"] = connectAs
			tflog.Debug(ctx, "Fixed behavior structure: wrapped profiles in connectAs", map[string]interface{}{
				"connectAs": fmt.Sprintf("%+v", connectAs),
			})
		}
	} else {
		tflog.Error(ctx, "Behavior is not a map!", map[string]interface{}{
			"type": fmt.Sprintf("%T", policyMap["behavior"]),
		})
	}

	// Step 5: Clean up metadata - remove server-managed fields and fix various issues
	// Note: Keys are now camelCase after ConvertToCamelCase()
	// For UPDATE, the policy was READ from API so has full server-managed fields populated
	if metadata, ok := policyMap["metadata"].(map[string]interface{}); ok {
		// Remove server-managed fields (they exist with full data from API read, not just empty)
		// CRITICAL: policyId must NOT be in the request body for UPDATE - it's in the URL only
		for _, key := range []string{"createdBy", "updatedOn", "timeFrame", "policyId"} {
			if _, exists := metadata[key]; exists {
				delete(metadata, key)
				tflog.Debug(ctx, "Removed server-managed metadata field for UPDATE", map[string]interface{}{
					"field": key,
				})
			}
		}

		// Remove duplicate policy_tags (snake_case version) and fix policyTags
		delete(metadata, "policy_tags")
		if metadata["policyTags"] == nil {
			metadata["policyTags"] = []string{}
			tflog.Debug(ctx, "Fixed policyTags from null to empty array", nil)
		}

		// Fix locationType: AZURE → Azure (ReadAzureVMPolicyDirect converts Azure→AZURE for SDK)
		if entitlement, ok := metadata["policyEntitlement"].(map[string]interface{}); ok {
			if locationType, ok := entitlement["locationType"].(string); ok && locationType == "AZURE" {
				entitlement["locationType"] = "Azure"
				tflog.Debug(ctx, "Fixed locationType: AZURE → Azure for API", nil)
			}
		}

		// Clean up status object - remove server-managed fields that should not be in UPDATE request
		// The API returns statusCode and statusDescription, but CREATE only sends status: { status: "Active" }
		if status, ok := metadata["status"].(map[string]interface{}); ok {
			// Remove server-managed status fields (not present in CREATE, causes HTTP 500 in UPDATE)
			for _, key := range []string{"statusCode", "statusDescription", "link"} {
				if _, exists := status[key]; exists {
					delete(status, key)
					tflog.Debug(ctx, "Removed server-managed status field for UPDATE", map[string]interface{}{
						"field": key,
					})
				}
			}
		}
	}

	// Step 6: Make PUT request with corrected JSON
	// Log the full JSON being sent for debugging
	if jsonBytes, err := json.MarshalIndent(policyMap, "", "  "); err == nil {
		tflog.Debug(ctx, "Sending Azure VM policy UPDATE JSON to API", map[string]interface{}{
			"policy_id":   policyID,
			"policy_name": policy.Metadata.Name,
			"json":        string(jsonBytes),
		})
	}

	endpoint := fmt.Sprintf(vmPolicyUpdateURL, policyID)
	response, err := client.Put(ctx, endpoint, policyMap)
	if err != nil {
		tflog.Error(ctx, "Azure VM policy UPDATE workaround request failed", map[string]interface{}{
			"policy_id":   policyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("PUT request failed: %w", err)
	}
	defer response.Body.Close()

	tflog.Debug(ctx, "Azure VM policy UPDATE workaround response received", map[string]interface{}{
		"policy_id":   policyID,
		"policy_name": policy.Metadata.Name,
		"status_code": response.StatusCode,
	})

	// Check response status (API returns 200 for successful update)
	if response.StatusCode != http.StatusOK {
		tflog.Error(ctx, "Azure VM policy UPDATE workaround failed with unexpected status", map[string]interface{}{
			"policy_id":   policyID,
			"policy_name": policy.Metadata.Name,
			"status_code": response.StatusCode,
		})
		return nil, fmt.Errorf("failed to update Azure VM policy %s - [%d] - [%s]",
			policyID, response.StatusCode, common.SerializeResponseToJSON(response.Body))
	}

	// API returns 200 with empty body for PUT operations
	// After successful UPDATE, fetch the updated policy using GET to return the current state
	tflog.Debug(ctx, "Azure VM policy UPDATE successful, fetching updated policy via GET", map[string]interface{}{
		"policy_id":   policyID,
		"policy_name": policy.Metadata.Name,
	})

	// Reuse ReadAzureVMPolicyDirect to fetch the updated policy
	updatedPolicy, err := ReadAzureVMPolicyDirect(ctx, authCtx, policyID)
	if err != nil {
		tflog.Error(ctx, "Failed to fetch updated policy after UPDATE", map[string]interface{}{
			"policy_id":   policyID,
			"policy_name": policy.Metadata.Name,
			"error":       err.Error(),
		})
		return nil, fmt.Errorf("update succeeded but failed to fetch updated policy: %w", err)
	}

	tflog.Debug(ctx, "Azure VM policy UPDATE workaround successful", map[string]interface{}{
		"policy_id":   policyID,
		"policy_name": policy.Metadata.Name,
	})

	return updatedPolicy, nil
}
