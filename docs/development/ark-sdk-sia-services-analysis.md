# ARK SDK SIA Services Analysis

**Research Date**: 2025-11-02
**ARK SDK Version**: v1.5.0
**Analysis Method**: Multi-perspective research (Claude Code + Gemini + Codex)

**Purpose**: Discovery and analysis of **available but not yet implemented** SDK services for future feature planning

**Related**: For usage patterns of **already implemented** resources, see [../sdk-integration.md](../sdk-integration.md)

---

## Executive Summary

This document provides a comprehensive analysis of all Secure Infrastructure Access (SIA) services available in the CyberArk ARK SDK for Go (v1.5.0) that could be added to the Terraform provider. The analysis was conducted using three independent perspectives to ensure completeness and identify potential blind spots.

**Key Findings**:
- **9 SIA services** verified in ARK SDK v1.5.0 ✅ **Confirmed by CyberArk's official `ark` CLI**
- **8 resources + 2 data sources currently implemented** (10 total Terraform components):
  - Resources: Database workspaces, database secrets, VM secrets, target sets, certificates, database policies, database policy principal assignments, database policy workspace assignments
  - Data sources: Database policies, principals
- **2 services available for implementation** (VM policies + assignments; SSH CA)
- **3 CLI-only services excluded** (end-user tools, not infrastructure management)
- **Critical SDK bugs REPRODUCED IN CYBERARK'S PRODUCTION CLI**:
  - DELETE panic bug ✅ **Production CLI crashes**: `ark exec sia secrets vm delete-secret` panics with nil pointer dereference during HTTP request construction. **Resource is NOT deleted** (panic occurs before API call).
  - VM filtering bug ✅ **Production CLI broken**: `list-secrets-by --secret-types X` returns `[]` for all filters. Filtering completely non-functional.
  - Serialization quirks ✅ **Verified**: Manual camel/snake conversions in VM policies
- **⚠️ Note**: 3 services initially identified by Gemini (K8s clusters, Accounts, Platforms) do not exist in SDK v1.5.0

**Implementation Status** (as of 2025-11-14):
1. **Phase 1** (40% Complete): VM Secrets ✅, Target Sets ✅ implemented. Remaining: VM Policies + assignments
2. **Phase 2** (Planned): Complete VM policies and assignments - Full VM infrastructure story
3. **Phase 3** (Planned): SSH CA - Certificate-based authentication
4. **Phase 4** (Optional): Data sources for automation (connector scripts, kubeconfig, SSO tokens)

**⚠️ Note**: Recent provider changes include resource renames (`cyberarksia_secret` → `cyberarksia_database_secret`, `database_policy_database_assignment` → `database_policy_workspace_assignment`). See [CHANGELOG.md](/CHANGELOG.md) for migration guide.

---

## Table of Contents

1. [Methodology](#methodology)
2. [Currently Implemented Services](#currently-implemented-services)
3. [Available Services (Not Yet Implemented)](#available-services-not-yet-implemented)
4. [CLI/End-User Services (Excluded)](#cliend-user-services-excluded)
5. [Critical SDK Issues](#critical-sdk-issues)
6. [Implementation Roadmap](#implementation-roadmap)
7. [Architectural Insights](#architectural-insights)
8. [Testing Considerations](#testing-considerations)
9. [Recommendations](#recommendations)

---

## Methodology

### Research Approach

This analysis employed a **multi-perspective research strategy** using three independent AI models to validate findings and identify blind spots:

1. **Claude Code (Primary)**: Direct SDK code exploration at `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/`
2. **Gemini**: Documentation-focused analysis with web search capabilities
3. **Codex**: Code-level deep dive examining implementation patterns and SDK quirks

### Research Scope

**Included**:
- Services in `pkg/services/sia/` (Secure Infrastructure Access)
- Services in `pkg/services/uap/sia/` (Unified Access Policies for SIA)
- Infrastructure management APIs suitable for Terraform resources/data sources

**Excluded**:
- End-user CLI tools (credential generation, shell execution)
- Services outside SIA domain
- Deprecated or internal-only services

### Validation Criteria

For each service, we evaluated:
- **Feature Purpose**: What SIA capability does it support?
- **Dependencies**: What other services or resources does it require?
- **Schemas/Models**: What data structures are involved?
- **CRUD Operations**: What lifecycle management is available?
- **Terraform Suitability**: Does it fit Terraform's declarative model?
- **Implementation Complexity**: What challenges exist?

### Validation Results

**Important Discovery**: The multi-perspective approach revealed that Gemini hallucinated 3 services that do not exist in ARK SDK v1.5.0:
- K8s Cluster Management (`sia.NewK8SClusters`)
- Account Management (`sia.NewAccounts`)
- Platform Management (`sia.NewPlatforms`)

These services were validated against the SDK source code at `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/sia/ark_sia_api.go` and confirmed absent. This document has been updated to reflect only **verified services**.

**11 Verified Services**:
- 9 SIA services (access, db, k8s, secrets/db, secrets/vm, sshca, sso, workspaces/db, workspaces/targetsets)
- 2 UAP SIA services (uap/sia/db, uap/sia/vm)

---

## Currently Implemented Services

These services are **already available** in the Terraform provider (v0.1.0).

**For detailed CRUD operations, schemas, and implementation patterns**, see the [SDK Integration Reference](../sdk-integration.md).

### Quick Reference Table

| Service | Resource/Data Source | SDK Service | Purpose | Implementation File |
|---------|---------------------|-------------|---------|---------------------|
| **Database Workspaces** | `cyberarksia_database_workspace` | `WorkspacesDB()` | Database target configurations (60+ engines) | `database_workspace_resource.go` |
| **Database Secrets** | `cyberarksia_database_secret` | `SecretsDB()` | Database credentials (username/password, AWS IAM) | `database_secret_resource.go` |
| **Certificates** | `cyberarksia_certificate` | Custom Client | TLS/mTLS certificates for database connections | `certificate_resource.go` |
| **Database Policies** | `cyberarksia_database_policy` | `Db()` (UAP) | Access policies with time-based conditions | `database_policy_resource.go` |
| **DB Policy Principal Assignments** | `cyberarksia_database_policy_principal_assignment` | `Db()` (UAP) | Assign users/groups/roles to DB policies (WHO) | `database_policy_principal_assignment_resource.go` |
| **DB Policy Workspace Assignments** | `cyberarksia_database_policy_workspace_assignment` | `Db()` (UAP) | Assign database workspaces to policies (WHAT) | `database_policy_workspace_assignment_resource.go` |
| **Principal Lookup** | `cyberarksia_principal` (data source) | Identity services | Look up users/groups/roles by name | `principal_data_source.go` |
| **VM Secrets** ✅ | `cyberarksia_virtual_machine_secret` | `SecretsVM()` | VM credentials (ProvisionerUser, PCloudAccount) | `virtual_machine_secret_resource.go` |
| **Target Sets** ✅ | `cyberarksia_target_set` | `WorkspacesTargetSets()` | VM/server target groupings with credentials | `target_set_resource.go` |
| **VM Policies** ✅ | `cyberarksia_vm_policy` | `VM()` (UAP) | VM access policies (FQDN/IP, AWS, Azure, GCP) | `vm_policy_resource.go` |
| **VM Policy Principal Assignments** ✅ | `cyberarksia_vm_policy_principal_assignment` | `VM()` (UAP) | Assign users/groups/roles to VM policies (WHO) | `vm_policy_principal_assignment_resource.go` |

### Implementation Notes (High-Level)

**Database Resources**:
- Full CRUD operations documented in [SDK Integration: Database Workspaces](../sdk-integration.md#sia-database-workspace-crud-operations)
- Secrets documented in [SDK Integration: Database Secrets](../sdk-integration.md#sia-database-secrets-crud-operations)
- Policies documented in [SDK Integration: Policy Assignment Deletion Pattern](../sdk-integration.md#policy-assignment-deletion-pattern)

**VM Resources** (Implemented):
- VM Secrets: Full CRUD with DELETE workaround (`DeleteVMSecretDirect`), client-side filtering
  - Details: [SDK Integration: VM Secrets](../sdk-integration.md#sia-vm-secrets-crud-operations)
- Target Sets: Name-as-ID pattern, DELETE and UPDATE workarounds
  - Details: [SDK Integration: Target Sets](../sdk-integration.md#sia-target-sets-crud-operations)
- VM Policies: Full CRUD with Azure workarounds for CREATE/READ/UPDATE
  - Supports all location types: FQDN/IP, AWS, Azure, GCP
  - Azure policies use `sdk_workarounds.go` functions (see Priority 5 in SDK Issues)
- VM Policy Principal Assignments: Full CRUD with Azure fallback pattern
  - Uses Read-Modify-Write pattern for updates
  - ImportState supports Azure policies via fallback

**Critical SDK Workarounds**:
All DELETE operations use workarounds from `internal/client/sdk_workarounds.go` to avoid nil pointer panic bug. Azure VM policies require additional workarounds for CREATE/READ/UPDATE operations. See [SDK Integration: SDK Limitations](../sdk-integration.md#sdk-limitations-and-workarounds) for complete workaround reference

---

## Available Services (Not Yet Implemented)

These services are **available in ARK SDK v1.5.0** and suitable for Terraform implementation:

### 8. VM Access Policies (`uap/sia/vm`) 🔥 **HIGH PRIORITY**

**Proposed Resources**:
- `cyberarksia_vm_policy`
- `cyberarksia_vm_policy_principal_assignment`
- `cyberarksia_vm_policy_target_assignment`

**SDK Service**: `ArkUAPSIAVMService`
- **API Accessor**: `VM()`
- **Location**: `pkg/services/uap/sia/vm/`

**Purpose**: Define access policies for VM/server targets with conditions and principals

**Schemas** (`pkg/services/uap/sia/vm/models/`):
- `ArkUAPSIAVMAccessPolicy` - VM policy structure
- `ArkUAPSIAVMFilters` - Query filters
- Uses shared UAP policy models (same as DB policies)

**Policy Structure**:
- `Metadata` - Name, description, status
- `Targets` - Map of workspace types to target sets (e.g., `{"FQDN/IP": [target-set-names]}`)
- `Principals` - Users, groups, roles with permissions
- `Conditions` - Time-based access rules (days of week, hours)

**CRUD Operations**:
```go
AddPolicy(policy *models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)
UpdatePolicy(policy *models.ArkUAPSIAVMAccessPolicy) (*models.ArkUAPSIAVMAccessPolicy, error)
DeletePolicy(request *models.ArkUAPDeletePolicyRequest) error  // Uses BaseDeletePolicy
Policy(request *models.ArkUAPGetPolicyRequest) (*models.ArkUAPSIAVMAccessPolicy, error)
ListPolicies() (<-chan *ArkUAPVMPolicyPage, error)  // Returns channel!
ListPoliciesBy(filter *models.ArkUAPSIAVMFilters) (<-chan *ArkUAPVMPolicyPage, error)  // Returns channel!
PolicyStatus(request *models.ArkUAPGetPolicyStatus) (string, error)
PoliciesStats() (*models.ArkUAPPoliciesStats, error)
```

**Dependencies**: Target Sets (references in policy targets)

**Implementation Notes**:
- Same UAP framework as database policies
- ⚠️ **PROVEN API Constraint**: Read-Modify-Write pattern REQUIRED
  - **Proof**: Working database policy implementation at `internal/provider/database_policy_workspace_assignment_resource.go:384-390`
  - **Evidence**: Comment states "CRITICAL: API only accepts ONE workspace type in Targets per update"
  - **Implementation**: Code explicitly constructs policy with ONLY modified workspace type in Targets map
  - **Pattern**: Fetch full policy → Modify specific element (principal/target) → Send back with ALL metadata/conditions but ONLY modified workspace type
  - **Reason**: This preserves unmanaged assignments (UI-created, other workspaces) - without this, updates would wipe out all other workspace types
  - **Testing**: Validated in `examples/testing/TESTING-GUIDE.md` and `examples/testing/azure-postgresql/TEST-RESULTS.md`
  - **Specification**: Formal requirement FR-018, FR-025 in `specs/002-sia-policy-lifecycle/spec.md`
- Composite ID pattern for assignments (3-part for principals, 2-part for targets)
- ⚠️ **Serialization Quirk**: Manual camel/snake conversions in SDK
  - Use set/hash functions in Terraform to avoid drift from ordering
- Can reuse assignment resource patterns from database policies (proven to work)

**Complexity**: Medium-High (serialization quirks + Read-Modify-Write)

**Value**: High (completes VM access management)

**Discovery Credit**: Claude (primary), validated by Gemini and Codex
**CLI Validation**: ✅ Confirmed via `ark exec uap vm` (all 8 operations present)

#### PoC Validation Results (2025-11-16)

**Validation Status**: ✅ **BASIC CRUD VALIDATED** - Happy-path scenario confirmed working
**Validation Date**: 2025-11-16
**Validation Scope**: ⚠️ **Single happy-path scenario only** (USER principal, FQDN/IP SUFFIX, SSH behavior)

**Test Results Summary**:
- ✅ **CREATE**: Policy created with USER principal, FQDN/IP SUFFIX rule, SSH behavior
- ✅ **READ**: Policy retrieved, targets and behavior deserialized correctly
- ✅ **UPDATE**: Description updated, targets/behavior preserved (read-modify-write pattern)
- ✅ **DELETE**: DELETE workaround successful, no panic, 404 confirmed after deletion
- ✅ **Principal UUID Lookup**: Identity API integration via `UserByName()` working
- ✅ **Required Metadata**: status, timeZone, daysOfTheWeek, delegationClassification all identified

**Critical Findings**:

1. **Required Metadata Fields** ✅ (Discovered via HTTP 400 error)
   - `TimeZone`: "GMT" (or other valid timezone)
   - `Status.Status`: "Active"
   - `DelegationClassification`: "Restricted" or "Unrestricted"
   - `Conditions.AccessWindow.DaysOfTheWeek`: []int{0,1,2,3,4,5,6} (required array)
   - `PolicyType`: "Recurring" or "OnDemand"
   - **API Error**: `UAP1005: invalid status, timeZone, daysOfTheWeek, delegationClassification`

2. **AccessWindow Type Discovery** ✅ (Compilation error revealed)
   - **Field Type**: `uapcommonmodels.ArkUAPTimeCondition` (VALUE, not pointer)
   - **Common Mistake**: Using `&uapcommonmodels.ArkUAPTimeCondition{...}` causes compilation error
   - **Correct**: `AccessWindow: uapcommonmodels.ArkUAPTimeCondition{DaysOfTheWeek: [...]}`

3. **LocationType Initialization** ⚠️
   - Must be set BEFORE calling `Serialize()`
   - Error if missing: "unsupported workspace type"
   - Location: `pkg/services/uap/sia/vm/models/ark_uap_sia_vm_targets.go:404-417`

4. **DELETE Workaround Confirmed** ✅
   - Existing `DeleteDatabasePolicyDirect()` works for VM policies (uses same `BaseDeletePolicy`)
   - Service: "uap", Endpoint: `/api/policies/{id}`
   - No new workaround needed

5. **Principal UUID Requirement** ✅
   - API requires valid UUIDs (not usernames)
   - Use Identity API: `users.UserByName(&usersmodels.ArkIdentityUserByName{Username: name})`
   - Same pattern as database policies
   - HTTP 500 error if username used instead of UUID

6. **SDK Method Signatures Validated** ✅
   ```go
   AddPolicy(*uapsiavmmodels.ArkUAPSIAVMAccessPolicy) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error)
   Policy(*uapcommonmodels.ArkUAPGetPolicyRequest) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error)
   UpdatePolicy(*uapsiavmmodels.ArkUAPSIAVMAccessPolicy) (*uapsiavmmodels.ArkUAPSIAVMAccessPolicy, error)
   DeletePolicy(*uapcommonmodels.ArkUAPDeletePolicyRequest) error // Uses BaseDeletePolicy
   ```

**Validated Patterns**:
```go
// Principal UUID Lookup (Identity API)
usersService, err := users.NewArkIdentityUsersService(ispAuth)
user, err := usersService.UserByName(&usersmodels.ArkIdentityUserByName{Username: username})
principalUUID := user.UserID

// Policy Construction with Required Fields
policy := &uapsiavmmodels.ArkUAPSIAVMAccessPolicy{
    ArkUAPSIACommonAccessPolicy: siacommonmodels.ArkUAPSIACommonAccessPolicy{
        ArkUAPCommonAccessPolicy: uapcommonmodels.ArkUAPCommonAccessPolicy{
            Metadata: uapcommonmodels.ArkUAPMetadata{
                TimeZone: "GMT", // REQUIRED
                PolicyEntitlement: uapcommonmodels.ArkUAPPolicyEntitlement{
                    LocationType:   commonmodels.WorkspaceTypeFQDNIP, // REQUIRED
                    PolicyType:     "Recurring", // REQUIRED
                },
                Status: uapcommonmodels.ArkUAPPolicyStatus{
                    Status: "Active", // REQUIRED
                },
            },
            DelegationClassification: "Unrestricted", // REQUIRED
        },
        Conditions: siacommonmodels.ArkUAPSIACommonConditions{
            ArkUAPConditions: uapcommonmodels.ArkUAPConditions{
                AccessWindow: uapcommonmodels.ArkUAPTimeCondition{ // VALUE, not pointer
                    DaysOfTheWeek: []int{0, 1, 2, 3, 4, 5, 6}, // REQUIRED
                },
            },
        },
    },
}

// DELETE Workaround
client, err := isp.FromISPAuth(ispAuth, "uap", ".", "", nil)
response, err := client.Delete(ctx, fmt.Sprintf("/api/policies/%s", policyID), map[string]string{})
```

**Known Limitations** ⚠️ (NOT tested in PoC - require acceptance testing during implementation):
- Multi-principal policies (GROUP/ROLE types, AD/LDAP directories)
- IP rules, DOMAIN/TARGET operators, AWS/Azure/GCP location types
- RDP profiles, combined SSH+RDP behaviors
- Import/list operations for `terraform import`
- OnDemand policy type, policy tag CRUD
- Error scenarios (invalid UUIDs, network errors, token expiration)

**Reusable Patterns** (from database policies):
- ✅ Authentication (`internal/client/auth.go`)
- ✅ DELETE workaround (`DeleteDatabasePolicyDirect`)
- ✅ Error handling (`MapError`, `RetryWithBackoff`)
- ✅ Principal lookup (via `cyberarksia_principal` data source or direct Identity API)
- ✅ Read-modify-write for updates

---

### 9. SSH CA Management (`sshca`) ⭐ **MEDIUM PRIORITY**

**Proposed Resources**:
- `cyberarksia_ssh_ca` (lifecycle management)
- `cyberarksia_ssh_ca_public_key` (data source)

**SDK Service**: `ArkSIASSHCAService`
- **API Accessor**: `SSHCa()`
- **Location**: `pkg/services/sia/sshca/`

**Purpose**: SSH Certificate Authority for certificate-based SSH authentication

**Schemas** (`pkg/services/sia/sshca/models/`):
- `ArkSIAGetSSHPublicKey` - Public key retrieval parameters

**Operations**:
```go
GenerateNewCA() error  // Create new CA key version
DeactivatePreviousCa() error  // Deactivate old CA key
ReactivatePreviousCa() error  // Reactivate old CA key
PublicKey(params *models.ArkSIAGetSSHPublicKey) (string, error)  // Get public key
PublicKeyScript(params *models.ArkSIAGetSSHPublicKey) (string, error)  // Get deployment script
```

**Use Case**: SSH certificate-based authentication for server access

**Dependencies**: None

**Implementation Notes**:
- Lifecycle management (generate, deactivate, reactivate)
- Public key retrieval for deployment to SSH servers
- Versioned CA keys (can have multiple versions)
- ⚠️ **Security**: Key rotations are irreversible
  - Consider `lifecycle { prevent_destroy = true }` in Terraform
- May be better split:
  - **Resource**: `cyberarksia_ssh_ca` (lifecycle management)
  - **Data Source**: `cyberarksia_ssh_ca_public_key` (key retrieval)
- Deployment script generation useful for automation

**Complexity**: Medium (lifecycle management + security safeguards)

**Value**: Medium-High (enables certificate-based SSH access)

**Discovery Credit**: Claude (primary), validated by Gemini and Codex
**CLI Validation**: ✅ Confirmed via `ark exec sia ssh-ca` (all 5 operations present)

---

### 10. Connectors (`access`) ⚠️ **QUESTIONABLE FIT**

**Proposed Resource**: `cyberarksia_connector` (NOT recommended)
**Alternative**: `cyberarksia_connector_setup_script` (data source)

**SDK Service**: `ArkSIAAccessService`
- **API Accessor**: `Access()`
- **Location**: `pkg/services/sia/access/`

**Purpose**: Manage SIA connectors for private network access to targets

**Schemas** (`pkg/services/sia/access/models/`):
- `ArkSIAInstallConnector` - Installation parameters (SSH/WinRM credentials)
- `ArkSIAUninstallConnector` - Uninstallation parameters
- `ArkSIAConnectorSetupScript` - Installation script
- `ArkSIATestConnectorReachability` - Connectivity test

**Operations**:
```go
InstallConnector(connector *models.ArkSIAInstallConnector) (*models.ArkSIAAccessConnectorID, error)
UninstallConnector(connector *models.ArkSIAUninstallConnector) error
DeleteConnector(connector *models.ArkSIADeleteConnector) error
ConnectorSetupScript(params *models.ArkSIAGetConnectorSetupScript) (*models.ArkSIAConnectorSetupScript, error)
TestConnectorReachability(test *models.ArkSIATestConnectorReachability) (*models.ArkSIAReachabilityTestResponse, error)
```

**Connection Types**:
- SSH (Linux/Darwin)
- WinRM (Windows)

**Dependencies**: None (infrastructure component)

**Implementation Notes**:
- ⚠️ **Complex**: Requires remote execution on target machines via SSH/WinRM
- ⚠️ **Side Effects**: Automated installation via remote commands, long-running operations
- ⚠️ **Partial Idempotency**: Operations only partially idempotent
- ⚠️ **Drift Detection**: Heavy side effects make drift detection risky
- Includes retry logic and health checks in SDK
- **Recommendation** (engineering judgment): Data source for setup scripts preferred over full lifecycle management
- **Note**: Infrastructure component with operational complexity

**Complexity**: High (remote execution + side effects + partial idempotency)

**Value**: Medium (infrastructure component, may fit better outside Terraform)

**Note**: These operational concerns are engineering judgments based on Terraform best practices, not SDK limitations.

**Recommendations**:
1. **Phase 1**: Implement `cyberarksia_connector_setup_script` data source (returns script content)
2. **Phase 2**: Evaluate need for full lifecycle management based on user feedback
3. **Alternative**: Consider external provisioning tool or Terraform provisioner

**Discovery Credit**: Claude (primary), Codex validated concerns, Gemini noted complexity

---

## CLI/End-User Services (Excluded)

These services are **available in ARK SDK v1.5.0** but are **NOT suitable for Terraform** (end-user CLI functionality, not infrastructure management):

### SSO Token Generation (`sso`)

**SDK Service**: `ArkSIASSOService`
- **API Accessor**: `Sso()`
- **Location**: `pkg/services/sia/sso/`

**Purpose**: Generate short-lived credentials for end-user access

**Operations**:
- `ShortLivedPassword()` - Generate temporary password
- `ShortLivedClientCertificate()` - Generate client certificate
- `ShortLivedOracleWallet()` - Generate Oracle wallet
- `ShortLivedRdpFile()` - Generate RDP file
- `ShortLivedSSHKey()` - Generate SSH key

**Why Excluded**:
- End-user CLI tool functionality
- Outputs are ephemeral (not infrastructure state)
- Often written to local filesystem
- Poor fit for Terraform's declarative model

**Alternative**: Could be exposed as **data source** with `Sensitive: true` for automation workflows (Phase 4, optional)

---

### Kubeconfig Generation (`k8s`)

**SDK Service**: `ArkSIAK8SService`
- **API Accessor**: `K8s()`
- **Location**: `pkg/services/sia/k8s/`

**Purpose**: Generate kubeconfig for Kubernetes access

**Operations**:
- `GenerateKubeconfig()` - Download kubeconfig for DPA-managed clusters

**CLI Routing Bug** ⚠️:
- The CLI incorrectly routes this to `ark exec sia db` instead of `ark exec sia k8s`
- Running `ark exec sia db --help` shows "SIA K8S Actions" with `generate-kubeconfig`
- `ark exec sia k8s` does not exist as a command
- This appears to shadow/conflict with the DB CLI tools service

**Why Excluded**:
- End-user CLI tool functionality
- Writes directly to `~/.kube/config` (local filesystem manipulation)
- Ephemeral configuration (not infrastructure state)

**Alternative**: Could be exposed as **data source** returning content (without forcing filesystem writes) (Phase 4, optional)

---

### DB CLI Tools (`db`)

**SDK Service**: `ArkSIADBService`
- **API Accessor**: `Db()`
- **Location**: `pkg/services/sia/db/`

**Purpose**: Execute database CLI commands and generate connection assets

**Operations** (in SDK):
- `Psql()` - Execute PostgreSQL psql command
- `Mysql()` - Execute MySQL mysql command
- `Sqlcmd()` - Execute SQL Server sqlcmd command
- `GenerateOracleTnsNames()` - Generate Oracle tnsnames.ora
- `GenerateProxyFullChain()` - Generate proxy certificate chain

**CLI Availability** ⚠️:
- ❌ **NOT exposed in CLI** - These operations are not available via `ark` command
- The `ark exec sia db` command is incorrectly routed to K8s functionality (see above)
- SDK service exists and works, but CLI doesn't expose these operations

**Why Excluded** (even if CLI access existed):
- End-user CLI functionality
- Wraps local command execution (requires installed DB clients)
- Depends on local filesystem and shell
- Brittle automation (varies by host environment)

**Alternative**: Leave to external tooling or shell scripts (not Terraform-appropriate)

---

## Critical SDK Issues

### Priority 1: DELETE Panic Bug ⚠️ **CRITICAL** - ✅ **PROVEN**

**Validation Status**: ✅ **REPRODUCED IN PRODUCTION & PoCs**
- CyberArk's own CLI panics with this bug (tested 2025-11-02)
- Dual independent PoCs confirmed (tested 2025-11-15)
  - Primary PoC: `/tmp/target-sets-poc/`
  - Codex PoC: `/tmp/target-sets-poc-codex/`

**CyberArk CLI Testing**: Confirmed `delete-secret` and `delete-target-set` crash with identical panic (nil pointer dereference in `http.NewRequestWithContext`). **CRITICAL: The API call NEVER succeeds and resources are NOT deleted** - panic occurs during HTTP request construction, BEFORE the request is sent to the API.

**PoC Validation (2025-11-15)**:
- ✅ **Both PoCs**: VM Secret DELETE panics with nil pointer dereference
- ✅ **Both PoCs**: Target Set DELETE panics with nil pointer dereference
- ✅ **Both PoCs**: Panic occurs in `bytes.(*Buffer).Len` called from `http.NewRequestWithContext`
- ✅ **Both PoCs**: Workarounds using ISP client with empty body `{}` succeed

**Affected Services**:
- `SecretsVM().DeleteSecret()` ✅ Confirmed in CLI + both PoCs
- `WorkspacesTargetSets().DeleteTargetSet()` ✅ Confirmed in CLI + both PoCs
- Potentially `VM().DeletePolicy()` (verify if uses `BaseDeletePolicy()`)

**Root Cause**:
- SDK file: `pkg/common/ark_client.go:556-576`
- Problem: `doRequest()` doesn't handle nil body pointer
- Delete methods pass `nil` body → panic in request construction

**Impact**:
- **ALL** new VM resources will hit this bug on Delete operations
- Provider will panic instead of graceful error handling

**Validation Testing** - Reproduced in CyberArk's Production CLI (2025-11-02):

**Test 1: VM Secrets Delete**
```bash
# Create test secret
$ ark exec sia secrets vm add-secret --secret-type ProvisionerUser \
  --secret-name "claude-test-delete-bug-1762104557" \
  --provisioner-username testuser --provisioner-password "TestPass123!" --raw
# Result: Created secret ID 636046ff-7690-4874-b046-a5ea614e23a3

# Attempt delete
$ ark exec sia secrets vm delete-secret --secret-id 636046ff-7690-4874-b046-a5ea614e23a3 --raw
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x8 pc=0x6c7504]

goroutine 1 [running]:
bytes.(*Buffer).Len(...)
	/usr/local/go/src/bytes/buffer.go:85
net/http.NewRequestWithContext({0xd94100, 0x1361500}, {0xc5f116?, 0xc0004948d0?}, ...)
	/usr/local/go/src/net/http/request.go:926 +0x384
github.com/cyberark/ark-sdk-golang/pkg/common.(*ArkClient).doRequest(...)
	.../ark-sdk-golang@v1.5.0/pkg/common/ark_client.go:576 +0x4df
github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/vm.(*ArkSIASecretsVMService).DeleteSecret(...)
	.../ark-sdk-golang@v1.5.0/pkg/services/sia/secrets/vm/ark_sia_secrets_vm_service.go:181 +0x12b

# Verify deletion status
$ ark exec sia secrets vm list-secrets --raw | grep "636046ff-7690-4874-b046-a5ea614e23a3"
# Result: SECRET STILL EXISTS (not deleted)
```

**Test 2: Target Sets Delete**
```bash
# Create test target set
$ ark exec sia workspaces target-sets add-target-set \
  --name "claude-test-targetset-1762104635" \
  --secret-type ProvisionerUser --secret-id "636046ff-7690-4874-b046-a5ea614e23a3" \
  --type Target --raw
# Result: Created target set "claude-test-targetset-1762104635"

# Attempt delete
$ ark exec sia workspaces target-sets delete-target-set --id "claude-test-targetset-1762104635" --raw
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x8 pc=0x6c7504]

goroutine 1 [running]:
bytes.(*Buffer).Len(...)
	/usr/local/go/src/bytes/buffer.go:85
net/http.NewRequestWithContext(...)
	/usr/local/go/src/net/http/request.go:926 +0x384
github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/targetsets.(*ArkSIAWorkspacesTargetSetsService).DeleteTargetSet(...)
	.../ark-sdk-golang@v1.5.0/pkg/services/sia/workspaces/targetsets/ark_sia_workspaces_target_sets_service.go:137 +0x12b

# Verify deletion status
$ ark exec sia workspaces target-sets list-target-sets --raw | grep "claude-test-targetset-1762104635"
# Result: TARGET SET STILL EXISTS (not deleted)
```

**Why It Happens**: `nil *bytes.Buffer` cast to `io.Reader` becomes non-nil interface with nil pointer value → panic when `http.NewRequestWithContext()` calls `.Len()`

**Critical Finding**: Panic occurs in `NewRequestWithContext()` during HTTP request construction, **BEFORE** any network I/O. The HTTP DELETE request is never sent to the API, so resources remain unchanged.

**Current Workaround**:
- File: `internal/client/sdk_workarounds.go`
- Functions: `DeleteDatabaseWorkspaceDirect()`, `DeleteSecretDirect()`, `DeleteDatabasePolicyDirect()`
- Pattern: Bypass SDK methods, call API directly with empty map `map[string]string{}` instead of `nil`

**Required Action for New Resources**:
```go
// ✅ CORRECT - Use workaround
err := client.DeleteDatabaseWorkspaceDirect(ctx, providerData.AuthContext, databaseID)

// ❌ WRONG - Will panic
err := siaAPI.WorkspacesDB().DeleteDatabase(databaseID)
```

**Long-term Solution**: Remove workaround when ARK SDK v1.6.0+ fixes nil body handling

**Note on Policy Assignments**: Database policy assignment resources (`cyberarksia_database_policy_workspace_assignment`, `cyberarksia_database_policy_principal_assignment`) do NOT hit this bug because they use a different deletion pattern:
- They call `UpdatePolicy()` (Read-Modify-Write pattern), not `DeletePolicy()`
- Delete operation: Fetch policy → Remove assignment from array → Update policy via API
- API validates constraints (≥1 principal, ≥1 target) during UpdatePolicy call
- If constraint violated, API returns clear error that we translate to helpful message
- This approach also avoids race conditions and blocking valid destroy flows
- See: `internal/provider/database_policy_workspace_assignment_resource.go:560-636`

**Discovery Credit**: Claude (existing knowledge), confirmed impact on new services

---

### Priority 2: VM Secrets Filtering Bug ⚠️ **FUNCTIONAL** - ✅ **PROVEN**

**Validation Status**: ✅ **CONFIRMED IN PRODUCTION** - Filtering completely broken in CyberArk's own CLI

**Affected Service**: `SecretsVM().ListSecretsBy()`

**Validation Testing** - Reproduced in CyberArk's Production CLI (2025-11-02):

```bash
# Step 1: List all secrets to confirm data exists
$ ark exec sia secrets vm list-secrets --raw
[
  {
    "secret_id": "501d42eb-213b-4cb4-be22-f8b938e87026",
    "secret_type": "ProvisionerUser",
    "secret_name": "ark-cli-delete-test-1762103925"
  },
  {
    "secret_id": "9989a180-9922-4ee1-9a0f-78ffd85abad2",
    "secret_type": "PCloudAccount",
    "secret_name": "Operating System-cyberiamtechWindowsDomainAccountsviaLDAP-..."
  }
]

# Test 1: Filter by secret_type "PCloudAccount"
$ ark exec sia secrets vm list-secrets-by --secret-types PCloudAccount --raw
[]  # WRONG: Should return the PCloudAccount secret

# Test 2: Filter by secret_type "ProvisionerUser"
$ ark exec sia secrets vm list-secrets-by --secret-types ProvisionerUser --raw
[]  # WRONG: Should return the ProvisionerUser secret

# Test 3: Filter by name
$ ark exec sia secrets vm list-secrets-by --name "ark-cli-delete-test" --raw
[]  # WRONG: Should return the matching secret
```

**Result**: ALL filters return empty arrays despite matching data existing. Filtering is completely non-functional.

**Location**: `pkg/services/sia/secrets/vm/ark_sia_secrets_vm_service.go:191-204`

**Root Cause** (Lines 192-203):
```go
func (s *ArkSIASecretsVMService) listSecretsWithFilter(...) {
    filterJSON := map[string]string{}  // Line 192: Build filter
    if secretType != "" {
        filterJSON["secret_type"] = secretType  // Line 194: Populate
    }
    // Lines 196-201: Add more filter criteria

    response, err := s.client.Get(context.Background(), secretsURL, nil)  // Line 203: BUG!
    //                                                                              ^^^
    //                                                                     filterJSON NEVER USED!
}
```
- `listSecretsWithFilter()` builds filterJSON correctly
- BUT passes `nil` instead of filterJSON to API
- API receives no filter parameters, returns ALL secrets unfiltered

**Impact**:
- Filtering by secret type, name, or other criteria **does not work**
- `ListSecretsBy()` returns unfiltered results (same as `ListSecrets()`)

**Workaround**:
- Use `ListSecrets()` (no filtering)
- Implement client-side filtering in Terraform provider
- Example:
  ```go
  allSecrets, err := siaAPI.SecretsVM().ListSecrets()
  // Filter in Go code
  filtered := filterBySecretType(allSecrets, desiredType)
  ```

**Performance Impact**:
- Minimal for small deployments
- May be inefficient for large-scale environments (100s+ secrets)

**Long-term Solution**:
- Report bug to ARK SDK team
- Wait for SDK v1.6.0+ fix
- Consider switching to SDK filtering once available

**Discovery Credit**: Codex (code-level analysis)

---

### Priority 3: Target Sets Name-as-ID and UPDATE Behavior ⚠️ **DESIGN PATTERN** - ✅ **VALIDATED 2025-11-15**

**Validation Status**: ✅ **CONFIRMED via dual independent PoCs** - Both primary and Codex PoCs validated behavior

**Affected Service**: `WorkspacesTargetSets()`

#### Name-as-ID Pattern

**SDK Pattern**:
- API returns `name` field (string)
- SDK maps `name` to `id` field internally
- Updates and deletes use `name` parameter, not numeric ID

**Comparison to Database Workspaces**:
- Database workspaces use numeric `id` (e.g., 12345)
- Target sets use string `name` (e.g., "prod-web-servers")

**Terraform Impact**:
- Target set names must be **immutable**
- Name changes require `ForceNew = true` (destroy + recreate)
- Cannot use numeric ID reference patterns from database resources

**Implementation Pattern**:
```hcl
resource "cyberarksia_target_set" "example" {
  name = "prod-web-servers"  # IMMUTABLE - ForceNew on change
  # ... other fields
}
```

**Schema Consideration**:
```go
"name": schema.StringAttribute{
    Required:    true,
    Description: "Name of the target set (immutable, used as identifier)",
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(), // ForceNew
    },
}
```

#### UPDATE Behavior and omitempty Tags

**Validation Testing** - Dual PoC Results (2025-11-15):

**PoC 1 (Primary)** `/tmp/target-sets-poc/`:
- ✅ UPDATE with all fields populated: **SUCCESS**
- Initial test with partial fields: 403 error (later determined to be test setup issue)

**PoC 2 (Codex)** `/tmp/target-sets-poc-codex/`:
- ✅ UPDATE with all fields populated: **SUCCESS**
- ❌ UPDATE with only Description field: **500 Internal Server Error**
- **Root cause identified**: `mapstructure.Decode()` honors `omitempty` tags

**SDK Implementation Analysis**:

The SDK's `UpdateTargetSet()` method (line 180-215 in `ark_sia_workspaces_target_sets_service.go`):
1. Uses `mapstructure.Decode()` to convert struct to map
2. Removes `id` field from map
3. Sends map as JSON body via PUT request

**The `ArkSIAUpdateTargetSet` model** has `omitempty` tags on all fields except `ID`:
```go
type ArkSIAUpdateTargetSet struct {
    ID                          string `json:"id"` // No omitempty
    Name                        string `json:"name,omitempty"` // Dropped if ""
    Description                 string `json:"description,omitempty"` // Dropped if ""
    ProvisionFormat             string `json:"provision_format,omitempty"` // Dropped if ""
    EnableCertificateValidation bool   `json:"enable_certificate_validation,omitempty"` // Dropped if false
    SecretType                  string `json:"secret_type,omitempty"` // Dropped if ""
    SecretID                    string `json:"secret_id,omitempty"` // Dropped if ""
    Type                        string `json:"type,omitempty"` // Dropped if ""
}
```

**Impact**:
- Zero-value fields are omitted from JSON payload
- Partial updates fail with API validation errors (500)
- **SDK method WORKS when all fields are populated**

**Provider Workaround Analysis**:

Current implementation uses `UpdateTargetSetDirect()` workaround. **This workaround is REQUIRED and cannot be removed.**

**❌ Failed Refactor Attempt (2025-11-15)**:

Attempted to use SDK's `UpdateTargetSet()` method directly (Option B below) based on PoC validation showing "SDK works when all fields populated". The refactor FAILED with 3 test failures:

1. **TestAccTargetSet_descriptionUpdate**: "Provider produced inconsistent result - description was null, but now cty.StringVal(\"Updated description\")"
2. **TestAccTargetSet_certValidation**: "Provider produced inconsistent result - enable_certificate_validation was cty.False, but now cty.True"
3. **TestAccTargetSet_provisionFormatNoClearing**: "expected an error but got none"

**Root Cause** (confirmed by Codex analysis):
- SDK's `omitempty` tags drop ALL zero values: `""` for strings, `false` for booleans
- Even explicit `Description: ""` or `EnableCertificateValidation: false` are omitted from JSON
- API interprets missing fields as "leave unchanged"
- Users cannot clear descriptions or set booleans to false using the SDK struct
- State fallback approach violates Terraform's plan contract (plan must match result)

**PoC Limitation**: PoCs only tested non-clearing updates (all fields stayed populated). They didn't test:
- Clearing description to empty string
- Setting enable_certificate_validation to false
- Removing provision_format

**Option A (REQUIRED)**: Use workaround with manual map building
```go
updateRequest := map[string]interface{}{
    "name":        plan.Name.ValueString(),
    "description": plan.Description.ValueString(), // Sends "" to clear
    "enable_certificate_validation": false, // Sends false explicitly
    // ... all fields explicitly included
}
result, err := client.UpdateTargetSetDirect(ctx, authCtx, oldName, updateRequest)
```

**Option B (DOES NOT WORK)**: ~~Use SDK method with all fields from state/plan~~
```go
// ❌ FAILS - omitempty drops zero values, preventing field clearing
updateReq := &targetsetmodels.ArkSIAUpdateTargetSet{
    ID:                          state.Name.ValueString(),
    Name:                        plan.Name.ValueString(),
    Description:                 "", // DROPPED by omitempty
    EnableCertificateValidation: false, // DROPPED by omitempty
    // ...
}
updated, err := siaAPI.WorkspacesTargetSets().UpdateTargetSet(updateReq)
```

**Recommendation**: The workaround MUST remain until the SDK is fixed.

**Why Workaround Cannot Be Removed**:
- ❌ SDK's `omitempty` tags drop zero values even when explicitly set
- ❌ Cannot clear descriptions (send empty string)
- ❌ Cannot set enable_certificate_validation to false
- ❌ Users need ability to clear/disable fields
- ❌ Terraform plan contract requires exact planned values to be applied

**When SDK Can Be Used (Future)**:
- ✅ SDK removes `omitempty` tags from mutable fields
- ✅ SDK uses pointer types for optional fields (`*string`, `*bool`)
- ✅ Or we create custom struct without `omitempty` for updates

**Discovery Credit**:
- Initial PoC validation: Primary + Codex (2025-11-15)
- Refactor failure analysis: Claude + Codex (2025-11-15)
- Root cause (omitempty blocking zero values): Codex deep analysis

---

### Priority 4: Serialization Quirks ⚠️ **STABILITY** - ✅ **VERIFIED**

**Validation Status**: Confirmed manual camel/snake conversions exist in VM policies
**Impact**: Potential field ordering variations due to Go map iteration randomness

**Affected Services**: VM Policies (verified), potentially DB policies

**Location**: `pkg/services/uap/sia/vm/ark_uap_sia_vm_service.go:63,84,101` + `pkg/common/ark_serializer.go`

**Issue**: Manual camel/snake case conversions iterate over maps (random order in Go)

**Impact**:
- Field ordering in API responses may vary
- `mapstructure` decoding can produce different field orders across calls
- Terraform detects false drift if comparing ordered lists

**Workaround**:
- Use `schema.SetAttribute` instead of `schema.ListAttribute` where order doesn't matter
- Implement custom hash functions for complex nested objects
- Example:
  ```go
  // Use Set for unordered collections
  "principals": schema.SetNestedAttribute{
      // ... hash function ensures stable comparison
  }
  ```

**Most SIA Services Pattern**:
- Most services use `snake_case` payloads decoded via `common.DeserializeJSONSnake` and `mapstructure`
- **Exception**: SSH CA service uses raw `io.ReadAll` without mapstructure
- Pattern is common but not universal across SDK

**Best Practice**:
- Mirror SDK's snake_case in Terraform schema
- Test for drift with multiple Read operations
- Use set/hash functions liberally

**Discovery Credit**: Codex (code-level analysis)

---

### Priority 5: Azure VM Policy Serialization Bug ⚠️ **CRITICAL** - ✅ **FIXED**

**Validation Status**: ✅ **REPRODUCED AND FIXED** (2025-11-25)
**GitHub Issue**: https://github.com/cyberark/ark-sdk-golang/issues/32

**Affected Services**: `VM().AddPolicy()`, `VM().Policy()`, `VM().UpdatePolicy()` for Azure location type

**Root Cause**: Multiple SDK serialization issues with Azure VM policies:

1. **Targets Key Casing**: SDK uses `"AZURE"` (uppercase) but API expects `"Azure"` (mixed case)
   - SDK's `Serialize()` produces `targets.azureResource` or `targets.AZURE`
   - API expects `targets.Azure` in JSON payload
   - Causes HTTP 500 on CREATE/UPDATE

2. **LocationType Casing**: Same issue in `metadata.policyEntitlement.locationType`
   - SDK uses `"AZURE"`, API expects `"Azure"`

3. **Behavior Structure**: SDK serializes SSH/RDP profiles incorrectly
   - SDK produces: `behavior.sshProfile`
   - API expects: `behavior.connectAs.ssh`

4. **UPDATE Server-Managed Fields**: API returns HTTP 500 if UPDATE request includes:
   - `metadata.policyId` (should be in URL only)
   - `metadata.status.statusCode` (read-only)
   - `metadata.status.statusDescription` (read-only)
   - `metadata.createdBy`, `metadata.updatedOn` (server-managed timestamps)

5. **UPDATE Empty Response**: API returns HTTP 200 with empty body for PUT requests
   - SDK's `UpdatePolicy()` tries to decode empty body → EOF error

**Impact**:
- SDK's `AddPolicy()` fails for Azure location type (HTTP 500)
- SDK's `Policy()` fails to deserialize Azure policies ("unsupported workspace type")
- SDK's `UpdatePolicy()` fails for Azure policies (HTTP 500 or EOF)
- Principal assignment resources fail for Azure policies

**Workarounds Implemented** (`internal/client/sdk_workarounds.go`):

```go
// CREATE: Fix JSON structure before sending
CreateAzureVMPolicyDirect(ctx, authCtx, policy *ArkUAPSIAVMAccessPolicy)
// - Converts targets.azureResource → targets.Azure
// - Fixes behavior.sshProfile → behavior.connectAs.ssh
// - Fixes locationType: AZURE → Azure

// READ: Fix response before deserialization
ReadAzureVMPolicyDirect(ctx, authCtx, policyID string)
// - Converts targets.Azure → targets.AZURE (for SDK compatibility)
// - Fixes locationType: Azure → AZURE

// UPDATE: Remove server-managed fields + handle empty response
UpdateAzureVMPolicyDirect(ctx, authCtx, policyID string, policy *ArkUAPSIAVMAccessPolicy)
// - Removes policyId, statusCode, statusDescription from request
// - Removes createdBy, updatedOn, timeFrame
// - Does follow-up GET after successful UPDATE (empty response handling)
```

**Principal Assignment Resource Pattern**:
```go
// All methods (Create/Read/Delete/ImportState) use fallback pattern:
policy, err := vmService.Policy(...)
if err != nil && strings.Contains(err.Error(), "unsupported workspace type") {
    // Fallback to Azure workaround
    policy, err = client.ReadAzureVMPolicyDirect(ctx, authCtx, policyID)
}
```

**Detection Logic**: Check `plan.LocationType.ValueString() == "Azure"` before API calls

**Test Validation**: `TestAccVMPolicyPrincipalAssignment_azure` - Full CRUD + ImportState passing

**Long-term Solution**: Remove workarounds when ARK SDK v1.6.0+ fixes Azure serialization

**Discovery Credit**: Claude (implementation debugging), validated via acceptance tests

---

### Other Known Issues

**No Context Support in Authenticate()**:
- Cannot cancel authentication mid-flight via context
- First parameter is `*ArkProfile` (optional), NOT `context.Context`
- Impact: Long authentication hangs cannot be interrupted
- Workaround: Use timeout at HTTP client level

**No Structured Errors**:
- SDK returns generic `error` interface with string messages
- No error type checking (no `errors.As()` support)
- Workaround: Use `internal/client.MapError()` for classification

**15-Minute Token Expiration**:
- SDK handles automatic token refresh
- In-memory profile pattern (stateless, container-friendly)
- No action needed, but aware of refresh behavior

---

## Implementation Roadmap

### Phase 1: VM Infrastructure (Complete ✅)

**Goal**: Extend provider to VM/server management (mirrors existing DB resources)

**Resources**:
1. ✅ `cyberarksia_virtual_machine_secret` (VM credentials) - **COMPLETED** (commit 0fabb1c)
2. ✅ `cyberarksia_target_set` (VM workspaces) - **COMPLETED** (commit e309c9a)
3. ✅ `cyberarksia_vm_policy` (VM access policies) - **COMPLETED** (commit de11dda)
4. ✅ `cyberarksia_vm_policy_principal_assignment` (WHO gets access) - **COMPLETED** (commit 4571f9c)
5. ❌ `cyberarksia_vm_policy_target_assignment` - **NOT NEEDED** (targets managed inline via vm_policy resource)

**Dependencies**:
```
✅ VM Secrets → ✅ Target Sets → ✅ VM Policies → ✅ Principal Assignments
```

**Progress**: 4 of 4 resources completed (100%)

**Completed Implementation Notes**:
- ✅ DELETE workarounds implemented (`sdk_workarounds.go`)
  - `DeleteVMSecretDirect()` - Bypasses nil body panic
  - `DeleteTargetSetDirect()` - Bypasses nil body panic
  - `UpdateTargetSetDirect()` - Bypasses omitempty serialization issues
  - `ChangeVMSecretDirect()` - Fixes POST→PUT bug
- ✅ Azure VM policy workarounds implemented (`sdk_workarounds.go`)
  - `CreateAzureVMPolicyDirect()` - Fixes Azure targets key casing ("AZURE" → "Azure")
  - `ReadAzureVMPolicyDirect()` - Handles Azure response deserialization
  - `UpdateAzureVMPolicyDirect()` - Removes server-managed fields, handles empty response
- ✅ Target sets use string name as ID (ForceNew pattern implemented)
- ✅ Full CRUD validation completed (27 VM policy tests passing)
- ✅ Azure fallback pattern in principal assignment Create/Read/Delete/ImportState

**Note**: Client-side filtering not needed for VM secrets resource (uses single-secret reads only, not list operations)

---

### Phase 2: SSH CA (Medium Priority)

**Goal**: Add certificate-based SSH authentication

**Resources to Implement**:
7. `cyberarksia_ssh_ca` (SSH CA lifecycle)
8. `cyberarksia_ssh_ca_public_key` (data source - public key retrieval)

**Dependencies**: None (standalone)

**Effort Estimate**: Low-Medium (1-2 weeks)
- Lifecycle management patterns
- Security safeguards implementation

**Value**: Medium-High (enables certificate-based SSH access)

**Testing Requirements**:
- SSH CA key rotation tests with safeguards
- Public key retrieval validation
- Lifecycle management validation

**Critical Implementation Notes**:
- Consider `prevent_destroy` lifecycle rule (irreversible rotations)
- Split resource (lifecycle) and data source (key retrieval)
- Security-critical component

---

### Phase 3: Data Sources for Automation (Optional)

**Goal**: Provide convenience data sources for automation workflows

**Data Sources to Implement**:
9. `cyberarksia_connector_setup_script` (connector installation script)
10. `cyberarksia_kubeconfig` (kubeconfig content)
11. `cyberarksia_sso_credential` (short-lived credentials)

**Dependencies**:
- Connector script: None
- Kubeconfig: None
- SSO credential: None

**Effort Estimate**: Low (1-2 weeks)
- Read-only data sources
- No lifecycle management
- Sensitive data handling for credentials

**Value**: Low-Medium (convenience features)

**Testing Requirements**:
- Data source retrieval validation
- Sensitive data masking tests
- Integration with existing resources

**Critical Implementation Notes**:
- Mark all outputs as `Sensitive: true`
- Avoid filesystem writes (return content only)
- Document use cases clearly

---

### Explicitly Out of Scope

**Services NOT Planned for Implementation**:
- ❌ Full Connector lifecycle management (too many side effects)
- ❌ DB CLI tools (psql/mysql/sqlcmd execution)
- ❌ Local filesystem manipulation tools

**Rationale**:
- Poor fit for Terraform's declarative model
- Heavy side effects or local dependencies
- Better served by external tooling (Ansible, shell scripts)

---

## Architectural Insights

### Pattern Reuse Opportunities

**VM Secrets ← Database Secrets**:
- Similar structure to `database_secret_resource.go`
- Reuse sensitive data handling patterns
- Client-side filtering not needed (resource only reads single secrets)
- DELETE workaround implemented in sdk_workarounds.go

**Target Sets ← Database Workspaces**:
- Similar schema patterns
- Adjust for string ID vs numeric ID
- Handle ForceNew on name changes
- Implement DELETE workaround

**VM Policies ← Database Policies**:
- Reuse UAP policy framework
- Same Read-Modify-Write pattern
- Same composite ID patterns for assignments
- Add set/hash functions for stability

**SSH CA ← Certificate Resource**:
- Similar security considerations
- Lifecycle management patterns
- Sensitive data handling

### Authentication & Authorization

**All Services Use Same Pattern**:
- OAuth2/ISP authentication
- Same `ProviderData` struct
- 15-minute token expiration with automatic refresh
- No additional credentials needed

**Implication**: No provider configuration changes needed for new services

### Error Handling Strategy

**Consistent Pattern Across Services**:
```go
// Wrap SDK calls with retry logic
err := client.RetryWithBackoff(ctx, func() error {
    _, err := siaAPI.Service().Operation(...)
    return err
})

// Convert to Terraform diagnostics
if err != nil {
    resp.Diagnostics.Append(client.MapError(err, "Operation failed")...)
    return
}
```

**Benefits**:
- Automatic exponential backoff (3 retries, 30s max delay)
- Error classification (auth, permission, network, etc.)
- Actionable user messages

### Testing Strategy

**Primary: Acceptance Tests**:
- Test against real SIA API when `TF_ACC=1`
- Verify CRUD operations end-to-end
- Test ImportState functionality
- Test ForceNew behavior and drift detection

**Prerequisites** (`examples/testing/TESTING-GUIDE.md`):
- Environment variables: `CYBERARK_USERNAME`, `CYBERARK_PASSWORD`, `TF_ACC=1`
- Service account scopes: `sia`, `identity`
- CyberArk tenant with SIA enabled

**CRUD Validation**:
- Automated: `make test-crud DESC=<resource-description>`
- Manual: Follow `examples/testing/TESTING-GUIDE.md` workflow
- All validation checks must pass

**Selective: Unit Tests**:
- Complex validators only
- Error classification logic
- Retry logic
- Helper utilities

---

## Testing Considerations

### Real API Testing

**Requirements**:
- CyberArk Identity tenant with SIA enabled
- OAuth2 service account with `sia` and `identity` scopes
- Test data (secrets, workspaces, policies)

**Challenges**:
- Cross-service dependencies (secrets → workspaces → policies)
- Cleanup after failed tests
- Rate limiting considerations
- Concurrent test execution

**Mitigation**:
- Use `examples/testing/TESTING-GUIDE.md` templates
- Copy templates to `/tmp` for isolation
- Implement proper cleanup in test teardown
- Use unique names with timestamps

### Sensitive Data Handling

**Requirements**:
- Mask passwords, tokens, client secrets in test logs
- Use `Sensitive: true` attribute in schemas
- Redact sensitive data in state
- Avoid logging sensitive fields

**Testing**:
- Verify sensitive attributes don't appear in plans/applies
- Test state file doesn't contain plaintext secrets
- Verify logs don't contain credentials

### Drift Detection

**Requirements**:
- Test Read after Create (no unexpected changes)
- Test Read after Update (only intended changes)
- Test multiple Read operations (stable results)
- Test external modifications (drift detection)

**Special Cases**:
- VM policies: Test for false drift (serialization quirks)
- Target sets: Test name immutability
- Secrets: Test sensitive data stability

### Bug-Specific Testing

**VM Secrets Filtering**:
- Test that client-side filtering works correctly
- Verify all secret types can be filtered
- Test performance with large secret lists

**DELETE Workarounds**:
- Verify DELETE operations succeed (no panic)
- Test DELETE with non-existent resources (404 handling)
- Test DELETE with dependencies (error handling)

**Target Sets Name-as-ID**:
- Test that name changes trigger ForceNew
- Test that name is properly used for Read/Update/Delete
- Test import with name-based ID

---

## Recommendations

### Immediate Actions

1. **Prioritize VM Infrastructure (Phase 1)**
   - Natural extension of existing DB resources
   - High user demand
   - Can reuse established patterns

2. **Verify SDK v1.6.0 Roadmap**
   - Check if DELETE panic bug is fixed
   - Check if VM filtering bug is fixed
   - Plan for workaround removal

3. **Create Implementation Specs**
   - Use SpecKit slash commands for each resource
   - Document schemas, dependencies, testing requirements
   - Get user validation before implementation

4. **Enhance Delete Workarounds**
   - Add VM secret delete function
   - Add target set delete function
   - Verify VM policy delete behavior

### Strategic Decisions

**Connector Resource (Complex)**:
- Codex and Claude both flag implementation concerns
- Consider data source only (setup scripts)
- Evaluate user demand for full lifecycle
- May be better served by external tooling

**SSO Service Clarification**:
- SDK provides SSO token generation only (CLI-appropriate)
- No separate SSO settings management service exists in SDK v1.5.0
- Tokens: Can be exposed as data source (Phase 3, optional)

### Documentation Needs

1. **Update CLAUDE.md**
   - Reference this research document
   - Add VM resource patterns to anti-patterns section
   - Document new SDK bugs and workarounds

2. **Update docs/sdk-integration.md**
   - Add VM services SDK mappings
   - Document serialization patterns
   - Add filtering bug notes

3. **Update docs/troubleshooting.md**
   - Add VM-specific troubleshooting
   - Document client-side filtering workaround
   - Add name-as-ID pattern guidance

4. **Create Implementation Guides**
   - VM resources step-by-step guide
   - K8s cluster management guide
   - SSH CA management guide

### Long-term Planning

**SDK Dependency Management**:
- Track ARK SDK releases for bug fixes
- Plan migration away from workarounds
- Consider contributing fixes upstream

**Feature Prioritization**:
- Gather user feedback on Phase 2+ features
- Assess K8s vs. Accounts priority
- Evaluate enterprise features demand (SSO settings)

**Testing Infrastructure**:
- Invest in robust acceptance test framework
- Consider test environment automation
- Implement comprehensive CRUD templates

---

## Conclusion

The ARK SDK v1.5.0 analysis identified 9 SIA services. The Terraform provider currently implements **11 resources and 1 data source** (12 total components) covering database management, VM/server management, access policies, and principal lookup. **1 service remains available** for implementation: SSH CA (medium priority).

**Key Takeaways**:
1. **Phase 1 complete**: VM Secrets, Target Sets, VM Policies, and VM Policy Principal Assignments all implemented (100% complete).
2. **Validation is critical**: Multi-perspective research identified 3 hallucinated services (K8s clusters, Accounts, Platforms)
3. **SDK bugs are manageable**: Workarounds implemented in sdk_workarounds.go (including Azure serialization fix)
4. **Clear implementation path**: Phase-based roadmap focused on verified services
5. **Provider scope**: 11 resources + 1 data source currently implemented across database and VM management. 1 service available for future implementation (SSH CA). 3 CLI-only services excluded.

**Next Steps**:
1. Create implementation specs using SpecKit for Phase 2 (SSH CA)
2. Validate Phase 2/3 priorities with stakeholders
3. Monitor ARK SDK v1.6.0+ for bug fixes to potentially remove workarounds

---

**Document Version**: 2.0 (Phase 1 Complete + Azure Bug Fix)
**Last Updated**: 2025-11-25
**Research Methodology**: Multi-perspective (Claude + Gemini + Codex) with SDK source code validation + **CyberArk ark CLI production testing** + **Dual independent PoCs**
**Implementation Update**: Phase 1 VM Infrastructure complete (VM Secrets, Target Sets, VM Policies, VM Policy Principal Assignments)
**PoC Validation (2025-11-15)**:
- Primary PoC: `/tmp/target-sets-poc/` - Full CRUD validation, SDK method testing
- Codex PoC: `/tmp/target-sets-poc-codex/` - Independent validation, partial update failure proof
- Key Finding: UpdateTargetSet() SDK method works when all fields populated (workaround unnecessary for providers)
**Azure Bug Fix (2025-11-25)**:
- Identified and fixed Azure VM policy serialization issues (GitHub #32)
- Created workarounds for CREATE/READ/UPDATE operations
- Added Azure fallback pattern to principal assignment resource
- All 27 VM policy tests passing
**ARK SDK Version Analyzed**: v1.5.0
**Validation Status**:
- ✅ All services confirmed against CyberArk's official `ark` CLI at `/home/tim/go/bin/ark`
- ✅ All operations verified in production CLI (VM secrets, target sets, SSH CA, VM policies)
- ✅ **SDK bugs FULLY REPRODUCED in CyberArk's production CLI** (Validation Date: 2025-11-02):
  - DELETE panic: ✅ Confirmed with `delete-secret` and `delete-target-set` (crashes with nil pointer, resources NOT deleted)
  - VM filtering: ✅ Confirmed with `list-secrets-by` (all filters return `[]` despite matching data existing)
- ✅ SDK source code verified at `/home/tim/go/pkg/mod/github.com/cyberark/ark-sdk-golang@v1.5.0/pkg/services/sia/ark_sia_api.go`
- ✅ Hallucinated services verified as non-existent (Platforms, Accounts, K8s Clusters management, SSO Settings)
- ✅ Document inconsistency corrected: Removed `cyberarksia_platform` from Phase 1 roadmap (was correctly identified as hallucinated in appendix)
- ⚠️ **CLI routing bug identified**: `ark exec sia db` incorrectly routes to K8s functionality; DB CLI tools not exposed in CLI
- ⚠️ Operational recommendations are based on Terraform best practices and proven API constraints

---

## Appendix: Service Summary Table

### Currently Implemented Services

| Service | Type | Status | Complexity | SDK Location | DELETE Bug | Notes |
|---------|------|--------|------------|--------------|------------|-------|
| Database Workspaces | Resource | ✅ Implemented | Medium | `sia/workspaces/db/` | ✅ Workaround | 60+ database engines, DeleteDatabaseWorkspaceDirect |
| Database Secrets | Resource | ✅ Implemented | Low-Medium | `sia/secrets/db/` | ✅ Workaround | Username/password, AWS IAM, DeleteSecretDirect |
| VM Secrets | Resource | ✅ Implemented | Medium | `sia/secrets/vm/` | ✅ Workaround | ProvisionerUser/PCloudAccount, DeleteVMSecretDirect, ChangeVMSecretDirect |
| Target Sets | Resource | ✅ Implemented | Medium | `sia/workspaces/targetsets/` | ⚠️ Partial | Name-as-ID pattern, DeleteTargetSetDirect (required), UpdateTargetSetDirect (optional - can use SDK) |
| Certificates | Resource | ✅ Implemented | Low | `certificates` (custom client) | N/A | TLS/mTLS certificates |
| Database Policies | Resource | ✅ Implemented | High | `uap/sia/db/` | ✅ Workaround | Access policies with time conditions, DeleteDatabasePolicyDirect |
| Policy Assignments | Resource | ✅ Implemented | Medium | `uap/sia/db/` | N/A | Principal & workspace assignments (Read-Modify-Write pattern) |
| Principal Lookup | Data Source | ✅ Implemented | Low | `identity/directories/`, `identity/users/` | N/A | Lookup users/groups/roles by name (hybrid lookup) |

### Services Available for Implementation

| Service | Type | Priority | Complexity | SDK Location | DELETE Bug | Notes |
|---------|------|----------|------------|--------------|------------|-------|
| VM Policies | Resource | High | Med-High | `uap/sia/vm/` | ❓ TBD | Serialization quirks, verify delete |
| SSH CA | Resource | Medium | Medium | `sia/sshca/` | ❓ TBD | Lifecycle mgmt, verify delete |
| SSH CA Public Key | Data Source | Medium | Low | `sia/sshca/` | N/A | Key retrieval |
| Connector Script | Data Source | Low | Low | `sia/access/` | N/A | Read-only, returns script |
| Kubeconfig | Data Source | Low | Low | `sia/k8s/` | N/A | Returns content |
| SSO Credentials | Data Source | Low | Low | `sia/sso/` | N/A | Ephemeral tokens |

### Hallucinated Services (DO NOT EXIST in SDK v1.5.0)

| Service | Claimed Source | Why It Doesn't Exist |
|---------|---------------|----------------------|
| K8s Clusters (`sia.NewK8SClusters`) | Gemini | No cluster management service; only kubeconfig generation exists |
| Accounts (`sia.NewAccounts`) | Gemini | No account management service in SIA |
| Platforms (`sia.NewPlatforms`) | Gemini | No platform management service in SIA |
| SSO Settings (`sia.NewSSO` for settings) | Gemini | Only token generation exists, no settings management |

**Legend**:
- ✅ Workaround: DELETE bug workaround implemented in sdk_workarounds.go
- ⚠️ Yes: DELETE panic bug confirmed, workaround required
- ❓ TBD: Needs verification (likely affected)
- N/A: Not applicable (data sources, no delete operation)

---

**End of Document**
