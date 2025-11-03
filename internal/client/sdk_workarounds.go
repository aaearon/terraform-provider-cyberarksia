// Package client provides CyberArk SIA API client wrappers
package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cyberark/ark-sdk-golang/pkg/common"
	"github.com/cyberark/ark-sdk-golang/pkg/common/isp"
	vmsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm/models"
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
// Pattern: All workarounds bypass SDK methods and make direct HTTP calls using ISP client
//
// TODO: Remove this file when ARK SDK v1.6.0+ fixes these HTTP method bugs

const (
	// Database workspace DELETE endpoint (from SDK source)
	databaseWorkspaceDeleteURL = "/api/adb/resources/%d"

	// Secret DELETE endpoint (from SDK source)
	secretDeleteURL = "/api/adb/secretsmgmt/secrets/%s" //nolint:gosec // URL path template, not a credential

	// VM Secret DELETE endpoint (from SDK source)
	vmSecretDeleteURL = "/api/secrets/%s" //nolint:gosec // URL path template, not a credential

	// Policy DELETE endpoint (from SDK source)
	policyDeleteURL = "/api/policies/%s"
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
