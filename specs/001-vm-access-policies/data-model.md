# Data Model: VM Access Policy Resources

**Date**: 2025-11-16
**Purpose**: Terraform state models and SDK mappings for VM access policy resources

---

## Overview

This document defines the Terraform state models for VM access policy management, including field-by-field mappings to ARK SDK v1.5.0 models.

**Resources**:
1. `cyberarksia_vm_policy` - Main VM access policy resource
2. `cyberarksia_vm_policy_principal_assignment` - Principal-to-policy assignment resource

---

## 1. VMPolicyResourceModel

**Purpose**: Terraform state model for `cyberarksia_vm_policy` resource

**Go Struct**:
```go
type VMPolicyResourceModel struct {
    // Identity
    ID       types.String `tfsdk:"id"`        // Computed, same as PolicyID
    PolicyID types.String `tfsdk:"policy_id"` // Computed, UUID from API

    // Metadata
    Name        types.String `tfsdk:"name"`         // Required, ForceNew, 1-200 chars
    Description types.String `tfsdk:"description"`  // Optional, max 200 chars
    TimeZone    types.String `tfsdk:"time_zone"`    // Optional+Computed, default "GMT"
    Tags        types.List   `tfsdk:"tags"`         // Optional+Computed, []string, max 20 items

    // Policy Configuration
    LocationType types.String `tfsdk:"location_type"` // Required, ForceNew, enum
    Status       types.String `tfsdk:"status"`        // Required, enum
    PolicyType   types.String `tfsdk:"policy_type"`   // Optional+Computed, default "Recurring"

    // Assignments
    Principals types.List `tfsdk:"principals"` // Required, min 1, []PrincipalModel

    // Conditions
    MaxSessionDuration types.Int64  `tfsdk:"max_session_duration"` // Optional+Computed, default 1
    IdleTime           types.Int64  `tfsdk:"idle_time"`            // Optional+Computed, default 10
    AccessWindow       types.Object `tfsdk:"access_window"`        // Optional, AccessWindowModel
    TimeFrame          types.Object `tfsdk:"time_frame"`           // Optional, TimeFrameModel

    // Behavior
    Behavior types.Object `tfsdk:"behavior"` // Required, BehaviorModel

    // Targets (oneOf)
    FQDNIPTargets types.Object `tfsdk:"fqdn_ip_targets"` // Optional, FQDNIPTargetsModel
    AWSTargets    types.Object `tfsdk:"aws_targets"`     // Optional, AWSTargetsModel
    AzureTargets  types.Object `tfsdk:"azure_targets"`   // Optional, AzureTargetsModel
    GCPTargets    types.Object `tfsdk:"gcp_targets"`     // Optional, GCPTargetsModel

    // Computed Fields
    DelegationClassification types.String `tfsdk:"delegation_classification"` // Computed
    CreatedBy                types.Object `tfsdk:"created_by"`                // Computed, UserTimestampModel
    UpdatedBy                types.Object `tfsdk:"updated_by"`                // Computed, UserTimestampModel
}
```

### Field Mappings: Terraform ↔ SDK

| Terraform Attribute | SDK Field Path | Type | Notes |
|---------------------|----------------|------|-------|
| `id` | `Metadata.PolicyID` | string | Alias for policy_id |
| `policy_id` | `Metadata.PolicyID` | string | UUID, API-generated |
| `name` | `Metadata.Name` | string | Unique per tenant, ForceNew |
| `description` | `Metadata.Description` | string | Optional |
| `time_zone` | `Metadata.TimeZone` | string | Default "GMT" |
| `tags` | `Metadata.PolicyTags` | []string | Max 20 items |
| `location_type` | `Metadata.PolicyEntitlement.LocationType` | string | ForceNew, determines target type |
| `status` | `Metadata.Status.Status` | string | Active/Suspended |
| `policy_type` | `Metadata.PolicyEntitlement.PolicyType` | string | Recurring/OnDemand |
| `principals` | `Principals` | []ArkUAPPrincipal | Min 1 required |
| `max_session_duration` | `Conditions.MaxSessionDuration` | int | 1-24 hours |
| `idle_time` | `Conditions.IdleTime` | int | 1-120 minutes |
| `access_window` | `Conditions.AccessWindow` | object | Optional time restrictions |
| `time_frame` | `Metadata.TimeFrame` | object | Optional activation period |
| `behavior` | `Behavior` | object | SSH/RDP connection profiles |
| `fqdn_ip_targets` | `Targets.FQDNIPResource` | object | When LocationType = "FQDN/IP" |
| `aws_targets` | `Targets.AWSResource` | object | When LocationType = "AWS" |
| `azure_targets` | `Targets.AzureResource` | object | When LocationType = "Azure" |
| `gcp_targets` | `Targets.GCPResource` | object | When LocationType = "GCP" |
| `delegation_classification` | `DelegationClassification` | string | Computed by server |
| `created_by` | `Metadata.CreatedBy` | object | Read-only |
| `updated_by` | `Metadata.UpdatedBy` | object | Read-only |

---

## 2. PrincipalModel

**Purpose**: Nested model for principal assignments (inline principals in policy resource)

**Go Struct**:
```go
type PrincipalModel struct {
    PrincipalID           types.String `tfsdk:"principal_id"`            // Required, UUID, max 40 chars
    PrincipalName         types.String `tfsdk:"principal_name"`          // Required, 1-512 chars
    PrincipalType         types.String `tfsdk:"principal_type"`          // Required, enum: USER/GROUP/ROLE
    SourceDirectoryName   types.String `tfsdk:"source_directory_name"`   // Conditional, max 50 chars
    SourceDirectoryID     types.String `tfsdk:"source_directory_id"`     // Conditional
}
```

### SDK Mapping: `ArkUAPPrincipal`

| Terraform Field | SDK Field | Validation |
|----------------|-----------|------------|
| `principal_id` | `ID` | UUID format, max 40 chars |
| `principal_name` | `Name` | 1-512 chars, pattern `[\w.\-+]+` |
| `principal_type` | `Type` | Enum: USER, GROUP, ROLE |
| `source_directory_name` | `SourceDirectoryName` | Required for USER/GROUP, optional for ROLE |
| `source_directory_id` | `SourceDirectoryID` | Required for USER/GROUP, optional for ROLE |

**Conditional Validation Logic**:
```go
if principalType == "USER" || principalType == "GROUP" {
    // source_directory_name and source_directory_id are REQUIRED
    if sourceDirectoryName == "" || sourceDirectoryID == "" {
        return error
    }
}
// For ROLE, source directory fields are OPTIONAL
```

---

## 3. BehaviorModel

**Purpose**: Connection behavior configuration (SSH/RDP profiles)

**Go Struct**:
```go
type BehaviorModel struct {
    SSH types.Object `tfsdk:"ssh"` // Optional, SSHProfileModel
    RDP types.Object `tfsdk:"rdp"` // Optional, RDPProfileModel
}

type SSHProfileModel struct {
    Username types.String `tfsdk:"username"` // Required if SSH block present
}

type RDPProfileModel struct {
    LocalEphemeralUser  types.Object `tfsdk:"local_ephemeral_user"`  // Optional, LocalEphemeralUserModel
    DomainEphemeralUser types.Object `tfsdk:"domain_ephemeral_user"` // Optional, DomainEphemeralUserModel
}

type LocalEphemeralUserModel struct {
    AssignGroups                 types.List `tfsdk:"assign_groups"`                    // []string
    EnableEphemeralUserReconnect types.Bool `tfsdk:"enable_ephemeral_user_reconnect"`  // bool
}

type DomainEphemeralUserModel struct {
    AssignGroups                 types.List `tfsdk:"assign_groups"`                    // []string (local groups)
    AssignDomainGroups           types.List `tfsdk:"assign_domain_groups"`             // []string (domain groups)
    EnableEphemeralUserReconnect types.Bool `tfsdk:"enable_ephemeral_user_reconnect"`  // bool
}
```

### SDK Mapping

| Terraform Path | SDK Path | Notes |
|----------------|----------|-------|
| `behavior.ssh.username` | `Behavior.SSHProfile.Username` | Required if SSH profile present |
| `behavior.rdp.local_ephemeral_user.assign_groups` | `Behavior.RDPProfile.LocalEphemeralUser.AssignGroups` | Local Windows groups |
| `behavior.rdp.local_ephemeral_user.enable_ephemeral_user_reconnect` | `Behavior.RDPProfile.LocalEphemeralUser.EnableEphemeralUserReconnect` | Default false |
| `behavior.rdp.domain_ephemeral_user.assign_groups` | `Behavior.RDPProfile.DomainEphemeralUser.AssignGroups` | Local groups |
| `behavior.rdp.domain_ephemeral_user.assign_domain_groups` | `Behavior.RDPProfile.DomainEphemeralUser.AssignDomainGroups` | Domain groups |
| `behavior.rdp.domain_ephemeral_user.enable_ephemeral_user_reconnect` | `Behavior.RDPProfile.DomainEphemeralUser.EnableEphemeralUserReconnect` | Default false |

**Validation**: At least one of SSH or RDP must be present (server-validated)

---

## 4. Target Models

### 4.1 FQDNIPTargetsModel

**Go Struct**:
```go
type FQDNIPTargetsModel struct {
    FQDNRules types.List `tfsdk:"fqdn_rule"` // []FQDNRuleModel
    IPRules   types.List `tfsdk:"ip_rule"`   // []IPRuleModel
}

type FQDNRuleModel struct {
    Operator            types.String `tfsdk:"operator"`              // Required, enum
    ComputernamePattern types.String `tfsdk:"computername_pattern"`  // Required, max 300 chars
    Domain              types.String `tfsdk:"domain"`                // Optional, max 1000 chars
}

type IPRuleModel struct {
    Operator    types.String `tfsdk:"operator"`      // Required, enum
    IPAddresses types.List   `tfsdk:"ip_addresses"`  // Required, []string, max 1000 items
    LogicalName types.String `tfsdk:"logical_name"`  // Required, 1-256 chars
}
```

**SDK Mapping**: `Targets.FQDNIPResource`

**FQDN Operators**: EXACTLY, WILDCARD, PREFIX, SUFFIX, CONTAINS
**IP Operators**: EXACTLY, WILDCARD

### 4.2 AWSTargetsModel

**Go Struct**:
```go
type AWSTargetsModel struct {
    Regions    types.List `tfsdk:"regions"`     // Optional, []string
    Tags       types.List `tfsdk:"tags"`        // Optional, []TagModel
    VPCIDs     types.List `tfsdk:"vpc_ids"`     // Optional, []string
    AccountIDs types.List `tfsdk:"account_ids"` // Optional, []string
}

type TagModel struct {
    Key   types.String `tfsdk:"key"`   // Required
    Value types.List   `tfsdk:"value"` // Optional, []string
}
```

**SDK Mapping**: `Targets.AWSResource`

### 4.3 GCPTargetsModel

**Go Struct**:
```go
type GCPTargetsModel struct {
    Regions  types.List `tfsdk:"regions"`  // Optional, []string
    Labels   types.List `tfsdk:"labels"`   // Optional, []TagModel (NOTE: Labels, not Tags!)
    VPCIDs   types.List `tfsdk:"vpc_ids"`  // Optional, []string
    Projects types.List `tfsdk:"projects"` // Optional, []string
}
```

**SDK Mapping**: `Targets.GCPResource`

**NOTE**: GCP uses "Labels" field name in SDK, not "Tags"

### 4.4 AzureTargetsModel

**Go Struct**:
```go
type AzureTargetsModel struct {
    Regions        types.List `tfsdk:"regions"`         // Optional, []string
    Tags           types.List `tfsdk:"tags"`            // Optional, []TagModel
    ResourceGroups types.List `tfsdk:"resource_groups"` // Optional, []string
    VNetIDs        types.List `tfsdk:"vnet_ids"`        // Optional, []string
    Subscriptions  types.List `tfsdk:"subscriptions"`   // Optional, []string
}
```

**SDK Mapping**: `Targets.AzureResource`

---

## 5. Condition Models

### 5.1 AccessWindowModel

**Purpose**: Daily access schedule configuration

**Go Struct**:
```go
type AccessWindowModel struct {
    DaysOfTheWeek types.List   `tfsdk:"days_of_the_week"` // Optional, []int (0-6)
    FromHour      types.String `tfsdk:"from_hour"`        // Optional, time format
    ToHour        types.String `tfsdk:"to_hour"`          // Optional, time format
}
```

**SDK Mapping**: `Conditions.AccessWindow`

**DaysOfTheWeek**: 0=Sunday, 1=Monday, ..., 6=Saturday
**Time Format**: Pattern `\w+` (e.g., "09:00", "17:00")

### 5.2 TimeFrameModel

**Purpose**: Policy activation period

**Go Struct**:
```go
type TimeFrameModel struct {
    FromTime types.String `tfsdk:"from_time"` // Optional, ISO 8601 date-time
    ToTime   types.String `tfsdk:"to_time"`   // Optional, ISO 8601 date-time
}
```

**SDK Mapping**: `Metadata.TimeFrame`

**Format**: ISO 8601 (`yyyy-MM-ddTHH:mm:ss`), e.g., "2025-01-01T00:00:00"

---

## 6. VMPolicyPrincipalAssignmentResourceModel

**Purpose**: Terraform state model for `cyberarksia_vm_policy_principal_assignment` resource

**Go Struct**:
```go
type VMPolicyPrincipalAssignmentResourceModel struct {
    // Composite ID: "policy-id:principal-id:principal-type"
    ID types.String `tfsdk:"id"` // Computed, composite ID

    // Reference to policy
    PolicyID types.String `tfsdk:"policy_id"` // Required, ForceNew

    // Principal details (all ForceNew)
    PrincipalID           types.String `tfsdk:"principal_id"`            // Required, ForceNew, UUID
    PrincipalName         types.String `tfsdk:"principal_name"`          // Required, ForceNew
    PrincipalType         types.String `tfsdk:"principal_type"`          // Required, ForceNew, enum
    SourceDirectoryName   types.String `tfsdk:"source_directory_name"`   // Optional, ForceNew
    SourceDirectoryID     types.String `tfsdk:"source_directory_id"`     // Optional, ForceNew
}
```

### Composite ID Format

**Format**: `policy-id:principal-id:principal-type`

**Example**: `a1b2c3d4-5678-90ab-cdef-1234567890ab:e5f6a7b8-9012-34cd-ef56-7890abcdef12:USER`

**Parsing**: Use `helpers.ParseVMPolicyPrincipalID(id string) (policyID, principalID, principalType string, err error)`

**Building**: Use `helpers.BuildVMPolicyPrincipalID(policyID, principalID, principalType string) string`

### CRUD Implementation Notes

**CREATE**:
1. Read existing policy via `vmService.Policy()`
2. Check for duplicate principal (same ID + Type)
3. Append new principal to `policy.Principals` array
4. Update policy via `vmService.UpdatePolicy()`
5. Verify assignment succeeded

**READ**:
1. Parse composite ID
2. Read policy via `vmService.Policy()`
3. Find principal in `policy.Principals` array
4. If not found, remove from state (drift detection)

**DELETE**:
1. Parse composite ID
2. Read existing policy
3. Remove principal from `policy.Principals` array
4. Update policy via `vmService.UpdatePolicy()`

---

## 7. Validation Constraints Summary

### Required Fields (Schema-Level)

| Field | Constraint | Validator |
|-------|-----------|-----------|
| `name` | 1-200 chars, unique | `stringvalidator.LengthBetween(1, 200)` |
| `status` | Enum | `stringvalidator.OneOf("Active", "Suspended")` |
| `location_type` | Enum | `stringvalidator.OneOf("AWS", "Azure", "GCP", "FQDN/IP")` |
| `principals` | Min 1 element | `listvalidator.SizeAtLeast(1)` |
| `behavior` | Required object | Required in schema |

### Conditional Requirements

| Field | Condition | Validation |
|-------|-----------|------------|
| `source_directory_name` | When `principal_type` = USER or GROUP | Runtime validation in ValidateConfig |
| `source_directory_id` | When `principal_type` = USER or GROUP | Runtime validation in ValidateConfig |
| `ssh.username` | When `ssh` block present | Schema Required within nested block |

### Server-Validated Constraints

| Constraint | Error | Validation |
|------------|-------|------------|
| At least one connection profile (SSH or RDP) | 400 | ValidateConfig method |
| SSH username non-empty | 400 | Server validates |
| MaxSessionDuration ≥ 1 | 400 | `int64validator.Between(1, 24)` |
| IdleTime > 0 | 400 | `int64validator.AtLeast(1)` |
| IP rule LogicalName present | 400 | Schema Required |
| Exactly ONE location type | 400 | Custom ValidateConfig |
| Principal uniqueness | 400 | Server validates (duplicate detection in assignment resource) |

### OneOf Constraints

| Constraint | Implementation |
|------------|---------------|
| Exactly one location type (FQDN/IP, AWS, Azure, GCP) | Custom ValidateConfig logic |
| Exactly one RDP ephemeral user type (Local, Domain) | Custom ValidateConfig logic (AtLeastOneOf acceptable) |

---

## 8. State Transitions

### Policy Status

```
Active ←→ Suspended (user-controlled)
   ↓
Validating → Active/Error (server-managed)
   ↓
Expired (server-managed, read-only)
```

**User-settable**: Active, Suspended
**Server-managed** (read-only): Validating, Error, Warning, Expired

### Principal Assignment

```
Not Assigned → [CREATE] → Assigned
Assigned → [DELETE] → Not Assigned
```

**No UPDATE**: All principal assignment fields are ForceNew (changing any field requires destroy + recreate)

---

## 9. SDK Conversion Helpers

### 9.1 Terraform → SDK (Create/Update)

```go
// Convert Terraform principals to SDK principals
func buildSDKPrincipals(tfPrincipals []PrincipalModel) []uapcommonmodels.ArkUAPPrincipal {
    sdkPrincipals := make([]uapcommonmodels.ArkUAPPrincipal, len(tfPrincipals))
    for i, p := range tfPrincipals {
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

// Convert Terraform behavior to SDK behavior
func buildSDKBehavior(tfBehavior BehaviorModel) uapsiavmmodels.ArkUAPSSIAVMBehavior {
    behavior := uapsiavmmodels.ArkUAPSSIAVMBehavior{}

    if !tfBehavior.SSH.IsNull() {
        var ssh SSHProfileModel
        tfBehavior.SSH.As(ctx, &ssh, basetypes.ObjectAsOptions{})
        behavior.SSHProfile = &uapsiavmmodels.ArkUAPSSIAVMSSHProfile{
            Username: ssh.Username.ValueString(),
        }
    }

    if !tfBehavior.RDP.IsNull() {
        var rdp RDPProfileModel
        tfBehavior.RDP.As(ctx, &rdp, basetypes.ObjectAsOptions{})
        behavior.RDPProfile = &uapsiavmmodels.ArkUAPSSIAVMRDPProfile{}

        if !rdp.LocalEphemeralUser.IsNull() {
            // Build local ephemeral user
        }
        if !rdp.DomainEphemeralUser.IsNull() {
            // Build domain ephemeral user
        }
    }

    return behavior
}
```

### 9.2 SDK → Terraform (Read)

```go
// Convert SDK principals to Terraform principals
func mapSDKPrincipalsToTerraform(ctx context.Context, sdkPrincipals []uapcommonmodels.ArkUAPPrincipal) types.List {
    principalModels := make([]PrincipalModel, len(sdkPrincipals))
    for i, p := range sdkPrincipals {
        principalModels[i] = PrincipalModel{
            PrincipalID:           types.StringValue(p.ID),
            PrincipalName:         types.StringValue(p.Name),
            PrincipalType:         types.StringValue(p.Type),
            SourceDirectoryName:   types.StringValue(p.SourceDirectoryName),
            SourceDirectoryID:     types.StringValue(p.SourceDirectoryID),
        }
    }

    principalList, diags := types.ListValueFrom(ctx, types.ObjectType{
        AttrTypes: principalAttrTypes,
    }, principalModels)

    return principalList
}
```

---

## 10. Example HCL → State Mapping

**HCL Configuration**:
```hcl
resource "cyberarksia_vm_policy" "example" {
  name          = "Production Servers"
  location_type = "FQDN/IP"
  status        = "Active"

  principal {
    principal_id              = "abc-123"
    principal_name            = "admin@example.com"
    principal_type            = "USER"
    source_directory_name     = "CyberArk"
    source_directory_id       = "dir-456"
  }

  fqdn_ip_targets {
    fqdn_rule {
      operator             = "SUFFIX"
      computername_pattern = "-prod"
      domain               = "example.com"
    }
  }

  behavior {
    ssh {
      username = "ec2-user"
    }
  }

  max_session_duration = 4
}
```

**Terraform State** (partial):
```json
{
  "id": "policy-uuid-123",
  "policy_id": "policy-uuid-123",
  "name": "Production Servers",
  "location_type": "FQDN/IP",
  "status": "Active",
  "time_zone": "GMT",
  "principals": [
    {
      "principal_id": "abc-123",
      "principal_name": "admin@example.com",
      "principal_type": "USER",
      "source_directory_name": "CyberArk",
      "source_directory_id": "dir-456"
    }
  ],
  "max_session_duration": 4,
  "idle_time": 10,
  "delegation_classification": "Restricted"
}
```

**SDK Model** (simplified):
```json
{
  "metadata": {
    "policyId": "policy-uuid-123",
    "name": "Production Servers",
    "timeZone": "GMT",
    "policyEntitlement": {
      "locationType": "FQDN/IP"
    },
    "status": {
      "status": "Active"
    }
  },
  "principals": [
    {
      "id": "abc-123",
      "name": "admin@example.com",
      "type": "USER",
      "sourceDirectoryName": "CyberArk",
      "sourceDirectoryId": "dir-456"
    }
  ],
  "conditions": {
    "maxSessionDuration": 4,
    "idleTime": 10
  },
  "behavior": {
    "connectAs": {
      "ssh": {
        "username": "ec2-user"
      }
    }
  },
  "targets": {
    "FQDN/IP": {
      "fqdnRules": [
        {
          "operator": "SUFFIX",
          "computername_pattern": "-prod",
          "domain": "example.com"
        }
      ]
    }
  }
}
```

---

**Data Model Complete**: This document provides complete field mappings for implementation.
