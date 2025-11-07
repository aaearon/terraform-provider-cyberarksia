package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// VirtualMachineSecretModel represents a VM secret resource in Terraform
type VirtualMachineSecretModel struct {
	// Identifiers (computed)
	ID       types.String `tfsdk:"id"`
	SecretID types.String `tfsdk:"secret_id"`

	// Required attributes
	SecretName types.String `tfsdk:"secret_name"`
	SecretType types.String `tfsdk:"secret_type"`

	// ProvisionerUser Type Fields (conditional)
	ProvisionerUsername types.String `tfsdk:"provisioner_username"`
	ProvisionerPassword types.String `tfsdk:"provisioner_password"`

	// PCloudAccount Type Fields (conditional)
	PCloudSafeName    types.String `tfsdk:"pcloud_safe_name"`
	PCloudAccountName types.String `tfsdk:"pcloud_account_name"`
}
