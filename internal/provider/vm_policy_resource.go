package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/models"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/validators"
	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
	uapcommondels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/common/models"
	vm "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm"
	uapsiavmmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/vm/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VMPolicyResource{}
var _ resource.ResourceWithImportState = &VMPolicyResource{}
var _ resource.ResourceWithValidateConfig = &VMPolicyResource{}
var _ resource.ResourceWithModifyPlan = &VMPolicyResource{}

func NewVMPolicyResource() resource.Resource {
	return &VMPolicyResource{}
}

// VMPolicyResource defines the resource implementation.
type VMPolicyResource struct {
	providerData *ProviderData
}

func (r *VMPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_policy"
}

func (r *VMPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CyberArk SIA virtual machine access policy. " +
			"Defines WHO can access (principals), WHAT they access (targets), " +
			"WHEN they can access (conditions), and HOW they connect (behavior).\n\n" +
			"**Required**: At least one principal MUST be assigned at policy creation. " +
			"Additional principals can be added via `cyberarksia_vm_policy_principal_assignment` resource.\n\n" +
			"**Constraint**: Exactly ONE location type per policy (FQDN/IP, AWS, Azure, or GCP).\n\n" +
			"**User Story**: Basic FQDN/IP policies with SSH behavior (MVP foundation).",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Policy identifier (same as policy_id).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy_id": schema.StringAttribute{
				MarkdownDescription: "Unique policy identifier (UUID, API-generated).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Policy name (1-200 characters, unique). **ForceNew**: Changing creates new policy.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Policy description (max 200 characters).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Policy status. Valid values: `Active`, `Suspended`. " +
					"**Note**: `Expired`, `Validating`, `Error`, and `Warning` are server-managed statuses.",
				Required: true,
				Validators: []validator.String{
					validators.PolicyStatus(),
				},
			},
			"location_type": schema.StringAttribute{
				MarkdownDescription: "Location type. Valid: `AWS`, `Azure`, `GCP`, `FQDN/IP`. **ForceNew**: Changing requires new policy.",
				Required:            true,
				Validators: []validator.String{
					validators.VMLocationType(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"time_zone": schema.StringAttribute{
				MarkdownDescription: "Timezone for access conditions (max 50 characters). " +
					"Supports IANA names (e.g., `America/New_York`) or GMT offsets (e.g., `GMT+05:00`). Default: `GMT`.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("GMT"),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(50),
				},
			},
			"tags": schema.ListAttribute{
				MarkdownDescription: "List of tags for policy organization (max 20 tags).",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(20),
				},
			},
			"policy_type": schema.StringAttribute{
				MarkdownDescription: "Policy type. Valid values: `Recurring` (default), `OnDemand`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Recurring"),
				Validators: []validator.String{
					stringvalidator.OneOf("Recurring", "OnDemand"),
				},
			},
			"max_session_duration": schema.Int64Attribute{
				MarkdownDescription: "Maximum session duration in hours (1-24). Default: 1.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 24),
				},
			},
			"idle_time": schema.Int64Attribute{
				MarkdownDescription: "Session idle timeout in minutes (1-120). Default: 10.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(10),
				Validators: []validator.Int64{
					int64validator.Between(1, 120),
				},
			},
			"delegation_classification": schema.StringAttribute{
				MarkdownDescription: "Delegation classification (server-computed).",
				Computed:            true,
			},
			"created_by": schema.SingleNestedAttribute{
				MarkdownDescription: "Policy creator metadata (computed).",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"user": schema.StringAttribute{
						MarkdownDescription: "Creator username.",
						Computed:            true,
					},
					"timestamp": schema.StringAttribute{
						MarkdownDescription: "Creation timestamp.",
						Computed:            true,
					},
				},
			},
			"updated_by": schema.SingleNestedAttribute{
				MarkdownDescription: "Policy updater metadata (computed).",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"user": schema.StringAttribute{
						MarkdownDescription: "Last updater username.",
						Computed:            true,
					},
					"timestamp": schema.StringAttribute{
						MarkdownDescription: "Last update timestamp.",
						Computed:            true,
					},
				},
			},
		},

		Blocks: map[string]schema.Block{
			"principals": schema.ListNestedBlock{
				MarkdownDescription: "Principal assignment (repeatable block). **Required**: At least 1 principal.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"principal_id": schema.StringAttribute{
							MarkdownDescription: "Principal identifier (UUID format).",
							Required:            true,
							Validators: []validator.String{
								validators.UUID(),
							},
						},
						"principal_name": schema.StringAttribute{
							MarkdownDescription: "Principal name (1-512 chars).",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 512),
							},
						},
						"principal_type": schema.StringAttribute{
							MarkdownDescription: "Principal type. Valid: `USER`, `GROUP`, `ROLE`.",
							Required:            true,
							Validators: []validator.String{
								validators.PrincipalType(),
							},
						},
						"source_directory_name": schema.StringAttribute{
							MarkdownDescription: "Source directory name (max 50 chars). **Required** for USER/GROUP types.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(50),
							},
						},
						"source_directory_id": schema.StringAttribute{
							MarkdownDescription: "Source directory ID. **Required** for USER/GROUP types.",
							Optional:            true,
						},
					},
				},
			},

			"access_window": schema.SingleNestedBlock{
				MarkdownDescription: "Time-based access restrictions (days and hours). Optional.",
				Attributes: map[string]schema.Attribute{
					"days_of_the_week": schema.SetAttribute{
						MarkdownDescription: "Days access is allowed (0=Sunday through 6=Saturday). Specify days in any order - order is automatically normalized. Example: `[1, 2, 3, 4, 5]` for weekdays.",
						Required:            true,
						ElementType:         types.Int64Type,
						Validators: []validator.Set{
							setvalidator.ValueInt64sAre(int64validator.Between(0, 6)),
							setvalidator.SizeBetween(1, 7),
						},
					},
					"from_hour": schema.StringAttribute{
						MarkdownDescription: "Start time in HH:MM format (e.g., `09:00`).",
						Optional:            true,
					},
					"to_hour": schema.StringAttribute{
						MarkdownDescription: "End time in HH:MM format (e.g., `17:00`).",
						Optional:            true,
					},
				},
			},

			"time_frame": schema.SingleNestedBlock{
				MarkdownDescription: "Policy validity period. Optional - if not specified, policy never expires.",
				Attributes: map[string]schema.Attribute{
					"from_time": schema.StringAttribute{
						MarkdownDescription: "Start time (ISO 8601 format, e.g., `2024-01-01T00:00:00Z`).",
						Optional:            true,
					},
					"to_time": schema.StringAttribute{
						MarkdownDescription: "End time (ISO 8601 format, e.g., `2024-12-31T23:59:59Z`).",
						Optional:            true,
					},
				},
			},

			"behavior": schema.SingleNestedBlock{
				MarkdownDescription: "Connection behavior (SSH/RDP profiles). **Required**: At least one profile.",
				Blocks: map[string]schema.Block{
					"ssh": schema.SingleNestedBlock{
						MarkdownDescription: "SSH connection profile.",
						Attributes: map[string]schema.Attribute{
							"username": schema.StringAttribute{
								MarkdownDescription: "SSH username. Required if ssh block present.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.LengthAtLeast(1),
								},
							},
						},
					},
					"rdp": schema.SingleNestedBlock{
						MarkdownDescription: "RDP connection profile.",
						Blocks: map[string]schema.Block{
							"local_ephemeral_user": schema.SingleNestedBlock{
								MarkdownDescription: "Local Windows ephemeral user configuration.",
								Attributes: map[string]schema.Attribute{
									"assign_groups": schema.ListAttribute{
										MarkdownDescription: "Local Windows groups to assign.",
										Optional:            true,
										ElementType:         types.StringType,
									},
									"enable_ephemeral_user_reconnect": schema.BoolAttribute{
										MarkdownDescription: "Enable reconnection to same ephemeral user.",
										Optional:            true,
									},
								},
							},
							"domain_ephemeral_user": schema.SingleNestedBlock{
								MarkdownDescription: "Domain-joined ephemeral user configuration. **Note**: SDK-supported, OpenAPI undocumented.",
								Attributes: map[string]schema.Attribute{
									"assign_groups": schema.ListAttribute{
										MarkdownDescription: "Local Windows groups to assign.",
										Optional:            true,
										ElementType:         types.StringType,
									},
									"assign_domain_groups": schema.ListAttribute{
										MarkdownDescription: "Domain groups to assign.",
										Optional:            true,
										ElementType:         types.StringType,
									},
									"enable_ephemeral_user_reconnect": schema.BoolAttribute{
										MarkdownDescription: "Enable reconnection to same ephemeral user.",
										Optional:            true,
									},
								},
							},
						},
					},
				},
			},

			"fqdn_ip_targets": schema.SingleNestedBlock{
				MarkdownDescription: "FQDN/IP target rules. Use when location_type = `FQDN/IP`.",
				Blocks: map[string]schema.Block{
					"fqdn_rule": schema.ListNestedBlock{
						MarkdownDescription: "FQDN matching rules.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"operator": schema.StringAttribute{
									MarkdownDescription: "FQDN operator. Valid: `EXACTLY`, `WILDCARD`, `PREFIX`, `SUFFIX`, `CONTAINS`.",
									Required:            true,
									Validators: []validator.String{
										validators.FQDNOperator(),
									},
								},
								"computername_pattern": schema.StringAttribute{
									MarkdownDescription: "Computername pattern (max 300 chars).",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.LengthAtMost(300),
									},
								},
								"domain": schema.StringAttribute{
									MarkdownDescription: "Domain name (max 1000 chars). Optional.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.LengthAtMost(1000),
									},
								},
							},
						},
					},
					"ip_rule": schema.ListNestedBlock{
						MarkdownDescription: "IP address matching rules.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"operator": schema.StringAttribute{
									MarkdownDescription: "IP operator. Valid: `EXACTLY`, `WILDCARD`.",
									Required:            true,
									Validators: []validator.String{
										validators.IPOperator(),
									},
								},
								"ip_addresses": schema.ListAttribute{
									MarkdownDescription: "IP addresses (max 1000 items).",
									Required:            true,
									ElementType:         types.StringType,
									Validators: []validator.List{
										listvalidator.SizeAtMost(1000),
									},
								},
								"logical_name": schema.StringAttribute{
									MarkdownDescription: "Logical name for IP rule (1-256 chars).",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 256),
									},
								},
							},
						},
					},
				},
			},

			"aws_targets": schema.SingleNestedBlock{
				MarkdownDescription: "AWS cloud target criteria. Mutually exclusive with fqdn_ip_targets, azure_targets, and gcp_targets.",
				Blocks: map[string]schema.Block{
					"tags": schema.ListNestedBlock{
						MarkdownDescription: "AWS resource tags for target matching.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Tag key.",
								},
								"value": schema.ListAttribute{
									ElementType:         types.StringType,
									Optional:            true,
									MarkdownDescription: "Tag values (optional, empty means any value).",
								},
							},
						},
					},
				},
				Attributes: map[string]schema.Attribute{
					"regions": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "AWS regions (e.g., us-east-1, eu-west-1). Empty means all regions.",
					},
					"vpc_ids": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "VPC IDs for target matching. Empty means all VPCs.",
					},
					"account_ids": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "AWS account IDs. Empty means all accounts.",
					},
				},
			},

			"azure_targets": schema.SingleNestedBlock{
				MarkdownDescription: "Azure cloud target criteria. Mutually exclusive with fqdn_ip_targets, aws_targets, and gcp_targets.",
				Blocks: map[string]schema.Block{
					"tags": schema.ListNestedBlock{
						MarkdownDescription: "Azure resource tags for target matching.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Tag key.",
								},
								"value": schema.ListAttribute{
									ElementType:         types.StringType,
									Optional:            true,
									MarkdownDescription: "Tag values (optional, empty means any value).",
								},
							},
						},
					},
				},
				Attributes: map[string]schema.Attribute{
					"regions": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "Azure regions (e.g., eastus, westeurope). Empty means all regions.",
					},
					"resource_groups": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "Azure resource groups. Empty means all resource groups.",
					},
					"vnet_ids": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "Virtual Network IDs. Empty means all VNets.",
					},
					"subscriptions": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "Azure subscription IDs. Empty means all subscriptions.",
					},
				},
			},

			"gcp_targets": schema.SingleNestedBlock{
				MarkdownDescription: "GCP cloud target criteria. Mutually exclusive with fqdn_ip_targets, aws_targets, and azure_targets. **Note**: GCP uses 'labels' not 'tags'.",
				Blocks: map[string]schema.Block{
					"labels": schema.ListNestedBlock{
						MarkdownDescription: "GCP resource labels for target matching.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Required:            true,
									MarkdownDescription: "Label key.",
								},
								"value": schema.ListAttribute{
									ElementType:         types.StringType,
									Optional:            true,
									MarkdownDescription: "Label values (optional, empty means any value).",
								},
							},
						},
					},
				},
				Attributes: map[string]schema.Attribute{
					"regions": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "GCP regions (e.g., us-central1, europe-west1). Empty means all regions.",
					},
					"vpc_ids": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "VPC network IDs. Empty means all VPCs.",
					},
					"projects": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "GCP project IDs. Empty means all projects.",
					},
				},
			},
		},
	}
}

func (r *VMPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VMPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config models.VMPolicyResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate exactly ONE location type and that it matches location_type attribute
	locationTypeCount := 0
	var configuredTargetType string

	if !config.FQDNIPTargets.IsNull() {
		locationTypeCount++
		configuredTargetType = "FQDN/IP"
	}
	if !config.AWSTargets.IsNull() {
		locationTypeCount++
		configuredTargetType = "AWS"
	}
	if !config.AzureTargets.IsNull() {
		locationTypeCount++
		configuredTargetType = "Azure"
	}
	if !config.GCPTargets.IsNull() {
		locationTypeCount++
		configuredTargetType = "GCP"
	}

	if locationTypeCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid Location Type Configuration",
			"Exactly one location type must be specified: fqdn_ip_targets, aws_targets, azure_targets, or gcp_targets",
		)
	}

	// Verify location_type attribute matches the configured targets block
	if locationTypeCount == 1 && !config.LocationType.IsNull() {
		declaredType := config.LocationType.ValueString()
		if declaredType != configuredTargetType {
			resp.Diagnostics.AddError(
				"Location Type Mismatch",
				fmt.Sprintf("The location_type attribute is set to %q but %s targets are configured. "+
					"Please ensure location_type matches the configured targets block.",
					declaredType, configuredTargetType),
			)
		}
	}

	// Validate at least one connection profile (SSH or RDP)
	var behavior models.BehaviorModel
	diags := config.Behavior.As(ctx, &behavior, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diags...)

	if !resp.Diagnostics.HasError() {
		hasSSH := !behavior.SSH.IsNull()
		hasRDP := !behavior.RDP.IsNull()

		if !hasSSH && !hasRDP {
			resp.Diagnostics.AddError(
				"Invalid Behavior Configuration",
				"At least one connection profile (ssh or rdp) must be configured in behavior block",
			)
		}
	}

	// Validate conditional source directory fields for principals
	if !config.Principals.IsNull() {
		var principals []models.PrincipalModel
		diags := config.Principals.ElementsAs(ctx, &principals, false)
		resp.Diagnostics.Append(diags...)

		if !resp.Diagnostics.HasError() {
			for i, p := range principals {
				principalType := p.PrincipalType.ValueString()
				if principalType == "USER" || principalType == "GROUP" {
					if p.SourceDirectoryName.IsNull() || p.SourceDirectoryName.ValueString() == "" {
						resp.Diagnostics.AddAttributeError(
							path.Root("principals").AtListIndex(i).AtName("source_directory_name"),
							"Missing Required Field",
							fmt.Sprintf("source_directory_name is required for %s principals", principalType),
						)
					}
					if p.SourceDirectoryID.IsNull() || p.SourceDirectoryID.ValueString() == "" {
						resp.Diagnostics.AddAttributeError(
							path.Root("principals").AtListIndex(i).AtName("source_directory_id"),
							"Missing Required Field",
							fmt.Sprintf("source_directory_id is required for %s principals", principalType),
						)
					}
				}
			}
		}
	}

	// Validate access_window: if from_hour or to_hour is set, both must be set
	if !config.AccessWindow.IsNull() {
		var accessWindow models.VMAccessWindowModel
		diags := config.AccessWindow.As(ctx, &accessWindow, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)

		if !resp.Diagnostics.HasError() {
			fromHourSet := !accessWindow.FromHour.IsNull() && !accessWindow.FromHour.IsUnknown()
			toHourSet := !accessWindow.ToHour.IsNull() && !accessWindow.ToHour.IsUnknown()

			if fromHourSet != toHourSet {
				resp.Diagnostics.AddAttributeError(
					path.Root("access_window"),
					"Invalid Access Window Configuration",
					"Both from_hour and to_hour must be specified together, or both must be omitted. "+
						"If you want to restrict access to specific hours, provide both fields. "+
						"If you only want to restrict by days, omit both fields.",
				)
			}
		}
	}
}

func (r *VMPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.VMPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil || r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	// Type assert VMService to *vm.ArkUAPSIAVMService
	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	// Build SDK policy model
	policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{}

	// Set delegation classification (required, server validates)
	policy.DelegationClassification = "Unrestricted"

	// Build metadata
	metadata := buildSDKMetadata(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	policy.Metadata = metadata

	// Build principals
	principals := buildSDKPrincipals(ctx, plan.Principals, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	policy.Principals = principals

	// Build targets
	targets := buildSDKTargets(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	policy.Targets = targets

	// Build behavior
	behavior := buildSDKBehavior(ctx, plan.Behavior, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	policy.Behavior = behavior

	// Build conditions
	policy.Conditions = buildSDKConditions(plan)

	// Create policy with retry logic
	var created *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var createErr error
		created, createErr = vmService.AddPolicy(policy)
		return createErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "create VM policy"))
		return
	}

	// Set only the ID fields from API response
	// Don't call mapSDKPolicyToState() here - it tries to populate computed metadata fields
	// (created_by, updated_by) which causes "unknown value" errors during CREATE.
	// Terraform will automatically call Read() after Create() to populate all fields.
	plan.ID = types.StringValue(created.Metadata.PolicyID)
	plan.PolicyID = types.StringValue(created.Metadata.PolicyID)

	// Set computed fields that must be known after apply
	plan.DelegationClassification = types.StringValue(created.DelegationClassification)

	// Set tags (Optional+Computed) - use empty list if not provided
	if len(created.Metadata.PolicyTags) > 0 {
		tagsList, diagsTags := types.ListValueFrom(ctx, types.StringType, created.Metadata.PolicyTags)
		if diagsTags.HasError() {
			resp.Diagnostics.Append(diagsTags...)
		} else {
			plan.Tags = tagsList
		}
	} else {
		plan.Tags = types.ListNull(types.StringType)
	}

	// Explicitly set computed metadata fields to null to avoid "unknown value" errors
	// These will be populated by the automatic Read() call after Create()
	plan.CreatedBy = types.ObjectNull(map[string]attr.Type{
		"user":      types.StringType,
		"timestamp": types.StringType,
	})
	plan.UpdatedBy = types.ObjectNull(map[string]attr.Type{
		"user":      types.StringType,
		"timestamp": types.StringType,
	})

	tflog.Info(ctx, "Created VM policy", map[string]interface{}{
		"policy_id":   plan.PolicyID.ValueString(),
		"policy_name": plan.Name.ValueString(),
		"principals":  len(plan.Principals.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *VMPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.VMPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil || r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	// Type assert VMService to *vm.ArkUAPSIAVMService
	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	policyID := state.PolicyID.ValueString()

	// Fetch policy from API with retry logic
	var policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var fetchErr error
		policy, fetchErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return fetchErr
	})

	if err != nil {
		// Drift detection: 404 = policy deleted externally
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "read VM policy"))
		return
	}

	// Map SDK response to state
	newState := mapSDKPolicyToState(ctx, policy, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Read VM policy", map[string]interface{}{
		"policy_id": newState.PolicyID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *VMPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state models.VMPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil || r.providerData.VMService == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	// Type assert VMService to *vm.ArkUAPSIAVMService
	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	policyID := state.PolicyID.ValueString()

	// Step 1: READ existing policy from API with retry logic
	var existingPolicy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var fetchErr error
		existingPolicy, fetchErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return fetchErr
	})
	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "read VM policy for update"))
		return
	}

	// Step 2: IDENTIFY inline principals from plan
	inlinePrincipalKeys := make(map[string]bool)
	var planPrincipals []models.PrincipalModel
	if !plan.Principals.IsNull() {
		diags := plan.Principals.ElementsAs(ctx, &planPrincipals, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, p := range planPrincipals {
			key := fmt.Sprintf("%s:%s", p.PrincipalID.ValueString(), p.PrincipalType.ValueString())
			inlinePrincipalKeys[key] = true
		}
	}

	// Step 3: PRESERVE assigned principals (not in inline config)
	preservedPrincipals := []uapcommonmodels.ArkUAPPrincipal{}
	for _, p := range existingPolicy.Principals {
		key := fmt.Sprintf("%s:%s", p.ID, p.Type)
		if !inlinePrincipalKeys[key] {
			// This principal was added via assignment resource - preserve it
			preservedPrincipals = append(preservedPrincipals, p)
		}
	}

	// Step 4: BUILD new principals: inline from plan + preserved assigned
	newPrincipals := buildSDKPrincipals(ctx, plan.Principals, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	newPrincipals = append(newPrincipals, preservedPrincipals...)

	// Step 5: UPDATE other fields from plan
	metadata := buildSDKMetadata(plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve PolicyID from existing
	metadata.PolicyID = existingPolicy.Metadata.PolicyID
	existingPolicy.Metadata = metadata

	existingPolicy.Principals = newPrincipals

	// Update targets
	targets := buildSDKTargets(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	existingPolicy.Targets = targets

	// Update behavior
	behavior := buildSDKBehavior(ctx, plan.Behavior, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	existingPolicy.Behavior = behavior

	// Update conditions
	existingPolicy.Conditions = buildSDKConditions(plan)

	// Step 6: WRITE back to API
	var updated *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err = client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var updateErr error
		updated, updateErr = vmService.UpdatePolicy(existingPolicy)
		return updateErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "update VM policy"))
		return
	}

	// Map updated policy to state
	newState := mapSDKPolicyToState(ctx, updated, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updated VM policy", map[string]interface{}{
		"policy_id":   newState.PolicyID.ValueString(),
		"policy_name": newState.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// ModifyPlan implements plan-time validation with API awareness to prevent removing
// the last principal from a policy. This method queries the API to count both inline
// and externally-managed principals (via cyberarksia_vm_policy_principal_assignment),
// ensuring accurate validation without false positives.
func (r *VMPolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip if resource being destroyed (plan is null)
	if req.Plan.Raw.IsNull() {
		return
	}

	// Skip if resource being created (state is null)
	if req.State.Raw.IsNull() {
		return
	}

	// Skip if plan not fully known (computed values, data source failures)
	if !req.Plan.Raw.IsFullyKnown() {
		return
	}

	// Get plan and state models
	var plan, state models.VMPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only validate if inline principals are being REDUCED
	var planPrincipals []models.PrincipalModel
	var statePrincipals []models.PrincipalModel
	plan.Principals.ElementsAs(ctx, &planPrincipals, false)
	state.Principals.ElementsAs(ctx, &statePrincipals, false)

	principalsReducing := len(planPrincipals) < len(statePrincipals)

	if !principalsReducing {
		return // No risk of constraint violation
	}

	// Ensure provider is configured
	if r.providerData == nil || r.providerData.VMService == nil {
		tflog.Warn(ctx, "ModifyPlan: Provider not configured, skipping validation")
		return
	}

	// Type assert VMService
	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		tflog.Warn(ctx, "ModifyPlan: Invalid VMService type, skipping validation")
		return
	}

	// Fetch actual policy from API to get TOTAL principal count (inline + external)
	policyID := state.PolicyID.ValueString()

	tflog.Debug(ctx, "ModifyPlan: Fetching policy to validate principal constraints", map[string]interface{}{
		"policy_id":           policyID,
		"principals_reducing": principalsReducing,
	})

	actualPolicy, err := vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})
	if err != nil {
		// If can't fetch policy, skip validation (Update will handle it)
		tflog.Warn(ctx, "ModifyPlan: Could not fetch policy for validation, skipping", map[string]interface{}{
			"policy_id": policyID,
			"error":     err.Error(),
		})
		return
	}

	// Validate principals constraint
	totalPrincipals := len(actualPolicy.Principals)
	inlinePrincipals := len(statePrincipals)
	externalPrincipals := totalPrincipals - inlinePrincipals
	principalsAfterChange := len(planPrincipals) + externalPrincipals

	tflog.Debug(ctx, "ModifyPlan: Principal count analysis", map[string]interface{}{
		"total_principals":    totalPrincipals,
		"inline_principals":   inlinePrincipals,
		"external_principals": externalPrincipals,
		"principals_after":    principalsAfterChange,
	})

	if principalsAfterChange < 1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("principals"),
			"Cannot Remove Last Principal",
			fmt.Sprintf(
				"This change would remove the last principal from the policy.\n\n"+
					"Current state:\n"+
					"  • Inline principals (in this resource): %d\n"+
					"  • External principals (via cyberarksia_vm_policy_principal_assignment): %d\n"+
					"  • Total principals: %d\n\n"+
					"After this change: %d principal(s) remaining\n\n"+
					"CyberArk SIA policies require at least 1 principal.\n\n"+
					"To resolve:\n"+
					"  1. Add another principal via cyberarksia_vm_policy_principal_assignment first\n"+
					"  2. Delete the entire policy resource instead: terraform destroy <resource_name>",
				inlinePrincipals, externalPrincipals, totalPrincipals, principalsAfterChange,
			),
		)
	}
}

func (r *VMPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.VMPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.providerData == nil || r.providerData.AuthContext == nil {
		resp.Diagnostics.AddError(
			"Unconfigured Provider",
			"Provider was not configured. Please ensure provider configuration is complete before using resources.",
		)
		return
	}

	policyID := state.PolicyID.ValueString()

	// Delete policy using workaround (ARK SDK v1.5.0 nil body bug)
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		return client.DeleteDatabasePolicyDirect(ctx, r.providerData.AuthContext, policyID)
	})

	if err != nil {
		// 404 = already deleted (drift detection) - treat as success
		if client.IsNotFoundError(err) {
			tflog.Info(ctx, "Policy already deleted", map[string]interface{}{
				"policy_id": policyID,
			})
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "delete VM policy"))
		return
	}

	tflog.Info(ctx, "Deleted VM policy", map[string]interface{}{
		"policy_id": policyID,
	})
}

func (r *VMPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policyID := req.ID

	// Type assert VMService to *vm.ArkUAPSIAVMService
	vmService, ok := r.providerData.VMService.(*vm.ArkUAPSIAVMService)
	if !ok {
		resp.Diagnostics.AddError(
			"Invalid VMService Type",
			fmt.Sprintf("Expected *vm.ArkUAPSIAVMService, got: %T. Please report this issue to the provider developers.", r.providerData.VMService),
		)
		return
	}

	// Fetch policy from API with retry logic
	var policy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var fetchErr error
		policy, fetchErr = vmService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
			PolicyID: policyID,
		})
		return fetchErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "import VM policy"))
		return
	}

	// Convert to state model
	state := mapSDKPolicyToState(ctx, policy, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Imported VM policy", map[string]interface{}{
		"policy_id":   state.PolicyID.ValueString(),
		"policy_name": state.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// SDK Helper Functions

// buildSDKMetadata creates SDK metadata from Terraform plan
func buildSDKMetadata(plan models.VMPolicyResourceModel, diags *diag.Diagnostics) uapcommonmodels.ArkUAPMetadata {
	metadata := uapcommonmodels.ArkUAPMetadata{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		TimeZone:    plan.TimeZone.ValueString(),
		PolicyEntitlement: uapcommonmodels.ArkUAPPolicyEntitlement{
			LocationType:   plan.LocationType.ValueString(),
			PolicyType:     plan.PolicyType.ValueString(),
			TargetCategory: "VM",
		},
		Status: uapcommonmodels.ArkUAPPolicyStatus{
			Status: plan.Status.ValueString(),
		},
	}

	// Add policy tags if provided
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		var tags []string
		tagsDiags := plan.Tags.ElementsAs(context.Background(), &tags, false)
		diags.Append(tagsDiags...)
		if !diags.HasError() {
			metadata.PolicyTags = tags
		}
	}

	// Add time frame if provided
	if !plan.TimeFrame.IsNull() && !plan.TimeFrame.IsUnknown() {
		var timeFrame models.TimeFrameModel
		tfDiags := plan.TimeFrame.As(context.Background(), &timeFrame, basetypes.ObjectAsOptions{})
		diags.Append(tfDiags...)
		if !diags.HasError() {
			metadata.TimeFrame = uapcommonmodels.ArkUAPTimeFrame{
				FromTime: timeFrame.FromTime.ValueString(),
				ToTime:   timeFrame.ToTime.ValueString(),
			}
		}
	}

	return metadata
}

// buildSDKPrincipals converts Terraform principals to SDK principals
func buildSDKPrincipals(ctx context.Context, tfPrincipals types.List, diags *diag.Diagnostics) []uapcommonmodels.ArkUAPPrincipal {
	if tfPrincipals.IsNull() || tfPrincipals.IsUnknown() {
		return []uapcommonmodels.ArkUAPPrincipal{}
	}

	var principalModels []models.PrincipalModel
	elemDiags := tfPrincipals.ElementsAs(ctx, &principalModels, false)
	diags.Append(elemDiags...)
	if diags.HasError() {
		return []uapcommonmodels.ArkUAPPrincipal{}
	}

	sdkPrincipals := make([]uapcommonmodels.ArkUAPPrincipal, len(principalModels))
	for i, p := range principalModels {
		sdkPrincipals[i] = uapcommonmodels.ArkUAPPrincipal{
			ID:                  p.PrincipalID.ValueString(),
			Name:                p.PrincipalName.ValueString(),
			Type:                p.PrincipalType.ValueString(),
			SourceDirectoryName: p.SourceDirectoryName.ValueString(),
			SourceDirectoryID:   p.SourceDirectoryID.ValueString(),
		}
	}

	return sdkPrincipals
}

// buildSDKBehavior converts Terraform behavior to SDK behavior
func buildSDKBehavior(ctx context.Context, tfBehavior types.Object, diags *diag.Diagnostics) uapsiavmmodels.ArkUAPSSIAVMBehavior {
	behavior := uapsiavmmodels.ArkUAPSSIAVMBehavior{}

	if tfBehavior.IsNull() || tfBehavior.IsUnknown() {
		return behavior
	}

	var behaviorModel models.BehaviorModel
	behaviorDiags := tfBehavior.As(ctx, &behaviorModel, basetypes.ObjectAsOptions{})
	diags.Append(behaviorDiags...)
	if diags.HasError() {
		return behavior
	}

	// Build SSH profile if present
	if !behaviorModel.SSH.IsNull() {
		var ssh models.SSHProfileModel
		sshDiags := behaviorModel.SSH.As(ctx, &ssh, basetypes.ObjectAsOptions{})
		diags.Append(sshDiags...)
		if !diags.HasError() {
			behavior.SSHProfile = &uapsiavmmodels.ArkUAPSSIAVMSSHProfile{
				Username: ssh.Username.ValueString(),
			}
		}
	}

	// Build RDP profile if present
	if !behaviorModel.RDP.IsNull() {
		var rdp models.RDPProfileModel
		rdpDiags := behaviorModel.RDP.As(ctx, &rdp, basetypes.ObjectAsOptions{})
		diags.Append(rdpDiags...)
		if !diags.HasError() {
			behavior.RDPProfile = &uapsiavmmodels.ArkUAPSSIAVMRDPProfile{}

			// Build local ephemeral user if present
			if !rdp.LocalEphemeralUser.IsNull() {
				var localUser models.LocalEphemeralUserModel
				localUserDiags := rdp.LocalEphemeralUser.As(ctx, &localUser, basetypes.ObjectAsOptions{})
				diags.Append(localUserDiags...)

				var assignGroups []string
				if !localUser.AssignGroups.IsNull() {
					groupsDiags := localUser.AssignGroups.ElementsAs(ctx, &assignGroups, false)
					diags.Append(groupsDiags...)
				}

				if !diags.HasError() {
					behavior.RDPProfile.LocalEphemeralUser = &uapsiavmmodels.ArkUAPSSIAVMEphemeralUser{
						AssignGroups:                 assignGroups,
						EnableEphemeralUserReconnect: localUser.EnableEphemeralUserReconnect.ValueBool(),
					}
				}
			}

			// Build domain ephemeral user if present
			if !rdp.DomainEphemeralUser.IsNull() {
				var domainUser models.DomainEphemeralUserModel
				domainUserDiags := rdp.DomainEphemeralUser.As(ctx, &domainUser, basetypes.ObjectAsOptions{})
				diags.Append(domainUserDiags...)

				var assignGroups []string
				if !domainUser.AssignGroups.IsNull() {
					groupsDiags := domainUser.AssignGroups.ElementsAs(ctx, &assignGroups, false)
					diags.Append(groupsDiags...)
				}

				var assignDomainGroups []string
				if !domainUser.AssignDomainGroups.IsNull() {
					domainGroupsDiags := domainUser.AssignDomainGroups.ElementsAs(ctx, &assignDomainGroups, false)
					diags.Append(domainGroupsDiags...)
				}

				if !diags.HasError() {
					behavior.RDPProfile.DomainEphemeralUser = &uapsiavmmodels.ArkUAPSSIAVMDomainEphemeralUser{
						ArkUAPSSIAVMEphemeralUser: uapsiavmmodels.ArkUAPSSIAVMEphemeralUser{
							AssignGroups:                 assignGroups,
							EnableEphemeralUserReconnect: domainUser.EnableEphemeralUserReconnect.ValueBool(),
						},
						AssignDomainGroups: assignDomainGroups,
					}
				}
			}
		}
	}

	return behavior
}

// buildSDKTargets converts Terraform targets to SDK targets (FQDN/IP only for User Story 1)
func buildSDKTargets(ctx context.Context, plan models.VMPolicyResourceModel, diags *diag.Diagnostics) uapsiavmmodels.ArkUAPSIAVMPlatformTargets {
	targets := uapsiavmmodels.ArkUAPSIAVMPlatformTargets{}

	// Build FQDN/IP targets if present
	if !plan.FQDNIPTargets.IsNull() {
		var fqdnIPTargets models.FQDNIPTargetsModel
		fqdnDiags := plan.FQDNIPTargets.As(ctx, &fqdnIPTargets, basetypes.ObjectAsOptions{})
		diags.Append(fqdnDiags...)
		if diags.HasError() {
			return targets
		}

		resource := &uapsiavmmodels.ArkUAPSIAVMFQDNIPResource{}

		// Build FQDN rules
		if !fqdnIPTargets.FQDNRules.IsNull() {
			var fqdnRules []models.FQDNRuleModel
			fqdnRulesDiags := fqdnIPTargets.FQDNRules.ElementsAs(ctx, &fqdnRules, false)
			diags.Append(fqdnRulesDiags...)
			if !diags.HasError() {
				resource.FQDNRules = make([]uapsiavmmodels.ArkUAPSIAVMFQDNRule, len(fqdnRules))
				for i, rule := range fqdnRules {
					resource.FQDNRules[i] = uapsiavmmodels.ArkUAPSIAVMFQDNRule{
						Operator:            rule.Operator.ValueString(),
						ComputernamePattern: rule.ComputernamePattern.ValueString(),
						Domain:              rule.Domain.ValueString(),
					}
				}
			}
		}

		// Build IP rules
		if !fqdnIPTargets.IPRules.IsNull() {
			var ipRules []models.IPRuleModel
			ipRulesDiags := fqdnIPTargets.IPRules.ElementsAs(ctx, &ipRules, false)
			diags.Append(ipRulesDiags...)
			if !diags.HasError() {
				resource.IPRules = make([]uapsiavmmodels.ArkUAPSIAVMIPRule, len(ipRules))
				for i, rule := range ipRules {
					var ipAddresses []string
					ipAddressesDiags := rule.IPAddresses.ElementsAs(ctx, &ipAddresses, false)
					diags.Append(ipAddressesDiags...)

					if !diags.HasError() {
						resource.IPRules[i] = uapsiavmmodels.ArkUAPSIAVMIPRule{
							Operator:    rule.Operator.ValueString(),
							IPAddresses: ipAddresses,
							LogicalName: rule.LogicalName.ValueString(),
						}
					}
				}
			}
		}

		targets.FQDNIPResource = resource
	}

	// Build AWS targets if present
	if !plan.AWSTargets.IsNull() {
		var awsTargets models.AWSTargetsModel
		awsDiags := plan.AWSTargets.As(ctx, &awsTargets, basetypes.ObjectAsOptions{})
		diags.Append(awsDiags...)
		if diags.HasError() {
			return targets
		}

		awsResource := &uapsiavmmodels.ArkUAPSIAVMAWSResource{}

		// Regions
		if !awsTargets.Regions.IsNull() {
			var regions []string
			regionsDiags := awsTargets.Regions.ElementsAs(ctx, &regions, false)
			diags.Append(regionsDiags...)
			if !diags.HasError() {
				awsResource.Regions = regions
			}
		}

		// Tags
		if !awsTargets.Tags.IsNull() {
			var tags []models.TagModel
			tagsDiags := awsTargets.Tags.ElementsAs(ctx, &tags, false)
			diags.Append(tagsDiags...)

			if !diags.HasError() {
				sdkTags := make([]uapsiavmmodels.ArkUAPSIAVMKeyValTag, len(tags))
				for i, tag := range tags {
					sdkTags[i] = uapsiavmmodels.ArkUAPSIAVMKeyValTag{
						Key: tag.Key.ValueString(),
					}
					if !tag.Value.IsNull() {
						var values []string
						valuesDiags := tag.Value.ElementsAs(ctx, &values, false)
						diags.Append(valuesDiags...)
						if !diags.HasError() {
							sdkTags[i].Value = values
						}
					}
				}
				awsResource.Tags = sdkTags
			}
		}

		// VPC IDs
		if !awsTargets.VPCIDs.IsNull() {
			var vpcIDs []string
			vpcIDsDiags := awsTargets.VPCIDs.ElementsAs(ctx, &vpcIDs, false)
			diags.Append(vpcIDsDiags...)
			if !diags.HasError() {
				awsResource.VPCIDs = vpcIDs
			}
		}

		// Account IDs
		if !awsTargets.AccountIDs.IsNull() {
			var accountIDs []string
			accountIDsDiags := awsTargets.AccountIDs.ElementsAs(ctx, &accountIDs, false)
			diags.Append(accountIDsDiags...)
			if !diags.HasError() {
				awsResource.AccountIDs = accountIDs
			}
		}

		targets.AWSResource = awsResource
	}

	// Build Azure targets if present
	if !plan.AzureTargets.IsNull() {
		var azureTargets models.AzureTargetsModel
		azureDiags := plan.AzureTargets.As(ctx, &azureTargets, basetypes.ObjectAsOptions{})
		diags.Append(azureDiags...)
		if diags.HasError() {
			return targets
		}

		azureResource := &uapsiavmmodels.ArkUAPSIAVMAzureResource{}

		// Regions
		if !azureTargets.Regions.IsNull() {
			var regions []string
			regionsDiags := azureTargets.Regions.ElementsAs(ctx, &regions, false)
			diags.Append(regionsDiags...)
			if !diags.HasError() {
				azureResource.Regions = regions
			}
		}

		// Tags
		if !azureTargets.Tags.IsNull() {
			var tags []models.TagModel
			tagsDiags := azureTargets.Tags.ElementsAs(ctx, &tags, false)
			diags.Append(tagsDiags...)

			if !diags.HasError() {
				sdkTags := make([]uapsiavmmodels.ArkUAPSIAVMKeyValTag, len(tags))
				for i, tag := range tags {
					sdkTags[i] = uapsiavmmodels.ArkUAPSIAVMKeyValTag{
						Key: tag.Key.ValueString(),
					}
					if !tag.Value.IsNull() {
						var values []string
						valuesDiags := tag.Value.ElementsAs(ctx, &values, false)
						diags.Append(valuesDiags...)
						if !diags.HasError() {
							sdkTags[i].Value = values
						}
					}
				}
				azureResource.Tags = sdkTags
			}
		}

		// Resource Groups
		if !azureTargets.ResourceGroups.IsNull() {
			var resourceGroups []string
			rgDiags := azureTargets.ResourceGroups.ElementsAs(ctx, &resourceGroups, false)
			diags.Append(rgDiags...)
			if !diags.HasError() {
				azureResource.ResourceGroups = resourceGroups
			}
		}

		// VNet IDs
		if !azureTargets.VNetIDs.IsNull() {
			var vnetIDs []string
			vnetIDsDiags := azureTargets.VNetIDs.ElementsAs(ctx, &vnetIDs, false)
			diags.Append(vnetIDsDiags...)
			if !diags.HasError() {
				azureResource.VNetIDs = vnetIDs
			}
		}

		// Subscriptions
		if !azureTargets.Subscriptions.IsNull() {
			var subscriptions []string
			subsDiags := azureTargets.Subscriptions.ElementsAs(ctx, &subscriptions, false)
			diags.Append(subsDiags...)
			if !diags.HasError() {
				azureResource.Subscriptions = subscriptions
			}
		}

		targets.AzureResource = azureResource
	}

	// Build GCP targets if present (NOTE: uses Labels not Tags!)
	if !plan.GCPTargets.IsNull() {
		var gcpTargets models.GCPTargetsModel
		gcpDiags := plan.GCPTargets.As(ctx, &gcpTargets, basetypes.ObjectAsOptions{})
		diags.Append(gcpDiags...)
		if diags.HasError() {
			return targets
		}

		gcpResource := &uapsiavmmodels.ArkUAPSIAVMGCPResource{}

		// Regions
		if !gcpTargets.Regions.IsNull() {
			var regions []string
			regionsDiags := gcpTargets.Regions.ElementsAs(ctx, &regions, false)
			diags.Append(regionsDiags...)
			if !diags.HasError() {
				gcpResource.Regions = regions
			}
		}

		// Labels (NOTE: GCP uses Labels not Tags!)
		if !gcpTargets.Labels.IsNull() {
			var labels []models.TagModel
			labelsDiags := gcpTargets.Labels.ElementsAs(ctx, &labels, false)
			diags.Append(labelsDiags...)

			if !diags.HasError() {
				sdkLabels := make([]uapsiavmmodels.ArkUAPSIAVMKeyValTag, len(labels))
				for i, label := range labels {
					sdkLabels[i] = uapsiavmmodels.ArkUAPSIAVMKeyValTag{
						Key: label.Key.ValueString(),
					}
					if !label.Value.IsNull() {
						var values []string
						valuesDiags := label.Value.ElementsAs(ctx, &values, false)
						diags.Append(valuesDiags...)
						if !diags.HasError() {
							sdkLabels[i].Value = values
						}
					}
				}
				gcpResource.Labels = sdkLabels
			}
		}

		// VPC IDs
		if !gcpTargets.VPCIDs.IsNull() {
			var vpcIDs []string
			vpcIDsDiags := gcpTargets.VPCIDs.ElementsAs(ctx, &vpcIDs, false)
			diags.Append(vpcIDsDiags...)
			if !diags.HasError() {
				gcpResource.VPCIDs = vpcIDs
			}
		}

		// Projects
		if !gcpTargets.Projects.IsNull() {
			var projects []string
			projectsDiags := gcpTargets.Projects.ElementsAs(ctx, &projects, false)
			diags.Append(projectsDiags...)
			if !diags.HasError() {
				gcpResource.Projects = projects
			}
		}

		targets.GCPResource = gcpResource
	}

	return targets
}

// buildSDKConditions creates SDK conditions from Terraform plan
func buildSDKConditions(plan models.VMPolicyResourceModel) uapcommondels.ArkUAPSIACommonConditions {
	conditions := uapcommondels.ArkUAPSIACommonConditions{
		ArkUAPConditions: uapcommonmodels.ArkUAPConditions{
			MaxSessionDuration: int(plan.MaxSessionDuration.ValueInt64()),
		},
		IdleTime: int(plan.IdleTime.ValueInt64()),
	}

	// Add access window if provided
	if !plan.AccessWindow.IsNull() && !plan.AccessWindow.IsUnknown() {
		var accessWindow models.VMAccessWindowModel
		plan.AccessWindow.As(context.Background(), &accessWindow, basetypes.ObjectAsOptions{})

		var daysOfWeek []int
		if !accessWindow.DaysOfTheWeek.IsNull() {
			var daysInt64 []int64
			accessWindow.DaysOfTheWeek.ElementsAs(context.Background(), &daysInt64, false)
			daysOfWeek = make([]int, len(daysInt64))
			for i, d := range daysInt64 {
				daysOfWeek[i] = int(d)
			}
		}

		conditions.AccessWindow = uapcommonmodels.ArkUAPTimeCondition{
			DaysOfTheWeek: daysOfWeek,
			FromHour:      accessWindow.FromHour.ValueString(),
			ToHour:        accessWindow.ToHour.ValueString(),
		}
	}

	return conditions
}

// mapSDKPolicyToState converts SDK policy response to Terraform state
func mapSDKPolicyToState(ctx context.Context, sdkPolicy *uapsiavmmodels.ArkUAPSIAVMAccessPolicy, diags *diag.Diagnostics) models.VMPolicyResourceModel {
	state := models.VMPolicyResourceModel{}

	// Identity
	state.ID = types.StringValue(sdkPolicy.Metadata.PolicyID)
	state.PolicyID = types.StringValue(sdkPolicy.Metadata.PolicyID)

	// Metadata
	state.Name = types.StringValue(sdkPolicy.Metadata.Name)
	// Normalize empty description to null
	if sdkPolicy.Metadata.Description != "" {
		state.Description = types.StringValue(sdkPolicy.Metadata.Description)
	} else {
		state.Description = types.StringNull()
	}
	state.TimeZone = types.StringValue(sdkPolicy.Metadata.TimeZone)
	state.LocationType = types.StringValue(sdkPolicy.Metadata.PolicyEntitlement.LocationType)
	state.Status = types.StringValue(sdkPolicy.Metadata.Status.Status)
	state.PolicyType = types.StringValue(sdkPolicy.Metadata.PolicyEntitlement.PolicyType)
	state.DelegationClassification = types.StringValue(sdkPolicy.DelegationClassification)

	// Tags
	if len(sdkPolicy.Metadata.PolicyTags) > 0 {
		tagsList, diagsTags := types.ListValueFrom(ctx, types.StringType, sdkPolicy.Metadata.PolicyTags)
		if diagsTags.HasError() {
			diags.Append(diagsTags...)
		} else {
			state.Tags = tagsList
		}
	} else {
		state.Tags = types.ListNull(types.StringType)
	}

	// Time frame
	if sdkPolicy.Metadata.TimeFrame.FromTime != "" || sdkPolicy.Metadata.TimeFrame.ToTime != "" {
		timeFrameObj, diagsTimeFrame := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"from_time": types.StringType,
			"to_time":   types.StringType,
		}, models.TimeFrameModel{
			FromTime: types.StringValue(sdkPolicy.Metadata.TimeFrame.FromTime),
			ToTime:   types.StringValue(sdkPolicy.Metadata.TimeFrame.ToTime),
		})
		if diagsTimeFrame.HasError() {
			diags.Append(diagsTimeFrame...)
		} else {
			state.TimeFrame = timeFrameObj
		}
	} else {
		state.TimeFrame = types.ObjectNull(map[string]attr.Type{
			"from_time": types.StringType,
			"to_time":   types.StringType,
		})
	}

	// Principals - map ALL principals (inline + assigned)
	if len(sdkPolicy.Principals) > 0 {
		principalModels := make([]models.PrincipalModel, len(sdkPolicy.Principals))
		for i, p := range sdkPolicy.Principals {
			principalModels[i] = models.PrincipalModel{
				PrincipalID:         types.StringValue(p.ID),
				PrincipalName:       types.StringValue(p.Name),
				PrincipalType:       types.StringValue(p.Type),
				SourceDirectoryName: types.StringValue(p.SourceDirectoryName),
				SourceDirectoryID:   types.StringValue(p.SourceDirectoryID),
			}
		}

		principalsList, diagsPrincipals := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"principal_id":          types.StringType,
				"principal_name":        types.StringType,
				"principal_type":        types.StringType,
				"source_directory_name": types.StringType,
				"source_directory_id":   types.StringType,
			},
		}, principalModels)
		if diagsPrincipals.HasError() {
			diags.Append(diagsPrincipals...)
		} else {
			state.Principals = principalsList
		}
	}

	// Conditions
	state.MaxSessionDuration = types.Int64Value(int64(sdkPolicy.Conditions.MaxSessionDuration))
	state.IdleTime = types.Int64Value(int64(sdkPolicy.Conditions.IdleTime))

	// Access window
	if len(sdkPolicy.Conditions.AccessWindow.DaysOfTheWeek) > 0 ||
		sdkPolicy.Conditions.AccessWindow.FromHour != "" || sdkPolicy.Conditions.AccessWindow.ToHour != "" {

		var daysInt64 []int64
		for _, d := range sdkPolicy.Conditions.AccessWindow.DaysOfTheWeek {
			daysInt64 = append(daysInt64, int64(d))
		}

		daysSet, diagsDays := types.SetValueFrom(ctx, types.Int64Type, daysInt64)
		if diagsDays.HasError() {
			diags.Append(diagsDays...)
		}

		// Normalize empty from_hour/to_hour to null
		fromHour := types.StringNull()
		if sdkPolicy.Conditions.AccessWindow.FromHour != "" {
			fromHour = types.StringValue(sdkPolicy.Conditions.AccessWindow.FromHour)
		}
		toHour := types.StringNull()
		if sdkPolicy.Conditions.AccessWindow.ToHour != "" {
			toHour = types.StringValue(sdkPolicy.Conditions.AccessWindow.ToHour)
		}

		accessWindowObj, diagsAW := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"days_of_the_week": types.SetType{ElemType: types.Int64Type},
			"from_hour":        types.StringType,
			"to_hour":          types.StringType,
		}, models.VMAccessWindowModel{
			DaysOfTheWeek: daysSet,
			FromHour:      fromHour,
			ToHour:        toHour,
		})
		if diagsAW.HasError() {
			diags.Append(diagsAW...)
		} else {
			state.AccessWindow = accessWindowObj
		}
	} else {
		state.AccessWindow = types.ObjectNull(map[string]attr.Type{
			"days_of_the_week": types.SetType{ElemType: types.Int64Type},
			"from_hour":        types.StringType,
			"to_hour":          types.StringType,
		})
	}

	// Behavior - always read from API (drift detection)
	behaviorModel := models.BehaviorModel{}

	if sdkPolicy.Behavior.SSHProfile != nil {
		sshObj, diagsSSH := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"username": types.StringType,
		}, models.SSHProfileModel{
			Username: types.StringValue(sdkPolicy.Behavior.SSHProfile.Username),
		})
		if diagsSSH.HasError() {
			diags.Append(diagsSSH...)
		} else {
			behaviorModel.SSH = sshObj
		}
	} else {
		behaviorModel.SSH = types.ObjectNull(map[string]attr.Type{
			"username": types.StringType,
		})
	}

	// Map RDP profile if present
	if sdkPolicy.Behavior.RDPProfile != nil {
		rdpModel := models.RDPProfileModel{}

		// Map local ephemeral user if present
		if sdkPolicy.Behavior.RDPProfile.LocalEphemeralUser != nil {
			assignGroups := sdkPolicy.Behavior.RDPProfile.LocalEphemeralUser.AssignGroups
			assignGroupsList, diagsGroups := types.ListValueFrom(ctx, types.StringType, assignGroups)
			if diagsGroups.HasError() {
				diags.Append(diagsGroups...)
			}

			localUserObj, diagsLocalUser := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"assign_groups":                   types.ListType{ElemType: types.StringType},
				"enable_ephemeral_user_reconnect": types.BoolType,
			}, models.LocalEphemeralUserModel{
				AssignGroups:                 assignGroupsList,
				EnableEphemeralUserReconnect: types.BoolValue(sdkPolicy.Behavior.RDPProfile.LocalEphemeralUser.EnableEphemeralUserReconnect),
			})
			if diagsLocalUser.HasError() {
				diags.Append(diagsLocalUser...)
			} else {
				rdpModel.LocalEphemeralUser = localUserObj
			}
		}

		// Map domain ephemeral user if present
		if sdkPolicy.Behavior.RDPProfile.DomainEphemeralUser != nil {
			assignGroups := sdkPolicy.Behavior.RDPProfile.DomainEphemeralUser.AssignGroups
			assignGroupsList, diagsGroups := types.ListValueFrom(ctx, types.StringType, assignGroups)
			if diagsGroups.HasError() {
				diags.Append(diagsGroups...)
			}

			assignDomainGroups := sdkPolicy.Behavior.RDPProfile.DomainEphemeralUser.AssignDomainGroups
			assignDomainGroupsList, diagsDomainGroups := types.ListValueFrom(ctx, types.StringType, assignDomainGroups)
			if diagsDomainGroups.HasError() {
				diags.Append(diagsDomainGroups...)
			}

			domainUserObj, diagsDomainUser := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"assign_groups":                   types.ListType{ElemType: types.StringType},
				"assign_domain_groups":            types.ListType{ElemType: types.StringType},
				"enable_ephemeral_user_reconnect": types.BoolType,
			}, models.DomainEphemeralUserModel{
				AssignGroups:                 assignGroupsList,
				AssignDomainGroups:           assignDomainGroupsList,
				EnableEphemeralUserReconnect: types.BoolValue(sdkPolicy.Behavior.RDPProfile.DomainEphemeralUser.EnableEphemeralUserReconnect),
			})
			if diagsDomainUser.HasError() {
				diags.Append(diagsDomainUser...)
			} else {
				rdpModel.DomainEphemeralUser = domainUserObj
			}
		}

		rdpObj, diagsRDP := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"local_ephemeral_user":  types.ObjectType{AttrTypes: map[string]attr.Type{"assign_groups": types.ListType{ElemType: types.StringType}, "enable_ephemeral_user_reconnect": types.BoolType}},
			"domain_ephemeral_user": types.ObjectType{AttrTypes: map[string]attr.Type{"assign_groups": types.ListType{ElemType: types.StringType}, "assign_domain_groups": types.ListType{ElemType: types.StringType}, "enable_ephemeral_user_reconnect": types.BoolType}},
		}, rdpModel)
		if diagsRDP.HasError() {
			diags.Append(diagsRDP...)
		} else {
			behaviorModel.RDP = rdpObj
		}
	} else {
		behaviorModel.RDP = types.ObjectNull(map[string]attr.Type{
			"local_ephemeral_user": types.ObjectType{AttrTypes: map[string]attr.Type{
				"assign_groups":                   types.ListType{ElemType: types.StringType},
				"enable_ephemeral_user_reconnect": types.BoolType,
			}},
			"domain_ephemeral_user": types.ObjectType{AttrTypes: map[string]attr.Type{
				"assign_groups":                   types.ListType{ElemType: types.StringType},
				"assign_domain_groups":            types.ListType{ElemType: types.StringType},
				"enable_ephemeral_user_reconnect": types.BoolType,
			}},
		})
	}

	behaviorObj, diagsBehavior := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"ssh": types.ObjectType{AttrTypes: map[string]attr.Type{"username": types.StringType}},
		"rdp": types.ObjectType{AttrTypes: map[string]attr.Type{
			"local_ephemeral_user": types.ObjectType{AttrTypes: map[string]attr.Type{
				"assign_groups":                   types.ListType{ElemType: types.StringType},
				"enable_ephemeral_user_reconnect": types.BoolType,
			}},
			"domain_ephemeral_user": types.ObjectType{AttrTypes: map[string]attr.Type{
				"assign_groups":                   types.ListType{ElemType: types.StringType},
				"assign_domain_groups":            types.ListType{ElemType: types.StringType},
				"enable_ephemeral_user_reconnect": types.BoolType,
			}},
		}},
	}, behaviorModel)
	if diagsBehavior.HasError() {
		diags.Append(diagsBehavior...)
	} else {
		state.Behavior = behaviorObj
	}

	// Targets - always read from API (drift detection)
	if sdkPolicy.Targets.FQDNIPResource != nil {
		fqdnIPTargets := models.FQDNIPTargetsModel{}

		// Map FQDN rules
		if len(sdkPolicy.Targets.FQDNIPResource.FQDNRules) > 0 {
			fqdnRuleModels := make([]models.FQDNRuleModel, len(sdkPolicy.Targets.FQDNIPResource.FQDNRules))
			for i, rule := range sdkPolicy.Targets.FQDNIPResource.FQDNRules {
				// Normalize empty domain string to null
				domain := types.StringNull()
				if rule.Domain != "" {
					domain = types.StringValue(rule.Domain)
				}
				fqdnRuleModels[i] = models.FQDNRuleModel{
					Operator:            types.StringValue(rule.Operator),
					ComputernamePattern: types.StringValue(rule.ComputernamePattern),
					Domain:              domain,
				}
			}
			fqdnRulesList, diagsFQDNRules := types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"operator":             types.StringType,
					"computername_pattern": types.StringType,
					"domain":               types.StringType,
				},
			}, fqdnRuleModels)
			if diagsFQDNRules.HasError() {
				diags.Append(diagsFQDNRules...)
			} else {
				fqdnIPTargets.FQDNRules = fqdnRulesList
			}
		}

		// Map IP rules
		if len(sdkPolicy.Targets.FQDNIPResource.IPRules) > 0 {
			ipRuleModels := make([]models.IPRuleModel, len(sdkPolicy.Targets.FQDNIPResource.IPRules))
			for i, rule := range sdkPolicy.Targets.FQDNIPResource.IPRules {
				ipAddressesList, diagsIPs := types.ListValueFrom(ctx, types.StringType, rule.IPAddresses)
				if diagsIPs.HasError() {
					diags.Append(diagsIPs...)
				}
				ipRuleModels[i] = models.IPRuleModel{
					Operator:    types.StringValue(rule.Operator),
					IPAddresses: ipAddressesList,
					LogicalName: types.StringValue(rule.LogicalName),
				}
			}
			ipRulesList, diagsIPRules := types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"operator":     types.StringType,
					"ip_addresses": types.ListType{ElemType: types.StringType},
					"logical_name": types.StringType,
				},
			}, ipRuleModels)
			if diagsIPRules.HasError() {
				diags.Append(diagsIPRules...)
			} else {
				fqdnIPTargets.IPRules = ipRulesList
			}
		} else {
			// Initialize empty list when no IP rules
			fqdnIPTargets.IPRules = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"operator":     types.StringType,
					"ip_addresses": types.ListType{ElemType: types.StringType},
					"logical_name": types.StringType,
				},
			})
		}

		fqdnIPTargetsObj, diagsFQDNIP := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"fqdn_rule": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "computername_pattern": types.StringType, "domain": types.StringType}}},
			"ip_rule":   types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "ip_addresses": types.ListType{ElemType: types.StringType}, "logical_name": types.StringType}}},
		}, fqdnIPTargets)
		if diagsFQDNIP.HasError() {
			diags.Append(diagsFQDNIP...)
		} else {
			state.FQDNIPTargets = fqdnIPTargetsObj
		}
	}

	// Map AWS targets if present
	if sdkPolicy.Targets.AWSResource != nil {
		awsTargets := models.AWSTargetsModel{}
		var diagsTemp diag.Diagnostics

		// Map regions
		if len(sdkPolicy.Targets.AWSResource.Regions) > 0 {
			awsTargets.Regions, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AWSResource.Regions)
			diags.Append(diagsTemp...)
		} else {
			awsTargets.Regions = types.ListNull(types.StringType)
		}

		// Map tags
		if len(sdkPolicy.Targets.AWSResource.Tags) > 0 {
			tagModels := make([]models.TagModel, len(sdkPolicy.Targets.AWSResource.Tags))
			for i, sdkTag := range sdkPolicy.Targets.AWSResource.Tags {
				tagModels[i] = models.TagModel{
					Key: types.StringValue(sdkTag.Key),
				}
				if len(sdkTag.Value) > 0 {
					tagModels[i].Value, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkTag.Value)
					diags.Append(diagsTemp...)
				} else {
					tagModels[i].Value = types.ListNull(types.StringType)
				}
			}
			awsTargets.Tags, diagsTemp = types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}, tagModels)
			diags.Append(diagsTemp...)
		} else {
			awsTargets.Tags = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			})
		}

		// Map VPC IDs
		if len(sdkPolicy.Targets.AWSResource.VPCIDs) > 0 {
			awsTargets.VPCIDs, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AWSResource.VPCIDs)
			diags.Append(diagsTemp...)
		} else {
			awsTargets.VPCIDs = types.ListNull(types.StringType)
		}

		// Map Account IDs
		if len(sdkPolicy.Targets.AWSResource.AccountIDs) > 0 {
			awsTargets.AccountIDs, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AWSResource.AccountIDs)
			diags.Append(diagsTemp...)
		} else {
			awsTargets.AccountIDs = types.ListNull(types.StringType)
		}

		// Convert to object
		state.AWSTargets, diagsTemp = types.ObjectValueFrom(ctx, map[string]attr.Type{
			"regions": types.ListType{ElemType: types.StringType},
			"tags": types.ListType{ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}},
			"vpc_ids":     types.ListType{ElemType: types.StringType},
			"account_ids": types.ListType{ElemType: types.StringType},
		}, awsTargets)
		diags.Append(diagsTemp...)
	} else {
		state.AWSTargets = types.ObjectNull(map[string]attr.Type{
			"regions": types.ListType{ElemType: types.StringType},
			"tags": types.ListType{ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}},
			"vpc_ids":     types.ListType{ElemType: types.StringType},
			"account_ids": types.ListType{ElemType: types.StringType},
		})
	}

	// Map Azure targets if present
	if sdkPolicy.Targets.AzureResource != nil {
		azureTargets := models.AzureTargetsModel{}
		var diagsTemp diag.Diagnostics

		// Map regions
		if len(sdkPolicy.Targets.AzureResource.Regions) > 0 {
			azureTargets.Regions, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AzureResource.Regions)
			diags.Append(diagsTemp...)
		} else {
			azureTargets.Regions = types.ListNull(types.StringType)
		}

		// Map tags
		if len(sdkPolicy.Targets.AzureResource.Tags) > 0 {
			tagModels := make([]models.TagModel, len(sdkPolicy.Targets.AzureResource.Tags))
			for i, sdkTag := range sdkPolicy.Targets.AzureResource.Tags {
				tagModels[i] = models.TagModel{
					Key: types.StringValue(sdkTag.Key),
				}
				if len(sdkTag.Value) > 0 {
					tagModels[i].Value, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkTag.Value)
					diags.Append(diagsTemp...)
				} else {
					tagModels[i].Value = types.ListNull(types.StringType)
				}
			}
			azureTargets.Tags, diagsTemp = types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}, tagModels)
			diags.Append(diagsTemp...)
		} else {
			azureTargets.Tags = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			})
		}

		// Map Resource Groups
		if len(sdkPolicy.Targets.AzureResource.ResourceGroups) > 0 {
			azureTargets.ResourceGroups, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AzureResource.ResourceGroups)
			diags.Append(diagsTemp...)
		} else {
			azureTargets.ResourceGroups = types.ListNull(types.StringType)
		}

		// Map VNet IDs
		if len(sdkPolicy.Targets.AzureResource.VNetIDs) > 0 {
			azureTargets.VNetIDs, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AzureResource.VNetIDs)
			diags.Append(diagsTemp...)
		} else {
			azureTargets.VNetIDs = types.ListNull(types.StringType)
		}

		// Map Subscriptions
		if len(sdkPolicy.Targets.AzureResource.Subscriptions) > 0 {
			azureTargets.Subscriptions, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.AzureResource.Subscriptions)
			diags.Append(diagsTemp...)
		} else {
			azureTargets.Subscriptions = types.ListNull(types.StringType)
		}

		// Convert to object
		state.AzureTargets, diagsTemp = types.ObjectValueFrom(ctx, map[string]attr.Type{
			"regions": types.ListType{ElemType: types.StringType},
			"tags": types.ListType{ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}},
			"resource_groups": types.ListType{ElemType: types.StringType},
			"vnet_ids":        types.ListType{ElemType: types.StringType},
			"subscriptions":   types.ListType{ElemType: types.StringType},
		}, azureTargets)
		diags.Append(diagsTemp...)
	} else {
		state.AzureTargets = types.ObjectNull(map[string]attr.Type{
			"regions": types.ListType{ElemType: types.StringType},
			"tags": types.ListType{ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}},
			"resource_groups": types.ListType{ElemType: types.StringType},
			"vnet_ids":        types.ListType{ElemType: types.StringType},
			"subscriptions":   types.ListType{ElemType: types.StringType},
		})
	}

	// Map GCP targets if present (NOTE: uses Labels not Tags!)
	if sdkPolicy.Targets.GCPResource != nil {
		gcpTargets := models.GCPTargetsModel{}
		var diagsTemp diag.Diagnostics

		// Map regions
		if len(sdkPolicy.Targets.GCPResource.Regions) > 0 {
			gcpTargets.Regions, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.GCPResource.Regions)
			diags.Append(diagsTemp...)
		} else {
			gcpTargets.Regions = types.ListNull(types.StringType)
		}

		// Map labels (NOTE: GCP uses Labels not Tags!)
		if len(sdkPolicy.Targets.GCPResource.Labels) > 0 {
			labelModels := make([]models.TagModel, len(sdkPolicy.Targets.GCPResource.Labels))
			for i, sdkLabel := range sdkPolicy.Targets.GCPResource.Labels {
				labelModels[i] = models.TagModel{
					Key: types.StringValue(sdkLabel.Key),
				}
				if len(sdkLabel.Value) > 0 {
					labelModels[i].Value, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkLabel.Value)
					diags.Append(diagsTemp...)
				} else {
					labelModels[i].Value = types.ListNull(types.StringType)
				}
			}
			gcpTargets.Labels, diagsTemp = types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}, labelModels)
			diags.Append(diagsTemp...)
		} else {
			gcpTargets.Labels = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			})
		}

		// Map VPC IDs
		if len(sdkPolicy.Targets.GCPResource.VPCIDs) > 0 {
			gcpTargets.VPCIDs, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.GCPResource.VPCIDs)
			diags.Append(diagsTemp...)
		} else {
			gcpTargets.VPCIDs = types.ListNull(types.StringType)
		}

		// Map Projects
		if len(sdkPolicy.Targets.GCPResource.Projects) > 0 {
			gcpTargets.Projects, diagsTemp = types.ListValueFrom(ctx, types.StringType, sdkPolicy.Targets.GCPResource.Projects)
			diags.Append(diagsTemp...)
		} else {
			gcpTargets.Projects = types.ListNull(types.StringType)
		}

		// Convert to object
		state.GCPTargets, diagsTemp = types.ObjectValueFrom(ctx, map[string]attr.Type{
			"regions": types.ListType{ElemType: types.StringType},
			"labels": types.ListType{ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}},
			"vpc_ids":  types.ListType{ElemType: types.StringType},
			"projects": types.ListType{ElemType: types.StringType},
		}, gcpTargets)
		diags.Append(diagsTemp...)
	} else {
		state.GCPTargets = types.ObjectNull(map[string]attr.Type{
			"regions": types.ListType{ElemType: types.StringType},
			"labels": types.ListType{ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"key":   types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}},
			"vpc_ids":  types.ListType{ElemType: types.StringType},
			"projects": types.ListType{ElemType: types.StringType},
		})
	}

	// Created/Updated metadata
	if sdkPolicy.Metadata.CreatedBy.User != "" {
		createdByObj, diagsCreated := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"user":      types.StringType,
			"timestamp": types.StringType,
		}, models.UserTimestampModel{
			User:      types.StringValue(sdkPolicy.Metadata.CreatedBy.User),
			Timestamp: types.StringValue(sdkPolicy.Metadata.CreatedBy.Time),
		})
		if diagsCreated.HasError() {
			diags.Append(diagsCreated...)
		} else {
			state.CreatedBy = createdByObj
		}
	} else {
		state.CreatedBy = types.ObjectNull(map[string]attr.Type{
			"user":      types.StringType,
			"timestamp": types.StringType,
		})
	}

	if sdkPolicy.Metadata.UpdatedOn.User != "" {
		updatedByObj, diagsUpdated := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"user":      types.StringType,
			"timestamp": types.StringType,
		}, models.UserTimestampModel{
			User:      types.StringValue(sdkPolicy.Metadata.UpdatedOn.User),
			Timestamp: types.StringValue(sdkPolicy.Metadata.UpdatedOn.Time),
		})
		if diagsUpdated.HasError() {
			diags.Append(diagsUpdated...)
		} else {
			state.UpdatedBy = updatedByObj
		}
	} else {
		state.UpdatedBy = types.ObjectNull(map[string]attr.Type{
			"user":      types.StringType,
			"timestamp": types.StringType,
		})
	}

	return state
}
