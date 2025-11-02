// Package provider implements the virtual machine secret resource
package provider

import (
	"context"
	"fmt"

	"github.com/aaearon/terraform-provider-cyberark-sia/internal/client"
	"github.com/aaearon/terraform-provider-cyberark-sia/internal/models"
	vmsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                   = &virtualMachineSecretResource{}
	_ resource.ResourceWithConfigure      = &virtualMachineSecretResource{}
	_ resource.ResourceWithImportState    = &virtualMachineSecretResource{}
	_ resource.ResourceWithValidateConfig = &virtualMachineSecretResource{}
)

// NewVirtualMachineSecretResource is a helper function to simplify the provider implementation
func NewVirtualMachineSecretResource() resource.Resource {
	return &virtualMachineSecretResource{}
}

// virtualMachineSecretResource is the resource implementation
type virtualMachineSecretResource struct {
	providerData *ProviderData
}

// Metadata returns the resource type name
func (r *virtualMachineSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_secret"
}

// Schema defines the schema for the resource
func (r *virtualMachineSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a virtual machine secret (credential) in CyberArk SIA. VM secrets are credentials " +
			"used for VM/server access provisioning. Supports ProvisionerUser (self-contained username/password) " +
			"and PCloudAccount (PAM vault account references).",
		MarkdownDescription: "Manages a virtual machine secret (credential) in CyberArk SIA. VM secrets are credentials " +
			"used for VM/server access provisioning.\n\n" +
			"**Secret Types**:\n" +
			"- `ProvisionerUser`: Username/password stored directly in SIA\n" +
			"- `PCloudAccount`: Reference to an existing PAM vault account",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "SIA-assigned unique identifier for the secret (equals secret_id)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_id": schema.StringAttribute{
				Description: "SIA-assigned UUID for the secret, immutable unique identifier. Maps to SecretID in SDK.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_name": schema.StringAttribute{
				Description: "User-friendly name for the secret (1-200 characters). Maps to SecretName in SDK.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"secret_type": schema.StringAttribute{
				Description: "Type of VM secret credentials. Valid values: ProvisionerUser, PCloudAccount. " +
					"Immutable - changing this triggers resource replacement (ForceNew). " +
					"Maps to SecretType in SDK.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("ProvisionerUser", "PCloudAccount"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// ProvisionerUser authentication fields
			"provisioner_username": schema.StringAttribute{
				Description: "Username for VM provisioning. Required when secret_type=ProvisionerUser. " +
					"Not allowed for PCloudAccount secrets. Maps to ProvisionerUsername in SDK. " +
					"Validated in ValidateConfig method.",
				Optional: true,
			},
			"provisioner_password": schema.StringAttribute{
				Description: "Password for VM provisioning. Required when secret_type=ProvisionerUser. " +
					"Not allowed for PCloudAccount secrets. Min 8 characters. NEVER logged or displayed in outputs. " +
					"Write-only field - API never returns passwords. Maps to ProvisionerPassword in SDK. " +
					"Validated in ValidateConfig method.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(8),
				},
			},

			// PCloudAccount authentication fields
			"pcloud_safe_name": schema.StringAttribute{
				Description: "PAM vault safe name containing the account. Required when secret_type=PCloudAccount. " +
					"Not allowed for ProvisionerUser secrets. Maps to PCloudAccountSafe in SDK. " +
					"Validated in ValidateConfig method.",
				Optional: true,
			},
			"pcloud_account_name": schema.StringAttribute{
				Description: "PAM account name within the safe. Required when secret_type=PCloudAccount. " +
					"Not allowed for ProvisionerUser secrets. Maps to PCloudAccountName in SDK. " +
					"Validated in ValidateConfig method.",
				Optional: true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource
func (r *virtualMachineSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.providerData = providerData
}

// ValidateConfig performs cross-attribute validation based on secret_type
func (r *virtualMachineSecretResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config models.VirtualMachineSecretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretType := config.SecretType.ValueString()

	switch secretType {
	case "ProvisionerUser":
		// Both fields required
		if config.ProvisionerUsername.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("provisioner_username"),
				"Missing Required Field",
				"provisioner_username is required when secret_type=ProvisionerUser",
			)
		}
		if config.ProvisionerPassword.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("provisioner_password"),
				"Missing Required Field",
				"provisioner_password is required when secret_type=ProvisionerUser",
			)
		}

		// Prevent PCloud fields
		if !config.PCloudSafeName.IsNull() && !config.PCloudSafeName.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("pcloud_safe_name"),
				"Invalid Field Combination",
				"pcloud_safe_name cannot be set when secret_type=ProvisionerUser",
			)
		}
		if !config.PCloudAccountName.IsNull() && !config.PCloudAccountName.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("pcloud_account_name"),
				"Invalid Field Combination",
				"pcloud_account_name cannot be set when secret_type=ProvisionerUser",
			)
		}

	case "PCloudAccount":
		// Both fields required
		if config.PCloudSafeName.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("pcloud_safe_name"),
				"Missing Required Field",
				"pcloud_safe_name is required when secret_type=PCloudAccount",
			)
		}
		if config.PCloudAccountName.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("pcloud_account_name"),
				"Missing Required Field",
				"pcloud_account_name is required when secret_type=PCloudAccount",
			)
		}

		// Prevent Provisioner fields
		if !config.ProvisionerUsername.IsNull() && !config.ProvisionerUsername.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("provisioner_username"),
				"Invalid Field Combination",
				"provisioner_username cannot be set when secret_type=PCloudAccount",
			)
		}
		if !config.ProvisionerPassword.IsNull() && !config.ProvisionerPassword.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("provisioner_password"),
				"Invalid Field Combination",
				"provisioner_password cannot be set when secret_type=PCloudAccount",
			)
		}
	}
}

// Create creates the resource and sets the initial Terraform state
func (r *virtualMachineSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.VirtualMachineSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Guard against nil provider data
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Expected configured provider data but got nil. Please report this issue to the provider developers.",
		)
		return
	}

	tflog.Info(ctx, "Creating VM secret", map[string]interface{}{
		"name":        plan.SecretName.ValueString(),
		"secret_type": plan.SecretType.ValueString(),
	})

	// Build SDK request
	addSecretReq := &vmsecretsmodels.ArkSIAVMAddSecret{
		SecretName: plan.SecretName.ValueString(),
		SecretType: plan.SecretType.ValueString(),
	}

	// Add type-specific fields
	if plan.SecretType.ValueString() == "ProvisionerUser" {
		addSecretReq.ProvisionerUsername = plan.ProvisionerUsername.ValueString()
		addSecretReq.ProvisionerPassword = plan.ProvisionerPassword.ValueString()
	} else if plan.SecretType.ValueString() == "PCloudAccount" {
		addSecretReq.PCloudAccountSafe = plan.PCloudSafeName.ValueString()
		addSecretReq.PCloudAccountName = plan.PCloudAccountName.ValueString()
	}

	// Call SDK with retry
	var secret *vmsecretsmodels.ArkSIAVMSecret
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		secret, apiErr = r.providerData.SIAAPI.SecretsVM().AddSecret(addSecretReq)
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "create VM secret"))
		return
	}

	// Map response to state
	plan.ID = types.StringValue(secret.SecretID)
	plan.SecretID = plan.ID

	tflog.Debug(ctx, "VM secret created successfully", map[string]interface{}{
		"secret_id": secret.SecretID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data
func (r *virtualMachineSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.VirtualMachineSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Guard against nil provider data
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Expected configured provider data but got nil. Please report this issue to the provider developers.",
		)
		return
	}

	secretID := state.ID.ValueString()
	tflog.Debug(ctx, "Reading VM secret", map[string]interface{}{
		"secret_id": secretID,
	})

	// Build SDK request
	getSecretReq := &vmsecretsmodels.ArkSIAVMGetSecret{
		SecretID: secretID,
	}

	// Call SDK with retry
	var secret *vmsecretsmodels.ArkSIAVMSecret
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		secret, apiErr = r.providerData.SIAAPI.SecretsVM().Secret(getSecretReq)
		return apiErr
	})

	// Handle 404 - secret deleted outside Terraform
	if client.IsNotFoundError(err) {
		tflog.Info(ctx, "VM secret not found, removing from state", map[string]interface{}{
			"secret_id": secretID,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "read VM secret"))
		return
	}

	// Update computed identifiers from API response (critical for import)
	state.ID = types.StringValue(secret.SecretID)
	state.SecretID = types.StringValue(secret.SecretID)

	// Update mutable fields from API response
	state.SecretName = types.StringValue(secret.SecretName)

	// Passwords are write-only - preserve existing state value
	// Username can be updated from API
	switch secret.SecretType {
	case "ProvisionerUser":
		// Update username from API if available in Secret.SecretData
		if secretData, ok := secret.Secret.SecretData.(map[string]interface{}); ok {
			if username, hasUsername := secretData["username"].(string); hasUsername {
				state.ProvisionerUsername = types.StringValue(username)
			}
		}
		// Password remains in state as-is (API never returns it)
	case "PCloudAccount":
		// Update PAM references from API if available in Secret.SecretData
		if secretData, ok := secret.Secret.SecretData.(map[string]interface{}); ok {
			if safe, hasSafe := secretData["safe"].(string); hasSafe {
				state.PCloudSafeName = types.StringValue(safe)
			}
			if accountName, hasAccount := secretData["account_name"].(string); hasAccount {
				state.PCloudAccountName = types.StringValue(accountName)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success
func (r *virtualMachineSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state models.VirtualMachineSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Guard against nil provider data
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Expected configured provider data but got nil. Please report this issue to the provider developers.",
		)
		return
	}

	secretID := state.ID.ValueString()
	tflog.Info(ctx, "Updating VM secret", map[string]interface{}{
		"secret_id": secretID,
	})

	// Build SDK request
	changeSecretReq := &vmsecretsmodels.ArkSIAVMChangeSecret{
		SecretID:   secretID,
		SecretName: plan.SecretName.ValueString(),
	}

	// Add type-specific fields if changed
	if plan.SecretType.ValueString() == "ProvisionerUser" {
		if !plan.ProvisionerUsername.Equal(state.ProvisionerUsername) || !plan.ProvisionerPassword.Equal(state.ProvisionerPassword) {
			changeSecretReq.ProvisionerUsername = plan.ProvisionerUsername.ValueString()
			changeSecretReq.ProvisionerPassword = plan.ProvisionerPassword.ValueString()
		}
	} else if plan.SecretType.ValueString() == "PCloudAccount" {
		if !plan.PCloudSafeName.Equal(state.PCloudSafeName) || !plan.PCloudAccountName.Equal(state.PCloudAccountName) {
			changeSecretReq.PCloudAccountSafe = plan.PCloudSafeName.ValueString()
			changeSecretReq.PCloudAccountName = plan.PCloudAccountName.ValueString()
		}
	}

	// Call SDK with retry
	var secret *vmsecretsmodels.ArkSIAVMSecret
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		secret, apiErr = r.providerData.SIAAPI.SecretsVM().ChangeSecret(changeSecretReq)
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "update VM secret"))
		return
	}

	// Update state with response
	plan.SecretName = types.StringValue(secret.SecretName)

	tflog.Debug(ctx, "VM secret updated successfully", map[string]interface{}{
		"secret_id": secretID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success
func (r *virtualMachineSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.VirtualMachineSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Guard against nil provider data
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Expected configured provider data but got nil. Please report this issue to the provider developers.",
		)
		return
	}

	secretID := state.ID.ValueString()
	tflog.Info(ctx, "Deleting VM secret", map[string]interface{}{
		"secret_id": secretID,
	})

	// CRITICAL: Use delete workaround - SDK DELETE has nil body panic bug
	// Wrap in retry for consistency with other operations
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		return client.DeleteVMSecretDirect(ctx, r.providerData.AuthContext, secretID)
	})

	// Handle 404 as success (idempotent delete)
	if client.IsNotFoundError(err) {
		tflog.Info(ctx, "VM secret already deleted (404), considering delete successful")
		return
	}

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "delete VM secret"))
		return
	}

	tflog.Debug(ctx, "VM secret deleted successfully", map[string]interface{}{
		"secret_id": secretID,
	})
}

// ImportState imports an existing resource by ID
func (r *virtualMachineSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by secret_id (UUID)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
