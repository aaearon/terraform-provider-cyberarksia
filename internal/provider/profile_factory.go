// Package provider implements authentication profile factory for policy database assignments
package provider

import (
	"context"
	"fmt"

	"github.com/aaearon/terraform-provider-cyberark-sia/internal/models"
	uapsiadbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/db/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BuildAuthenticationProfile converts Terraform plan data to SDK instance target with profile
// This function centralizes the logic for building all 6 authentication profile types,
// eliminating duplication across Create/Read/Update methods.
//
// Parameters:
//   - ctx: Context for logging and diagnostics
//   - authMethod: Authentication method type (db_auth, ldap_auth, etc.)
//   - data: Terraform state/plan model containing profile data
//   - diagnostics: Diagnostic collection for validation errors
//
// Returns:
//   - Profile pointer to set on instanceTarget (nil if validation fails)
//   - Updates diagnostics with any errors encountered
func BuildAuthenticationProfile(
	ctx context.Context,
	authMethod string,
	data *models.DatabasePolicyWorkspaceAssignmentModel,
	diagnostics *diag.Diagnostics,
) interface{} {
	switch authMethod {
	case "db_auth":
		return buildDBAuthProfile(ctx, data, diagnostics)
	case "ldap_auth":
		return buildLDAPAuthProfile(ctx, data, diagnostics)
	case "oracle_auth":
		return buildOracleAuthProfile(ctx, data, diagnostics)
	case "mongo_auth":
		return buildMongoAuthProfile(ctx, data, diagnostics)
	case "sqlserver_auth":
		return buildSQLServerAuthProfile(ctx, data, diagnostics)
	case "rds_iam_user_auth":
		return buildRDSIAMUserAuthProfile(ctx, data, diagnostics)
	default:
		diagnostics.AddError(
			"Unsupported Authentication Method",
			fmt.Sprintf("Authentication method %q is not implemented", authMethod),
		)
		return nil
	}
}

// buildDBAuthProfile handles db_auth profile building
func buildDBAuthProfile(ctx context.Context, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) *uapsiadbmodels.ArkUAPSIADBDBAuthProfile {
	if data.DBAuthProfile == nil {
		diagnostics.AddError("Missing db_auth Profile", "db_auth_profile block is required when authentication_method is 'db_auth'")
		return nil
	}
	var roles []string
	diagnostics.Append(data.DBAuthProfile.Roles.ElementsAs(ctx, &roles, false)...)
	if diagnostics.HasError() {
		return nil
	}
	return &uapsiadbmodels.ArkUAPSIADBDBAuthProfile{Roles: roles}
}

// buildLDAPAuthProfile handles ldap_auth profile building
func buildLDAPAuthProfile(ctx context.Context, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) *uapsiadbmodels.ArkUAPSIADBLDAPAuthProfile {
	if data.LDAPAuthProfile == nil {
		diagnostics.AddError("Missing ldap_auth Profile", "ldap_auth_profile block is required when authentication_method is 'ldap_auth'")
		return nil
	}
	var assignGroups []string
	diagnostics.Append(data.LDAPAuthProfile.AssignGroups.ElementsAs(ctx, &assignGroups, false)...)
	if diagnostics.HasError() {
		return nil
	}
	return &uapsiadbmodels.ArkUAPSIADBLDAPAuthProfile{AssignGroups: assignGroups}
}

// buildOracleAuthProfile handles oracle_auth profile building
func buildOracleAuthProfile(ctx context.Context, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) *uapsiadbmodels.ArkUAPSIADBOracleAuthProfile {
	if data.OracleAuthProfile == nil {
		diagnostics.AddError("Missing oracle_auth Profile", "oracle_auth_profile block is required when authentication_method is 'oracle_auth'")
		return nil
	}
	var roles []string
	diagnostics.Append(data.OracleAuthProfile.Roles.ElementsAs(ctx, &roles, false)...)
	if diagnostics.HasError() {
		return nil
	}
	return &uapsiadbmodels.ArkUAPSIADBOracleAuthProfile{
		Roles:       roles,
		DbaRole:     data.OracleAuthProfile.DbaRole.ValueBool(),
		SysdbaRole:  data.OracleAuthProfile.SysdbaRole.ValueBool(),
		SysoperRole: data.OracleAuthProfile.SysoperRole.ValueBool(),
	}
}

// buildMongoAuthProfile handles mongo_auth profile building
func buildMongoAuthProfile(ctx context.Context, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) *uapsiadbmodels.ArkUAPSIADBMongoAuthProfile {
	if data.MongoAuthProfile == nil {
		diagnostics.AddError("Missing mongo_auth Profile", "mongo_auth_profile block is required when authentication_method is 'mongo_auth'")
		return nil
	}

	mongoProfile := &uapsiadbmodels.ArkUAPSIADBMongoAuthProfile{}

	// Global builtin roles
	if !data.MongoAuthProfile.GlobalBuiltinRoles.IsNull() {
		var globalRoles []string
		diagnostics.Append(data.MongoAuthProfile.GlobalBuiltinRoles.ElementsAs(ctx, &globalRoles, false)...)
		if diagnostics.HasError() {
			return nil
		}
		mongoProfile.GlobalBuiltinRoles = globalRoles
	}

	// Database builtin roles
	if !data.MongoAuthProfile.DatabaseBuiltinRoles.IsNull() {
		dbBuiltinRoles := make(map[string][]string)
		diagnostics.Append(data.MongoAuthProfile.DatabaseBuiltinRoles.ElementsAs(ctx, &dbBuiltinRoles, false)...)
		if diagnostics.HasError() {
			return nil
		}
		mongoProfile.DatabaseBuiltinRoles = dbBuiltinRoles
	}

	// Database custom roles
	if !data.MongoAuthProfile.DatabaseCustomRoles.IsNull() {
		dbCustomRoles := make(map[string][]string)
		diagnostics.Append(data.MongoAuthProfile.DatabaseCustomRoles.ElementsAs(ctx, &dbCustomRoles, false)...)
		if diagnostics.HasError() {
			return nil
		}
		mongoProfile.DatabaseCustomRoles = dbCustomRoles
	}

	return mongoProfile
}

// buildSQLServerAuthProfile handles sqlserver_auth profile building
func buildSQLServerAuthProfile(ctx context.Context, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) *uapsiadbmodels.ArkUAPSIADBSqlServerAuthProfile {
	if data.SQLServerAuthProfile == nil {
		diagnostics.AddError("Missing sqlserver_auth Profile", "sqlserver_auth_profile block is required when authentication_method is 'sqlserver_auth'")
		return nil
	}

	sqlProfile := &uapsiadbmodels.ArkUAPSIADBSqlServerAuthProfile{}

	// Global builtin roles
	if !data.SQLServerAuthProfile.GlobalBuiltinRoles.IsNull() {
		var globalBuiltin []string
		diagnostics.Append(data.SQLServerAuthProfile.GlobalBuiltinRoles.ElementsAs(ctx, &globalBuiltin, false)...)
		if diagnostics.HasError() {
			return nil
		}
		sqlProfile.GlobalBuiltinRoles = globalBuiltin
	}

	// Global custom roles
	if !data.SQLServerAuthProfile.GlobalCustomRoles.IsNull() {
		var globalCustom []string
		diagnostics.Append(data.SQLServerAuthProfile.GlobalCustomRoles.ElementsAs(ctx, &globalCustom, false)...)
		if diagnostics.HasError() {
			return nil
		}
		sqlProfile.GlobalCustomRoles = globalCustom
	}

	// Database builtin roles
	if !data.SQLServerAuthProfile.DatabaseBuiltinRoles.IsNull() {
		dbBuiltin := make(map[string][]string)
		diagnostics.Append(data.SQLServerAuthProfile.DatabaseBuiltinRoles.ElementsAs(ctx, &dbBuiltin, false)...)
		if diagnostics.HasError() {
			return nil
		}
		sqlProfile.DatabaseBuiltinRoles = dbBuiltin
	}

	// Database custom roles
	if !data.SQLServerAuthProfile.DatabaseCustomRoles.IsNull() {
		dbCustom := make(map[string][]string)
		diagnostics.Append(data.SQLServerAuthProfile.DatabaseCustomRoles.ElementsAs(ctx, &dbCustom, false)...)
		if diagnostics.HasError() {
			return nil
		}
		sqlProfile.DatabaseCustomRoles = dbCustom
	}

	return sqlProfile
}

// buildRDSIAMUserAuthProfile handles rds_iam_user_auth profile building
func buildRDSIAMUserAuthProfile(_ context.Context, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) *uapsiadbmodels.ArkUAPSIADBRDSIAMUserAuthProfile {
	if data.RDSIAMUserAuthProfile == nil {
		diagnostics.AddError("Missing rds_iam_user_auth Profile", "rds_iam_user_auth_profile block is required when authentication_method is 'rds_iam_user_auth'")
		return nil
	}
	return &uapsiadbmodels.ArkUAPSIADBRDSIAMUserAuthProfile{
		DBUser: data.RDSIAMUserAuthProfile.DBUser.ValueString(),
	}
}

// SetProfileOnInstanceTarget sets the appropriate profile field on instanceTarget based on auth method
// This is a helper to cleanly set the profile after building it
// Returns an error if profile type doesn't match the auth method (indicates a programming error)
func SetProfileOnInstanceTarget(
	instanceTarget *uapsiadbmodels.ArkUAPSIADBInstanceTarget,
	authMethod string,
	profile interface{},
) error {
	// Clear all profiles first
	instanceTarget.DBAuthProfile = nil
	instanceTarget.LDAPAuthProfile = nil
	instanceTarget.OracleAuthProfile = nil
	instanceTarget.MongoAuthProfile = nil
	instanceTarget.SQLServerAuthProfile = nil
	instanceTarget.RDSIAMUserAuthProfile = nil

	// Set the appropriate one
	// Type assertions should never fail if BuildAuthenticationProfile works correctly
	// Return error on mismatch so it can be handled gracefully
	switch authMethod {
	case "db_auth":
		p, ok := profile.(*uapsiadbmodels.ArkUAPSIADBDBAuthProfile)
		if !ok {
			return fmt.Errorf("internal error: profile type mismatch for db_auth - got %T, expected *ArkUAPSIADBDBAuthProfile. Please report this issue to the provider developers", profile)
		}
		instanceTarget.DBAuthProfile = p
	case "ldap_auth":
		p, ok := profile.(*uapsiadbmodels.ArkUAPSIADBLDAPAuthProfile)
		if !ok {
			return fmt.Errorf("internal error: profile type mismatch for ldap_auth - got %T, expected *ArkUAPSIADBLDAPAuthProfile. Please report this issue to the provider developers", profile)
		}
		instanceTarget.LDAPAuthProfile = p
	case "oracle_auth":
		p, ok := profile.(*uapsiadbmodels.ArkUAPSIADBOracleAuthProfile)
		if !ok {
			return fmt.Errorf("internal error: profile type mismatch for oracle_auth - got %T, expected *ArkUAPSIADBOracleAuthProfile. Please report this issue to the provider developers", profile)
		}
		instanceTarget.OracleAuthProfile = p
	case "mongo_auth":
		p, ok := profile.(*uapsiadbmodels.ArkUAPSIADBMongoAuthProfile)
		if !ok {
			return fmt.Errorf("internal error: profile type mismatch for mongo_auth - got %T, expected *ArkUAPSIADBMongoAuthProfile. Please report this issue to the provider developers", profile)
		}
		instanceTarget.MongoAuthProfile = p
	case "sqlserver_auth":
		p, ok := profile.(*uapsiadbmodels.ArkUAPSIADBSqlServerAuthProfile)
		if !ok {
			return fmt.Errorf("internal error: profile type mismatch for sqlserver_auth - got %T, expected *ArkUAPSIADBSqlServerAuthProfile. Please report this issue to the provider developers", profile)
		}
		instanceTarget.SQLServerAuthProfile = p
	case "rds_iam_user_auth":
		p, ok := profile.(*uapsiadbmodels.ArkUAPSIADBRDSIAMUserAuthProfile)
		if !ok {
			return fmt.Errorf("internal error: profile type mismatch for rds_iam_user_auth - got %T, expected *ArkUAPSIADBRDSIAMUserAuthProfile. Please report this issue to the provider developers", profile)
		}
		instanceTarget.RDSIAMUserAuthProfile = p
	}
	return nil
}

// ParseAuthenticationProfile converts SDK instance target profile back to Terraform state
// This function centralizes the logic for parsing all 6 authentication profile types,
// eliminating duplication in the Read() method.
//
// Parameters:
//   - ctx: Context for type conversion
//   - target: SDK instance target containing the profile
//   - data: Terraform state model to populate
//   - diagnostics: Diagnostic collection for conversion errors
//
// Returns:
//   - Updates data with parsed profile information
//   - Updates diagnostics with any errors encountered
func ParseAuthenticationProfile(
	ctx context.Context,
	target *uapsiadbmodels.ArkUAPSIADBInstanceTarget,
	data *models.DatabasePolicyWorkspaceAssignmentModel,
	diagnostics *diag.Diagnostics,
) {
	// CRITICAL: Clear all profile pointers before parsing to prevent stale data
	// When authentication method changes (e.g., db_auth → ldap_auth), the old
	// profile pointer must be removed to avoid perpetual Terraform diffs
	data.DBAuthProfile = nil
	data.LDAPAuthProfile = nil
	data.OracleAuthProfile = nil
	data.MongoAuthProfile = nil
	data.SQLServerAuthProfile = nil
	data.RDSIAMUserAuthProfile = nil

	switch target.AuthenticationMethod {
	case "db_auth":
		parseDBAuthProfile(ctx, target, data, diagnostics)
	case "ldap_auth":
		parseLDAPAuthProfile(ctx, target, data, diagnostics)
	case "oracle_auth":
		parseOracleAuthProfile(ctx, target, data, diagnostics)
	case "mongo_auth":
		parseMongoAuthProfile(ctx, target, data, diagnostics)
	case "sqlserver_auth":
		parseSQLServerAuthProfile(ctx, target, data, diagnostics)
	case "rds_iam_user_auth":
		parseRDSIAMUserAuthProfile(ctx, target, data)
	}
}

// parseDBAuthProfile handles db_auth profile parsing
func parseDBAuthProfile(ctx context.Context, target *uapsiadbmodels.ArkUAPSIADBInstanceTarget, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) {
	if target.DBAuthProfile != nil {
		rolesList, diags := types.ListValueFrom(ctx, types.StringType, target.DBAuthProfile.Roles)
		diagnostics.Append(diags...)
		if !diagnostics.HasError() {
			data.DBAuthProfile = &models.DBAuthProfileModel{Roles: rolesList}
		}
	}
}

// parseLDAPAuthProfile handles ldap_auth profile parsing
func parseLDAPAuthProfile(ctx context.Context, target *uapsiadbmodels.ArkUAPSIADBInstanceTarget, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) {
	if target.LDAPAuthProfile != nil {
		assignGroupsList, diags := types.ListValueFrom(ctx, types.StringType, target.LDAPAuthProfile.AssignGroups)
		diagnostics.Append(diags...)
		if !diagnostics.HasError() {
			data.LDAPAuthProfile = &models.LDAPAuthProfileModel{AssignGroups: assignGroupsList}
		}
	}
}

// parseOracleAuthProfile handles oracle_auth profile parsing
func parseOracleAuthProfile(ctx context.Context, target *uapsiadbmodels.ArkUAPSIADBInstanceTarget, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) {
	if target.OracleAuthProfile != nil {
		rolesList, diags := types.ListValueFrom(ctx, types.StringType, target.OracleAuthProfile.Roles)
		diagnostics.Append(diags...)
		if !diagnostics.HasError() {
			data.OracleAuthProfile = &models.OracleAuthProfileModel{
				Roles:       rolesList,
				DbaRole:     types.BoolValue(target.OracleAuthProfile.DbaRole),
				SysdbaRole:  types.BoolValue(target.OracleAuthProfile.SysdbaRole),
				SysoperRole: types.BoolValue(target.OracleAuthProfile.SysoperRole),
			}
		}
	}
}

// parseMongoAuthProfile handles mongo_auth profile parsing
func parseMongoAuthProfile(ctx context.Context, target *uapsiadbmodels.ArkUAPSIADBInstanceTarget, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) {
	if target.MongoAuthProfile != nil {
		mongoModel := &models.MongoAuthProfileModel{}

		// Global builtin roles
		if len(target.MongoAuthProfile.GlobalBuiltinRoles) > 0 {
			globalList, diags := types.ListValueFrom(ctx, types.StringType, target.MongoAuthProfile.GlobalBuiltinRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			mongoModel.GlobalBuiltinRoles = globalList
		}

		// Database builtin roles
		if len(target.MongoAuthProfile.DatabaseBuiltinRoles) > 0 {
			dbBuiltinMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, target.MongoAuthProfile.DatabaseBuiltinRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			mongoModel.DatabaseBuiltinRoles = dbBuiltinMap
		}

		// Database custom roles
		if len(target.MongoAuthProfile.DatabaseCustomRoles) > 0 {
			dbCustomMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, target.MongoAuthProfile.DatabaseCustomRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			mongoModel.DatabaseCustomRoles = dbCustomMap
		}

		data.MongoAuthProfile = mongoModel
	}
}

// parseSQLServerAuthProfile handles sqlserver_auth profile parsing
func parseSQLServerAuthProfile(ctx context.Context, target *uapsiadbmodels.ArkUAPSIADBInstanceTarget, data *models.DatabasePolicyWorkspaceAssignmentModel, diagnostics *diag.Diagnostics) {
	if target.SQLServerAuthProfile != nil {
		sqlModel := &models.SQLServerAuthProfileModel{}

		// Global builtin roles
		if len(target.SQLServerAuthProfile.GlobalBuiltinRoles) > 0 {
			globalBuiltinList, diags := types.ListValueFrom(ctx, types.StringType, target.SQLServerAuthProfile.GlobalBuiltinRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			sqlModel.GlobalBuiltinRoles = globalBuiltinList
		}

		// Global custom roles
		if len(target.SQLServerAuthProfile.GlobalCustomRoles) > 0 {
			globalCustomList, diags := types.ListValueFrom(ctx, types.StringType, target.SQLServerAuthProfile.GlobalCustomRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			sqlModel.GlobalCustomRoles = globalCustomList
		}

		// Database builtin roles
		if len(target.SQLServerAuthProfile.DatabaseBuiltinRoles) > 0 {
			dbBuiltinMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, target.SQLServerAuthProfile.DatabaseBuiltinRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			sqlModel.DatabaseBuiltinRoles = dbBuiltinMap
		}

		// Database custom roles
		if len(target.SQLServerAuthProfile.DatabaseCustomRoles) > 0 {
			dbCustomMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, target.SQLServerAuthProfile.DatabaseCustomRoles)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			sqlModel.DatabaseCustomRoles = dbCustomMap
		}

		data.SQLServerAuthProfile = sqlModel
	}
}

// parseRDSIAMUserAuthProfile handles rds_iam_user_auth profile parsing
func parseRDSIAMUserAuthProfile(_ context.Context, target *uapsiadbmodels.ArkUAPSIADBInstanceTarget, data *models.DatabasePolicyWorkspaceAssignmentModel) {
	if target.RDSIAMUserAuthProfile != nil {
		data.RDSIAMUserAuthProfile = &models.RDSIAMUserAuthProfileModel{
			DBUser: types.StringValue(target.RDSIAMUserAuthProfile.DBUser),
		}
	}
}
