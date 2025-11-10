package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/models"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/validators"
	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatabasePolicyPrincipalAssignmentResource{}
var _ resource.ResourceWithImportState = &DatabasePolicyPrincipalAssignmentResource{}

func NewDatabasePolicyPrincipalAssignmentResource() resource.Resource {
	return &DatabasePolicyPrincipalAssignmentResource{}
}

// DatabasePolicyPrincipalAssignmentResource defines the resource implementation.
type DatabasePolicyPrincipalAssignmentResource struct {
	providerData *ProviderData
}

func (r *DatabasePolicyPrincipalAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_policy_principal_assignment"
}

func (r *DatabasePolicyPrincipalAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the assignment of a principal (user/group/role) to a database access policy. " +
			"This resource follows the modular assignment pattern - manage individual principal assignments " +
			"rather than managing the entire policy.\n\n" +
			"**Composite ID Format**: `policy-id:principal-id:principal-type` (3-part format required to handle duplicate principal IDs across types).\n\n" +
			"**Conditional Validation**: `source_directory_name` and `source_directory_id` are required for USER and GROUP principal types, but optional for ROLE.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Composite identifier in the format `policy-id:principal-id:principal-type`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the database access policy. Use `cyberarksia_database_policy.example.policy_id`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_id": schema.StringAttribute{
				MarkdownDescription: "Principal identifier in UUID format (e.g., `c2c7bcc6-9560-44e0-8dff-5be221cd37ee`). This is the unique identifier returned by the SIA API.",
				Required:            true,
				Validators: []validator.String{
					validators.UUID(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_type": schema.StringAttribute{
				MarkdownDescription: "Principal type. Valid values: `USER`, `GROUP`, `ROLE`.",
				Required:            true,
				Validators: []validator.String{
					validators.PrincipalType(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"principal_name": schema.StringAttribute{
				MarkdownDescription: "Principal name. For USER principals, typically in email format (e.g., `user@example.com` or `tim.schindler@cyberark.cloud.40562`). For GROUP and ROLE principals, may be a display name or identifier without email format.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(256),
				},
			},
			"source_directory_name": schema.StringAttribute{
				MarkdownDescription: "Source identity directory name (max 50 characters). **Required** for USER and GROUP types. Examples: `AzureAD`, `LDAP`, `Okta`.",
				Optional:            true,
			},
			"source_directory_id": schema.StringAttribute{
				MarkdownDescription: "Source identity directory ID. **Required** for USER and GROUP types.",
				Optional:            true,
			},
			"last_modified": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last modification.",
				Computed:            true,
			},
		},
	}
}

func (r *DatabasePolicyPrincipalAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ProviderData, got: %T", req.ProviderData),
		)
		return
	}

	r.providerData = providerData
}

func (r *DatabasePolicyPrincipalAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.PolicyPrincipalAssignmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	// Validate conditional requirements
	if err := validatePrincipalDirectory(data.PrincipalType.ValueString(), data.SourceDirectoryName.ValueString(), data.SourceDirectoryID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Validation Error", err.Error())
		return
	}

	policyID := data.PolicyID.ValueString()

	// Read-modify-write: Fetch existing policy
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "fetch policy for principal assignment"))
		return
	}

	// Check for duplicate principal
	principalID := data.PrincipalID.ValueString()
	principalType := data.PrincipalType.ValueString()
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			resp.Diagnostics.AddError(
				"Principal Already Assigned",
				fmt.Sprintf("Principal %s (type: %s) is already assigned to policy %s", principalID, principalType, policyID),
			)
			return
		}
	}

	// Add new principal
	newPrincipal := data.ToSDKPrincipal()
	policy.Principals = append(policy.Principals, newPrincipal)

	// Update policy with retry
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		_, err := r.providerData.UAPClient.Db().UpdatePolicy(policy)
		return err
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "assign principal to policy"))
		return
	}

	// Populate state
	data.FromSDKPrincipal(policyID, newPrincipal)

	tflog.Info(ctx, "Created principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePolicyPrincipalAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.PolicyPrincipalAssignmentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	policyID := data.PolicyID.ValueString()
	principalID := data.PrincipalID.ValueString()
	principalType := data.PrincipalType.ValueString()

	// Fetch policy
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(client.MapError(err, "fetch policy for read"))
		return
	}

	// Find principal
	var found bool
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			data.FromSDKPrincipal(policyID, p)
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Debug(ctx, "Read principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePolicyPrincipalAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.PolicyPrincipalAssignmentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	// Validate conditional requirements
	if err := validatePrincipalDirectory(data.PrincipalType.ValueString(), data.SourceDirectoryName.ValueString(), data.SourceDirectoryID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Validation Error", err.Error())
		return
	}

	policyID := data.PolicyID.ValueString()
	principalID := data.PrincipalID.ValueString()
	principalType := data.PrincipalType.ValueString()

	// Read-modify-write: Fetch policy
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "fetch policy for update"))
		return
	}

	// Find and update principal
	found := false
	for i, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			policy.Principals[i] = data.ToSDKPrincipal()
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Principal Not Found",
			fmt.Sprintf("Principal %s (type: %s) not found in policy %s", principalID, principalType, policyID),
		)
		return
	}

	// Update policy with retry
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		_, err := r.providerData.UAPClient.Db().UpdatePolicy(policy)
		return err
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "update principal assignment"))
		return
	}

	// Update state
	data.FromSDKPrincipal(policyID, data.ToSDKPrincipal())

	tflog.Info(ctx, "Updated principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePolicyPrincipalAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.PolicyPrincipalAssignmentModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. "+
				"Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	policyID := data.PolicyID.ValueString()
	principalID := data.PrincipalID.ValueString()
	principalType := data.PrincipalType.ValueString()

	// Read-modify-write: Fetch policy
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})
	if err != nil {
		if client.IsNotFoundError(err) {
			return // Already deleted
		}
		resp.Diagnostics.Append(client.MapError(err, "update principal assignment"))
		return
	}

	// Remove principal
	found := false
	newPrincipals := make([]uapcommonmodels.ArkUAPPrincipal, 0, len(policy.Principals))
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			found = true
			continue // Skip this principal
		}
		newPrincipals = append(newPrincipals, p)
	}

	if !found {
		// Already removed
		return
	}

	policy.Principals = newPrincipals

	// Update policy with retry
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		_, err := r.providerData.UAPClient.Db().UpdatePolicy(policy)
		return err
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "remove principal from policy"))
		return
	}

	tflog.Info(ctx, "Deleted principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})
}

func (r *DatabasePolicyPrincipalAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse composite ID
	policyID, principalID, principalType, err := models.ParseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	// Fetch policy
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "operation"))
		return
	}

	// Find principal
	var found bool
	var data models.PolicyPrincipalAssignmentModel
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			data.FromSDKPrincipal(policyID, p)
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Principal Not Found",
			fmt.Sprintf("Principal %s (type: %s) not found in policy %s", principalID, principalType, policyID),
		)
		return
	}

	tflog.Info(ctx, "Imported principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// validatePrincipalDirectory validates that USER/GROUP have required directory fields
func validatePrincipalDirectory(principalType, directoryName, directoryID string) error {
	if principalType == "USER" || principalType == "GROUP" {
		if directoryName == "" || directoryID == "" {
			return fmt.Errorf("source_directory_name and source_directory_id are required for %s principal type", principalType)
		}
	}
	return nil
}
