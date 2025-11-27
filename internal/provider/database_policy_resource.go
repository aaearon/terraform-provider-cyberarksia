package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/client"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/models"
	"github.com/aaearon/terraform-provider-cyberarksia/internal/validators"
	dbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/db/models"
	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
	uapsiadbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/db/models"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DatabasePolicyResource{}
var _ resource.ResourceWithImportState = &DatabasePolicyResource{}
var _ resource.ResourceWithValidateConfig = &DatabasePolicyResource{}
var _ resource.ResourceWithModifyPlan = &DatabasePolicyResource{}

func NewDatabasePolicyResource() resource.Resource {
	return &DatabasePolicyResource{}
}

// DatabasePolicyResource defines the resource implementation.
type DatabasePolicyResource struct {
	providerData *ProviderData
}

func (r *DatabasePolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_policy"
}

func (r *DatabasePolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CyberArk SIA database access policy including metadata and access conditions. " +
			"This resource manages policy-level configuration only. Use `cyberarksia_database_policy_principal_assignment` " +
			"to assign principals (users/groups/roles) and `cyberarksia_database_policy_workspace_assignment` to assign database workspaces.\n\n" +
			"**Pattern**: Follows the modular assignment pattern for distributed team workflows - security teams manage policies " +
			"and principals, application teams manage database assignments independently.\n\n" +
			"**Constraints**: This resource requires at least 1 `principal` block and 1 `target_database` block. " +
			"Additional principals and targets can be managed via separate assignment resources. " +
			"The provider validates constraints by checking both inline and externally-managed assignments during planning.\n\n" +
			"**Important**: When removing inline principals or targets, the provider queries the API to ensure the constraint " +
			"won't be violated. If you need to remove the last inline item, either add another via assignment resources first, " +
			"or delete the entire policy resource (which automatically cascades to all assignments).",

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
				MarkdownDescription: "Policy name (1-200 characters, unique per tenant). **ForceNew**: Changing this creates a new policy.",
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
				MarkdownDescription: "Policy status. Valid values: `active` (enabled), `suspended` (disabled). **Note**: `expired`, `validating`, and `error` are server-managed statuses and cannot be set by users.",
				Required:            true,
				Validators: []validator.String{
					validators.PolicyStatus(),
				},
			},
			"delegation_classification": schema.StringAttribute{
				MarkdownDescription: "Delegation classification. Valid values: `restricted`, `unrestricted`. Default: `unrestricted`. **Note**: Currently, SIA only supports `unrestricted` for database policies regardless of the value set. This attribute is available for future compatibility.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("unrestricted"),
				Validators: []validator.String{
					stringvalidator.OneOf("restricted", "unrestricted", "Restricted", "Unrestricted"),
				},
			},
			"time_zone": schema.StringAttribute{
				MarkdownDescription: "Timezone for access window conditions (max 50 characters). Supports IANA timezone names (e.g., `America/New_York`) or GMT offsets (e.g., `GMT+05:00`). Default: `GMT`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("GMT"),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(50),
				},
			},
			"policy_tags": schema.SetAttribute{
				MarkdownDescription: "Set of tags for policy organization (max 20 tags).",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtMost(20),
				},
			},
			"last_modified": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last modification to the policy.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata about policy creation (set by API).",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"user": schema.StringAttribute{
						MarkdownDescription: "Username of the user who created the policy.",
						Computed:            true,
					},
					"timestamp": schema.StringAttribute{
						MarkdownDescription: "Creation timestamp in ISO 8601 format.",
						Computed:            true,
					},
				},
			},
			"updated_on": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata about the last policy update (set by API).",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"user": schema.StringAttribute{
						MarkdownDescription: "Username of the user who last updated the policy.",
						Computed:            true,
					},
					"timestamp": schema.StringAttribute{
						MarkdownDescription: "Last update timestamp in ISO 8601 format.",
						Computed:            true,
					},
				},
			},
		},

		Blocks: map[string]schema.Block{
			"target_database": schema.ListNestedBlock{
				MarkdownDescription: "Database workspace assignment (repeatable block). **Required**: At least 1 target_database block is required. " +
					"Follows familiar Terraform patterns (aws_security_group ingress/egress). " +
					"Use `lifecycle { ignore_changes = [target_database] }` if managing assignments via separate `cyberarksia_policy_database_assignment` resources.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"database_workspace_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the database workspace to assign.",
							Required:            true,
						},
						"authentication_method": schema.StringAttribute{
							MarkdownDescription: "Authentication method. Valid values: `db_auth`, `ldap_auth`, `oracle_auth`, `mongo_auth`, `sqlserver_auth`, `rds_iam_user_auth`.",
							Required:            true,
							Validators: []validator.String{
								validators.AuthenticationMethod(),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"db_auth_profile": schema.SingleNestedBlock{
							MarkdownDescription: "Database authentication profile. **Required** when `authentication_method` is `db_auth`.",
							Attributes: map[string]schema.Attribute{
								"roles": schema.SetAttribute{
									MarkdownDescription: "Set of database roles to assign.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
						"ldap_auth_profile": schema.SingleNestedBlock{
							MarkdownDescription: "LDAP authentication profile. **Required** when `authentication_method` is `ldap_auth`.",
							Attributes: map[string]schema.Attribute{
								"assign_groups": schema.SetAttribute{
									MarkdownDescription: "Set of LDAP groups to assign.",
									Optional:            true,
									ElementType:         types.StringType,
								},
							},
						},
						"oracle_auth_profile": schema.SingleNestedBlock{
							MarkdownDescription: "Oracle authentication profile. **Required** when `authentication_method` is `oracle_auth`.",
							Attributes: map[string]schema.Attribute{
								"roles": schema.SetAttribute{
									MarkdownDescription: "Set of Oracle roles to assign.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"dba_role": schema.BoolAttribute{
									MarkdownDescription: "Grant DBA role.",
									Optional:            true,
								},
								"sysdba_role": schema.BoolAttribute{
									MarkdownDescription: "Grant SYSDBA role.",
									Optional:            true,
								},
								"sysoper_role": schema.BoolAttribute{
									MarkdownDescription: "Grant SYSOPER role.",
									Optional:            true,
								},
							},
						},
						"mongo_auth_profile": schema.SingleNestedBlock{
							MarkdownDescription: "MongoDB authentication profile. **Required** when `authentication_method` is `mongo_auth`.",
							Attributes: map[string]schema.Attribute{
								"global_builtin_roles": schema.SetAttribute{
									MarkdownDescription: "Set of global built-in roles.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"database_builtin_roles": schema.MapAttribute{
									MarkdownDescription: "Map of database names to built-in roles.",
									Optional:            true,
									ElementType:         types.ListType{ElemType: types.StringType},
								},
								"database_custom_roles": schema.MapAttribute{
									MarkdownDescription: "Map of database names to custom roles.",
									Optional:            true,
									ElementType:         types.ListType{ElemType: types.StringType},
								},
							},
						},
						"sqlserver_auth_profile": schema.SingleNestedBlock{
							MarkdownDescription: "SQL Server authentication profile. **Required** when `authentication_method` is `sqlserver_auth`.",
							Attributes: map[string]schema.Attribute{
								"global_builtin_roles": schema.SetAttribute{
									MarkdownDescription: "Set of global built-in roles.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"global_custom_roles": schema.SetAttribute{
									MarkdownDescription: "Set of global custom roles.",
									Optional:            true,
									ElementType:         types.StringType,
								},
								"database_builtin_roles": schema.MapAttribute{
									MarkdownDescription: "Map of database names to built-in roles.",
									Optional:            true,
									ElementType:         types.ListType{ElemType: types.StringType},
								},
								"database_custom_roles": schema.MapAttribute{
									MarkdownDescription: "Map of database names to custom roles.",
									Optional:            true,
									ElementType:         types.ListType{ElemType: types.StringType},
								},
							},
						},
						"rds_iam_user_auth_profile": schema.SingleNestedBlock{
							MarkdownDescription: "RDS IAM User authentication profile. **Required** when `authentication_method` is `rds_iam_user_auth`.",
							Attributes: map[string]schema.Attribute{
								"db_user": schema.StringAttribute{
									MarkdownDescription: "Database user for RDS IAM authentication.",
									Optional:            true,
								},
							},
						},
					},
				},
			},
			"principal": schema.SetNestedBlock{
				MarkdownDescription: "Principal assignment (repeatable block). **Required**: At least 1 principal block is required. " +
					"Follows familiar Terraform patterns (aws_security_group ingress/egress). Order-independent (principals may be returned in any order by the API). " +
					"Use `lifecycle { ignore_changes = [principal] }` if managing assignments via separate `cyberarksia_database_policy_principal_assignment` resources.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"principal_id": schema.StringAttribute{
							MarkdownDescription: "Principal identifier in UUID format (e.g., `c2c7bcc6-9560-44e0-8dff-5be221cd37ee`). This is the unique identifier returned by the SIA API.",
							Required:            true,
							Validators: []validator.String{
								validators.UUID(),
							},
						},
						"principal_type": schema.StringAttribute{
							MarkdownDescription: "Principal type. Valid values: `USER`, `GROUP`, `ROLE`.",
							Required:            true,
							Validators: []validator.String{
								validators.PrincipalType(),
							},
						},
						"principal_name": schema.StringAttribute{
							MarkdownDescription: "Principal name (SystemName). For USER: email format (e.g., `user@example.com`). For GROUP/ROLE: display name (e.g., `CyberIAM Guardians`, `Database Administrators`).",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 255),
							},
						},
						"source_directory_name": schema.StringAttribute{
							MarkdownDescription: "Source identity directory name (max 50 characters). **Required** for USER and GROUP types.",
							Optional:            true,
						},
						"source_directory_id": schema.StringAttribute{
							MarkdownDescription: "Source identity directory ID. **Required** for USER and GROUP types.",
							Optional:            true,
						},
					},
				},
			},
			"time_frame": schema.SingleNestedBlock{
				MarkdownDescription: "Policy validity period. **Optional**: If not specified, policy never expires (valid indefinitely). When specified, both `from_time` and `to_time` must be provided.",
				Attributes: map[string]schema.Attribute{
					"from_time": schema.StringAttribute{
						MarkdownDescription: "Start time (ISO 8601 format, e.g., `2024-01-01T00:00:00Z`). Required when `time_frame` block is present.",
						Optional:            true,
					},
					"to_time": schema.StringAttribute{
						MarkdownDescription: "End time (ISO 8601 format, e.g., `2024-12-31T23:59:59Z`). Required when `time_frame` block is present.",
						Optional:            true,
					},
				},
			},
			"conditions": schema.SingleNestedBlock{
				MarkdownDescription: "Policy access conditions (session limits, idle timeouts, time windows).",
				Attributes: map[string]schema.Attribute{
					"max_session_duration": schema.Int64Attribute{
						MarkdownDescription: "Maximum session duration in hours (1-24). **Required**.",
						Required:            true,
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
				},
				Blocks: map[string]schema.Block{
					"access_window": schema.SingleNestedBlock{
						MarkdownDescription: "Time-based access restrictions (days and hours).",
						Attributes: map[string]schema.Attribute{
							"days_of_the_week": schema.SetAttribute{
								MarkdownDescription: "Days access is allowed (0=Sunday through 6=Saturday). Specify days in any order - order is automatically normalized. Example: `[1, 2, 3, 4, 5]` for weekdays.",
								Required:            true,
								ElementType:         types.Int64Type,
								Validators: []validator.Set{
									setvalidator.ValueInt64sAre(int64validator.Between(0, 6)), // 0=Sunday through 6=Saturday (0-indexed)
									setvalidator.SizeBetween(1, 7),                            // At least 1 day required, max 7 days (e.g., all week = [0,1,2,3,4,5,6])
								},
							},
							"from_hour": schema.StringAttribute{
								MarkdownDescription: "Start time in HH:MM format (e.g., `09:00`). Optional - if both `from_hour` and `to_hour` are omitted, access is allowed all day. If one is specified, the other must also be specified.",
								Optional:            true,
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										mustCompileRegex(`^([01]\d|2[0-3]):([0-5]\d)$`),
										"must be in HH:MM format (e.g., 09:00)",
									),
								},
							},
							"to_hour": schema.StringAttribute{
								MarkdownDescription: "End time in HH:MM format (e.g., `17:00`). Optional - if both `from_hour` and `to_hour` are omitted, access is allowed all day. If one is specified, the other must also be specified.",
								Optional:            true,
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										mustCompileRegex(`^([01]\d|2[0-3]):([0-5]\d)$`),
										"must be in HH:MM format (e.g., 17:00)",
									),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *DatabasePolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatabasePolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data models.DatabasePolicyModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate at least 1 target database
	if len(data.TargetDatabase) == 0 {
		resp.Diagnostics.AddError(
			"Missing Target Databases",
			"At least one target_database block is required. Database access policies must have at least one target database.",
		)
	}

	// Validate at least 1 principal
	if len(data.Principal) == 0 {
		resp.Diagnostics.AddError(
			"Missing Principals",
			"At least one principal block is required. Database access policies must have at least one principal (user/group/role).",
		)
	}

	// Validate principal directory requirements (USER/GROUP need source_directory)
	// Skip validation if values are unknown (will be resolved at apply time from data sources)
	for i, principal := range data.Principal {
		principalType := principal.PrincipalType.ValueString()
		if principalType == "USER" || principalType == "GROUP" {
			// Skip validation if value is unknown (e.g., from data source reference)
			if !principal.SourceDirectoryName.IsUnknown() {
				if principal.SourceDirectoryName.IsNull() || principal.SourceDirectoryName.ValueString() == "" {
					resp.Diagnostics.AddError(
						"Missing Source Directory Name",
						fmt.Sprintf("principals[%d]: source_directory_name is required for principal_type %s", i, principalType),
					)
				}
			}
			if !principal.SourceDirectoryID.IsUnknown() {
				if principal.SourceDirectoryID.IsNull() || principal.SourceDirectoryID.ValueString() == "" {
					resp.Diagnostics.AddError(
						"Missing Source Directory ID",
						fmt.Sprintf("principals[%d]: source_directory_id is required for principal_type %s", i, principalType),
					)
				}
			}
		}
	}

	// Validate authentication method profiles match
	for i, targetDB := range data.TargetDatabase {
		authMethod := targetDB.AuthenticationMethod.ValueString()
		switch authMethod {
		case "db_auth":
			if targetDB.DBAuthProfile == nil {
				resp.Diagnostics.AddError(
					"Missing Authentication Profile",
					fmt.Sprintf("target_databases[%d]: db_auth_profile block is required when authentication_method is 'db_auth'", i),
				)
			}
		case "ldap_auth":
			if targetDB.LDAPAuthProfile == nil {
				resp.Diagnostics.AddError(
					"Missing Authentication Profile",
					fmt.Sprintf("target_databases[%d]: ldap_auth_profile block is required when authentication_method is 'ldap_auth'", i),
				)
			}
		case "oracle_auth":
			if targetDB.OracleAuthProfile == nil {
				resp.Diagnostics.AddError(
					"Missing Authentication Profile",
					fmt.Sprintf("target_databases[%d]: oracle_auth_profile block is required when authentication_method is 'oracle_auth'", i),
				)
			}
		case "mongo_auth":
			if targetDB.MongoAuthProfile == nil {
				resp.Diagnostics.AddError(
					"Missing Authentication Profile",
					fmt.Sprintf("target_databases[%d]: mongo_auth_profile block is required when authentication_method is 'mongo_auth'", i),
				)
			}
		case "sqlserver_auth":
			if targetDB.SQLServerAuthProfile == nil {
				resp.Diagnostics.AddError(
					"Missing Authentication Profile",
					fmt.Sprintf("target_databases[%d]: sqlserver_auth_profile block is required when authentication_method is 'sqlserver_auth'", i),
				)
			}
		case "rds_iam_user_auth":
			if targetDB.RDSIAMUserAuthProfile == nil {
				resp.Diagnostics.AddError(
					"Missing Authentication Profile",
					fmt.Sprintf("target_databases[%d]: rds_iam_user_auth_profile block is required when authentication_method is 'rds_iam_user_auth'", i),
				)
			}
		}
	}

	// Validate access_window: if from_hour or to_hour is set, both must be set
	if data.Conditions != nil && data.Conditions.AccessWindow != nil {
		fromHourSet := !data.Conditions.AccessWindow.FromHour.IsNull() && !data.Conditions.AccessWindow.FromHour.IsUnknown()
		toHourSet := !data.Conditions.AccessWindow.ToHour.IsNull() && !data.Conditions.AccessWindow.ToHour.IsUnknown()

		if fromHourSet != toHourSet {
			resp.Diagnostics.AddError(
				"Invalid Access Window Configuration",
				"Both from_hour and to_hour must be specified together, or both must be omitted. "+
					"When both are omitted, access is allowed all day (00:00-23:59).",
			)
		}
	}
}

// ModifyPlan implements plan-time validation with API awareness to prevent removing
// the last principal or target from a policy. This method queries the API to count
// both inline and externally-managed assignments (via separate assignment resources),
// ensuring accurate validation without false positives.
func (r *DatabasePolicyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
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
	var plan, state models.DatabasePolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only validate if inline items are being REDUCED
	principalsReducing := len(plan.Principal) < len(state.Principal)
	targetsReducing := len(plan.TargetDatabase) < len(state.TargetDatabase)

	if !principalsReducing && !targetsReducing {
		return // No risk of constraint violation
	}

	// Fetch actual policy from API to get TOTAL counts (inline + external assignments)
	policyID := state.PolicyID.ValueString()

	tflog.Debug(ctx, "ModifyPlan: Fetching policy to validate constraints", map[string]interface{}{
		"policy_id":           policyID,
		"principals_reducing": principalsReducing,
		"targets_reducing":    targetsReducing,
	})

	actualPolicy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
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
	if principalsReducing {
		totalPrincipals := len(actualPolicy.Principals)
		inlinePrincipals := len(state.Principal)
		externalPrincipals := totalPrincipals - inlinePrincipals
		principalsAfterChange := len(plan.Principal) + externalPrincipals

		tflog.Debug(ctx, "ModifyPlan: Principal count analysis", map[string]interface{}{
			"total_principals":    totalPrincipals,
			"inline_principals":   inlinePrincipals,
			"external_principals": externalPrincipals,
			"principals_after":    principalsAfterChange,
		})

		if principalsAfterChange < 1 {
			resp.Diagnostics.AddAttributeError(
				path.Root("principal"),
				"Cannot Remove Last Principal",
				fmt.Sprintf(
					"This change would remove the last principal from the policy.\n\n"+
						"Current state:\n"+
						"  • Inline principals (in this resource): %d\n"+
						"  • External principals (via cyberarksia_database_policy_principal_assignment): %d\n"+
						"  • Total principals: %d\n\n"+
						"After this change: %d principal(s) remaining\n\n"+
						"CyberArk SIA policies require at least 1 principal.\n\n"+
						"To resolve:\n"+
						"  1. Add another principal via cyberarksia_database_policy_principal_assignment first\n"+
						"  2. Delete the entire policy resource instead: terraform destroy cyberarksia_database_policy.comprehensive_test",
					inlinePrincipals, externalPrincipals, totalPrincipals, principalsAfterChange,
				),
			)
		}
	}

	// Validate targets constraint
	if targetsReducing {
		totalTargets := 0
		for _, targets := range actualPolicy.Targets {
			totalTargets += len(targets.Instances)
		}
		inlineTargets := len(state.TargetDatabase)
		externalTargets := totalTargets - inlineTargets
		targetsAfterChange := len(plan.TargetDatabase) + externalTargets

		tflog.Debug(ctx, "ModifyPlan: Target count analysis", map[string]interface{}{
			"total_targets":    totalTargets,
			"inline_targets":   inlineTargets,
			"external_targets": externalTargets,
			"targets_after":    targetsAfterChange,
		})

		if targetsAfterChange < 1 {
			resp.Diagnostics.AddAttributeError(
				path.Root("target_database"),
				"Cannot Remove Last Target Database",
				fmt.Sprintf(
					"This change would remove the last target database from the policy.\n\n"+
						"Current state:\n"+
						"  • Inline targets (in this resource): %d\n"+
						"  • External targets (via cyberarksia_database_policy_workspace_assignment): %d\n"+
						"  • Total targets: %d\n\n"+
						"After this change: %d target(s) remaining\n\n"+
						"CyberArk SIA policies require at least 1 target database.\n\n"+
						"To resolve:\n"+
						"  1. Add another target via cyberarksia_database_policy_workspace_assignment first\n"+
						"  2. Delete the entire policy resource instead: terraform destroy cyberarksia_database_policy.comprehensive_test",
					inlineTargets, externalTargets, totalTargets, targetsAfterChange,
				),
			)
		}
	}
}

func (r *DatabasePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.DatabasePolicyModel

	// Read Terraform plan data
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

	// Convert Terraform state to SDK policy (metadata only)
	policy := data.ToSDK()

	// Build inline target databases
	if len(data.TargetDatabase) > 0 {
		policy.Targets = make(map[string]uapsiadbmodels.ArkUAPSIADBTargets)

		for i, targetDB := range data.TargetDatabase {
			// Check for unknown or null database workspace ID
			if targetDB.DatabaseWorkspaceID.IsNull() || targetDB.DatabaseWorkspaceID.IsUnknown() {
				resp.Diagnostics.AddError(
					"Unknown Database Workspace ID",
					fmt.Sprintf("target_databases[%d]: database_workspace_id is not yet known. "+
						"This typically occurs when the database workspace is created in the same Terraform configuration. "+
						"Ensure the database workspace resource is created before the policy resource references it.", i),
				)
				return
			}

			// Fetch database workspace to get instance details
			databaseID := targetDB.DatabaseWorkspaceID.ValueString()
			databaseIDInt, err := strconv.Atoi(databaseID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Database ID",
					fmt.Sprintf("target_databases[%d]: database_workspace_id must be a valid integer: %s", i, err.Error()),
				)
				return
			}

			database, err := r.providerData.SIAAPI.WorkspacesDB().Database(&dbmodels.ArkSIADBGetDatabase{
				ID: databaseIDInt,
			})
			if err != nil {
				resp.Diagnostics.Append(client.MapError(err, fmt.Sprintf("fetch database workspace for target_databases[%d]", i)))
				return
			}

			// Determine workspace type (always "FQDN/IP" for all databases)
			workspaceType := "FQDN/IP"

			// Build instance target with authentication profile
			instanceTarget, buildErr := buildInstanceTarget(ctx, database, targetDB)
			if buildErr != nil {
				resp.Diagnostics.AddError(
					"Failed to Build Target",
					fmt.Sprintf("target_databases[%d]: %s", i, buildErr.Error()),
				)
				return
			}

			// Add to targets map
			targets := policy.Targets[workspaceType]
			targets.Instances = append(targets.Instances, *instanceTarget)
			policy.Targets[workspaceType] = targets

			tflog.Debug(ctx, "Added target database to policy", map[string]interface{}{
				"database_id":    databaseID,
				"workspace_type": workspaceType,
				"auth_method":    targetDB.AuthenticationMethod.ValueString(),
			})
		}
	}

	// Build inline principals
	if len(data.Principal) > 0 {
		policy.Principals = make([]uapcommonmodels.ArkUAPPrincipal, len(data.Principal))
		for i, principal := range data.Principal {
			policy.Principals[i] = uapcommonmodels.ArkUAPPrincipal{
				ID:                  principal.PrincipalID.ValueString(),
				Name:                principal.PrincipalName.ValueString(),
				Type:                principal.PrincipalType.ValueString(),
				SourceDirectoryName: principal.SourceDirectoryName.ValueString(),
				SourceDirectoryID:   principal.SourceDirectoryID.ValueString(),
			}

			tflog.Info(ctx, "SENDING PRINCIPAL TO API", map[string]interface{}{
				"principal_id":          principal.PrincipalID.ValueString(),
				"principal_name":        principal.PrincipalName.ValueString(),
				"principal_type":        principal.PrincipalType.ValueString(),
				"source_directory_name": principal.SourceDirectoryName.ValueString(),
				"source_directory_id":   principal.SourceDirectoryID.ValueString(),
			})
		}
	}

	// Create policy with retry logic
	var createdPolicy *uapsiadbmodels.ArkUAPSIADBAccessPolicy
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		var createErr error
		createdPolicy, createErr = r.providerData.UAPClient.Db().AddPolicy(policy)
		return createErr
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "create database policy"))
		return
	}

	// Log what API returned
	tflog.Info(ctx, "RECEIVED FROM API", map[string]interface{}{
		"principals_count": len(createdPolicy.Principals),
	})
	for i, p := range createdPolicy.Principals {
		tflog.Info(ctx, fmt.Sprintf("API RETURNED PRINCIPAL %d", i), map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
			"type": p.Type,
		})
	}

	// Set only the ID fields from API response
	// Don't call FromSDK() here - it tries to populate computed metadata fields
	// (created_by, updated_on) which causes "unknown value" errors during CREATE.
	// Terraform will automatically call Read() after Create() to populate all fields.
	data.ID = types.StringValue(createdPolicy.Metadata.PolicyID)
	data.PolicyID = types.StringValue(createdPolicy.Metadata.PolicyID)

	// Set last_modified to empty string (API doesn't return this field on create)
	data.LastModified = types.StringValue("")

	// Explicitly set computed metadata fields to null to avoid "unknown value" errors
	// These will be populated by the automatic Read() call after Create()
	data.CreatedBy = types.ObjectNull(models.ChangeInfoAttrTypes())
	data.UpdatedOn = types.ObjectNull(models.ChangeInfoAttrTypes())

	tflog.Info(ctx, "Created database policy", map[string]interface{}{
		"policy_id":        data.PolicyID.ValueString(),
		"policy_name":      data.Name.ValueString(),
		"target_databases": len(data.TargetDatabase),
		"principals":       len(data.Principal),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.DatabasePolicyModel

	// Read Terraform state
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

	// Fetch policy from API
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})

	if err != nil {
		// If policy not found, remove from state
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "read database policy"))
		return
	}

	// Update state with fetched policy
	if err := data.FromSDK(ctx, policy); err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Policy Response",
			fmt.Sprintf("Failed to convert API response to state: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Read database policy", map[string]interface{}{
		"policy_id": data.PolicyID.ValueString(),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.DatabasePolicyModel

	// Read Terraform plan data
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

	policyID := data.PolicyID.ValueString()

	// Convert new state to SDK (metadata only)
	updatedPolicy := data.ToSDK()

	// Build inline target databases (if provided)
	if len(data.TargetDatabase) > 0 {
		updatedPolicy.Targets = make(map[string]uapsiadbmodels.ArkUAPSIADBTargets)

		for i, targetDB := range data.TargetDatabase {
			// Check for unknown or null database workspace ID
			if targetDB.DatabaseWorkspaceID.IsNull() || targetDB.DatabaseWorkspaceID.IsUnknown() {
				resp.Diagnostics.AddError(
					"Unknown Database Workspace ID",
					fmt.Sprintf("target_databases[%d]: database_workspace_id is not yet known. "+
						"This typically occurs when the database workspace is created in the same Terraform configuration. "+
						"Ensure the database workspace resource is created before the policy resource references it.", i),
				)
				return
			}

			// Fetch database workspace to get instance details
			databaseID := targetDB.DatabaseWorkspaceID.ValueString()
			databaseIDInt, err := strconv.Atoi(databaseID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Database ID",
					fmt.Sprintf("target_databases[%d]: database_workspace_id must be a valid integer: %s", i, err.Error()),
				)
				return
			}

			database, err := r.providerData.SIAAPI.WorkspacesDB().Database(&dbmodels.ArkSIADBGetDatabase{
				ID: databaseIDInt,
			})
			if err != nil {
				resp.Diagnostics.Append(client.MapError(err, fmt.Sprintf("fetch database workspace for target_databases[%d]", i)))
				return
			}

			// Determine workspace type (always "FQDN/IP" for all databases)
			workspaceType := "FQDN/IP"

			// Build instance target with authentication profile
			instanceTarget, buildErr := buildInstanceTarget(ctx, database, targetDB)
			if buildErr != nil {
				resp.Diagnostics.AddError(
					"Failed to Build Target",
					fmt.Sprintf("target_databases[%d]: %s", i, buildErr.Error()),
				)
				return
			}

			// Add to targets map
			targets := updatedPolicy.Targets[workspaceType]
			targets.Instances = append(targets.Instances, *instanceTarget)
			updatedPolicy.Targets[workspaceType] = targets

			tflog.Debug(ctx, "Updated target database in policy", map[string]interface{}{
				"database_id":    databaseID,
				"workspace_type": workspaceType,
				"auth_method":    targetDB.AuthenticationMethod.ValueString(),
			})
		}
	}

	// Build inline principals (if provided)
	if len(data.Principal) > 0 {
		updatedPolicy.Principals = make([]uapcommonmodels.ArkUAPPrincipal, len(data.Principal))
		for i, principal := range data.Principal {
			updatedPolicy.Principals[i] = uapcommonmodels.ArkUAPPrincipal{
				ID:                  principal.PrincipalID.ValueString(),
				Name:                principal.PrincipalName.ValueString(),
				Type:                principal.PrincipalType.ValueString(),
				SourceDirectoryName: principal.SourceDirectoryName.ValueString(),
				SourceDirectoryID:   principal.SourceDirectoryID.ValueString(),
			}

			tflog.Debug(ctx, "Updated principal in policy", map[string]interface{}{
				"principal_id":   principal.PrincipalID.ValueString(),
				"principal_type": principal.PrincipalType.ValueString(),
			})
		}
	}

	// Update policy with retry logic
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		_, err := r.providerData.UAPClient.Db().UpdatePolicy(updatedPolicy)
		return err
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "update database policy"))
		return
	}

	// Fetch updated policy to get computed fields
	refreshedPolicy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "refresh policy after update"))
		return
	}

	// Update state with refreshed policy
	if err := data.FromSDK(ctx, refreshedPolicy); err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Policy Response",
			fmt.Sprintf("Failed to convert API response to state: %s", err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Updated database policy", map[string]interface{}{
		"policy_id":        data.PolicyID.ValueString(),
		"target_databases": len(data.TargetDatabase),
		"principals":       len(data.Principal),
	})

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatabasePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.DatabasePolicyModel

	// Read Terraform state
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

	// Delete policy with retry logic using workaround (ARK SDK v1.5.0 bug)
	// Note: API automatically cascades deletion to principals and targets
	err := client.RetryWithBackoff(ctx, &client.RetryConfig{
		MaxRetries: client.DefaultMaxRetries,
		BaseDelay:  client.BaseDelay,
		MaxDelay:   client.MaxDelay,
	}, func() error {
		return client.DeleteDatabasePolicyDirect(ctx, r.providerData.AuthContext, policyID)
	})

	if err != nil {
		// If already deleted, treat as success
		if client.IsNotFoundError(err) {
			tflog.Info(ctx, "Policy already deleted", map[string]interface{}{
				"policy_id": policyID,
			})
			return
		}

		resp.Diagnostics.Append(client.MapError(err, "delete database policy"))
		return
	}

	tflog.Info(ctx, "Deleted database policy", map[string]interface{}{
		"policy_id": policyID,
	})
}

func (r *DatabasePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by policy ID
	policyID := req.ID

	// Fetch policy from API
	policy, err := r.providerData.UAPClient.Db().Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
		PolicyID: policyID,
	})

	if err != nil {
		resp.Diagnostics.Append(client.MapError(err, "import database policy"))
		return
	}

	// Convert to state model
	var data models.DatabasePolicyModel
	if err := data.FromSDK(ctx, policy); err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Policy Response",
			fmt.Sprintf("Failed to convert API response to state: %s", err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Imported database policy", map[string]interface{}{
		"policy_id":   data.PolicyID.ValueString(),
		"policy_name": data.Name.ValueString(),
	})

	// Save imported state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mustCompileRegex compiles a regex pattern and panics if it fails (for use in validators)
func mustCompileRegex(pattern string) *regexp.Regexp {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Sprintf("failed to compile regex pattern %q: %v", pattern, err))
	}
	return re
}

// buildInstanceTarget creates an ArkUAPSIADBInstanceTarget from database workspace and assignment data
// This function leverages the profile factory to eliminate code duplication
func buildInstanceTarget(ctx context.Context, database *dbmodels.ArkSIADBDatabase, targetDB models.InlineDatabaseAssignmentModel) (*uapsiadbmodels.ArkUAPSIADBInstanceTarget, error) {
	authMethod := targetDB.AuthenticationMethod.ValueString()

	// Convert inline assignment model to standard assignment model for profile factory
	assignmentModel := convertInlineToAssignmentModel(&targetDB)

	// Build authentication profile using centralized factory
	var diagnostics diag.Diagnostics
	profile := BuildAuthenticationProfile(ctx, authMethod, assignmentModel, &diagnostics)

	// Check for errors from profile building
	if diagnostics.HasError() {
		// Extract first error message for return
		for _, d := range diagnostics.Errors() {
			return nil, fmt.Errorf("%s: %s", d.Summary(), d.Detail())
		}
		return nil, fmt.Errorf("failed to build authentication profile")
	}

	// Create instance target with database metadata
	instanceTarget := &uapsiadbmodels.ArkUAPSIADBInstanceTarget{
		InstanceName:         database.Name,
		InstanceType:         database.ProviderDetails.Family,
		InstanceID:           strconv.Itoa(database.ID),
		AuthenticationMethod: authMethod,
	}

	// Set profile on instance target using centralized setter
	if err := SetProfileOnInstanceTarget(instanceTarget, authMethod, profile); err != nil {
		return nil, err
	}

	return instanceTarget, nil
}

// convertInlineToAssignmentModel adapts an inline assignment model to the standard assignment model
// This allows reuse of the profile factory with inline target_database blocks
func convertInlineToAssignmentModel(inline *models.InlineDatabaseAssignmentModel) *models.DatabasePolicyWorkspaceAssignmentModel {
	return &models.DatabasePolicyWorkspaceAssignmentModel{
		DatabaseWorkspaceID:   inline.DatabaseWorkspaceID,
		AuthenticationMethod:  inline.AuthenticationMethod,
		DBAuthProfile:         inline.DBAuthProfile,
		LDAPAuthProfile:       inline.LDAPAuthProfile,
		OracleAuthProfile:     inline.OracleAuthProfile,
		MongoAuthProfile:      inline.MongoAuthProfile,
		SQLServerAuthProfile:  inline.SQLServerAuthProfile,
		RDSIAMUserAuthProfile: inline.RDSIAMUserAuthProfile,
		// PolicyID, ID, and LastModified are not needed for profile building
	}
}
