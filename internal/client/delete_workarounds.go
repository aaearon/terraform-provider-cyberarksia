// Package client provides CyberArk SIA API client wrappers
package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cyberark/ark-sdk-golang/pkg/common"
	"github.com/cyberark/ark-sdk-golang/pkg/common/isp"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ARK SDK v1.5.0 BUG WORKAROUND
//
// The SDK's DeleteDatabase() and DeleteSecret() methods pass nil body to HTTP DELETE requests,
// causing a panic in doRequest() when http.NewRequestWithContext() tries to call bodyBytes.Len()
// on a nil *bytes.Buffer pointer.
//
// Root Cause: pkg/common/ark_client.go:556-576
//   - Line 556: var bodyBytes *bytes.Buffer (defaults to nil)
//   - Line 557-575: Only initializes bodyBytes if body != nil
//   - Line 576: Passes nil pointer to http.NewRequestWithContext()
//   - Result: Panic when http.Request calls bodyBytes.Len()
//
// Affected SDK Methods:
//   - pkg/services/sia/workspaces/db/ark_sia_workspaces_db_service.go:188 - DeleteDatabase()
//   - pkg/services/sia/secrets/db/ark_sia_secrets_db_service.go:343 - DeleteSecret()
//
// Workaround: Pass empty map map[string]string{} instead of nil
//   - Empty map is non-nil → gets JSON-marshaled to "{}"
//   - Creates valid bytes.Buffer → no panic
//   - SIA API ignores empty JSON body on DELETE → functionally equivalent
//
// Same workaround used successfully in certificates.go:570
//
// TODO: Remove this file when ARK SDK v1.6.0+ fixes the nil body handling

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
