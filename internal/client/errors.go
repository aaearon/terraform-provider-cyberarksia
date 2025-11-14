// Package client provides CyberArk SIA API client wrappers
package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// ErrorCategory represents the classification of an error
type ErrorCategory int

const (
	ErrorCategoryAuth ErrorCategory = iota
	ErrorCategoryPermission
	ErrorCategoryNotFound
	ErrorCategoryConflict
	ErrorCategoryValidation
	ErrorCategoryNetwork
	ErrorCategoryTimeout
	ErrorCategoryRateLimit
	ErrorCategoryServer
	ErrorCategoryUnknown
)

// String returns a string representation of the error category
func (ec ErrorCategory) String() string {
	switch ec {
	case ErrorCategoryAuth:
		return "authentication"
	case ErrorCategoryPermission:
		return "permission"
	case ErrorCategoryNotFound:
		return "not_found"
	case ErrorCategoryConflict:
		return "conflict"
	case ErrorCategoryValidation:
		return "validation"
	case ErrorCategoryNetwork:
		return "network"
	case ErrorCategoryTimeout:
		return "timeout"
	case ErrorCategoryRateLimit:
		return "rate_limit"
	case ErrorCategoryServer:
		return "server"
	default:
		return "unknown"
	}
}

// classifyError determines the error category using multiple detection strategies
// Note: ARK SDK v1.5.0 does not expose structured error types or HTTP status codes,
// so we rely on error message patterns and standard Go error type detection
func classifyError(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}

	errorMsg := strings.ToLower(err.Error())

	// Note: Cannot use tflog here as we don't have context
	// Logging is done in MapError and IsRetryable which have context

	// 1. Check for standard Go error types (most reliable)

	// Context errors (timeout/cancellation)
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorCategoryTimeout
	}
	if errors.Is(err, context.Canceled) {
		return ErrorCategoryNetwork // Treat as network since operation was interrupted
	}

	// Network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrorCategoryTimeout
		}
		return ErrorCategoryNetwork
	}

	// 2. Pattern matching (ordered by specificity - most specific first)

	// Authentication (very specific patterns first)
	if strings.Contains(errorMsg, "authentication failed") ||
		strings.Contains(errorMsg, "invalid credentials") ||
		strings.Contains(errorMsg, "unauthorized") ||
		strings.Contains(errorMsg, "401") ||
		strings.Contains(errorMsg, "invalid_client") ||
		strings.Contains(errorMsg, "invalid client") ||
		strings.Contains(errorMsg, "access denied") {
		return ErrorCategoryAuth
	}

	// Permission errors (403 Forbidden)
	if strings.Contains(errorMsg, "insufficient permissions") ||
		strings.Contains(errorMsg, "forbidden") ||
		strings.Contains(errorMsg, "403") ||
		strings.Contains(errorMsg, "permission denied") ||
		strings.Contains(errorMsg, "not authorized") {
		return ErrorCategoryPermission
	}

	// Rate limiting (429 Too Many Requests)
	if strings.Contains(errorMsg, "rate limit") ||
		strings.Contains(errorMsg, "too many requests") ||
		strings.Contains(errorMsg, "429") ||
		strings.Contains(errorMsg, "throttled") ||
		strings.Contains(errorMsg, "quota exceeded") {
		return ErrorCategoryRateLimit
	}

	// Resource not found (404)
	if strings.Contains(errorMsg, "not found") ||
		strings.Contains(errorMsg, "404") ||
		strings.Contains(errorMsg, "does not exist") ||
		strings.Contains(errorMsg, "no such") {
		return ErrorCategoryNotFound
	}

	// Conflict/duplicate (409 Conflict)
	// Certificate-specific patterns (check before generic patterns)
	if strings.Contains(errorMsg, "certificate_in_use") ||
		strings.Contains(errorMsg, "certificate in use") ||
		strings.Contains(errorMsg, "certificate is currently in use") ||
		strings.Contains(errorMsg, "duplicate_name") ||
		strings.Contains(errorMsg, "duplicate name") ||
		strings.Contains(errorMsg, "already exists") ||
		strings.Contains(errorMsg, "duplicate") ||
		strings.Contains(errorMsg, "409") ||
		strings.Contains(errorMsg, "conflict") {
		return ErrorCategoryConflict
	}

	// Validation errors (400 Bad Request, 422 Unprocessable Entity)
	if strings.Contains(errorMsg, "validation") ||
		strings.Contains(errorMsg, "invalid") ||
		strings.Contains(errorMsg, "400") ||
		strings.Contains(errorMsg, "422") ||
		strings.Contains(errorMsg, "bad request") ||
		strings.Contains(errorMsg, "malformed") {
		return ErrorCategoryValidation
	}

	// Server errors (5xx)
	if strings.Contains(errorMsg, "server error") ||
		strings.Contains(errorMsg, "service unavailable") ||
		strings.Contains(errorMsg, "internal error") ||
		strings.Contains(errorMsg, "500") ||
		strings.Contains(errorMsg, "502") ||
		strings.Contains(errorMsg, "503") ||
		strings.Contains(errorMsg, "504") {
		return ErrorCategoryServer
	}

	// Network/connectivity (check after other patterns)
	if strings.Contains(errorMsg, "connection refused") ||
		strings.Contains(errorMsg, "timeout") ||
		strings.Contains(errorMsg, "timed out") ||
		strings.Contains(errorMsg, "network") ||
		strings.Contains(errorMsg, "dial") ||
		strings.Contains(errorMsg, "no such host") ||
		strings.Contains(errorMsg, "connection reset") {
		return ErrorCategoryNetwork
	}

	// 3. Fallback for unknown errors
	return ErrorCategoryUnknown
}

// IsNotFoundError returns true if the error represents a 404 Not Found response
// Used for drift detection in Read() methods to determine if resource was deleted
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return classifyError(err) == ErrorCategoryNotFound
}

// IsForbiddenError checks if error is a 403 Forbidden/permission error
func IsForbiddenError(err error) bool {
	if err == nil {
		return false
	}
	return classifyError(err) == ErrorCategoryPermission
}

// MapError converts ARK SDK errors to Terraform diagnostics with actionable guidance
// Returns nil if err is nil (caller should check before appending)
//
// Note: ARK SDK v1.5.0 does not provide structured error types with HTTP status codes.
// Error classification relies on standard Go error types and string pattern matching.
// For robustness, all patterns are ordered by specificity with comprehensive fallback.
func MapError(err error, operation string) diag.Diagnostic {
	if err == nil {
		// Return an empty error diagnostic (won't be appended by caller check)
		return diag.NewErrorDiagnostic("", "")
	}

	errorMsg := err.Error()
	category := classifyError(err)

	// Log error classification for debugging (using background context since MapError doesn't receive one)
	// In a real scenario, the caller should log with their context before calling MapError

	switch category {
	case ErrorCategoryAuth:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Authentication Failed - %s", operation),
			fmt.Sprintf("Invalid username or password.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Verify credentials in provider configuration\n"+
				"2. Check username format: service-account@cyberark.cloud.tenant-id\n"+
				"3. Ensure service account has SIA role memberships", errorMsg),
		)

	case ErrorCategoryPermission:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Insufficient Permissions - %s", operation),
			fmt.Sprintf("ISPSS service account lacks required permissions.\n\n"+
				"Error: %s\n\n"+
				"Recommended action:\n"+
				"Verify service account has SIA Database Administrator role or equivalent", errorMsg),
		)

	case ErrorCategoryNotFound:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Resource Not Found - %s", operation),
			fmt.Sprintf("The requested resource was not found in SIA.\n\n"+
				"Error: %s\n\n"+
				"This may occur if:\n"+
				"- Resource was deleted outside Terraform\n"+
				"- Resource ID is incorrect\n\n"+
				"Run 'terraform refresh' to sync state", errorMsg),
		)

	case ErrorCategoryConflict:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Resource Conflict - %s", operation),
			fmt.Sprintf("A resource with this identifier already exists.\n\n"+
				"Error: %s\n\n"+
				"Use 'terraform import' to manage the existing resource", errorMsg),
		)

	case ErrorCategoryValidation:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Validation Failed - %s", operation),
			fmt.Sprintf("SIA API validation failed.\n\n"+
				"Error: %s\n\n"+
				"Check configuration values match SIA requirements", errorMsg),
		)

	case ErrorCategoryNetwork:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Network Error - %s", operation),
			fmt.Sprintf("Unable to connect to SIA API.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Check network connectivity\n"+
				"2. Verify identity_url is correct\n"+
				"3. Check firewall rules", errorMsg),
		)

	case ErrorCategoryTimeout:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Request Timeout - %s", operation),
			fmt.Sprintf("Request to SIA API exceeded timeout limit.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Check network latency to SIA API\n"+
				"2. Increase request_timeout in provider configuration\n"+
				"3. Verify SIA API is responsive", errorMsg),
		)

	case ErrorCategoryRateLimit:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Rate Limit Exceeded - %s", operation),
			fmt.Sprintf("Too many requests to SIA API.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Reduce parallelism in Terraform configuration\n"+
				"2. Wait before retrying\n"+
				"3. Contact CyberArk support if rate limits are too restrictive", errorMsg),
		)

	case ErrorCategoryServer:
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("SIA API Server Error - %s", operation),
			fmt.Sprintf("SIA API encountered an internal error.\n\n"+
				"Error: %s\n\n"+
				"This is typically a transient issue. Terraform will retry automatically.\n"+
				"If the problem persists, contact CyberArk support.", errorMsg),
		)

	case ErrorCategoryUnknown:
		fallthrough
	default:
		// Comprehensive fallback for unknown error types
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("SIA API Error - %s", operation),
			fmt.Sprintf("An error occurred communicating with SIA API.\n\n"+
				"Error: %s\n\n"+
				"If this error persists, please report it with the full error message above.", errorMsg),
		)
	}
}

// MapCertificateError converts certificate-specific errors to Terraform diagnostics.
// Extends MapError with certificate-specific error handling patterns.
//
// Certificate-Specific Errors:
//   - CERTIFICATE_IN_USE: Certificate cannot be deleted (database workspaces reference it)
//   - DUPLICATE_NAME: Certificate name already exists
//   - INVALID_CERTIFICATE: PEM/DER validation failed server-side
//
// Parameters:
//   - err: Error from certificate API call
//   - operation: Operation name for context (e.g., "Create Certificate")
//
// Returns:
//   - diag.Diagnostic: Actionable error message for Terraform
func MapCertificateError(err error, operation string) diag.Diagnostic {
	if err == nil {
		return diag.NewErrorDiagnostic("", "")
	}

	errorMsg := err.Error()
	errorLower := strings.ToLower(errorMsg)

	// Certificate-specific error patterns (check before generic MapError)

	// CERTIFICATE_IN_USE: Cannot delete certificate referenced by database workspaces
	if strings.Contains(errorLower, "certificate_in_use") ||
		strings.Contains(errorLower, "certificate in use") ||
		strings.Contains(errorLower, "certificate is currently in use") {
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Certificate In Use - %s", operation),
			fmt.Sprintf("Cannot delete certificate: currently in use by database workspaces.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Identify database workspaces referencing this certificate\n"+
				"2. Update those workspaces to remove certificate_id reference\n"+
				"3. Then retry certificate deletion", errorMsg),
		)
	}

	// DUPLICATE_NAME: Certificate name already exists
	if strings.Contains(errorLower, "duplicate_name") ||
		strings.Contains(errorLower, "duplicate name") ||
		(strings.Contains(errorLower, "certificate") && strings.Contains(errorLower, "already exists")) {
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Duplicate Certificate Name - %s", operation),
			fmt.Sprintf("A certificate with this name already exists.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Use 'terraform import' to manage the existing certificate\n"+
				"2. Or choose a different cert_name value", errorMsg),
		)
	}

	// INVALID_CERTIFICATE: Server-side certificate validation failed
	if strings.Contains(errorLower, "invalid certificate") ||
		strings.Contains(errorLower, "invalid cert") ||
		strings.Contains(errorLower, "malformed certificate") {
		return diag.NewErrorDiagnostic(
			fmt.Sprintf("Invalid Certificate - %s", operation),
			fmt.Sprintf("Certificate validation failed.\n\n"+
				"Error: %s\n\n"+
				"Recommended actions:\n"+
				"1. Verify cert_body contains valid PEM encoded certificate\n"+
				"2. Ensure certificate does not contain private key material\n"+
				"3. Check certificate is not expired or corrupted", errorMsg),
		)
	}

	// Fallback to generic error mapping
	return MapError(err, operation)
}
