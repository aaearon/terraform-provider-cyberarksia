package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/models"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/provider/helpers"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/validators"
	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
	vm "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm"
	uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VMPolicyPrincipalAssignmentResource{}
var _ resource.ResourceWithImportState = &VMPolicyPrincipalAssignmentResource{}

func NewVMPolicyPrincipalAssignmentResource() resource.Resource {
	return &VMPolicyPrincipalAssignmentResource{}
}

// VMPolicyPrincipalAssignmentResource defines the resource implementation.
type VMPolicyPrincipalAssignmentResource struct {
	providerData *ProviderData
}

func (r *VMPolicyPrincipalAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_policy_principal_assignment"
}

func (r *VMPolicyPrincipalAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the assignment of a principal (user/group/role) to a VM access policy. " +
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
				MarkdownDescription: "The ID of the VM access policy. Use `cyberarksia_vm_policy.example.policy_id`.",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_directory_name": schema.StringAttribute{
				MarkdownDescription: "Source identity directory name (max 50 characters). **Required** for USER and GROUP types. Examples: `AzureAD`, `LDAP`, `Okta`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_directory_id": schema.StringAttribute{
				MarkdownDescription: "Source identity directory ID. **Required** for USER and GROUP types.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *VMPolicyPrincipalAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VMPolicyPrincipalAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.VMPolicyPrincipalAssignmentResourceModel

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

	if r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured VMService",
			"VMService was not configured. Please ensure the provider is properly configured.",
		)
		return
	}

	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	// Read-modify-write: Fetch existing policy
	var policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var retryErr error
		policy, retryErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return retryErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "fetch VM policy for principal assignment"))
		return
	}

	// Check for duplicate principal (research.md §4.3 duplicate detection algorithm)
	principalID := data.PrincipalID.ValueString()
	principalType := data.PrincipalType.ValueString()
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			resp.Diagnostics.AddError(
				"Principal Already Assigned",
				fmt.Sprintf("Principal %s (type: %s) is already assigned to VM policy %s", principalID, principalType, policyID),
			)
			return
		}
	}

	// Add new principal
	newPrincipal := uapcommonmodels.ArkUAPPrincipal{
		ID:                  principalID,
		Name:                data.PrincipalName.ValueString(),
		Type:                principalType,
		SourceDirectoryName: data.SourceDirectoryName.ValueString(),
		SourceDirectoryID:   data.SourceDirectoryID.ValueString(),
	}
	policy.Principals = append(policy.Principals, newPrincipal)

	// Update policy with retry
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		_, err := vmService.UpdatePolicy(policy)
		return err
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "assign principal to VM policy"))
		return
	}

	// Build composite ID
	compositeID := helpers.BuildVMPolicyPrincipalID(policyID, principalID, principalType)
	data.ID = types.StringValue(compositeID)

	tflog.Info(ctx, "Created VM policy principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMPolicyPrincipalAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.VMPolicyPrincipalAssignmentResourceModel

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

	if r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured VMService",
			"VMService was not configured. Please ensure the provider is properly configured.",
		)
		return
	}

	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	// Fetch policy with retry
	var policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var retryErr error
		policy, retryErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return retryErr
	})

	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.Append(client.MapError(err, "fetch VM policy for read"))
		return
	}

	// Find principal
	var found bool
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			// Update state with current values from API
			data.PrincipalName = types.StringValue(p.Name)
			// Use StringNull() for optional fields when empty (prevents perpetual replace plans)
			if p.SourceDirectoryName != "" {
				data.SourceDirectoryName = types.StringValue(p.SourceDirectoryName)
			} else {
				data.SourceDirectoryName = types.StringNull()
			}
			if p.SourceDirectoryID != "" {
				data.SourceDirectoryID = types.StringValue(p.SourceDirectoryID)
			} else {
				data.SourceDirectoryID = types.StringNull()
			}
			found = true
			break
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Debug(ctx, "Read VM policy principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is not supported - all attributes are immutable (ForceNew).
// Any change to principal assignment attributes will trigger resource replacement.
// This method exists to satisfy the resource.Resource interface but should never be called
// because all attributes have RequiresReplace plan modifiers.
func (r *VMPolicyPrincipalAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"This resource does not support updates. All attributes are immutable and changes require resource replacement. "+
			"This error should not occur under normal circumstances as all attributes have ForceNew modifiers.",
	)
}

func (r *VMPolicyPrincipalAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.VMPolicyPrincipalAssignmentResourceModel

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

	if r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured VMService",
			"VMService was not configured. Please ensure the provider is properly configured.",
		)
		return
	}

	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	// Read-modify-write: Fetch policy with retry
	var policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var retryErr error
		policy, retryErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return retryErr
	})

	if err != nil {
		if client.IsNotFoundError(err) {
			return // Already deleted
		}
		resp.Diagnostics.Append(client.MapError(err, "fetch VM policy for principal removal"))
		return
	}

	// Check if principal exists in policy
	found := false
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			found = true
			break
		}
	}

	if !found {
		// Already removed - idempotent success
		tflog.Info(ctx, "Principal not found in VM policy - considering delete successful")
		return
	}

	// Remove principal from policy
	newPrincipals := make([]uapcommonmodels.ArkUAPPrincipal, 0, len(policy.Principals))
	for _, p := range policy.Principals {
		if p.ID != principalID || p.Type != principalType {
			newPrincipals = append(newPrincipals, p)
		}
	}

	// Update policy with modified principals (let API validate ≥1 constraint)
	policy.Principals = newPrincipals

	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		_, updateErr := vmService.UpdatePolicy(policy)
		return updateErr
	})

	if err != nil {
		// Handle 404 as success - policy was deleted between check and update (race condition)
		if client.IsNotFoundError(err) {
			tflog.Info(ctx, "VM policy deleted during assignment removal - considering delete successful")
			return
		}

		// Check if API rejected due to constraint violation (must have ≥1 principal)
		errMsg := err.Error()
		if strings.Contains(errMsg, "at least 1 item") || strings.Contains(errMsg, "minimum") || strings.Contains(errMsg, "principals") {
			resp.Diagnostics.AddError(
				"Cannot Remove Last Principal",
				fmt.Sprintf("VM policy %s requires at least one principal assignment. "+
					"This error occurs because removing this assignment would leave the policy with no principals. "+
					"To resolve: either delete the policy itself, or add another principal before removing this one.\n\n"+
					"API Error: %s", policyID, errMsg),
			)
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "update VM policy while removing principal assignment"))
		return
	}

	tflog.Info(ctx, "Successfully removed principal assignment from VM policy", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})
}

func (r *VMPolicyPrincipalAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse composite ID
	policyID, principalID, principalType, err := helpers.ParseVMPolicyPrincipalID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	if r.providerData == nil || r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured VMService",
			"VMService was not configured. Please ensure the provider is properly configured.",
		)
		return
	}

	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	// Fetch policy with retry
	var policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var retryErr error
		policy, retryErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return retryErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "fetch VM policy for import"))
		return
	}

	// Find principal
	var found bool
	var data models.VMPolicyPrincipalAssignmentResourceModel
	for _, p := range policy.Principals {
		if p.ID == principalID && p.Type == principalType {
			data.ID = types.StringValue(req.ID)
			data.PolicyID = types.StringValue(policyID)
			data.PrincipalID = types.StringValue(principalID)
			data.PrincipalName = types.StringValue(p.Name)
			data.PrincipalType = types.StringValue(principalType)
			// Use StringNull() for optional fields when empty (prevents perpetual replace plans)
			if p.SourceDirectoryName != "" {
				data.SourceDirectoryName = types.StringValue(p.SourceDirectoryName)
			} else {
				data.SourceDirectoryName = types.StringNull()
			}
			if p.SourceDirectoryID != "" {
				data.SourceDirectoryID = types.StringValue(p.SourceDirectoryID)
			} else {
				data.SourceDirectoryID = types.StringNull()
			}
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Principal Not Found",
			fmt.Sprintf("Principal %s (type: %s) not found in VM policy %s", principalID, principalType, policyID),
		)
		return
	}

	tflog.Info(ctx, "Imported VM policy principal assignment", map[string]interface{}{
		"policy_id":      policyID,
		"principal_id":   principalID,
		"principal_type": principalType,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
