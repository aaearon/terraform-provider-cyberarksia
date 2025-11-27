// Package models defines Terraform state models
package models

import (
	"context"
	"fmt"
	"sort"
	"strings"

	uapcommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/common/models"
	uapsiacommonmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/common/models"
	uapsiadbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/db/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// stringValueOrNull returns types.StringNull() for empty strings, otherwise types.StringValue()
// This handles the case where the API returns null (which Go unmarshals to "") but the
// Terraform config has the field omitted (null). Without this, state would have "" while
// config has null, causing perpetual drift.
func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// changeInfoAttrTypes defines the attribute types for ChangeInfo objects (created_by, updated_on)
var changeInfoAttrTypes = map[string]attr.Type{
	"user":      types.StringType,
	"timestamp": types.StringType,
}

// ChangeInfoAttrTypes returns the attribute types for ChangeInfo objects
// Used for creating ObjectNull values in provider resource operations
func ChangeInfoAttrTypes() map[string]attr.Type {
	return changeInfoAttrTypes
}

// createChangeInfoObject creates a types.Object from user and timestamp strings
// Returns ObjectNull if user is empty, otherwise returns ObjectValue with the provided data
func createChangeInfoObject(user, timestamp string) types.Object {
	if user == "" {
		return types.ObjectNull(changeInfoAttrTypes)
	}

	attrs := map[string]attr.Value{
		"user":      types.StringValue(user),
		"timestamp": types.StringValue(timestamp),
	}

	objVal, diags := types.ObjectValue(changeInfoAttrTypes, attrs)
	if diags.HasError() {
		// Log error but return null object to avoid blocking operations
		// In practice, this should never happen with valid string inputs
		return types.ObjectNull(changeInfoAttrTypes)
	}

	return objVal
}

// DatabasePolicyModel represents the Terraform state for cyberarksia_database_policy resource
type DatabasePolicyModel struct {
	Conditions               *ConditionsModel                `tfsdk:"conditions"`
	TimeFrame                *TimeFrameModel                 `tfsdk:"time_frame"`
	PolicyTags               types.Set                       `tfsdk:"policy_tags"`
	UpdatedOn                types.Object                    `tfsdk:"updated_on"`
	CreatedBy                types.Object                    `tfsdk:"created_by"`
	ID                       types.String                    `tfsdk:"id"`
	PolicyID                 types.String                    `tfsdk:"policy_id"`
	Name                     types.String                    `tfsdk:"name"`
	Status                   types.String                    `tfsdk:"status"`
	DelegationClassification types.String                    `tfsdk:"delegation_classification"`
	Description              types.String                    `tfsdk:"description"`
	TimeZone                 types.String                    `tfsdk:"time_zone"`
	LastModified             types.String                    `tfsdk:"last_modified"`
	Principal                []InlinePrincipalModel          `tfsdk:"principal"`
	TargetDatabase           []InlineDatabaseAssignmentModel `tfsdk:"target_database"`
}

// TimeFrameModel represents policy validity period
type TimeFrameModel struct {
	FromTime types.String `tfsdk:"from_time"` // ISO 8601
	ToTime   types.String `tfsdk:"to_time"`   // ISO 8601
}

// ConditionsModel represents policy access conditions
type ConditionsModel struct {
	AccessWindow       *AccessWindowModel `tfsdk:"access_window"`        // 8 bytes (pointer)
	MaxSessionDuration types.Int64        `tfsdk:"max_session_duration"` // 8 bytes - 1-24 hours
	IdleTime           types.Int64        `tfsdk:"idle_time"`            // 8 bytes - 1-120 minutes, default 10
}

// AccessWindowModel represents time-based access restrictions
type AccessWindowModel struct {
	DaysOfTheWeek types.Set    `tfsdk:"days_of_the_week"` // Set of int, 0=Sunday through 6=Saturday (order automatically normalized)
	FromHour      types.String `tfsdk:"from_hour"`        // "HH:MM"
	ToHour        types.String `tfsdk:"to_hour"`          // "HH:MM"
}

// ChangeInfoModel represents user and timestamp for policy changes
type ChangeInfoModel struct {
	User      types.String `tfsdk:"user"`
	Timestamp types.String `tfsdk:"timestamp"` // ISO 8601
}

// InlineDatabaseAssignmentModel represents an inline target database assignment
type InlineDatabaseAssignmentModel struct {
	// Profile blocks (mutually exclusive based on authentication_method)
	DBAuthProfile         *DBAuthProfileModel         `tfsdk:"db_auth_profile"`           // 8 bytes (pointer)
	LDAPAuthProfile       *LDAPAuthProfileModel       `tfsdk:"ldap_auth_profile"`         // 8 bytes (pointer)
	OracleAuthProfile     *OracleAuthProfileModel     `tfsdk:"oracle_auth_profile"`       // 8 bytes (pointer)
	MongoAuthProfile      *MongoAuthProfileModel      `tfsdk:"mongo_auth_profile"`        // 8 bytes (pointer)
	SQLServerAuthProfile  *SQLServerAuthProfileModel  `tfsdk:"sqlserver_auth_profile"`    // 8 bytes (pointer)
	RDSIAMUserAuthProfile *RDSIAMUserAuthProfileModel `tfsdk:"rds_iam_user_auth_profile"` // 8 bytes (pointer)

	DatabaseWorkspaceID  types.String `tfsdk:"database_workspace_id"` // types.String
	AuthenticationMethod types.String `tfsdk:"authentication_method"` // types.String
}

// InlinePrincipalModel represents an inline principal assignment
type InlinePrincipalModel struct {
	PrincipalID         types.String `tfsdk:"principal_id"`
	PrincipalType       types.String `tfsdk:"principal_type"`
	PrincipalName       types.String `tfsdk:"principal_name"`
	SourceDirectoryName types.String `tfsdk:"source_directory_name"`
	SourceDirectoryID   types.String `tfsdk:"source_directory_id"`
}

// ToSDK converts Terraform state model to ARK SDK policy struct
func (m *DatabasePolicyModel) ToSDK() *uapsiadbmodels.ArkUAPSIADBAccessPolicy {
	policy := &uapsiadbmodels.ArkUAPSIADBAccessPolicy{
		ArkUAPSIACommonAccessPolicy: uapsiacommonmodels.ArkUAPSIACommonAccessPolicy{
			ArkUAPCommonAccessPolicy: uapcommonmodels.ArkUAPCommonAccessPolicy{
				Metadata: uapcommonmodels.ArkUAPMetadata{
					PolicyID:    m.PolicyID.ValueString(),
					Name:        m.Name.ValueString(),
					Description: m.Description.ValueString(),
					Status: uapcommonmodels.ArkUAPPolicyStatus{
						Status: m.Status.ValueString(),
					},
					PolicyEntitlement: uapcommonmodels.ArkUAPPolicyEntitlement{
						TargetCategory: "DB",
						LocationType:   "FQDN/IP",
						PolicyType:     "Recurring",
					},
					TimeZone: m.TimeZone.ValueString(),
				},
				// SIA currently only supports "unrestricted" for database policies
				// Send "unrestricted" regardless of configured value for now
				// When SIA supports this attribute, remove this override
				DelegationClassification: "unrestricted",
			},
		},
	}

	// Convert policy tags
	if !m.PolicyTags.IsNull() && !m.PolicyTags.IsUnknown() {
		var tags []string
		m.PolicyTags.ElementsAs(context.Background(), &tags, false)
		policy.Metadata.PolicyTags = tags
	}

	// Convert time frame
	if m.TimeFrame != nil {
		policy.Metadata.TimeFrame = uapcommonmodels.ArkUAPTimeFrame{
			FromTime: m.TimeFrame.FromTime.ValueString(),
			ToTime:   m.TimeFrame.ToTime.ValueString(),
		}
	}

	// Convert conditions
	if m.Conditions != nil {
		policy.Conditions = convertConditionsToSDK(m.Conditions)
	}

	return policy
}

// FromSDK populates Terraform state model from ARK SDK policy struct
func (m *DatabasePolicyModel) FromSDK(ctx context.Context, policy *uapsiadbmodels.ArkUAPSIADBAccessPolicy) error {
	m.ID = types.StringValue(policy.Metadata.PolicyID)
	m.PolicyID = types.StringValue(policy.Metadata.PolicyID)
	m.Name = types.StringValue(policy.Metadata.Name)
	m.Description = stringValueOrNull(policy.Metadata.Description)
	// Keep API values as-is (API returns "Active"/"Suspended" capitalized)
	// Normalize to lowercase to match user config (API returns titlecase)
	m.Status = types.StringValue(strings.ToLower(policy.Metadata.Status.Status))
	m.TimeZone = types.StringValue(policy.Metadata.TimeZone)
	// Normalize to lowercase to match user config (API returns titlecase)
	m.DelegationClassification = types.StringValue(strings.ToLower(policy.DelegationClassification))

	// Convert policy tags
	if len(policy.Metadata.PolicyTags) > 0 {
		tagValues := make([]attr.Value, len(policy.Metadata.PolicyTags))
		for i, tag := range policy.Metadata.PolicyTags {
			tagValues[i] = types.StringValue(tag)
		}
		tagSet, diags := types.SetValue(types.StringType, tagValues)
		if diags.HasError() {
			return fmt.Errorf("failed to convert policy tags: %v", diags.Errors())
		}
		m.PolicyTags = tagSet
	} else {
		m.PolicyTags = types.SetNull(types.StringType)
	}

	// Convert time frame
	if policy.Metadata.TimeFrame.FromTime != "" || policy.Metadata.TimeFrame.ToTime != "" {
		m.TimeFrame = &TimeFrameModel{
			FromTime: types.StringValue(policy.Metadata.TimeFrame.FromTime),
			ToTime:   types.StringValue(policy.Metadata.TimeFrame.ToTime),
		}
	}

	// Convert conditions
	m.Conditions = convertConditionsFromSDK(ctx, &policy.Conditions)

	// Convert inline principals from SDK
	m.Principal = make([]InlinePrincipalModel, 0, len(policy.Principals))
	for _, sdkPrincipal := range policy.Principals {
		m.Principal = append(m.Principal, InlinePrincipalModel{
			PrincipalID:         types.StringValue(sdkPrincipal.ID),
			PrincipalType:       types.StringValue(sdkPrincipal.Type),
			PrincipalName:       types.StringValue(sdkPrincipal.Name),
			SourceDirectoryName: stringValueOrNull(sdkPrincipal.SourceDirectoryName),
			SourceDirectoryID:   stringValueOrNull(sdkPrincipal.SourceDirectoryID),
		})
	}

	// Convert inline target databases from SDK
	m.TargetDatabase = make([]InlineDatabaseAssignmentModel, 0)
	for _, targets := range policy.Targets {
		for _, instance := range targets.Instances {
			// Parse the authentication profile back from SDK format
			assignment, err := parseInstanceTargetToAssignment(ctx, &instance)
			if err != nil {
				// Log error but continue - don't fail the entire read
				tflog.Warn(ctx, "Failed to parse instance target", map[string]interface{}{
					"instance_id": instance.InstanceID,
					"error":       err.Error(),
				})
				continue
			}
			m.TargetDatabase = append(m.TargetDatabase, *assignment)
		}
	}

	// Computed fields - convert to types.Object to handle unknown values properly
	m.CreatedBy = createChangeInfoObject(policy.Metadata.CreatedBy.User, policy.Metadata.CreatedBy.Time)
	m.UpdatedOn = createChangeInfoObject(policy.Metadata.UpdatedOn.User, policy.Metadata.UpdatedOn.Time)

	return nil
}

// convertConditionsToSDK converts Terraform conditions to SDK conditions
func convertConditionsToSDK(c *ConditionsModel) uapsiacommonmodels.ArkUAPSIACommonConditions {
	conditions := uapsiacommonmodels.ArkUAPSIACommonConditions{
		ArkUAPConditions: uapcommonmodels.ArkUAPConditions{
			MaxSessionDuration: int(c.MaxSessionDuration.ValueInt64()),
		},
		IdleTime: int(c.IdleTime.ValueInt64()),
	}

	// Convert access window if present
	if c.AccessWindow != nil {
		var days []int
		if !c.AccessWindow.DaysOfTheWeek.IsNull() && !c.AccessWindow.DaysOfTheWeek.IsUnknown() {
			// Convert set to slice
			var daysInt64 []int64
			c.AccessWindow.DaysOfTheWeek.ElementsAs(context.Background(), &daysInt64, false)

			// Sort to ensure canonical order (eliminates plan/state mismatch)
			sort.Slice(daysInt64, func(i, j int) bool { return daysInt64[i] < daysInt64[j] })

			// Convert []int64 to []int for SDK
			days = make([]int, len(daysInt64))
			for i, day := range daysInt64 {
				days[i] = int(day)
			}
		}

		conditions.AccessWindow = uapcommonmodels.ArkUAPTimeCondition{
			DaysOfTheWeek: days,
			FromHour:      c.AccessWindow.FromHour.ValueString(),
			ToHour:        c.AccessWindow.ToHour.ValueString(),
		}
	}

	return conditions
}

// convertConditionsFromSDK converts SDK conditions to Terraform conditions
func convertConditionsFromSDK(ctx context.Context, c *uapsiacommonmodels.ArkUAPSIACommonConditions) *ConditionsModel {
	if c == nil {
		return nil
	}

	conditions := &ConditionsModel{
		MaxSessionDuration: types.Int64Value(int64(c.MaxSessionDuration)),
		IdleTime:           types.Int64Value(int64(c.IdleTime)),
	}

	// Convert access window if present
	if len(c.AccessWindow.DaysOfTheWeek) > 0 || c.AccessWindow.FromHour != "" || c.AccessWindow.ToHour != "" {
		// Convert API response to slice
		daysInt64 := make([]int64, len(c.AccessWindow.DaysOfTheWeek))
		for i, day := range c.AccessWindow.DaysOfTheWeek {
			daysInt64[i] = int64(day)
		}

		// Sort to ensure canonical order (eliminates plan/state mismatch)
		sort.Slice(daysInt64, func(i, j int) bool { return daysInt64[i] < daysInt64[j] })

		// Create set from sorted days
		daysSet, _ := types.SetValueFrom(ctx, types.Int64Type, daysInt64)

		conditions.AccessWindow = &AccessWindowModel{
			DaysOfTheWeek: daysSet,
			FromHour:      stringValueOrNull(c.AccessWindow.FromHour),
			ToHour:        stringValueOrNull(c.AccessWindow.ToHour),
		}
	}

	return conditions
}

// parseInstanceTargetToAssignment converts an SDK instance target to an inline database assignment model
// This enables drift detection for inline target_database blocks
func parseInstanceTargetToAssignment(ctx context.Context, instance *uapsiadbmodels.ArkUAPSIADBInstanceTarget) (*InlineDatabaseAssignmentModel, error) {
	if instance == nil {
		return nil, fmt.Errorf("instance target is nil")
	}

	// Extract database workspace ID from InstanceID
	assignment := &InlineDatabaseAssignmentModel{
		DatabaseWorkspaceID:  types.StringValue(instance.InstanceID),
		AuthenticationMethod: types.StringValue(instance.AuthenticationMethod),
	}

	// Parse authentication profiles based on method
	// NOTE: For drift detection, we only populate basic profile information
	// Full profile drift is detected through separate policy_database_assignment resources
	// This simplified parsing allows us to detect database additions/removals in policies
	switch instance.AuthenticationMethod {
	case "db_auth":
		if instance.DBAuthProfile != nil {
			assignment.DBAuthProfile = &DBAuthProfileModel{
				Roles: convertStringSliceToSet(instance.DBAuthProfile.Roles),
			}
		}
	case "ldap_auth":
		if instance.LDAPAuthProfile != nil {
			assignment.LDAPAuthProfile = &LDAPAuthProfileModel{
				AssignGroups: convertStringSliceToSet(instance.LDAPAuthProfile.AssignGroups),
			}
		}
	case "oracle_auth":
		if instance.OracleAuthProfile != nil {
			assignment.OracleAuthProfile = &OracleAuthProfileModel{
				Roles:       convertStringSliceToSet(instance.OracleAuthProfile.Roles),
				DbaRole:     types.BoolValue(instance.OracleAuthProfile.DbaRole),
				SysdbaRole:  types.BoolValue(instance.OracleAuthProfile.SysdbaRole),
				SysoperRole: types.BoolValue(instance.OracleAuthProfile.SysoperRole),
			}
		}
	case "mongo_auth":
		if instance.MongoAuthProfile != nil {
			mongoProfile := &MongoAuthProfileModel{}
			if len(instance.MongoAuthProfile.GlobalBuiltinRoles) > 0 {
				mongoProfile.GlobalBuiltinRoles = convertStringSliceToSet(instance.MongoAuthProfile.GlobalBuiltinRoles)
			}
			if len(instance.MongoAuthProfile.DatabaseBuiltinRoles) > 0 {
				dbBuiltin, _ := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, instance.MongoAuthProfile.DatabaseBuiltinRoles)
				mongoProfile.DatabaseBuiltinRoles = dbBuiltin
			}
			if len(instance.MongoAuthProfile.DatabaseCustomRoles) > 0 {
				dbCustom, _ := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, instance.MongoAuthProfile.DatabaseCustomRoles)
				mongoProfile.DatabaseCustomRoles = dbCustom
			}
			assignment.MongoAuthProfile = mongoProfile
		}
	case "sqlserver_auth":
		if instance.SQLServerAuthProfile != nil {
			sqlProfile := &SQLServerAuthProfileModel{}
			if len(instance.SQLServerAuthProfile.GlobalBuiltinRoles) > 0 {
				sqlProfile.GlobalBuiltinRoles = convertStringSliceToSet(instance.SQLServerAuthProfile.GlobalBuiltinRoles)
			}
			if len(instance.SQLServerAuthProfile.GlobalCustomRoles) > 0 {
				sqlProfile.GlobalCustomRoles = convertStringSliceToSet(instance.SQLServerAuthProfile.GlobalCustomRoles)
			}
			if len(instance.SQLServerAuthProfile.DatabaseBuiltinRoles) > 0 {
				dbBuiltin, _ := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, instance.SQLServerAuthProfile.DatabaseBuiltinRoles)
				sqlProfile.DatabaseBuiltinRoles = dbBuiltin
			}
			if len(instance.SQLServerAuthProfile.DatabaseCustomRoles) > 0 {
				dbCustom, _ := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, instance.SQLServerAuthProfile.DatabaseCustomRoles)
				sqlProfile.DatabaseCustomRoles = dbCustom
			}
			assignment.SQLServerAuthProfile = sqlProfile
		}
	case "rds_iam_user_auth":
		if instance.RDSIAMUserAuthProfile != nil {
			assignment.RDSIAMUserAuthProfile = &RDSIAMUserAuthProfileModel{
				DBUser: types.StringValue(instance.RDSIAMUserAuthProfile.DBUser),
			}
		}
	default:
		tflog.Debug(ctx, "Unknown authentication method", map[string]interface{}{
			"method": instance.AuthenticationMethod,
		})
	}

	return assignment, nil
}

// convertStringSliceToSet converts a []string to types.Set for Terraform state
// Use for unordered collections like roles where order doesn't matter
func convertStringSliceToSet(values []string) types.Set {
	if len(values) == 0 {
		return types.SetNull(types.StringType)
	}

	attrs := make([]attr.Value, len(values))
	for i, v := range values {
		attrs[i] = types.StringValue(v)
	}

	set, _ := types.SetValue(types.StringType, attrs)
	return set
}
