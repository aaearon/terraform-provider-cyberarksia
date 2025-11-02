// Package provider implements the database secret resource
package provider

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                   = &databaseSecretResource{}
	_ resource.ResourceWithConfigure      = &databaseSecretResource{}
	_ resource.ResourceWithImportState    = &databaseSecretResource{}
	_ resource.ResourceWithValidateConfig = &databaseSecretResource{}
)

// NewDatabaseSecretResource is a helper function to simplify the provider implementation
func NewDatabaseSecretResource() resource.Resource {
	return &databaseSecretResource{}
}

// databaseSecretResource is the resource implementation
type databaseSecretResource struct {
	providerData *ProviderData
}

// Metadata returns the resource type name
func (r *databaseSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_secret"
}

// Schema defines the schema for the resource
func (r *databaseSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"aws_account": schema.StringAttribute{
				Description: "AWS account number (12 digits). Required when authentication_type=aws_iam. " +
					"Not allowed for local or domain authentication. " +
					"Maps to IAMAccount in SDK. Example: 123456789012. " +
					"Validated in ValidateConfig method.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(12, 12),
				},
			},
			"aws_username": schema.StringAttribute{
				Description: "AWS IAM username portion from the IAM user ARN. Required when authentication_type=aws_iam. " +
					"Not allowed for local or domain authentication. " +
					"Maps to IAMUsername in SDK. Example: database-admin. " +
					"Validated in ValidateConfig method.",
				Optional: true,
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
func (r *databaseSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *databaseSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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
		addSecretReq.IAMAccount = plan.AWSAccount.ValueString()
		addSecretReq.IAMUsername = plan.AWSUsername.ValueString()

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
func (r *databaseSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	// Map SecretType to AuthenticationType
	// Per Create() logic: "local"/"domain" → "username_password", "aws_iam" → "iam_user"
	// For Read/Import, we need to reverse this mapping
	switch secretMetadata.SecretType {
	case "username_password":
		// Extract username from SecretExposedData
		if username, ok := secretMetadata.SecretExposedData["username"].(string); ok {
			state.Username = types.StringValue(username)

			// Only infer authentication_type if it's currently unknown (e.g., during import)
			// Otherwise, respect the existing value from state to avoid perpetual drift
			if state.AuthenticationType.IsNull() || state.AuthenticationType.IsUnknown() {
				// Infer authentication_type based on username format
				// Domain authentication uses: DOMAIN\username or username@domain
				if strings.Contains(username, "\\") || strings.Contains(username, "@") {
					state.AuthenticationType = types.StringValue("domain")
				} else {
					state.AuthenticationType = types.StringValue("local")
				}
			}

			// Extract domain if authentication_type is "domain" (either from state or inferred)
			if state.AuthenticationType.ValueString() == "domain" {
				// Try to extract domain for user convenience
				if strings.Contains(username, "\\") {
					// Windows format: DOMAIN\username
					parts := strings.Split(username, "\\")
					if len(parts) == 2 {
						state.Domain = types.StringValue(parts[0])
					}
				} else if strings.Contains(username, "@") {
					// UPN format: username@domain
					parts := strings.Split(username, "@")
					if len(parts) == 2 {
						state.Domain = types.StringValue(parts[1])
					}
				}
			} else {
				// Clear domain if not domain auth
				state.Domain = types.StringNull()
			}
		} else {
			// Fallback if username not in exposed data
			if state.AuthenticationType.IsNull() || state.AuthenticationType.IsUnknown() {
				state.AuthenticationType = types.StringValue("local")
			}
		}
	case "iam_user":
		state.AuthenticationType = types.StringValue("aws_iam")
		// Extract IAM fields from SecretExposedData
		// Per SDK line 184-189: account, username, access_key_id, secret_access_key
		if secretData, ok := secretMetadata.SecretExposedData["account"].(string); ok {
			state.AWSAccount = types.StringValue(secretData)
		}
		if username, ok := secretMetadata.SecretExposedData["username"].(string); ok {
			state.AWSUsername = types.StringValue(username)
		}
		// Note: access_key_id and secret_access_key are sensitive and not returned
		// Keep existing values from state (during refresh) or leave empty (during import)
	default:
		tflog.Warn(ctx, "Unknown secret type returned by API", map[string]interface{}{
			"secret_type": secretMetadata.SecretType,
		})
		// Keep existing authentication_type from state
	}

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
func (r *databaseSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
		if !plan.AWSAccount.IsNull() {
			updateReq.IAMAccount = plan.AWSAccount.ValueString()
		}
		if !plan.AWSUsername.IsNull() {
			updateReq.IAMUsername = plan.AWSUsername.ValueString()
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
func (r *databaseSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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
func (r *databaseSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the ID from import to retrieve the resource
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Info(ctx, "Imported secret", map[string]interface{}{
		"id": req.ID,
	})
}

// ValidateConfig performs cross-field validation for the secret resource
func (r *databaseSecretResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config models.SecretModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate authentication type combinations
	authType := config.AuthenticationType.ValueString()

	switch authType {
	case "local":
		// Username and password are required for local authentication
		// Skip validation if values are unknown (e.g., from variables during plan phase)
		if config.Username.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing Required Field",
				"username is required when authentication_type=local",
			)
		}
		if config.Password.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing Required Field",
				"password is required when authentication_type=local",
			)
		}

		// Domain field should not be set for local authentication
		if !config.Domain.IsNull() && !config.Domain.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("domain"),
				"Invalid Field Combination",
				"domain cannot be set when authentication_type=local (use authentication_type=domain for Active Directory accounts)",
			)
		}

		// AWS IAM fields should not be set
		if !config.AWSAccessKeyID.IsNull() && !config.AWSAccessKeyID.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_access_key_id"),
				"Invalid Field Combination",
				"aws_access_key_id cannot be set when authentication_type=local",
			)
		}
		if !config.AWSSecretAccessKey.IsNull() && !config.AWSSecretAccessKey.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_secret_access_key"),
				"Invalid Field Combination",
				"aws_secret_access_key cannot be set when authentication_type=local",
			)
		}
		if !config.AWSAccount.IsNull() && !config.AWSAccount.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_account"),
				"Invalid Field Combination",
				"aws_account cannot be set when authentication_type=local",
			)
		}
		if !config.AWSUsername.IsNull() && !config.AWSUsername.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_username"),
				"Invalid Field Combination",
				"aws_username cannot be set when authentication_type=local",
			)
		}

	case "domain":
		// Username and password are required for domain authentication
		if config.Username.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing Required Field",
				"username is required when authentication_type=domain",
			)
		}
		if config.Password.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing Required Field",
				"password is required when authentication_type=domain",
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
		if !config.AWSAccount.IsNull() && !config.AWSAccount.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_account"),
				"Invalid Field Combination",
				fmt.Sprintf("aws_account cannot be set when authentication_type=%s", authType),
			)
		}
		if !config.AWSUsername.IsNull() && !config.AWSUsername.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_username"),
				"Invalid Field Combination",
				fmt.Sprintf("aws_username cannot be set when authentication_type=%s", authType),
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
		if config.AWSAccount.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_account"),
				"Missing Required Field",
				"aws_account is required when authentication_type=aws_iam (12-digit AWS account number)",
			)
		}
		if config.AWSUsername.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("aws_username"),
				"Missing Required Field",
				"aws_username is required when authentication_type=aws_iam (IAM username from ARN)",
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
