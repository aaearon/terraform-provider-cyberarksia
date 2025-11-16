package models

import "github.com/hashicorp/terraform-plugin-framework/types"

// VMPolicyResourceModel - Main VM policy state model
type VMPolicyResourceModel struct {
	// Identity
	ID       types.String `tfsdk:"id"`
	PolicyID types.String `tfsdk:"policy_id"`

	// Metadata
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	TimeZone    types.String `tfsdk:"time_zone"`
	Tags        types.List   `tfsdk:"tags"` // []string

	// Policy Configuration
	LocationType types.String `tfsdk:"location_type"`
	Status       types.String `tfsdk:"status"`
	PolicyType   types.String `tfsdk:"policy_type"`

	// Assignments
	Principals types.List `tfsdk:"principals"` // []PrincipalModel, min 1 required

	// Conditions
	MaxSessionDuration types.Int64  `tfsdk:"max_session_duration"`
	IdleTime           types.Int64  `tfsdk:"idle_time"`
	AccessWindow       types.Object `tfsdk:"access_window"` // AccessWindowModel
	TimeFrame          types.Object `tfsdk:"time_frame"`    // TimeFrameModel

	// Behavior
	Behavior types.Object `tfsdk:"behavior"` // BehaviorModel

	// Targets (oneOf - exactly one must be set)
	FQDNIPTargets types.Object `tfsdk:"fqdn_ip_targets"`
	AWSTargets    types.Object `tfsdk:"aws_targets"`
	AzureTargets  types.Object `tfsdk:"azure_targets"`
	GCPTargets    types.Object `tfsdk:"gcp_targets"`

	// Computed Fields
	DelegationClassification types.String `tfsdk:"delegation_classification"`
	CreatedBy                types.Object `tfsdk:"created_by"`
	UpdatedBy                types.Object `tfsdk:"updated_by"`
}

// PrincipalModel - Inline principal assignment
type PrincipalModel struct {
	PrincipalID         types.String `tfsdk:"principal_id"`
	PrincipalName       types.String `tfsdk:"principal_name"`
	PrincipalType       types.String `tfsdk:"principal_type"`
	SourceDirectoryName types.String `tfsdk:"source_directory_name"`
	SourceDirectoryID   types.String `tfsdk:"source_directory_id"`
}

// BehaviorModel - Connection behavior (SSH/RDP)
type BehaviorModel struct {
	SSH types.Object `tfsdk:"ssh"` // SSHProfileModel
	RDP types.Object `tfsdk:"rdp"` // RDPProfileModel
}

// SSHProfileModel - SSH connection profile
type SSHProfileModel struct {
	Username types.String `tfsdk:"username"`
}

// RDPProfileModel - RDP connection profile
type RDPProfileModel struct {
	LocalEphemeralUser  types.Object `tfsdk:"local_ephemeral_user"`  // LocalEphemeralUserModel
	DomainEphemeralUser types.Object `tfsdk:"domain_ephemeral_user"` // DomainEphemeralUserModel
}

// LocalEphemeralUserModel - Local Windows ephemeral user configuration
type LocalEphemeralUserModel struct {
	AssignGroups                 types.List `tfsdk:"assign_groups"`                   // []string
	EnableEphemeralUserReconnect types.Bool `tfsdk:"enable_ephemeral_user_reconnect"` // bool
}

// DomainEphemeralUserModel - Domain-joined ephemeral user configuration
type DomainEphemeralUserModel struct {
	AssignGroups                 types.List `tfsdk:"assign_groups"`                   // []string (local groups)
	AssignDomainGroups           types.List `tfsdk:"assign_domain_groups"`            // []string (domain groups)
	EnableEphemeralUserReconnect types.Bool `tfsdk:"enable_ephemeral_user_reconnect"` // bool
}

// FQDNIPTargetsModel - FQDN/IP target rules
type FQDNIPTargetsModel struct {
	FQDNRules types.List `tfsdk:"fqdn_rule"` // []FQDNRuleModel
	IPRules   types.List `tfsdk:"ip_rule"`   // []IPRuleModel
}

// FQDNRuleModel - FQDN matching rule
type FQDNRuleModel struct {
	Operator            types.String `tfsdk:"operator"`
	ComputernamePattern types.String `tfsdk:"computername_pattern"`
	Domain              types.String `tfsdk:"domain"`
}

// IPRuleModel - IP address matching rule
type IPRuleModel struct {
	Operator    types.String `tfsdk:"operator"`
	IPAddresses types.List   `tfsdk:"ip_addresses"` // []string
	LogicalName types.String `tfsdk:"logical_name"`
}

// AWSTargetsModel - AWS cloud target criteria
type AWSTargetsModel struct {
	Regions    types.List `tfsdk:"regions"`     // []string
	Tags       types.List `tfsdk:"tags"`        // []TagModel
	VPCIDs     types.List `tfsdk:"vpc_ids"`     // []string
	AccountIDs types.List `tfsdk:"account_ids"` // []string
}

// AzureTargetsModel - Azure cloud target criteria
type AzureTargetsModel struct {
	Regions        types.List `tfsdk:"regions"`         // []string
	Tags           types.List `tfsdk:"tags"`            // []TagModel
	ResourceGroups types.List `tfsdk:"resource_groups"` // []string
	VNetIDs        types.List `tfsdk:"vnet_ids"`        // []string
	Subscriptions  types.List `tfsdk:"subscriptions"`   // []string
}

// GCPTargetsModel - GCP cloud target criteria (uses Labels, not Tags!)
type GCPTargetsModel struct {
	Regions  types.List `tfsdk:"regions"`  // []string
	Labels   types.List `tfsdk:"labels"`   // []TagModel (NOTE: field name is "labels" not "tags")
	VPCIDs   types.List `tfsdk:"vpc_ids"`  // []string
	Projects types.List `tfsdk:"projects"` // []string
}

// TagModel - Key-value tag/label structure
type TagModel struct {
	Key   types.String `tfsdk:"key"`   // Required
	Value types.List   `tfsdk:"value"` // Optional, []string
}

// VMAccessWindowModel - Time-based access restrictions (VM-specific: uses List instead of Set)
type VMAccessWindowModel struct {
	DaysOfTheWeek types.List   `tfsdk:"days_of_the_week"` // List of int, 0=Sunday through 6=Saturday
	FromHour      types.String `tfsdk:"from_hour"`        // "HH:MM"
	ToHour        types.String `tfsdk:"to_hour"`          // "HH:MM"
}

// UserTimestampModel - Creator/updater metadata
type UserTimestampModel struct {
	Name      types.String `tfsdk:"name"`
	Timestamp types.String `tfsdk:"timestamp"`
}

// VMPolicyPrincipalAssignmentResourceModel - Principal-to-policy assignment resource
type VMPolicyPrincipalAssignmentResourceModel struct {
	// Composite ID: "policy-id:principal-id:principal-type"
	ID types.String `tfsdk:"id"` // Computed, composite ID

	// Reference to policy
	PolicyID types.String `tfsdk:"policy_id"` // Required, ForceNew

	// Principal details (all ForceNew)
	PrincipalID         types.String `tfsdk:"principal_id"`          // Required, ForceNew, UUID
	PrincipalName       types.String `tfsdk:"principal_name"`        // Required, ForceNew
	PrincipalType       types.String `tfsdk:"principal_type"`        // Required, ForceNew, enum
	SourceDirectoryName types.String `tfsdk:"source_directory_name"` // Optional, ForceNew
	SourceDirectoryID   types.String `tfsdk:"source_directory_id"`   // Optional, ForceNew
}
