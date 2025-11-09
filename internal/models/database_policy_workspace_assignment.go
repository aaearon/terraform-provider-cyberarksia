// Package models defines Terraform state models
package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DatabasePolicyWorkspaceAssignmentModel describes the resource data model for Terraform state
type DatabasePolicyWorkspaceAssignmentModel struct {
	// Required inputs (ForceNew)
	PolicyID             types.String `tfsdk:"policy_id"`
	DatabaseWorkspaceID  types.String `tfsdk:"database_workspace_id"`
	AuthenticationMethod types.String `tfsdk:"authentication_method"`

	// Profile blocks (mutually exclusive)
	DBAuthProfile         *DBAuthProfileModel         `tfsdk:"db_auth_profile"`
	LDAPAuthProfile       *LDAPAuthProfileModel       `tfsdk:"ldap_auth_profile"`
	OracleAuthProfile     *OracleAuthProfileModel     `tfsdk:"oracle_auth_profile"`
	MongoAuthProfile      *MongoAuthProfileModel      `tfsdk:"mongo_auth_profile"`
	SQLServerAuthProfile  *SQLServerAuthProfileModel  `tfsdk:"sqlserver_auth_profile"`
	RDSIAMUserAuthProfile *RDSIAMUserAuthProfileModel `tfsdk:"rds_iam_user_auth_profile"`

	// Computed
	ID           types.String `tfsdk:"id"`
	LastModified types.String `tfsdk:"last_modified"`
}

// DBAuthProfileModel represents the db_auth authentication profile
type DBAuthProfileModel struct {
	Roles types.List `tfsdk:"roles"` // list(string)
}

// LDAPAuthProfileModel represents the ldap_auth authentication profile
type LDAPAuthProfileModel struct {
	AssignGroups types.List `tfsdk:"assign_groups"` // list(string)
}

// OracleAuthProfileModel represents the oracle_auth authentication profile
type OracleAuthProfileModel struct {
	Roles       types.List `tfsdk:"roles"`        // list(string)
	DbaRole     types.Bool `tfsdk:"dba_role"`     // bool
	SysdbaRole  types.Bool `tfsdk:"sysdba_role"`  // bool
	SysoperRole types.Bool `tfsdk:"sysoper_role"` // bool
}

// MongoAuthProfileModel represents the mongo_auth authentication profile
type MongoAuthProfileModel struct {
	GlobalBuiltinRoles   types.List `tfsdk:"global_builtin_roles"`   // list(string)
	DatabaseBuiltinRoles types.Map  `tfsdk:"database_builtin_roles"` // map(list(string))
	DatabaseCustomRoles  types.Map  `tfsdk:"database_custom_roles"`  // map(list(string))
}

// SQLServerAuthProfileModel represents the sqlserver_auth authentication profile
type SQLServerAuthProfileModel struct {
	GlobalBuiltinRoles   types.List `tfsdk:"global_builtin_roles"`   // list(string)
	GlobalCustomRoles    types.List `tfsdk:"global_custom_roles"`    // list(string)
	DatabaseBuiltinRoles types.Map  `tfsdk:"database_builtin_roles"` // map(list(string))
	DatabaseCustomRoles  types.Map  `tfsdk:"database_custom_roles"`  // map(list(string))
}

// RDSIAMUserAuthProfileModel represents the rds_iam_user_auth authentication profile
type RDSIAMUserAuthProfileModel struct {
	DBUser types.String `tfsdk:"db_user"` // string
}
