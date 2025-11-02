package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// SecretModel represents a secret resource in Terraform.
// Maps to cyberarksia_database_secret resource.
// Secrets are standalone credentials that can be referenced by database workspaces, VM workspaces, etc.
type SecretModel struct {
	// Computed attributes
	ID           types.String `tfsdk:"id"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastModified types.String `tfsdk:"last_modified"`

	// Required attributes
	Name               types.String `tfsdk:"name"`
	AuthenticationType types.String `tfsdk:"authentication_type"`

	// Conditional attributes (based on authentication_type)
	// Local/Domain authentication
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"` // Sensitive

	// Domain authentication (Active Directory)
	Domain types.String `tfsdk:"domain"`

	// AWS IAM authentication
	AWSAccessKeyID     types.String `tfsdk:"aws_access_key_id"`     // Sensitive
	AWSSecretAccessKey types.String `tfsdk:"aws_secret_access_key"` // Sensitive

	// Optional metadata
	Tags types.Map `tfsdk:"tags"` // Key-value tags for organization
}
