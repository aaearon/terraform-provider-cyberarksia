// Package provider implements the secret resource
package provider

import (
	"context"
	"fmt"

	"github.com/aaearon/terraform-provider-cyberark-sia/internal/client"
	"github.com/aaearon/terraform-provider-cyberark-sia/internal/models"
	secretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/db/models"
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
	_ resource.Resource                   = &secretResource{}
	_ resource.ResourceWithConfigure      = &secretResource{}
	_ resource.ResourceWithImportState    = &secretResource{}
	_ resource.ResourceWithValidateConfig = &secretResource{}
)

// NewSecretResource is a helper function to simplify the provider implementation
func NewSecretResource() resource.Resource {
	return &secretResource{}
}

// secretResource is the resource implementation
type secretResource struct {
	providerData *ProviderData
}

// Metadata returns the resource type name
func (r *secretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_secret"
}

// Schema defines the schema for the resource
func (r *secretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a secret (credential) in CyberArk SIA. Secrets are credentials " +
			"that SIA uses to provision ephemeral database access for users. Supports local authentication, " +
			"Active Directory, and AWS IAM authentication methods.",
		MarkdownDescription: "Manages a secret (credential) in CyberArk SIA. Secrets are credentials " +
			"that SIA uses to provision ephemeral database access for users.\n\n" +
			"**Authentication Types**:\n" +
			"- `local`: Username/password stored in SIA (username_password secret type)\n" +
			"- `domain`: Active Directory account (username_password secret type with domain)\n" +
			"- `aws_iam`: AWS IAM credentials (iam_user secret type)",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "SIA-assigned unique identifier for the secret (secret_id)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "User-friendly name for the secret (1-255 characters, unique within SIA tenant). " +
					"Maps to SecretName in SDK.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"authentication_type": schema.StringAttribute{
				Description: "Type of authentication credentials. " +
					"Valid values: local (username/password), domain (Active Directory), aws_iam (AWS IAM user). " +
					"Maps to SecretType in SDK (username_password or iam_user). " +
					"Must match the authentication_method of the referenced database_target.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("local", "domain", "aws_iam"),
				},
			},

			// Local/Domain authentication fields
			"username": schema.StringAttribute{
				Description: "Account username. Required for local and domain authentication types. " +
					"Not allowed for aws_iam authentication. Max 255 characters. " +
					"Validated in ValidateConfig method.",
				Optional:  true,
				Sensitive: false, // Username is not sensitive, only password is
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"password": schema.StringAttribute{
				Description: "Account password. Required for local and domain authentication types. " +
					"Not allowed for aws_iam authentication. Min 8 characters. " +
					"NEVER logged or displayed in outputs. " +
					"Validated in ValidateConfig method.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(8),
				},
			},

			// Domain authentication field
			"domain": schema.StringAttribute{
				Description: "Active Directory domain (e.g., corp.example.com). " +
					"Optional field for documentation purposes when authentication_type=domain. " +
					"NOTE: The ARK SDK does not have a separate domain field. " +
					"Include the domain directly in the username field using either:\n" +
					"- Windows format: DOMAIN\\username\n" +
					"- UPN format: username@domain.com\n" +
					"This field is for user convenience and is not sent to the SDK. " +
					"Validated in ValidateConfig method.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			// AWS IAM authentication fields
			"aws_access_key_id": schema.StringAttribute{
				Description: "AWS IAM access key ID. Required when authentication_type=aws_iam. " +
					"Not allowed for local or domain authentication. " +
					"Maps to IAMAccessKeyID in SDK. Valid AWS access key format (20 characters). " +
					"Validated in ValidateConfig method.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"aws_secret_access_key": schema.StringAttribute{
				Description: "AWS IAM secret access key. Required when authentication_type=aws_iam. " +
					"Not allowed for local or domain authentication. " +
					"Maps to IAMSecretAccessKey in SDK. NEVER logged or displayed in outputs. " +
					"Validated in ValidateConfig method.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			// Optional metadata
			"tags": schema.MapAttribute{
				Description: "Key-value tags for organizing and categorizing secrets. Maps to Tags in SDK.",
				ElementType: types.StringType,
				Optional:    true,
			},

			// Computed attributes
			"created_at": schema.StringAttribute{
				Description: "Timestamp of creation (ISO 8601, computed by SIA). Maps to CreationTime in SDK.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_modified": schema.StringAttribute{
				Description: "Timestamp of last modification (ISO 8601, computed by SIA). Maps to LastUpdateTime in SDK.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource
func (r *secretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured
	if req.ProviderData == nil {
		return
	}

	// Type assertion with error handling
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

// Create creates the resource and sets the initial Terraform state
func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan models.SecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if provider is configured
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	tflog.Info(ctx, "Creating secret", map[string]interface{}{
		"name":                plan.Name.ValueString(),
		"authentication_type": plan.AuthenticationType.ValueString(),
	})

	// Build ARK SDK request model based on authentication type
	// Per docs/sdk-integration.md: Use siaAPI.SecretsDB().AddSecret()
	addSecretReq := &secretsmodels.ArkSIADBAddSecret{
		SecretName: plan.Name.ValueString(),
	}

	// Set authentication type-specific fields
	authType := plan.AuthenticationType.ValueString()
	switch authType {
	case "local", "domain":
		addSecretReq.SecretType = "username_password"
		addSecretReq.Username = plan.Username.ValueString()
		addSecretReq.Password = plan.Password.ValueString()
		// Note: ARK SDK v1.5.0 does not have a separate Domain field for username_password secrets.
		// For Active Directory authentication, include the domain in the username field:
		// - Windows format: "DOMAIN\username"
		// - UPN format: "username@domain.com"
		// The domain field in Terraform schema is for user convenience but not sent to SDK.

	case "aws_iam":
		addSecretReq.SecretType = "iam_user"
		addSecretReq.IAMAccessKeyID = plan.AWSAccessKeyID.ValueString()
		addSecretReq.IAMSecretAccessKey = plan.AWSSecretAccessKey.ValueString()
		// Note: IAMAccount and IAMUsername are optional fields in ARK SDK v1.5.0
		// They may be derived automatically from the database_workspace_id if not provided
		// The SDK associates the secret with the database workspace, which determines the account context

	default:
		resp.Diagnostics.AddError(
			"Invalid Authentication Type",
			fmt.Sprintf("Unsupported authentication type: %s. Valid values: local, domain, aws_iam", authType),
		)
		return
	}

	// Convert tags from types.Map to map[string]string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags := make(map[string]string)
		diag := plan.Tags.ElementsAs(ctx, &tags, false)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		addSecretReq.Tags = tags
	}

	// Wrap SDK call with retry logic per docs/sdk-integration.md
	var secretMetadata *secretsmodels.ArkSIADBSecretMetadata
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		secretMetadata, apiErr = r.providerData.SIAAPI.SecretsDB().AddSecret(addSecretReq)
		return apiErr
	})

	if err != nil {
		tflog.Error(ctx, "Failed to create secret", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.Append(client.MapError(err, "create secret"))
		return
	}

	// Map response to state
	plan.ID = types.StringValue(secretMetadata.SecretID)
	plan.CreatedAt = types.StringValue(secretMetadata.CreationTime)
	plan.LastModified = types.StringValue(secretMetadata.LastUpdateTime)

	tflog.Info(ctx, "Created secret", map[string]interface{}{
		"id": plan.ID.ValueString(),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data
func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state models.SecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if provider is configured
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	tflog.Debug(ctx, "Reading secret", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Per docs/sdk-integration.md: Use siaAPI.SecretsDB().GetSecret()
	// Note: Response contains metadata only, no sensitive credentials per contract
	// Handle 404 as resource deleted (drift detection)
	var secretMetadata *secretsmodels.ArkSIADBSecretMetadata
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		// SDK method signature: Secret(*ArkSIADBGetSecret) (*ArkSIADBSecretMetadata, error)
		secretMetadata, apiErr = r.providerData.SIAAPI.SecretsDB().Secret(&secretsmodels.ArkSIADBGetSecret{
			SecretID: state.ID.ValueString(),
		})
		return apiErr
	})

	if err != nil {
		// Check if resource was deleted outside Terraform (404)
		if client.IsNotFoundError(err) {
			tflog.Warn(ctx, "Secret not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		tflog.Error(ctx, "Failed to read secret", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.Append(client.MapError(err, "read secret"))
		return
	}

	// Map response to state - update non-sensitive fields from API response
	// NOTE: Sensitive credentials (password, secret keys) are NOT returned by API
	state.Name = types.StringValue(secretMetadata.SecretName)
	state.CreatedAt = types.StringValue(secretMetadata.CreationTime)
	state.LastModified = types.StringValue(secretMetadata.LastUpdateTime)

	// Convert tags from map[string]string to types.Map
	if len(secretMetadata.Tags) > 0 {
		tagsMap, diag := types.MapValueFrom(ctx, types.StringType, secretMetadata.Tags)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		state.Tags = tagsMap
	} else {
		state.Tags = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, "Successfully read secret", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success
func (r *secretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan and state
	var plan, state models.SecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if provider is configured
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	tflog.Info(ctx, "Updating secret", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Build update request - handle both metadata and credential updates
	// Per docs/sdk-integration.md: Use siaAPI.SecretsDB().UpdateSecret()
	// SDK signature: UpdateSecret(*ArkSIADBUpdateSecret) (*ArkSIADBSecretMetadata, error)
	// Note: SIA updates credentials immediately per FR-015a
	updateReq := &secretsmodels.ArkSIADBUpdateSecret{
		SecretID:      state.ID.ValueString(),
		NewSecretName: plan.Name.ValueString(),
	}

	// Update credentials if changed (based on authentication type)
	authType := plan.AuthenticationType.ValueString()
	switch authType {
	case "local", "domain":
		// Update username/password if provided
		if !plan.Username.IsNull() {
			updateReq.Username = plan.Username.ValueString()
		}
		if !plan.Password.IsNull() {
			updateReq.Password = plan.Password.ValueString()
		}

	case "aws_iam":
		// Update IAM credentials if provided
		if !plan.AWSAccessKeyID.IsNull() {
			updateReq.IAMAccessKeyID = plan.AWSAccessKeyID.ValueString()
		}
		if !plan.AWSSecretAccessKey.IsNull() {
			updateReq.IAMSecretAccessKey = plan.AWSSecretAccessKey.ValueString()
		}
	}

	// Convert tags from types.Map to map[string]string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags := make(map[string]string)
		diag := plan.Tags.ElementsAs(ctx, &tags, false)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		updateReq.Tags = tags
	}

	// Wrap SDK call with retry logic
	var updated *secretsmodels.ArkSIADBSecretMetadata
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		updated, apiErr = r.providerData.SIAAPI.SecretsDB().UpdateSecret(updateReq)
		return apiErr
	})

	if err != nil {
		tflog.Error(ctx, "Failed to update secret", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.Append(client.MapError(err, "update secret"))
		return
	}

	// Map response to state
	plan.LastModified = types.StringValue(updated.LastUpdateTime)

	tflog.Info(ctx, "Updated secret", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success
func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state models.SecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if provider is configured
	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	tflog.Info(ctx, "Deleting secret", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// WORKAROUND: ARK SDK v1.5.0 Bug - DeleteSecret() panics with nil body
	// Use direct HTTP DELETE with empty map workaround instead of SDK method
	// See internal/client/delete_workarounds.go for details
	// TODO: Revert to SDK method when v1.6.0+ fixes nil body handling
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		return client.DeleteSecretDirect(ctx, r.providerData.AuthContext, state.ID.ValueString())
	})

	if err != nil {
		// Gracefully handle already-deleted resource (404)
		if client.IsNotFoundError(err) {
			tflog.Warn(ctx, "Secret already deleted", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			return
		}

		tflog.Error(ctx, "Failed to delete secret", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.Append(client.MapError(err, "delete secret"))
		return
	}

	tflog.Info(ctx, "Deleted secret", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

// ImportState imports an existing resource into Terraform state
func (r *secretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the ID from import to retrieve the resource
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Info(ctx, "Imported secret", map[string]interface{}{
		"id": req.ID,
	})
}

// ValidateConfig performs cross-field validation for the secret resource
func (r *secretResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config models.SecretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate authentication type combinations
	authType := config.AuthenticationType.ValueString()

	switch authType {
	case "local", "domain":
		// Username and password are required for local/domain authentication
		// Skip validation if values are unknown (e.g., from variables during plan phase)
		if config.Username.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing Required Field",
				fmt.Sprintf("username is required when authentication_type=%s", authType),
			)
		}
		if config.Password.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing Required Field",
				fmt.Sprintf("password is required when authentication_type=%s", authType),
			)
		}

		// AWS IAM fields should not be set
		if !config.AWSAccessKeyID.IsNull() && !config.AWSAccessKeyID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_access_key_id"),
				"Invalid Field Combination",
				fmt.Sprintf("aws_access_key_id cannot be set when authentication_type=%s", authType),
			)
		}
		if !config.AWSSecretAccessKey.IsNull() && !config.AWSSecretAccessKey.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_secret_access_key"),
				"Invalid Field Combination",
				fmt.Sprintf("aws_secret_access_key cannot be set when authentication_type=%s", authType),
			)
		}

	case "aws_iam":
		// AWS IAM credentials are required
		// Skip validation if values are unknown (e.g., from variables during plan phase)
		if config.AWSAccessKeyID.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_access_key_id"),
				"Missing Required Field",
				"aws_access_key_id is required when authentication_type=aws_iam",
			)
		}
		if config.AWSSecretAccessKey.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_secret_access_key"),
				"Missing Required Field",
				"aws_secret_access_key is required when authentication_type=aws_iam",
			)
		}

		// Username/password/domain should not be set
		if !config.Username.IsNull() && !config.Username.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Invalid Field Combination",
				"username cannot be set when authentication_type=aws_iam",
			)
		}
		if !config.Password.IsNull() && !config.Password.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Invalid Field Combination",
				"password cannot be set when authentication_type=aws_iam",
			)
		}
		if !config.Domain.IsNull() && !config.Domain.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("domain"),
				"Invalid Field Combination",
				"domain cannot be set when authentication_type=aws_iam",
			)
		}
	}
}
