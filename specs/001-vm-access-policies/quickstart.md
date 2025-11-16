# Quick Start: VM Access Policy Implementation

**Audience**: Developers implementing VM access policy resources

**Prerequisites**:
- Go 1.25.0 installed
- ARK SDK v1.5.0 knowledge
- Terraform Plugin Framework v6 experience
- CyberArk SIA tenant with OAuth2 credentials

---

## 1. Development Setup

### 1.1 Clone Repository

```bash
git checkout -b 001-vm-access-policies
cd /home/tim/terraform-provider-cyberarksia
```

### 1.2 Install Dependencies

```bash
go mod download
go mod tidy
```

### 1.3 Environment Variables

Export these for acceptance testing:

```bash
export CYBERARK_USERNAME="service-account@cyberark.cloud.12345"
export CYBERARK_PASSWORD="<your-password>"
export TF_ACC=1
export TF_LOG=DEBUG  # Optional: verbose logs
```

### 1.4 Verify Setup

```bash
# Build provider
go build -v

# Run existing tests to verify environment
go test ./internal/provider -v -run TestAccProvider_Configure
```

---

## 2. Implementing the VM Policy Resource

### 2.1 Create Data Models

**File**: `internal/models/vm_policy_models.go`

```go
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

    // Computed
    DelegationClassification types.String `tfsdk:"delegation_classification"`
    CreatedBy                types.Object `tfsdk:"created_by"`
    UpdatedBy                types.Object `tfsdk:"updated_by"`
}

// PrincipalModel - Inline principal assignment
type PrincipalModel struct {
    PrincipalID           types.String `tfsdk:"principal_id"`
    PrincipalName         types.String `tfsdk:"principal_name"`
    PrincipalType         types.String `tfsdk:"principal_type"`
    SourceDirectoryName   types.String `tfsdk:"source_directory_name"`
    SourceDirectoryID     types.String `tfsdk:"source_directory_id"`
}

// BehaviorModel - Connection behavior (SSH/RDP)
type BehaviorModel struct {
    SSH types.Object `tfsdk:"ssh"` // SSHProfileModel
    RDP types.Object `tfsdk:"rdp"` // RDPProfileModel
}

type SSHProfileModel struct {
    Username types.String `tfsdk:"username"`
}

type RDPProfileModel struct {
    LocalEphemeralUser  types.Object `tfsdk:"local_ephemeral_user"`  // LocalEphemeralUserModel
    DomainEphemeralUser types.Object `tfsdk:"domain_ephemeral_user"` // DomainEphemeralUserModel
}

// FQDNIPTargetsModel - FQDN/IP target rules
type FQDNIPTargetsModel struct {
    FQDNRules types.List `tfsdk:"fqdn_rule"` // []FQDNRuleModel
    IPRules   types.List `tfsdk:"ip_rule"`   // []IPRuleModel
}

type FQDNRuleModel struct {
    Operator            types.String `tfsdk:"operator"`
    ComputernamePattern types.String `tfsdk:"computername_pattern"`
    Domain              types.String `tfsdk:"domain"`
}

type IPRuleModel struct {
    Operator    types.String `tfsdk:"operator"`
    IPAddresses types.List   `tfsdk:"ip_addresses"` // []string
    LogicalName types.String `tfsdk:"logical_name"`
}

// See data-model.md §3-5 for complete model definitions including:
// - AWSTargetsModel, AzureTargetsModel, GCPTargetsModel
// - AccessWindowModel, TimeFrameModel
// - LocalEphemeralUserModel, DomainEphemeralUserModel
```

**Reference**: `specs/001-vm-access-policies/data-model.md` for complete field list

### 2.2 Create Schema

**File**: `internal/provider/vm_policy_resource.go`

```go
package provider

import (
    "context"
    "fmt"

    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
    "github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"

    "github.com/aaearon/terraform-provider-cyberarksia/internal/models"
    "github.com/aaearon/terraform-provider-cyberarksia/internal/validators"
)

type VMPolicyResource struct {
    providerData *ProviderData
}

func NewVMPolicyResource() resource.Resource {
    return &VMPolicyResource{}
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
            "**Constraint**: Exactly ONE location type per policy (FQDN/IP, AWS, Azure, or GCP).",

        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "policy_id": schema.StringAttribute{
                MarkdownDescription: "Unique policy identifier (UUID, API-generated).",
                Computed: true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                MarkdownDescription: "Policy name (1-200 chars, unique). **ForceNew**: Changing creates new policy.",
                Required: true,
                Validators: []validator.String{
                    stringvalidator.LengthBetween(1, 200),
                },
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),  // ForceNew
                },
            },
            "status": schema.StringAttribute{
                MarkdownDescription: "Policy status. Valid: `Active`, `Suspended`.",
                Required: true,
                Validators: []validator.String{
                    stringvalidator.OneOf("Active", "Suspended"),
                },
            },
            "location_type": schema.StringAttribute{
                MarkdownDescription: "Location type. Valid: `AWS`, `Azure`, `GCP`, `FQDN/IP`. **ForceNew**: Changing requires new policy.",
                Required: true,
                Validators: []validator.String{
                    validators.LocationType(),  // Custom enum validator
                },
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),  // ForceNew
                },
            },
            "time_zone": schema.StringAttribute{
                Optional: true,
                Computed: true,
                Default: stringdefault.StaticString("GMT"),
            },
            "policy_type": schema.StringAttribute{
                Optional: true,
                Computed: true,
                Default: stringdefault.StaticString("Recurring"),
                Validators: []validator.String{
                    stringvalidator.OneOf("Recurring", "OnDemand"),
                },
            },
            "max_session_duration": schema.Int64Attribute{
                Optional: true,
                Computed: true,
                Default: int64default.StaticInt64(1),
                Validators: []validator.Int64{
                    int64validator.Between(1, 24),
                },
            },
            "idle_time": schema.Int64Attribute{
                Optional: true,
                Computed: true,
                Default: int64default.StaticInt64(10),
                Validators: []validator.Int64{
                    int64validator.AtLeast(1),
                },
            },
            "delegation_classification": schema.StringAttribute{
                Computed: true,
            },
            // See data-model.md §1 for complete attribute list
        },

        Blocks: map[string]schema.Block{
            "principals": schema.ListNestedBlock{
                MarkdownDescription: "Initial principal assignments. **Required** - minimum 1 principal.",
                Validators: []validator.List{
                    listvalidator.SizeAtLeast(1),  // CRITICAL: Minimum 1 principal
                },
                NestedObject: schema.NestedBlockObject{
                    Attributes: map[string]schema.Attribute{
                        "principal_id": schema.StringAttribute{
                            Required: true,
                            Validators: []validator.String{
                                stringvalidator.LengthAtMost(40),
                                validators.UUID(),
                            },
                        },
                        "principal_name": schema.StringAttribute{
                            Required: true,
                            Validators: []validator.String{
                                stringvalidator.LengthBetween(1, 512),
                            },
                        },
                        "principal_type": schema.StringAttribute{
                            Required: true,
                            Validators: []validator.String{
                                validators.PrincipalType(),  // USER/GROUP/ROLE
                            },
                        },
                        "source_directory_name": schema.StringAttribute{
                            Optional: true,  // Conditionally required (validate in ValidateConfig)
                        },
                        "source_directory_id": schema.StringAttribute{
                            Optional: true,  // Conditionally required
                        },
                    },
                },
            },

            "behavior": schema.SingleNestedBlock{
                MarkdownDescription: "Connection behavior (SSH/RDP profiles). At least one profile required.",
                Blocks: map[string]schema.Block{
                    "ssh": schema.SingleNestedBlock{
                        Attributes: map[string]schema.Attribute{
                            "username": schema.StringAttribute{
                                Required: true,
                            },
                        },
                    },
                    "rdp": schema.SingleNestedBlock{
                        Blocks: map[string]schema.Block{
                            "local_ephemeral_user": schema.SingleNestedBlock{
                                Attributes: map[string]schema.Attribute{
                                    "assign_groups": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Optional: true,
                                    },
                                    "enable_ephemeral_user_reconnect": schema.BoolAttribute{
                                        Optional: true,
                                    },
                                },
                            },
                            "domain_ephemeral_user": schema.SingleNestedBlock{
                                Attributes: map[string]schema.Attribute{
                                    "assign_groups": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Optional: true,
                                    },
                                    "assign_domain_groups": schema.ListAttribute{
                                        ElementType: types.StringType,
                                        Optional: true,
                                    },
                                    "enable_ephemeral_user_reconnect": schema.BoolAttribute{
                                        Optional: true,
                                    },
                                },
                            },
                        },
                    },
                },
            },

            // Location type blocks (oneOf - see ValidateConfig for enforcement)
            "fqdn_ip_targets": schema.SingleNestedBlock{
                Blocks: map[string]schema.Block{
                    "fqdn_rule": schema.ListNestedBlock{
                        NestedObject: schema.NestedBlockObject{
                            Attributes: map[string]schema.Attribute{
                                "operator": schema.StringAttribute{
                                    Required: true,
                                    Validators: []validator.String{
                                        validators.FQDNOperator(),
                                    },
                                },
                                "computername_pattern": schema.StringAttribute{
                                    Required: true,
                                },
                                "domain": schema.StringAttribute{
                                    Optional: true,
                                },
                            },
                        },
                    },
                    "ip_rule": schema.ListNestedBlock{
                        NestedObject: schema.NestedBlockObject{
                            Attributes: map[string]schema.Attribute{
                                "operator": schema.StringAttribute{
                                    Required: true,
                                    Validators: []validator.String{
                                        validators.IPOperator(),
                                    },
                                },
                                "ip_addresses": schema.ListAttribute{
                                    ElementType: types.StringType,
                                    Required: true,
                                },
                                "logical_name": schema.StringAttribute{
                                    Required: true,
                                },
                            },
                        },
                    },
                },
            },
            // aws_targets, azure_targets, gcp_targets: See data-model.md §4.2-4.4 for complete schemas
        },
    }
}
```

**Tip**: Use `internal/provider/database_policy_resource.go` as reference for schema patterns

### 2.3 Implement Custom Validators

**File**: `internal/validators/vm_validators.go`

```go
package validators

import (
    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// LocationType validator
func LocationType() validator.String {
    return stringvalidator.OneOf("AWS", "Azure", "GCP", "FQDN/IP")
}

// FQDNOperator validator
func FQDNOperator() validator.String {
    return stringvalidator.OneOf("EXACTLY", "WILDCARD", "PREFIX", "SUFFIX", "CONTAINS")
}

// IPOperator validator
func IPOperator() validator.String {
    return stringvalidator.OneOf("EXACTLY", "WILDCARD")
}
```

### 2.4 Implement ValidateConfig

**Purpose**: Runtime validation for complex constraints

```go
func (r *VMPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var config models.VMPolicyResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

    // Validate exactly ONE location type
    locationTypeCount := 0
    if !config.FQDNIPTargets.IsNull() { locationTypeCount++ }
    if !config.AWSTargets.IsNull() { locationTypeCount++ }
    if !config.AzureTargets.IsNull() { locationTypeCount++ }
    if !config.GCPTargets.IsNull() { locationTypeCount++ }

    if locationTypeCount != 1 {
        resp.Diagnostics.AddError(
            "Invalid Location Type",
            "Exactly one location type must be specified: fqdn_ip_targets, aws_targets, azure_targets, or gcp_targets",
        )
    }

    // Validate at least one connection profile (SSH or RDP)
    var behavior models.BehaviorModel
    if !config.Behavior.IsNull() {
        config.Behavior.As(ctx, &behavior, basetypes.ObjectAsOptions{})
        if behavior.SSH.IsNull() && behavior.RDP.IsNull() {
            resp.Diagnostics.AddError(
                "Invalid Behavior",
                "At least one connection profile (ssh or rdp) must be configured",
            )
        }
    }

    // Validate conditional source directory fields for principals
    var principals []models.PrincipalModel
    config.Principals.ElementsAs(ctx, &principals, false)
    for i, p := range principals {
        if p.PrincipalType.ValueString() == "USER" || p.PrincipalType.ValueString() == "GROUP" {
            if p.SourceDirectoryName.IsNull() || p.SourceDirectoryName.ValueString() == "" {
                resp.Diagnostics.AddAttributeError(
                    path.Root("principals").AtListIndex(i).AtName("source_directory_name"),
                    "Missing Required Field",
                    fmt.Sprintf("source_directory_name is required for %s principals", p.PrincipalType.ValueString()),
                )
            }
            if p.SourceDirectoryID.IsNull() || p.SourceDirectoryID.ValueString() == "" {
                resp.Diagnostics.AddAttributeError(
                    path.Root("principals").AtListIndex(i).AtName("source_directory_id"),
                    "Missing Required Field",
                    fmt.Sprintf("source_directory_id is required for %s principals", p.PrincipalType.ValueString()),
                )
            }
        }
    }
}
```

### 2.5 Implement CRUD Methods

#### CREATE

```go
func (r *VMPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan models.VMPolicyResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Build SDK policy model
    policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{
        Metadata: /* map from plan */,
        Principals: /* map from plan.Principals */,
        Targets: /* map from plan location targets */,
        Behavior: /* map from plan.Behavior */,
        Conditions: /* map from plan */,
    }

    // Call SDK with retry
    var created *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
    err := client.RetryWithBackoff(ctx, func() error {
        var err error
        created, err = r.providerData.VMService.AddPolicy(policy)
        return err
    })
    if err != nil {
        resp.Diagnostics.Append(client.MapError(err, "create VM policy")...)
        return
    }

    // Map SDK response to state
    state := mapSDKPolicyToState(ctx, created)
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

**Reference**: `specs/001-vm-access-policies/research.md` Section 2.2 for SDK patterns

#### READ

```go
func (r *VMPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state models.VMPolicyResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    policyID := state.PolicyID.ValueString()

    policy, err := r.providerData.VMService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
        PolicyID: policyID,
    })
    if err != nil {
        // Drift detection: 404 = policy deleted externally
        if strings.Contains(err.Error(), "404") {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.Append(client.MapError(err, "read VM policy")...)
        return
    }

    // Map SDK response to state (includes ALL principals: inline + assigned)
    newState := mapSDKPolicyToState(ctx, policy)
    resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
```

#### UPDATE (Read-Modify-Write with Principal Preservation)

```go
func (r *VMPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan, state models.VMPolicyResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    policyID := state.PolicyID.ValueString()

    // Step 1: READ existing policy
    existingPolicy, err := r.providerData.VMService.Policy(&uapcommonmodels.ArkUAPGetPolicyRequest{
        PolicyID: policyID,
    })
    if err != nil {
        resp.Diagnostics.Append(client.MapError(err, "read VM policy for update")...)
        return
    }

    // Step 2: IDENTIFY inline principals from plan
    inlinePrincipalKeys := make(map[string]bool)
    var planPrincipals []models.PrincipalModel
    plan.Principals.ElementsAs(ctx, &planPrincipals, false)
    for _, p := range planPrincipals {
        key := fmt.Sprintf("%s:%s", p.PrincipalID.ValueString(), p.PrincipalType.ValueString())
        inlinePrincipalKeys[key] = true
    }

    // Step 3: PRESERVE assigned principals (not in inline config)
    preservedPrincipals := []uapcommonmodels.ArkUAPPrincipal{}
    for _, p := range existingPolicy.Principals {
        key := fmt.Sprintf("%s:%s", p.ID, p.Type)
        if !inlinePrincipalKeys[key] {
            preservedPrincipals = append(preservedPrincipals, p)
        }
    }

    // Step 4: BUILD new principals: inline from plan + preserved assigned
    newPrincipals := /* build from planPrincipals */
    newPrincipals = append(newPrincipals, preservedPrincipals...)

    // Step 5: UPDATE other fields from plan
    existingPolicy.Metadata.Description = plan.Description.ValueString()
    existingPolicy.Metadata.TimeZone = plan.TimeZone.ValueString()
    existingPolicy.Metadata.Status.Status = plan.Status.ValueString()
    existingPolicy.Conditions.MaxSessionDuration = int(plan.MaxSessionDuration.ValueInt64())
    existingPolicy.Conditions.IdleTime = int(plan.IdleTime.ValueInt64())
    existingPolicy.Principals = newPrincipals
    // Update targets, behavior, conditions from plan (see data-model.md §9 for conversion helpers)

    // Step 6: WRITE back
    var updated *uapsiavmmodels.ArkUAPSIAVMAccessPolicy
    err = client.RetryWithBackoff(ctx, func() error {
        var err error
        updated, err = r.providerData.VMService.UpdatePolicy(existingPolicy)
        return err
    })
    if err != nil {
        resp.Diagnostics.Append(client.MapError(err, "update VM policy")...)
        return
    }

    // Map to state
    newState := mapSDKPolicyToState(ctx, updated)
    resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}
```

**Critical**: See `specs/001-vm-access-policies/research.md` Section 4 for principal preservation algorithm

#### DELETE

```go
func (r *VMPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state models.VMPolicyResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    policyID := state.PolicyID.ValueString()

    // Use SDK method directly - NO workaround needed for VM policies
    err := r.providerData.VMService.DeletePolicy(&uapcommonmodels.ArkUAPDeletePolicyRequest{
        PolicyID: policyID,
    })
    if err != nil {
        // 404 = already deleted (drift detection) - treat as success
        if strings.Contains(err.Error(), "404") {
            return
        }
        resp.Diagnostics.Append(client.MapError(err, "delete VM policy")...)
        return
    }
}
```

**Note**: VM policies do NOT need `internal/client/delete_workarounds.go`

---

## 3. Implementing Principal Assignment Resource

### 3.1 Extend Composite ID Helpers

**File**: `internal/provider/helpers/composite_ids.go`

```go
// ParseVMPolicyPrincipalID parses composite ID for VM policy principal assignments
// Format: "policy-id:principal-id:principal-type"
func ParseVMPolicyPrincipalID(id string) (policyID, principalID, principalType string, err error) {
    parts := strings.Split(id, ":")
    if len(parts) != 3 {
        return "", "", "", fmt.Errorf(
            "invalid VM policy principal assignment ID: expected 'policy-id:principal-id:principal-type', got '%s'",
            id,
        )
    }

    policyID = parts[0]
    principalID = parts[1]
    principalType = parts[2]

    if principalType != "USER" && principalType != "GROUP" && principalType != "ROLE" {
        return "", "", "", fmt.Errorf("invalid principal type '%s': must be USER, GROUP, or ROLE", principalType)
    }

    return policyID, principalID, principalType, nil
}

// BuildVMPolicyPrincipalID creates composite ID
func BuildVMPolicyPrincipalID(policyID, principalID, principalType string) string {
    return fmt.Sprintf("%s:%s:%s", policyID, principalID, principalType)
}
```

### 3.2 Implement Assignment Resource

**File**: `internal/provider/vm_policy_principal_assignment_resource.go`

**Reference**: Follow `database_policy_principal_assignment_resource.go` pattern exactly

**Key CRUD Methods**:

- **CREATE**: Read policy → Check duplicates → Append principal → Update policy
- **READ**: Parse composite ID → Read policy → Find principal (404 if not found)
- **DELETE**: Parse composite ID → Read policy → Remove principal → Update policy

**Full implementation**: See reference docs for detailed patterns

---

## 4. Testing

### 4.1 Acceptance Tests

**File**: `internal/provider/vm_policy_resource_test.go`

```go
func TestAccVMPolicy_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:     func() { testAccPreCheck(t) },
        Providers:    testAccProviders,
        CheckDestroy: testAccCheckVMPolicyDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccVMPolicyConfig_basic(),
                Check: resource.ComposeTestCheckFunc(
                    testAccCheckVMPolicyExists("cyberarksia_vm_policy.test"),
                    resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "name", "test-policy"),
                    resource.TestCheckResourceAttr("cyberarksia_vm_policy.test", "status", "Active"),
                ),
            },
        },
    })
}
```

### 4.2 CRUD Validation

**Manual Testing**: Follow `examples/testing/TESTING-GUIDE.md`

**Automated**: Future enhancement - `make test-crud DESC=vm-policy`

---

## 5. Common Pitfalls

### Pitfall 1: Forgetting Principal Preservation

**Wrong**:
```go
existingPolicy.Principals = /* only inline principals from plan */
```

**Correct**:
```go
existingPolicy.Principals = append(inlinePrincipals, assignedPrincipals...)
```

### Pitfall 2: Using DELETE Workaround

**Wrong**:
```go
err := client.DeletePolicyDirect(ctx, providerData.AuthContext, policyID)  // Database policy workaround
```

**Correct**:
```go
err := vmService.DeletePolicy(&uapcommonmodels.ArkUAPDeletePolicyRequest{PolicyID: policyID})  // SDK works directly
```

### Pitfall 3: Incorrect Location Type Validation

**Wrong**: Allow multiple location type blocks (confusing error from API)

**Correct**: Validate exactly one location type in `ValidateConfig()`

---

## 6. Next Steps

1. **Register Resources**: Update `internal/provider/provider.go`
2. **Build & Install**: `make build && make install`
3. **Run Tests**: `TF_ACC=1 go test ./internal/provider -v -run TestAccVMPolicy`
4. **Create Examples**: Add to `examples/resources/cyberarksia_vm_policy/`
5. **Generate Docs**: `tfplugindocs generate`
6. **Update CLAUDE.md**: Add new resources to table

---

**Quick Start Complete**: You're now ready to implement VM access policy resources following established patterns.
