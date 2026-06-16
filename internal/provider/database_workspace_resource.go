// Package provider implements the database_target resource
package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/models"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/validators"
	dbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/db/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &databaseWorkspaceResource{}
	_ resource.ResourceWithConfigure   = &databaseWorkspaceResource{}
	_ resource.ResourceWithImportState = &databaseWorkspaceResource{}
)

// cloudProviderToAPI converts user-friendly cloud_provider values to API-expected Platform values
// Terraform uses lowercase with underscores (aws, azure, gcp, on_premise, atlas)
// SIA API expects uppercase with hyphens (AWS, AZURE, GCP, ON-PREMISE, ATLAS)
func cloudProviderToAPI(tfValue string) string {
	switch tfValue {
	case "on_premise":
		return "ON-PREMISE"
	default:
		return strings.ToUpper(tfValue)
	}
}

// cloudProviderFromAPI converts API Platform values to Terraform cloud_provider values
// Reverse of cloudProviderToAPI()
func cloudProviderFromAPI(apiValue string) string {
	switch apiValue {
	case "ON-PREMISE":
		return "on_premise"
	default:
		return strings.ToLower(apiValue)
	}
}

// stringValueOrNullWorkspace returns types.StringNull() for empty strings, otherwise types.StringValue()
// Used for optional string fields where API returns "" but config expects null
func stringValueOrNullWorkspace(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// NewDatabaseWorkspaceResource is a helper function to simplify the provider implementation
func NewDatabaseWorkspaceResource() resource.Resource {
	return &databaseWorkspaceResource{}
}

// databaseWorkspaceResource is the resource implementation
type databaseWorkspaceResource struct {
	providerData *ProviderData
}

// Metadata returns the resource type name
func (r *databaseWorkspaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_workspace"
}

// Schema defines the schema for the resource
func (r *databaseWorkspaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a database workspace in Idira SIA. Database workspaces represent existing databases " +
			"(AWS RDS, Azure SQL, on-premise) registered with SIA for secure access management.",
		MarkdownDescription: "Manages a database workspace in Idira SIA. Database workspaces represent existing databases " +
			"(AWS RDS, Azure SQL, on-premise) registered with SIA for secure access management.\n\n" +
			"**Note**: This resource does NOT create databases. It only registers existing databases with SIA.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "SIA-assigned unique identifier for the database workspace",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Database name on the database server (e.g., 'customers', 'inventory', 'myapp'). " +
					"This is the actual database/schema/catalog name that SIA will connect to. " +
					"For PostgreSQL: the database name in connection string (postgres://host:5432/DATABASE_NAME). " +
					"For MySQL: the database name (USE DATABASE_NAME). " +
					"For MongoDB: the database name to connect to. " +
					"Required, 1-255 characters, must be unique within SIA tenant.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"database_type": schema.StringAttribute{
				Description: "Type of database engine. REQUIRED due to SDK v1.5.0 validation constraints (empty strings fail validation). " +
					"Valid values include generic types (postgres, mysql, mariadb, mongo, oracle, mssql, sqlserver, db2) " +
					"and platform-specific variants (postgres-aws-rds, mysql-azure-managed, mongo-atlas-managed, etc.). " +
					"Validated against ARK SDK DatabaseEngineTypes - automatically stays in sync with SDK updates. " +
					"**Changing this value will force replacement of the resource.**",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validators.DatabaseEngine(), // Uses SDK's DatabaseEngineTypes list - stays in sync with SDK updates
				},
			},
			"address": schema.StringAttribute{
				Description: "Hostname, IP address, or FQDN of the database server (ReadWriteEndpoint in SDK). " +
					"Optional - some databases use service discovery.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"port": schema.Int64Attribute{
				Description: "TCP port for database connections (1-65535). Optional - SDK uses database family defaults if not provided.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"auth_database": schema.StringAttribute{
				Description: "Authentication database name (AuthDatabase in SDK). " +
					"Primarily used with MongoDB (default: 'admin'). Optional for other database types.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"services": schema.SetAttribute{
				Description: "Set of service names for the database (Services in SDK). " +
					"Used with Oracle and SQL Server for multi-service configurations. " +
					"Optional - only needed for databases with multiple services.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"account": schema.StringAttribute{
				Description: "Account name for provider-based databases (Account in SDK). " +
					"Used with Snowflake and MongoDB Atlas. Optional - only needed for these database types.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"network_name": schema.StringAttribute{
				Description: "Network name where the database resides (NetworkName in SDK). " +
					"Used for network segmentation and isolation. Defaults to 'ON-PREMISE' if not specified.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
			},
			"read_only_endpoint": schema.StringAttribute{
				Description: "Read-only endpoint for the database (ReadOnlyEndpoint in SDK). " +
					"Optional - used for read replica configurations to scale read operations.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"authentication_method": schema.StringAttribute{
				Description: "How SIA authenticates to the database (ConfiguredAuthMethodType in SDK). " +
					"Optional - SDK uses database family defaults if not provided. " +
					"Valid values: ad_ephemeral_user, local_ephemeral_user, rds_iam_authentication, atlas_ephemeral_user",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("ad_ephemeral_user", "local_ephemeral_user", "rds_iam_authentication", "atlas_ephemeral_user"),
				},
			},
			"secret_id": schema.StringAttribute{
				Description: "Reference to a secret stored in SIA's secret service. " +
					"Required for Zero Standing Privilege (ZSP) / Just-In-Time (JIT) access - SIA uses these credentials to provision ephemeral accounts. " +
					"Must reference an existing cyberarksia_database_secret resource. " +
					"Secret types: username_password, iam_user, cyberark_pam, atlas_access_keys",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"enable_certificate_validation": schema.BoolAttribute{
				Description: "Enforce TLS certificate validation for database connections (EnableCertificateValidation in SDK). " +
					"When true, requires valid TLS certificates. Defaults to true for security. " +
					"Set to false only if using self-signed certificates in non-production environments.",
				Optional: true,
			},
			"certificate_id": schema.StringAttribute{
				Description: "Certificate ID for TLS/mTLS connections (Certificate in SDK). " +
					"References a certificate stored in SIA's certificate service. " +
					"Optional - used for mutual TLS (mTLS) or custom CA certificates. " +
					"References cyberarksia_certificate resource ID.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Description: "Cloud provider hosting the database (Platform in SDK). " +
					"Valid values: aws, azure, gcp, on_premise, atlas. Defaults to on_premise.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("aws", "azure", "gcp", "on_premise", "atlas"),
				},
			},
			"region": schema.StringAttribute{
				Description: "Region of the database. Required for AWS RDS IAM authentication (rds_iam_authentication). " +
					"Used in AWS Signature Version 4 signing for generating temporary RDS authentication tokens. " +
					"Optional for other authentication methods and cloud providers.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			// Optional metadata
			"tags": schema.MapAttribute{
				Description: "Key-value tags for organizing and categorizing database workspaces. Maps to Tags in SDK.",
				ElementType: types.StringType,
				Optional:    true,
			},

			// Computed attributes
			"last_modified": schema.StringAttribute{
				Description: "Timestamp of last modification (ISO 8601, computed by SIA)",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource
func (r *databaseWorkspaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// handleCertificateError checks if an error is certificate-related and adds an actionable error diagnostic
// Returns true if a certificate error was detected and handled, false otherwise
func handleCertificateError(certificateID types.String, err error, resp interface{}) bool {
	// Only check if certificate ID is provided
	if certificateID.IsNull() {
		return false
	}

	// Extract diagnostics from response (works for both Create and Update)
	var diags *diag.Diagnostics
	switch r := resp.(type) {
	case *resource.CreateResponse:
		diags = &r.Diagnostics
	case *resource.UpdateResponse:
		diags = &r.Diagnostics
	default:
		return false
	}

	// Check for certificate-related error messages
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "certificate") &&
		(strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "does not exist") ||
			strings.Contains(errMsg, "invalid")) {
		diags.AddError(
			"Certificate Not Found",
			fmt.Sprintf(
				"The specified certificate (ID: %s) does not exist or is invalid.\n\n"+
					"Ensure the certificate exists before associating it with this database workspace.\n"+
					"You can verify the certificate exists with:\n"+
					"  terraform state show cyberarksia_certificate.<name>\n\n"+
					"Original error: %s",
				certificateID.ValueString(),
				err.Error(),
			),
		)
		return true
	}

	return false
}

// Create creates the resource and sets the initial Terraform state
func (r *databaseWorkspaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan models.DatabaseWorkspaceModel
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

	tflog.Info(ctx, "Creating database workspace", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	// Convert services set from Terraform to Go []string
	var services []string
	if !plan.Services.IsNull() {
		diag := plan.Services.ElementsAs(ctx, &services, false)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Build ARK SDK request model
	// Per docs/sdk-integration.md: Use siaAPI.WorkspacesDB().AddDatabase()
	// Convert cloud_provider from Terraform format to API format (on_premise -> ON-PREMISE, aws -> AWS, etc.)
	platformValue := ""
	if !plan.CloudProvider.IsNull() && !plan.CloudProvider.IsUnknown() {
		platformValue = cloudProviderToAPI(plan.CloudProvider.ValueString())
	}

	addDatabaseReq := &dbmodels.ArkSIADBAddDatabase{
		Name:                     plan.Name.ValueString(),
		NetworkName:              plan.NetworkName.ValueString(),
		Platform:                 platformValue,
		AuthDatabase:             plan.AuthDatabase.ValueString(),
		Services:                 services,
		Account:                  plan.Account.ValueString(),
		ProviderEngine:           plan.DatabaseType.ValueString(),
		Certificate:              plan.CertificateID.ValueString(),
		ReadWriteEndpoint:        plan.Address.ValueString(),
		ReadOnlyEndpoint:         plan.ReadOnlyEndpoint.ValueString(),
		Port:                     int(plan.Port.ValueInt64()),
		SecretID:                 plan.SecretID.ValueString(),
		ConfiguredAuthMethodType: plan.AuthenticationMethod.ValueString(),
		Region:                   plan.Region.ValueString(),
	}

	// SECURITY: Default to true if not explicitly set (secure by default)
	if !plan.EnableCertificateValidation.IsNull() {
		addDatabaseReq.EnableCertificateValidation = plan.EnableCertificateValidation.ValueBool()
	} else {
		addDatabaseReq.EnableCertificateValidation = true
	}

	// Convert tags from types.Map to map[string]string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags := make(map[string]string)
		diag := plan.Tags.ElementsAs(ctx, &tags, false)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		addDatabaseReq.Tags = tags
	}

	// Wrap SDK call with retry logic per docs/sdk-integration.md
	var database *dbmodels.ArkSIADBDatabase
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		database, apiErr = r.providerData.SIAAPI.WorkspacesDB().AddDatabase(addDatabaseReq)
		return apiErr
	})

	if err != nil {
		tflog.Error(ctx, "Failed to create database workspace", map[string]interface{}{
			"error": err.Error(),
		})

		// Check for certificate-related errors and provide actionable guidance
		if handleCertificateError(plan.CertificateID, err, resp) {
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "create database workspace"))
		return
	}

	// Map response to state
	plan.ID = types.StringValue(strconv.Itoa(database.ID))
	plan.DatabaseType = types.StringValue(database.ProviderDetails.Engine)
	// Note: ARK SDK v1.5.0 ArkSIADBDatabase model does not expose last_modified field
	// The API may track modification time internally, but it's not returned in the response
	plan.LastModified = types.StringNull()

	// Log certificate association if configured
	logFields := map[string]interface{}{
		"id": plan.ID.ValueString(),
	}
	if !plan.CertificateID.IsNull() && plan.CertificateID.ValueString() != "" {
		logFields["certificate_id"] = plan.CertificateID.ValueString()
		tflog.Info(ctx, "Database workspace associated with certificate", logFields)
	} else {
		tflog.Info(ctx, "Created database workspace", logFields)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data
func (r *databaseWorkspaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state models.DatabaseWorkspaceModel
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

	tflog.Debug(ctx, "Reading database workspace", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Convert string ID to int for SDK
	databaseID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Database ID",
			fmt.Sprintf("Unable to convert ID to integer: %s", err.Error()),
		)
		return
	}

	// Per docs/sdk-integration.md: Use siaAPI.WorkspacesDB().Database()
	// Note: SDK method is "Database", not "GetDatabase"
	// Handle 404 as resource deleted (drift detection)
	var database *dbmodels.ArkSIADBDatabase
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		database, apiErr = r.providerData.SIAAPI.WorkspacesDB().Database(&dbmodels.ArkSIADBGetDatabase{
			ID: databaseID,
		})
		return apiErr
	})

	if err != nil {
		// Check if resource was deleted outside Terraform (404)
		// Per sdk-integration.md: Handle 404 as resource deleted
		if client.IsNotFoundError(err) {
			tflog.Warn(ctx, "Database workspace not found, removing from state", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		tflog.Error(ctx, "Failed to read database workspace", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.Append(client.MapError(err, "read database workspace"))
		return
	}

	// Map response to state - update fields from API response
	// Use stringValueOrNullWorkspace for optional fields to prevent drift when API returns ""
	state.Name = types.StringValue(database.Name) // Required field
	state.NetworkName = stringValueOrNullWorkspace(database.NetworkName)
	// Convert Platform from API format back to Terraform format (ON-PREMISE -> on_premise, AWS -> aws, etc.)
	if database.Platform != "" {
		state.CloudProvider = types.StringValue(cloudProviderFromAPI(database.Platform))
	} else {
		state.CloudProvider = types.StringNull()
	}
	state.AuthDatabase = stringValueOrNullWorkspace(database.AuthDatabase)
	state.Account = stringValueOrNullWorkspace(database.Account)
	state.DatabaseType = types.StringValue(database.ProviderDetails.Engine) // Required field
	state.Address = stringValueOrNullWorkspace(database.ReadWriteEndpoint)
	state.ReadOnlyEndpoint = stringValueOrNullWorkspace(database.ReadOnlyEndpoint)
	state.Port = types.Int64Value(int64(database.Port))
	state.SecretID = types.StringValue(database.SecretID) // Required field
	state.EnableCertificateValidation = types.BoolValue(database.EnableCertificateValidation)
	state.CertificateID = stringValueOrNullWorkspace(database.Certificate)
	state.Region = stringValueOrNullWorkspace(database.Region)

	// Convert services []string from SDK to types.Set
	if len(database.Services) > 0 {
		servicesSet, diag := types.SetValueFrom(ctx, types.StringType, database.Services)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		state.Services = servicesSet
	} else {
		state.Services = types.SetNull(types.StringType)
	}

	// Convert tags from map[string]string to types.Map
	if len(database.Tags) > 0 {
		tagsMap, diag := types.MapValueFrom(ctx, types.StringType, database.Tags)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		state.Tags = tagsMap
	} else {
		state.Tags = types.MapNull(types.StringType)
	}

	tflog.Debug(ctx, "Successfully read database workspace", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success
func (r *databaseWorkspaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan and state
	var plan, state models.DatabaseWorkspaceModel
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

	tflog.Info(ctx, "Updating database workspace", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Convert string ID to int for SDK
	databaseID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Database ID",
			fmt.Sprintf("Unable to convert ID to integer: %s", err.Error()),
		)
		return
	}

	// Convert services set from Terraform to Go []string
	var services []string
	if !plan.Services.IsNull() {
		diag := plan.Services.ElementsAs(ctx, &services, false)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Build update request with only changed fields
	// Per docs/sdk-integration.md: Use siaAPI.WorkspacesDB().UpdateDatabase()
	// SDK signature: UpdateDatabase(*ArkSIADBUpdateDatabase) (*ArkSIADBDatabase, error)
	// Convert cloud_provider from Terraform format to API format (on_premise -> ON-PREMISE, aws -> AWS, etc.)
	platformValue := ""
	if !plan.CloudProvider.IsNull() && !plan.CloudProvider.IsUnknown() {
		platformValue = cloudProviderToAPI(plan.CloudProvider.ValueString())
	}

	updateReq := &dbmodels.ArkSIADBUpdateDatabase{
		ID:                       databaseID,
		NewName:                  plan.Name.ValueString(),
		NetworkName:              plan.NetworkName.ValueString(),
		Platform:                 platformValue,
		AuthDatabase:             plan.AuthDatabase.ValueString(),
		Services:                 services,
		Account:                  plan.Account.ValueString(),
		ProviderEngine:           plan.DatabaseType.ValueString(),
		Certificate:              plan.CertificateID.ValueString(),
		ReadWriteEndpoint:        plan.Address.ValueString(),
		ReadOnlyEndpoint:         plan.ReadOnlyEndpoint.ValueString(),
		Port:                     int(plan.Port.ValueInt64()),
		SecretID:                 plan.SecretID.ValueString(),
		ConfiguredAuthMethodType: plan.AuthenticationMethod.ValueString(),
		Region:                   plan.Region.ValueString(),
	}

	// SECURITY: Default to true if not explicitly set (secure by default)
	if !plan.EnableCertificateValidation.IsNull() {
		updateReq.EnableCertificateValidation = plan.EnableCertificateValidation.ValueBool()
	} else {
		updateReq.EnableCertificateValidation = true
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
	var updated *dbmodels.ArkSIADBDatabase
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		updated, apiErr = r.providerData.SIAAPI.WorkspacesDB().UpdateDatabase(updateReq)
		return apiErr
	})

	if err != nil {
		tflog.Error(ctx, "Failed to update database workspace", map[string]interface{}{
			"error": err.Error(),
		})

		// Check for certificate-related errors and provide actionable guidance
		if handleCertificateError(plan.CertificateID, err, resp) {
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "update database workspace"))
		return
	}

	// Map response to state
	plan.ID = types.StringValue(strconv.Itoa(updated.ID))
	plan.DatabaseType = types.StringValue(updated.ProviderDetails.Engine)
	// Note: ARK SDK v1.5.0 ArkSIADBDatabase model does not expose last_modified field
	// The API may track modification time internally, but it's not returned in the response
	plan.LastModified = types.StringNull()

	// Log certificate association changes if updated
	logFields := map[string]interface{}{
		"id": state.ID.ValueString(),
	}

	// Track certificate changes (added, updated, or removed)
	oldCertID := state.CertificateID.ValueString()
	newCertID := plan.CertificateID.ValueString()

	if oldCertID != newCertID {
		if newCertID != "" {
			logFields["certificate_id"] = newCertID
			if oldCertID != "" {
				logFields["old_certificate_id"] = oldCertID
				tflog.Info(ctx, "Database workspace certificate updated", logFields)
			} else {
				tflog.Info(ctx, "Database workspace associated with certificate", logFields)
			}
		} else {
			logFields["old_certificate_id"] = oldCertID
			tflog.Info(ctx, "Certificate removed from database workspace", logFields)
		}
	} else {
		tflog.Info(ctx, "Updated database workspace", logFields)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success
func (r *databaseWorkspaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state models.DatabaseWorkspaceModel
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

	tflog.Info(ctx, "Deleting database workspace", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	// Convert string ID to int for SDK
	databaseID, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Database ID",
			fmt.Sprintf("Unable to convert ID to integer: %s", err.Error()),
		)
		return
	}

	// WORKAROUND: ARK SDK v1.5.0 Bug - DeleteDatabase() panics with nil body
	// Use direct HTTP DELETE with empty map workaround instead of SDK method
	// See internal/client/delete_workarounds.go for details
	// TODO: Revert to SDK method when v1.6.0+ fixes nil body handling
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		return client.DeleteDatabaseWorkspaceDirect(ctx, r.providerData.AuthContext, databaseID)
	})

	if err != nil {
		// Gracefully handle already-deleted resource (404)
		if client.IsNotFoundError(err) {
			tflog.Warn(ctx, "Database workspace already deleted", map[string]interface{}{
				"id": state.ID.ValueString(),
			})
			return
		}

		tflog.Error(ctx, "Failed to delete database workspace", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.Append(client.MapError(err, "delete database workspace"))
		return
	}

	tflog.Info(ctx, "Deleted database workspace", map[string]interface{}{
		"id": state.ID.ValueString(),
	})
}

// ImportState imports an existing resource into Terraform state
func (r *databaseWorkspaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the ID from import to retrieve the resource
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	tflog.Info(ctx, "Imported database workspace", map[string]interface{}{
		"id": req.ID,
	})
}
