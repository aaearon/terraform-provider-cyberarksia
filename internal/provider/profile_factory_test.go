package provider

import (
	"context"
	"testing"

	"github.com/aaearon/terraform-provider-cyberarksia/internal/models"
	uapsiadbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/uap/sia/db/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestBuildAuthenticationProfile_DBAuth tests building a db_auth profile
func TestBuildAuthenticationProfile_DBAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data with db_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		DBAuthProfile: &models.DBAuthProfileModel{
			Roles: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("db_reader"),
				types.StringValue("db_writer"),
			}),
		},
	}

	// Build profile
	profile := BuildAuthenticationProfile(ctx, "db_auth", data, &diags)

	// Verify no errors
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags.Errors())
	}

	// Verify profile type
	dbProfile, ok := profile.(*uapsiadbmodels.ArkUAPSIADBDBAuthProfile)
	if !ok {
		t.Fatalf("Expected *ArkUAPSIADBDBAuthProfile, got %T", profile)
	}

	// Verify roles
	if len(dbProfile.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(dbProfile.Roles))
	}
	if dbProfile.Roles[0] != "db_reader" {
		t.Errorf("Expected first role 'db_reader', got %q", dbProfile.Roles[0])
	}
	if dbProfile.Roles[1] != "db_writer" {
		t.Errorf("Expected second role 'db_writer', got %q", dbProfile.Roles[1])
	}
}

// TestBuildAuthenticationProfile_DBAuth_Missing tests error when db_auth profile is missing
func TestBuildAuthenticationProfile_DBAuth_Missing(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data WITHOUT db_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		DBAuthProfile: nil,
	}

	// Build profile
	_ = BuildAuthenticationProfile(ctx, "db_auth", data, &diags)

	// Verify error occurred
	if !diags.HasError() {
		t.Fatal("Expected error for missing db_auth profile, got none")
	}

	// Verify error message contains expected text
	errors := diags.Errors()
	if len(errors) == 0 {
		t.Fatal("Expected error diagnostic, got none")
	}

	errorSummary := errors[0].Summary()
	if errorSummary != "Missing db_auth Profile" {
		t.Errorf("Expected error summary 'Missing db_auth Profile', got %q", errorSummary)
	}

	// Profile should be nil on error (already verified by error check above)
}

// TestBuildAuthenticationProfile_LDAPAuth tests building an ldap_auth profile
func TestBuildAuthenticationProfile_LDAPAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data with ldap_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		LDAPAuthProfile: &models.LDAPAuthProfileModel{
			AssignGroups: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("admins"),
				types.StringValue("developers"),
			}),
		},
	}

	// Build profile
	profile := BuildAuthenticationProfile(ctx, "ldap_auth", data, &diags)

	// Verify no errors
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags.Errors())
	}

	// Verify profile type
	ldapProfile, ok := profile.(*uapsiadbmodels.ArkUAPSIADBLDAPAuthProfile)
	if !ok {
		t.Fatalf("Expected *ArkUAPSIADBLDAPAuthProfile, got %T", profile)
	}

	// Verify groups
	if len(ldapProfile.AssignGroups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(ldapProfile.AssignGroups))
	}
	if ldapProfile.AssignGroups[0] != "admins" {
		t.Errorf("Expected first group 'admins', got %q", ldapProfile.AssignGroups[0])
	}
}

// TestBuildAuthenticationProfile_UnsupportedMethod tests error for unsupported auth method
func TestBuildAuthenticationProfile_UnsupportedMethod(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	data := &models.DatabasePolicyWorkspaceAssignmentModel{}

	// Build profile with invalid method
	_ = BuildAuthenticationProfile(ctx, "invalid_method", data, &diags)

	// Verify error occurred
	if !diags.HasError() {
		t.Fatal("Expected error for unsupported auth method, got none")
	}

	// Verify error message
	errors := diags.Errors()
	if len(errors) == 0 {
		t.Fatal("Expected error diagnostic, got none")
	}

	errorSummary := errors[0].Summary()
	if errorSummary != "Unsupported Authentication Method" {
		t.Errorf("Expected error summary 'Unsupported Authentication Method', got %q", errorSummary)
	}

	// Profile should be nil on error (already verified by error check above)
}

// TestSetProfileOnInstanceTarget_DBAuth tests setting db_auth profile on instance target
func TestSetProfileOnInstanceTarget_DBAuth(t *testing.T) {
	// Create instance target
	instanceTarget := &uapsiadbmodels.ArkUAPSIADBInstanceTarget{
		InstanceName:         "test-db",
		InstanceType:         "Postgres",
		AuthenticationMethod: "db_auth",
	}

	// Create profile
	profile := &uapsiadbmodels.ArkUAPSIADBDBAuthProfile{
		Roles: []string{"db_reader", "db_writer"},
	}

	// Set profile
	err := SetProfileOnInstanceTarget(instanceTarget, "db_auth", profile)
	if err != nil {
		t.Fatalf("SetProfileOnInstanceTarget failed: %v", err)
	}

	// Verify profile was set
	if instanceTarget.DBAuthProfile == nil {
		t.Fatal("Expected DBAuthProfile to be set, got nil")
	}

	if len(instanceTarget.DBAuthProfile.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(instanceTarget.DBAuthProfile.Roles))
	}

	// Verify other profiles are nil
	if instanceTarget.LDAPAuthProfile != nil {
		t.Error("Expected LDAPAuthProfile to be nil")
	}
	if instanceTarget.OracleAuthProfile != nil {
		t.Error("Expected OracleAuthProfile to be nil")
	}
	if instanceTarget.MongoAuthProfile != nil {
		t.Error("Expected MongoAuthProfile to be nil")
	}
	if instanceTarget.SQLServerAuthProfile != nil {
		t.Error("Expected SQLServerAuthProfile to be nil")
	}
	if instanceTarget.RDSIAMUserAuthProfile != nil {
		t.Error("Expected RDSIAMUserAuthProfile to be nil")
	}
}

// TestSetProfileOnInstanceTarget_ClearsOtherProfiles tests that setting a profile clears others
func TestSetProfileOnInstanceTarget_ClearsOtherProfiles(t *testing.T) {
	// Create instance target with existing profiles
	instanceTarget := &uapsiadbmodels.ArkUAPSIADBInstanceTarget{
		InstanceName:         "test-db",
		InstanceType:         "Postgres",
		AuthenticationMethod: "ldap_auth",
		DBAuthProfile:        &uapsiadbmodels.ArkUAPSIADBDBAuthProfile{Roles: []string{"old"}},
		LDAPAuthProfile:      &uapsiadbmodels.ArkUAPSIADBLDAPAuthProfile{AssignGroups: []string{"old"}},
	}

	// Create new LDAP profile
	profile := &uapsiadbmodels.ArkUAPSIADBLDAPAuthProfile{
		AssignGroups: []string{"new_group"},
	}

	// Set profile (should clear db_auth and update ldap_auth)
	err := SetProfileOnInstanceTarget(instanceTarget, "ldap_auth", profile)
	if err != nil {
		t.Fatalf("SetProfileOnInstanceTarget failed: %v", err)
	}

	// Verify old DB profile was cleared
	if instanceTarget.DBAuthProfile != nil {
		t.Error("Expected DBAuthProfile to be cleared, but it wasn't")
	}

	// Verify LDAP profile was set
	if instanceTarget.LDAPAuthProfile == nil {
		t.Fatal("Expected LDAPAuthProfile to be set, got nil")
	}

	if len(instanceTarget.LDAPAuthProfile.AssignGroups) != 1 {
		t.Errorf("Expected 1 group, got %d", len(instanceTarget.LDAPAuthProfile.AssignGroups))
	}

	if instanceTarget.LDAPAuthProfile.AssignGroups[0] != "new_group" {
		t.Errorf("Expected group 'new_group', got %q", instanceTarget.LDAPAuthProfile.AssignGroups[0])
	}
}

// TestBuildAuthenticationProfile_OracleAuth tests building an oracle_auth profile
func TestBuildAuthenticationProfile_OracleAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data with oracle_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		OracleAuthProfile: &models.OracleAuthProfileModel{
			Roles: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("oracle_reader"),
				types.StringValue("oracle_writer"),
			}),
			DbaRole:     types.BoolValue(true),
			SysdbaRole:  types.BoolValue(false),
			SysoperRole: types.BoolValue(true),
		},
	}

	// Build profile
	profile := BuildAuthenticationProfile(ctx, "oracle_auth", data, &diags)

	// Verify no errors
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags.Errors())
	}

	// Verify profile type
	oracleProfile, ok := profile.(*uapsiadbmodels.ArkUAPSIADBOracleAuthProfile)
	if !ok {
		t.Fatalf("Expected *ArkUAPSIADBOracleAuthProfile, got %T", profile)
	}

	// Verify roles
	if len(oracleProfile.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(oracleProfile.Roles))
	}
	if oracleProfile.Roles[0] != "oracle_reader" {
		t.Errorf("Expected first role 'oracle_reader', got %q", oracleProfile.Roles[0])
	}
	if oracleProfile.Roles[1] != "oracle_writer" {
		t.Errorf("Expected second role 'oracle_writer', got %q", oracleProfile.Roles[1])
	}

	// Verify boolean flags
	if !oracleProfile.DbaRole {
		t.Error("Expected DbaRole to be true")
	}
	if oracleProfile.SysdbaRole {
		t.Error("Expected SysdbaRole to be false")
	}
	if !oracleProfile.SysoperRole {
		t.Error("Expected SysoperRole to be true")
	}
}

// TestBuildAuthenticationProfile_OracleAuth_Missing tests error when oracle_auth profile is missing
func TestBuildAuthenticationProfile_OracleAuth_Missing(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data WITHOUT oracle_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		OracleAuthProfile: nil,
	}

	// Build profile
	_ = BuildAuthenticationProfile(ctx, "oracle_auth", data, &diags)

	// Verify error occurred
	if !diags.HasError() {
		t.Fatal("Expected error for missing oracle_auth profile, got none")
	}

	// Verify error message contains expected text
	errors := diags.Errors()
	if len(errors) == 0 {
		t.Fatal("Expected error diagnostic, got none")
	}

	errorSummary := errors[0].Summary()
	if errorSummary != "Missing oracle_auth Profile" {
		t.Errorf("Expected error summary 'Missing oracle_auth Profile', got %q", errorSummary)
	}
}

// TestBuildAuthenticationProfile_MongoAuth tests building a mongo_auth profile
func TestBuildAuthenticationProfile_MongoAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data with mongo_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		MongoAuthProfile: &models.MongoAuthProfileModel{
			GlobalBuiltinRoles: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("readAnyDatabase"),
				types.StringValue("readWriteAnyDatabase"),
			}),
			DatabaseBuiltinRoles: types.MapValueMust(
				types.ListType{ElemType: types.StringType},
				map[string]attr.Value{
					"db1": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("read"),
						types.StringValue("readWrite"),
					}),
				},
			),
			DatabaseCustomRoles: types.MapValueMust(
				types.ListType{ElemType: types.StringType},
				map[string]attr.Value{
					"db2": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("customRole1"),
					}),
				},
			),
		},
	}

	// Build profile
	profile := BuildAuthenticationProfile(ctx, "mongo_auth", data, &diags)

	// Verify no errors
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags.Errors())
	}

	// Verify profile type
	mongoProfile, ok := profile.(*uapsiadbmodels.ArkUAPSIADBMongoAuthProfile)
	if !ok {
		t.Fatalf("Expected *ArkUAPSIADBMongoAuthProfile, got %T", profile)
	}

	// Verify global builtin roles
	if len(mongoProfile.GlobalBuiltinRoles) != 2 {
		t.Errorf("Expected 2 global builtin roles, got %d", len(mongoProfile.GlobalBuiltinRoles))
	}
	if mongoProfile.GlobalBuiltinRoles[0] != "readAnyDatabase" {
		t.Errorf("Expected first global role 'readAnyDatabase', got %q", mongoProfile.GlobalBuiltinRoles[0])
	}

	// Verify database builtin roles
	if len(mongoProfile.DatabaseBuiltinRoles) != 1 {
		t.Errorf("Expected 1 database builtin role entry, got %d", len(mongoProfile.DatabaseBuiltinRoles))
	}
	if db1Roles, ok := mongoProfile.DatabaseBuiltinRoles["db1"]; ok {
		if len(db1Roles) != 2 {
			t.Errorf("Expected 2 roles for db1, got %d", len(db1Roles))
		}
	} else {
		t.Error("Expected 'db1' in DatabaseBuiltinRoles")
	}

	// Verify database custom roles
	if len(mongoProfile.DatabaseCustomRoles) != 1 {
		t.Errorf("Expected 1 database custom role entry, got %d", len(mongoProfile.DatabaseCustomRoles))
	}
	if db2Roles, ok := mongoProfile.DatabaseCustomRoles["db2"]; ok {
		if len(db2Roles) != 1 {
			t.Errorf("Expected 1 role for db2, got %d", len(db2Roles))
		}
	} else {
		t.Error("Expected 'db2' in DatabaseCustomRoles")
	}
}

// TestBuildAuthenticationProfile_MongoAuth_Missing tests error when mongo_auth profile is missing
func TestBuildAuthenticationProfile_MongoAuth_Missing(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data WITHOUT mongo_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		MongoAuthProfile: nil,
	}

	// Build profile
	_ = BuildAuthenticationProfile(ctx, "mongo_auth", data, &diags)

	// Verify error occurred
	if !diags.HasError() {
		t.Fatal("Expected error for missing mongo_auth profile, got none")
	}

	// Verify error message contains expected text
	errors := diags.Errors()
	if len(errors) == 0 {
		t.Fatal("Expected error diagnostic, got none")
	}

	errorSummary := errors[0].Summary()
	if errorSummary != "Missing mongo_auth Profile" {
		t.Errorf("Expected error summary 'Missing mongo_auth Profile', got %q", errorSummary)
	}
}

// TestBuildAuthenticationProfile_SQLServerAuth tests building a sqlserver_auth profile
func TestBuildAuthenticationProfile_SQLServerAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data with sqlserver_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		SQLServerAuthProfile: &models.SQLServerAuthProfileModel{
			GlobalBuiltinRoles: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("sysadmin"),
				types.StringValue("serveradmin"),
			}),
			GlobalCustomRoles: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("customGlobalRole"),
			}),
			DatabaseBuiltinRoles: types.MapValueMust(
				types.ListType{ElemType: types.StringType},
				map[string]attr.Value{
					"db1": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("db_owner"),
						types.StringValue("db_datareader"),
					}),
				},
			),
			DatabaseCustomRoles: types.MapValueMust(
				types.ListType{ElemType: types.StringType},
				map[string]attr.Value{
					"db2": types.ListValueMust(types.StringType, []attr.Value{
						types.StringValue("customDBRole"),
					}),
				},
			),
		},
	}

	// Build profile
	profile := BuildAuthenticationProfile(ctx, "sqlserver_auth", data, &diags)

	// Verify no errors
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags.Errors())
	}

	// Verify profile type
	sqlProfile, ok := profile.(*uapsiadbmodels.ArkUAPSIADBSqlServerAuthProfile)
	if !ok {
		t.Fatalf("Expected *ArkUAPSIADBSqlServerAuthProfile, got %T", profile)
	}

	// Verify global builtin roles
	if len(sqlProfile.GlobalBuiltinRoles) != 2 {
		t.Errorf("Expected 2 global builtin roles, got %d", len(sqlProfile.GlobalBuiltinRoles))
	}
	if sqlProfile.GlobalBuiltinRoles[0] != "sysadmin" {
		t.Errorf("Expected first global builtin role 'sysadmin', got %q", sqlProfile.GlobalBuiltinRoles[0])
	}

	// Verify global custom roles
	if len(sqlProfile.GlobalCustomRoles) != 1 {
		t.Errorf("Expected 1 global custom role, got %d", len(sqlProfile.GlobalCustomRoles))
	}
	if sqlProfile.GlobalCustomRoles[0] != "customGlobalRole" {
		t.Errorf("Expected global custom role 'customGlobalRole', got %q", sqlProfile.GlobalCustomRoles[0])
	}

	// Verify database builtin roles
	if len(sqlProfile.DatabaseBuiltinRoles) != 1 {
		t.Errorf("Expected 1 database builtin role entry, got %d", len(sqlProfile.DatabaseBuiltinRoles))
	}
	if db1Roles, ok := sqlProfile.DatabaseBuiltinRoles["db1"]; ok {
		if len(db1Roles) != 2 {
			t.Errorf("Expected 2 roles for db1, got %d", len(db1Roles))
		}
	} else {
		t.Error("Expected 'db1' in DatabaseBuiltinRoles")
	}

	// Verify database custom roles
	if len(sqlProfile.DatabaseCustomRoles) != 1 {
		t.Errorf("Expected 1 database custom role entry, got %d", len(sqlProfile.DatabaseCustomRoles))
	}
	if db2Roles, ok := sqlProfile.DatabaseCustomRoles["db2"]; ok {
		if len(db2Roles) != 1 {
			t.Errorf("Expected 1 role for db2, got %d", len(db2Roles))
		}
	} else {
		t.Error("Expected 'db2' in DatabaseCustomRoles")
	}
}

// TestBuildAuthenticationProfile_SQLServerAuth_Missing tests error when sqlserver_auth profile is missing
func TestBuildAuthenticationProfile_SQLServerAuth_Missing(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data WITHOUT sqlserver_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		SQLServerAuthProfile: nil,
	}

	// Build profile
	_ = BuildAuthenticationProfile(ctx, "sqlserver_auth", data, &diags)

	// Verify error occurred
	if !diags.HasError() {
		t.Fatal("Expected error for missing sqlserver_auth profile, got none")
	}

	// Verify error message contains expected text
	errors := diags.Errors()
	if len(errors) == 0 {
		t.Fatal("Expected error diagnostic, got none")
	}

	errorSummary := errors[0].Summary()
	if errorSummary != "Missing sqlserver_auth Profile" {
		t.Errorf("Expected error summary 'Missing sqlserver_auth Profile', got %q", errorSummary)
	}
}

// TestBuildAuthenticationProfile_RDSIAMUserAuth tests building an rds_iam_user_auth profile
func TestBuildAuthenticationProfile_RDSIAMUserAuth(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data with rds_iam_user_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		RDSIAMUserAuthProfile: &models.RDSIAMUserAuthProfileModel{
			DBUser: types.StringValue("rds_iam_user"),
		},
	}

	// Build profile
	profile := BuildAuthenticationProfile(ctx, "rds_iam_user_auth", data, &diags)

	// Verify no errors
	if diags.HasError() {
		t.Fatalf("Expected no errors, got: %v", diags.Errors())
	}

	// Verify profile type
	rdsProfile, ok := profile.(*uapsiadbmodels.ArkUAPSIADBRDSIAMUserAuthProfile)
	if !ok {
		t.Fatalf("Expected *ArkUAPSIADBRDSIAMUserAuthProfile, got %T", profile)
	}

	// Verify DBUser field
	if rdsProfile.DBUser != "rds_iam_user" {
		t.Errorf("Expected DBUser 'rds_iam_user', got %q", rdsProfile.DBUser)
	}
}

// TestBuildAuthenticationProfile_RDSIAMUserAuth_Missing tests error when rds_iam_user_auth profile is missing
func TestBuildAuthenticationProfile_RDSIAMUserAuth_Missing(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	// Create test data WITHOUT rds_iam_user_auth profile
	data := &models.DatabasePolicyWorkspaceAssignmentModel{
		RDSIAMUserAuthProfile: nil,
	}

	// Build profile
	_ = BuildAuthenticationProfile(ctx, "rds_iam_user_auth", data, &diags)

	// Verify error occurred
	if !diags.HasError() {
		t.Fatal("Expected error for missing rds_iam_user_auth profile, got none")
	}

	// Verify error message contains expected text
	errors := diags.Errors()
	if len(errors) == 0 {
		t.Fatal("Expected error diagnostic, got none")
	}

	errorSummary := errors[0].Summary()
	if errorSummary != "Missing rds_iam_user_auth Profile" {
		t.Errorf("Expected error summary 'Missing rds_iam_user_auth Profile', got %q", errorSummary)
	}
}

// TODO: Add tests for ParseAuthenticationProfile function:
// - TestParseAuthenticationProfile_DBAuth
// - TestParseAuthenticationProfile_LDAPAuth
// - TestParseAuthenticationProfile_OracleAuth
// - TestParseAuthenticationProfile_MongoAuth
// - TestParseAuthenticationProfile_SQLServerAuth
// - TestParseAuthenticationProfile_RDSIAMUserAuth

// TODO: Add round-trip tests:
// - TestProfileRoundTrip_DBAuth (Build -> Parse -> Build should be idempotent)
// - Similar tests for other auth methods
