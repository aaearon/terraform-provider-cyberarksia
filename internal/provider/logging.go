// Package provider implements the Idira SIA Terraform provider
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SensitiveFields are fields that should NEVER be logged
var SensitiveFields = []string{
	"password",
	"client_secret",
	"aws_secret_access_key",
	"token",
	"bearer",
	"secret",
}

// LogProviderConfig logs provider configuration (masking sensitive data)
func LogProviderConfig(ctx context.Context, config *CyberArkSIAProviderModel) {
	tflog.Debug(ctx, "Provider configuration loaded", map[string]interface{}{
		"identity_url": config.IdentityURL.ValueString(),
		// NEVER log: username, password
		// Note: Username contains tenant info - logged only when identity_url is not provided
	})
}

// LogAuthSuccess logs successful authentication
func LogAuthSuccess(ctx context.Context) {
	tflog.Info(ctx, "Successfully authenticated with Idira ISPSS")
}

// LogAuthStart logs authentication attempt
func LogAuthStart(ctx context.Context) {
	tflog.Debug(ctx, "Initializing authentication with Idira ISPSS")
}

// LogSIAClientInit logs SIA API client initialization
func LogSIAClientInit(ctx context.Context) {
	tflog.Debug(ctx, "Initializing SIA API client")
}

// LogSIAClientSuccess logs successful SIA API client creation
func LogSIAClientSuccess(ctx context.Context) {
	tflog.Info(ctx, "Successfully initialized SIA API client")
}

// LogUAPClientInit logs UAP API client initialization
func LogUAPClientInit(ctx context.Context) {
	tflog.Debug(ctx, "Initializing UAP API client")
}

// LogUAPClientSuccess logs successful UAP API client creation
func LogUAPClientSuccess(ctx context.Context) {
	tflog.Info(ctx, "Successfully initialized UAP API client")
}

// LogIdentityClientInit logs Identity API client initialization
func LogIdentityClientInit(ctx context.Context) {
	tflog.Debug(ctx, "Initializing Identity API client")
}

// LogIdentityClientSuccess logs successful Identity API client creation
func LogIdentityClientSuccess(ctx context.Context) {
	tflog.Info(ctx, "Successfully initialized Identity API client")
}

// LogOperationStart logs the start of an API operation
func LogOperationStart(ctx context.Context, operation string, resourceType string) {
	tflog.Debug(ctx, "Starting operation", map[string]interface{}{
		"operation":     operation,
		"resource_type": resourceType,
	})
}

// LogOperationSuccess logs successful completion of an API operation
func LogOperationSuccess(ctx context.Context, operation string, resourceType string, resourceID string) {
	tflog.Info(ctx, "Operation completed successfully", map[string]interface{}{
		"operation":     operation,
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}

// LogOperationError logs operation failure
func LogOperationError(ctx context.Context, operation string, resourceType string, err error) {
	tflog.Error(ctx, "Operation failed", map[string]interface{}{
		"operation":     operation,
		"resource_type": resourceType,
		"error":         err.Error(),
	})
}

// LogRetryAttempt logs retry attempt with backoff info
func LogRetryAttempt(ctx context.Context, attempt int, maxRetries int, delay string) {
	tflog.Warn(ctx, "Retrying operation after transient failure", map[string]interface{}{
		"attempt":     attempt,
		"max_retries": maxRetries,
		"delay":       delay,
	})
}

// LogDriftDetected logs when state drift is detected
func LogDriftDetected(ctx context.Context, resourceType string, resourceID string) {
	tflog.Warn(ctx, "State drift detected - resource modified outside Terraform", map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	})
}
