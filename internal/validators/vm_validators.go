package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"golang.org/x/exp/slices"
)

// vmLocationTypeValidator validates that VM policy location type is one of: "AWS", "Azure", "GCP", "FQDN/IP"
type vmLocationTypeValidator struct{}

// Description returns a plain text description of the validator's behavior
func (v vmLocationTypeValidator) Description(ctx context.Context) string {
	return "Value must be 'AWS', 'Azure', 'GCP', or 'FQDN/IP'"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior
func (v vmLocationTypeValidator) MarkdownDescription(ctx context.Context) string {
	return "Value must be `AWS`, `Azure`, `GCP`, or `FQDN/IP`"
}

// ValidateString validates the VM location type value
func (v vmLocationTypeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null (during plan phase)
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	validTypes := []string{"AWS", "Azure", "GCP", "FQDN/IP"}

	if !slices.Contains(validTypes, value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid VM Location Type",
			fmt.Sprintf("Value %q is not valid. Must be 'AWS', 'Azure', 'GCP', or 'FQDN/IP'.", value),
		)
	}
}

// VMLocationType returns a validator that ensures VM policy location type is valid
func VMLocationType() validator.String {
	return vmLocationTypeValidator{}
}

// fqdnOperatorValidator validates that FQDN operator is one of: "EXACTLY", "WILDCARD", "PREFIX", "SUFFIX", "CONTAINS"
type fqdnOperatorValidator struct{}

// Description returns a plain text description of the validator's behavior
func (v fqdnOperatorValidator) Description(ctx context.Context) string {
	return "Value must be 'EXACTLY', 'WILDCARD', 'PREFIX', 'SUFFIX', or 'CONTAINS'"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior
func (v fqdnOperatorValidator) MarkdownDescription(ctx context.Context) string {
	return "Value must be `EXACTLY`, `WILDCARD`, `PREFIX`, `SUFFIX`, or `CONTAINS`"
}

// ValidateString validates the FQDN operator value
func (v fqdnOperatorValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null (during plan phase)
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	validOperators := []string{"EXACTLY", "WILDCARD", "PREFIX", "SUFFIX", "CONTAINS"}

	if !slices.Contains(validOperators, value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid FQDN Operator",
			fmt.Sprintf("Value %q is not valid. Must be 'EXACTLY', 'WILDCARD', 'PREFIX', 'SUFFIX', or 'CONTAINS'.", value),
		)
	}
}

// FQDNOperator returns a validator that ensures FQDN operator is valid
func FQDNOperator() validator.String {
	return fqdnOperatorValidator{}
}

// ipOperatorValidator validates that IP operator is one of: "EXACTLY", "WILDCARD"
type ipOperatorValidator struct{}

// Description returns a plain text description of the validator's behavior
func (v ipOperatorValidator) Description(ctx context.Context) string {
	return "Value must be 'EXACTLY' or 'WILDCARD'"
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior
func (v ipOperatorValidator) MarkdownDescription(ctx context.Context) string {
	return "Value must be `EXACTLY` or `WILDCARD`"
}

// ValidateString validates the IP operator value
func (v ipOperatorValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if value is unknown or null (during plan phase)
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	validOperators := []string{"EXACTLY", "WILDCARD"}

	if !slices.Contains(validOperators, value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IP Operator",
			fmt.Sprintf("Value %q is not valid. Must be 'EXACTLY' or 'WILDCARD'.", value),
		)
	}
}

// IPOperator returns a validator that ensures IP operator is valid
func IPOperator() validator.String {
	return ipOperatorValidator{}
}
