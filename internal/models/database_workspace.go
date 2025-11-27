// Package models provides data structures for Terraform resources
package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DatabaseWorkspaceModel represents a database workspace resource in Terraform state.
// This model maps to the cyberark_sia_database_workspace resource schema.
// Note: SIA API uses integer IDs internally, we store as string in Terraform for consistency
type DatabaseWorkspaceModel struct {
	Services                    types.Set    `tfsdk:"services"`
	Tags                        types.Map    `tfsdk:"tags"`
	AuthDatabase                types.String `tfsdk:"auth_database"`
	Account                     types.String `tfsdk:"account"`
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	DatabaseType                types.String `tfsdk:"database_type"`
	Address                     types.String `tfsdk:"address"`
	LastModified                types.String `tfsdk:"last_modified"`
	Region                      types.String `tfsdk:"region"`
	NetworkName                 types.String `tfsdk:"network_name"`
	ReadOnlyEndpoint            types.String `tfsdk:"read_only_endpoint"`
	AuthenticationMethod        types.String `tfsdk:"authentication_method"`
	SecretID                    types.String `tfsdk:"secret_id"`
	CertificateID               types.String `tfsdk:"certificate_id"`
	CloudProvider               types.String `tfsdk:"cloud_provider"`
	Port                        types.Int64  `tfsdk:"port"`
	EnableCertificateValidation types.Bool   `tfsdk:"enable_certificate_validation"`
}
