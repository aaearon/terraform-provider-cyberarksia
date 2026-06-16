// Package provider implements the Idira SIA Terraform provider
package provider

// TestEnvVars documents the environment variables required for acceptance tests
// These variables must be set when running acceptance tests (TF_ACC=1)
const (
	// TF_ACC must be set to "1" to enable acceptance tests
	EnvTFAcc = "TF_ACC"

	// CYBERARK_USERNAME is the service account username in full format
	// Example: my-service-account@cyberark.cloud.12345
	EnvUsername = "CYBERARK_USERNAME"

	// CYBERARK_PASSWORD is the service account password
	EnvPassword = "CYBERARK_PASSWORD" //nolint:gosec // Environment variable name, not a credential

	// CYBERARK_IDENTITY_URL is the Idira Identity tenant URL (optional)
	// Example: https://example.cyberark.cloud
	// If not provided, automatically resolved from username by ARK SDK
	EnvIdentityURL = "CYBERARK_IDENTITY_URL"
)

// TestAccPreCheckVars lists the required environment variables for acceptance tests
var TestAccPreCheckVars = []string{
	EnvUsername,
	EnvPassword,
	// EnvIdentityURL is optional - omitted from required list
}
