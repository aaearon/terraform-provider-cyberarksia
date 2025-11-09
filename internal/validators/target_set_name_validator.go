// Package validators provides custom Terraform validators for the CyberArk SIA provider.
package validators

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NoForwardSlashes returns a validator that warns when a target set name contains forward slashes.
// Forward slashes are accepted by the API during creation but cause DELETE failures (403 errors)
// due to URL path interpretation issues.
func NoForwardSlashes() validator.String {
	return noForwardSlashesValidator{}
}

// noForwardSlashesValidator implements the validator interface.
type noForwardSlashesValidator struct{}

// Description returns a human-readable description of the validator.
func (v noForwardSlashesValidator) Description(_ context.Context) string {
	return "Warns when name contains forward slashes which cause deletion failures"
}

// MarkdownDescription returns a markdown description of the validator.
func (v noForwardSlashesValidator) MarkdownDescription(_ context.Context) string {
	return "Warns when name contains forward slashes which cause deletion failures"
}

// ValidateString performs the validation logic.
func (v noForwardSlashesValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is null or unknown, skip validation
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// Check if the name contains forward slashes
	if strings.Contains(value, "/") {
		resp.Diagnostics.AddWarning(
			"Target Set Name Contains Forward Slashes",
			"The name contains forward slashes which will cause deletion failures (403 errors). "+
				"While the API accepts this during creation, you will not be able to destroy this resource. "+
				"Consider using hyphens (-) or underscores (_) instead.\n\n"+
				"Current name: "+value,
		)
	}
}
