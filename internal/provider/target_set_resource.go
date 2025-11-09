// Package provider implements the target set resource
package provider

import (
	"context"
	"fmt"

	"github.com/aaearon/terraform-provider-cyberark-sia/internal/client"
	"github.com/aaearon/terraform-provider-cyberark-sia/internal/planmodifiers"
	"github.com/aaearon/terraform-provider-cyberark-sia/internal/validators"
	targetsetmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/targetsets/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &targetSetResource{}
	_ resource.ResourceWithConfigure   = &targetSetResource{}
	_ resource.ResourceWithImportState = &targetSetResource{}
)

// NewTargetSetResource is a helper function to simplify the provider implementation
func NewTargetSetResource() resource.Resource {
	return &targetSetResource{}
}

// targetSetResource is the resource implementation
type targetSetResource struct {
	providerData *ProviderData
}

// TargetSetModel describes the resource data model
type TargetSetModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Type                        types.String `tfsdk:"type"`
	SecretID                    types.String `tfsdk:"secret_id"`
	SecretType                  types.String `tfsdk:"secret_type"`
	ProvisionFormat             types.String `tfsdk:"provision_format"`
	Description                 types.String `tfsdk:"description"`
	EnableCertificateValidation types.Bool   `tfsdk:"enable_certificate_validation"`
}

// Metadata returns the resource type name
func (r *targetSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_target_set"
}

// Schema defines the schema for the resource
func (r *targetSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VM/server target set in CyberArk Secure Infrastructure Access (SIA). " +
			"Target sets define logical groupings of virtual machines and servers that share common " +
			"access credentials for Just-In-Time (JIT) privileged access.",
		MarkdownDescription: "Manages a VM/server target set in CyberArk Secure Infrastructure Access (SIA). " +
			"Target sets define logical groupings of virtual machines and servers that share common " +
			"access credentials for Just-In-Time (JIT) privileged access.\n\n" +
			"**Matching Patterns**:\n" +
			"- `Domain`: Matches all servers in a domain (e.g., *.example.com)\n" +
			"- `Suffix`: Matches servers with hostname suffix (e.g., *.dc1.example.com)\n" +
			"- `Target`: Matches specific server hostname",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier of the target set (equals name). " +
					"Changes when the target set is renamed.",
				MarkdownDescription: "Identifier of the target set (equals name). " +
					"Changes when the target set is renamed.",
				Computed: true,
				PlanModifiers: []planmodifier.String{
					planmodifiers.IDFollowsName(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the target set. Must be unique across the SIA tenant. " +
					"Avoid using forward slashes as they cause deletion issues.",
				MarkdownDescription: "Name of the target set. Must be unique across the SIA tenant. " +
					"Avoid using forward slashes as they cause deletion issues.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					validators.NoForwardSlashes(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Type of matching pattern: `Domain` (matches all servers in a domain), " +
					"`Suffix` (matches servers with hostname suffix), or `Target` (matches specific hostname).",
				MarkdownDescription: "Type of matching pattern: `Domain` (matches all servers in a domain), " +
					"`Suffix` (matches servers with hostname suffix), or `Target` (matches specific hostname).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("Domain", "Suffix", "Target"),
				},
			},
			"secret_id": schema.StringAttribute{
				Description: "ID of the VM secret containing credentials. " +
					"Reference `cyberarksia_virtual_machine_secret.example.id` for proper dependency ordering.",
				MarkdownDescription: "ID of the VM secret containing credentials. " +
					"Reference `cyberarksia_virtual_machine_secret.example.id` for proper dependency ordering.",
				Required: true,
			},
			"secret_type": schema.StringAttribute{
				Description: "Type of VM secret: `ProvisionerUser` (username/password credentials) or " +
					"`PCloudAccount` (PAM vault account reference).",
				MarkdownDescription: "Type of VM secret: `ProvisionerUser` (username/password credentials) or " +
					"`PCloudAccount` (PAM vault account reference).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("ProvisionerUser", "PCloudAccount"),
				},
			},
			"provision_format": schema.StringAttribute{
				Description: "Template for ephemeral account names. " +
					"Placeholders: `<user>` (requesting user), `<session-guid>` (unique session ID). " +
					"Cannot be removed once set (maintains audit trail consistency).",
				MarkdownDescription: "Template for ephemeral account names. " +
					"Placeholders: `<user>` (requesting user), `<session-guid>` (unique session ID). " +
					"Cannot be removed once set (maintains audit trail consistency).",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("<user>-<session-guid>"),
				PlanModifiers: []planmodifier.String{
					planmodifiers.PreventClearing(),
				},
			},
			"description": schema.StringAttribute{
				Description:         "Description of the target set.",
				MarkdownDescription: "Description of the target set.",
				Optional:            true,
			},
			"enable_certificate_validation": schema.BoolAttribute{
				Description:         "Whether to enable TLS/SSL certificate validation for connections to target servers.",
				MarkdownDescription: "Whether to enable TLS/SSL certificate validation for connections to target servers.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

// Configure adds the provider configured client to the resource
func (r *targetSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state
func (r *targetSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TargetSetModel

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

	tflog.Debug(ctx, "Creating target set", map[string]interface{}{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	// Build the API request
	addRequest := &targetsetmodels.ArkSIAAddTargetSet{
		Name:       plan.Name.ValueString(),
		Type:       plan.Type.ValueString(),
		SecretID:   plan.SecretID.ValueString(),
		SecretType: plan.SecretType.ValueString(),
	}

	// Add optional fields
	if !plan.ProvisionFormat.IsNull() && !plan.ProvisionFormat.IsUnknown() {
		addRequest.ProvisionFormat = plan.ProvisionFormat.ValueString()
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		addRequest.Description = plan.Description.ValueString()
	}
	if !plan.EnableCertificateValidation.IsNull() && !plan.EnableCertificateValidation.IsUnknown() {
		addRequest.EnableCertificateValidation = plan.EnableCertificateValidation.ValueBool()
	}

	// Call API with retry logic
	var targetSet *targetsetmodels.ArkSIATargetSet
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		targetSet, apiErr = r.providerData.SIAAPI.WorkspacesTargetSets().AddTargetSet(addRequest)
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "create target set"))
		return
	}

	tflog.Debug(ctx, "Target set created successfully", map[string]interface{}{
		"name": targetSet.Name,
	})

	// Map response to state
	plan.ID = types.StringValue(targetSet.Name)
	plan.Name = types.StringValue(targetSet.Name)
	plan.Type = types.StringValue(targetSet.Type)
	plan.SecretID = types.StringValue(targetSet.SecretID)
	plan.SecretType = types.StringValue(targetSet.SecretType)
	plan.ProvisionFormat = types.StringValue(targetSet.ProvisionFormat)

	if targetSet.Description != "" {
		plan.Description = types.StringValue(targetSet.Description)
	} else {
		plan.Description = types.StringNull()
	}

	plan.EnableCertificateValidation = types.BoolValue(targetSet.EnableCertificateValidation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data
func (r *targetSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TargetSetModel

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

	tflog.Debug(ctx, "Reading target set", map[string]interface{}{
		"name": state.Name.ValueString(),
	})

	// Call API with retry logic using direct workaround (handles URL-escaping for forward slashes)
	var targetSet *targetsetmodels.ArkSIATargetSet
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		targetSet, apiErr = client.GetTargetSetDirect(ctx, r.providerData.AuthContext, state.Name.ValueString())
		return apiErr
	})

	if err != nil {
		// Handle 404 as deleted resource (drift detection)
		if client.IsNotFoundError(err) {
			tflog.Debug(ctx, "Target set not found, removing from state", map[string]interface{}{
				"name": state.Name.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "read target set"))
		return
	}

	tflog.Debug(ctx, "Target set read successfully", map[string]interface{}{
		"name": targetSet.Name,
	})

	// Update state with latest values
	state.ID = types.StringValue(targetSet.Name)
	state.Name = types.StringValue(targetSet.Name)
	state.Type = types.StringValue(targetSet.Type)
	state.SecretID = types.StringValue(targetSet.SecretID)
	state.SecretType = types.StringValue(targetSet.SecretType)
	state.ProvisionFormat = types.StringValue(targetSet.ProvisionFormat)

	if targetSet.Description != "" {
		state.Description = types.StringValue(targetSet.Description)
	} else {
		state.Description = types.StringNull()
	}

	state.EnableCertificateValidation = types.BoolValue(targetSet.EnableCertificateValidation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success
func (r *targetSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TargetSetModel
	var state TargetSetModel

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

	tflog.Debug(ctx, "Updating target set", map[string]interface{}{
		"old_name": state.Name.ValueString(),
		"new_name": plan.Name.ValueString(),
	})

	// Build the API request as map[string]interface{} to preserve zero values
	// CRITICAL: API expects lowercase snake_case field names (not PascalCase!)
	updateRequest := map[string]interface{}{
		"name":        plan.Name.ValueString(), // New name (may be same or different)
		"type":        plan.Type.ValueString(),
		"secret_id":   plan.SecretID.ValueString(),
		"secret_type": plan.SecretType.ValueString(),
	}

	// Add optional fields - explicitly include empty/false values
	if !plan.ProvisionFormat.IsNull() && !plan.ProvisionFormat.IsUnknown() {
		updateRequest["provision_format"] = plan.ProvisionFormat.ValueString()
	}
	// Always include description - send "" to clear it (API may not support clearing)
	if !plan.Description.IsUnknown() {
		if plan.Description.IsNull() {
			updateRequest["description"] = ""
		} else {
			updateRequest["description"] = plan.Description.ValueString()
		}
	}
	// Always include enable_certificate_validation (even if false)
	// Fall back to state value if plan is unknown (field is Computed, so may be unknown)
	if !plan.EnableCertificateValidation.IsUnknown() {
		updateRequest["enable_certificate_validation"] = plan.EnableCertificateValidation.ValueBool()
	} else if !state.EnableCertificateValidation.IsNull() {
		updateRequest["enable_certificate_validation"] = state.EnableCertificateValidation.ValueBool()
	}

	// Call API with UPDATE workaround (bypasses SDK omitempty tags)
	// API returns {"target_set": {...}} with full updated resource
	var result map[string]interface{}
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var apiErr error
		result, apiErr = client.UpdateTargetSetDirect(ctx, r.providerData.AuthContext, state.Name.ValueString(), updateRequest)
		return apiErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "update target set"))
		return
	}

	tflog.Debug(ctx, "Target set updated successfully", map[string]interface{}{
		"name": result["name"],
	})

	// Update state - ID follows name for renames
	// API returns lowercase field names in JSON response
	if name, ok := result["name"].(string); ok {
		plan.ID = types.StringValue(name)
		plan.Name = types.StringValue(name)
	}
	if t, ok := result["type"].(string); ok {
		plan.Type = types.StringValue(t)
	}
	if sid, ok := result["secret_id"].(string); ok {
		plan.SecretID = types.StringValue(sid)
	}
	if st, ok := result["secret_type"].(string); ok {
		plan.SecretType = types.StringValue(st)
	}
	if pf, ok := result["provision_format"].(string); ok {
		plan.ProvisionFormat = types.StringValue(pf)
	}

	// description can be nil or empty string in JSON response
	// API treats "" and null as equivalent - always use null for consistency with Read()
	if desc, ok := result["description"].(string); ok && desc != "" {
		plan.Description = types.StringValue(desc)
	} else {
		plan.Description = types.StringNull()
	}

	if cv, ok := result["enable_certificate_validation"].(bool); ok {
		plan.EnableCertificateValidation = types.BoolValue(cv)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success
func (r *targetSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TargetSetModel

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

	tflog.Debug(ctx, "Deleting target set", map[string]interface{}{
		"name": state.Name.ValueString(),
	})

	// Use DELETE workaround to bypass SDK nil body panic bug with retry logic
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		return client.DeleteTargetSetDirect(ctx, r.providerData.AuthContext, state.Name.ValueString())
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "delete target set"))
		return
	}

	tflog.Debug(ctx, "Target set deleted successfully", map[string]interface{}{
		"name": state.Name.ValueString(),
	})
}

// ImportState imports the resource into Terraform state
func (r *targetSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using name as the identifier
	// The ID is computed from name, so we import by name
	tflog.Debug(ctx, "Importing target set", map[string]interface{}{
		"name": req.ID,
	})

	// Set both name and id to the import identifier
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
