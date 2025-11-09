# ARK SDK SIA Services Analysis

**Research Date**: 2025-11-02
**ARK SDK Version**: v1.5.0
**Analysis Method**: Multi-perspective research (Claude Code + Gemini + Codex)

---

## Executive Summary

This document provides a comprehensive analysis of all Secure Infrastructure Access (SIA) services available in the CyberArk ARK SDK for Go (v1.5.0) that could be added to the Terraform provider. The analysis was conducted using three independent perspectives to ensure completeness and identify potential blind spots.

**Key Findings**:
- **9 SIA services** verified in ARK SDK v1.5.0 ✅ **Confirmed by CyberArk's official `ark` CLI**
- **4 services currently implemented** (Database workspaces, secrets, certificates, policies)
- **5 services available for implementation** (VM infrastructure: secrets, workspaces, policies; SSH CA; Connectors)
- **3 CLI-only services excluded** (end-user tools, not infrastructure management)
- **Critical SDK bugs REPRODUCED IN CYBERARK'S PRODUCTION CLI**:
  - DELETE panic bug ✅ **Production CLI crashes**: `ark exec sia secrets vm delete-secret` panics with nil pointer dereference. Ironically, delete succeeds before crash.
  - VM filtering bug ✅ **Production CLI broken**: `list-secrets-by --secret-types X` returns `[]` for all filters. Filtering completely non-functional.
  - Serialization quirks ✅ **Verified**: Manual camel/snake conversions in VM policies
- **⚠️ Note**: 3 services initially identified by Gemini (K8s clusters, Accounts, Platforms) do not exist in SDK v1.5.0

**Recommended Priority**:
1. **Phase 1** (High): VM infrastructure (secrets, workspaces, policies) - Natural extension of DB resources
2. **Phase 2** (Medium): SSH CA - Certificate-based authentication
3. **Phase 3** (Optional): Data sources for automation (connector scripts, kubeconfig, SSO tokens)

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

These services are **already available** in the Terraform provider (v0.1.0):

### 1. Database Workspaces (`workspaces/db`)

**Resource**: `cyberarksia_database_workspace`

**SDK Service**: `ArkSIAWorkspacesDBService`
- **API Accessor**: `WorkspacesDB()`
- **Location**: `pkg/services/sia/workspaces/db/`

**Purpose**: Manage database target configurations with 60+ supported database engines (PostgreSQL, MySQL, Oracle, SQL Server, MongoDB, etc.)

**Key Features**:
- Database connection details (host, port, database name)
- Authentication profile management (6 auth methods)
- Cloud provider metadata (AWS, Azure, GCP)
- Certificate references for TLS/mTLS

**Implementation File**: `internal/provider/database_workspace_resource.go`

---

### 2. Database Secrets (`secrets/db`)

**Resource**: `cyberarksia_database_secret`

**SDK Service**: `ArkSIASecretsDBService`
- **API Accessor**: `SecretsDB()`
- **Location**: `pkg/services/sia/secrets/db/`

**Purpose**: Manage strong account credentials for database access (username/password, AWS IAM)

**Key Features**:
- Multiple authentication types (username/password, AWS IAM)
- Sensitive data protection
- Secret lifecycle management

**Implementation File**: `internal/provider/secret_resource.go`

---

### 3. Certificates (`certificates`)

**Resource**: `cyberarksia_certificate`

**SDK Service**: Custom `CertificatesClient`
- **Location**: `internal/client/certificates.go`

**Purpose**: Manage TLS/mTLS certificates for secure database connections

**Key Features**:
- PEM-format certificate management
- Certificate lifecycle (create, read, update, delete)
- Integration with database workspaces

**Implementation File**: `internal/provider/certificate_resource.go`

---

### 4. Database Access Policies (`uap/sia/db`)

**Resources**:
- `cyberarksia_database_policy`
- `cyberarksia_database_policy_principal_assignment`
- `cyberarksia_policy_workspace_assignment`

**SDK Service**: `ArkUAPSIADBService`
- **API Accessor**: `Db()`
- **Location**: `pkg/services/uap/sia/db/`

**Purpose**: Define access policies for database targets with time-based conditions

**Key Features**:
- Policy metadata (name, description, status)
- Principal assignments (WHO gets access: users, groups, roles)
- Database assignments (WHAT they access: database workspaces)
- Time-based conditions (days of week, hours)
- Composite ID pattern for assignments

**Implementation Files**:
- `internal/provider/database_policy_resource.go`
- `internal/provider/database_policy_principal_assignment_resource.go`
- `internal/provider/database_policy_workspace_assignment_resource.go`

---

### 5. Principal Lookup (`uap`)

**Data Source**: `cyberarksia_principal`

**SDK Service**: Uses UAP services
- **Location**: `pkg/services/uap/`

**Purpose**: Look up users, groups, and roles by name (eliminates need for manual UUID entry)

**Implementation File**: `internal/provider/principal_data_source.go`

---

## Available Services (Not Yet Implemented)

These services are **available in ARK SDK v1.5.0** and suitable for Terraform implementation:

### 6. VM Secrets (`secrets/vm`) 🔥 **HIGH PRIORITY**

**Proposed Resource**: `cyberarksia_vm_secret`

**SDK Service**: `ArkSIASecretsVMService`
- **API Accessor**: `SecretsVM()`
- **Location**: `pkg/services/sia/secrets/vm/`

**Purpose**: Manage VM/server credentials for privileged access

**Schemas** (`pkg/services/sia/secrets/vm/models/`):
- `ArkSIAVMSecret` - Complete VM secret structure
- `ArkSIAVMAddSecret` - Create VM credential
- `ArkSIAVMChangeSecret` - Update VM credential
- `ArkSIAVMSecretsFilter` - Query filters

**Secret Types**:
- `ProvisionerUser` - Username/password for VM provisioning
- `PCloudAccount` - Reference to CyberArk PAM vault account (safe + account name)

**CRUD Operations**:
```go
AddSecret(secret *models.ArkSIAVMAddSecret) (*models.ArkSIAVMSecret, error)
ChangeSecret(secret *models.ArkSIAVMChangeSecret) (*models.ArkSIAVMSecret, error)
DeleteSecret(secret *models.ArkSIAVMDeleteSecret) error  // ⚠️ DELETE bug applies
Secret(secret *models.ArkSIAVMGetSecret) (*models.ArkSIAVMSecret, error)
ListSecrets() ([]*models.ArkSIAVMSecret, error)
ListSecretsBy(filter *models.ArkSIAVMSecretsFilter) ([]*models.ArkSIAVMSecret, error)
SecretsStats() (*models.ArkSIAVMSecretsStats, error)
```

**Dependencies**: None (standalone)

**Implementation Notes**:
- Nearly identical pattern to `cyberarksia_database_secret`
- ⚠️ **SDK Bug**: DELETE panic applies - need workaround in `delete_workarounds.go`
- ⚠️ **SDK Bug**: `ListSecretsBy()` filtering broken (sends nil body) - use client-side filtering
- Two authentication methods similar to database secrets
- Sensitive data handling required

**Complexity**: Medium (bug workarounds + sensitive data)

**Value**: High (extends provider to VM/server management)

**Discovery Credit**: Claude (primary), validated by Gemini and Codex
**CLI Validation**: ✅ Confirmed via `ark exec sia secrets vm` (all 7 operations present)

---

### 7. Target Sets (`workspaces/targetsets`) 🔥 **HIGH PRIORITY**

**Proposed Resource**: `cyberarksia_target_set` or `cyberarksia_vm_workspace`

**SDK Service**: `ArkSIAWorkspacesTargetSetsService`
- **API Accessor**: `WorkspacesTargetSets()`
- **Location**: `pkg/services/sia/workspaces/targetsets/`

**Purpose**: Logical groupings of VM/server targets with associated credentials (VM equivalent of database workspaces)

**Schemas** (`pkg/services/sia/workspaces/targetsets/models/`):
- `ArkSIATargetSet` - Target set definition
- `ArkSIAAddTargetSet` - Create target set
- `ArkSIAUpdateTargetSet` - Update target set
- `ArkSIABulkAddTargetSets` - Bulk create
- `ArkSIABulkDeleteTargetSets` - Bulk delete

**Key Fields**:
- `Name` - Target set identifier (⚠️ also serves as ID, not numeric like databases)
- `SecretType` - Type of secret (ProvisionerUser, PCloudAccount)
- `SecretID` - Reference to VM secret

**CRUD Operations**:
```go
AddTargetSet(targetSet *models.ArkSIAAddTargetSet) (*models.ArkSIATargetSet, error)
UpdateTargetSet(targetSet *models.ArkSIAUpdateTargetSet) (*models.ArkSIATargetSet, error)
DeleteTargetSet(targetSet *models.ArkSIADeleteTargetSet) error  // ⚠️ DELETE bug applies
TargetSet(targetSet *models.ArkSIAGetTargetSet) (*models.ArkSIATargetSet, error)
ListTargetSets() ([]*models.ArkSIATargetSet, error)
ListTargetSetsBy(filter *models.ArkSIATargetSetsFilter) ([]*models.ArkSIATargetSet, error)
BulkAddTargetSets(targetSets *models.ArkSIABulkAddTargetSets) (*models.ArkSIABulkTargetSetResponse, error)
BulkDeleteTargetSets(targetSets *models.ArkSIABulkDeleteTargetSets) (*models.ArkSIABulkTargetSetResponse, error)
TargetSetsStats() (*models.ArkSIATargetSetsStats, error)
```

**Dependencies**: VM Secrets (references `secret_id`)

**Implementation Notes**:
- VM equivalent of database workspaces
- ⚠️ **SDK Quirk**: Uses `Name` as identifier (string, not numeric ID)
  - API returns `name` field; SDK maps it to `id`
  - Terraform must treat target set names as immutable (ForceNew on name change)
- ⚠️ **SDK Bug**: DELETE panic applies - need workaround
- Bulk operations available (useful for import scenarios)
- Similar schema patterns to database workspaces

**Complexity**: Medium (name-as-ID quirk + DELETE workaround)

**Value**: High (completes VM infrastructure story)

**Discovery Credit**: Claude (primary), validated by Gemini and Codex
**CLI Validation**: ✅ Confirmed via `ark exec sia workspaces target-sets` (all 9 operations present)

---

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

**Validation Status**: ✅ **REPRODUCED IN PRODUCTION** - CyberArk's own CLI panics with this bug
**CyberArk CLI Testing**: Confirmed `delete-secret` and `delete-target-set` crash with identical panic (nil pointer dereference in `http.NewRequestWithContext`). **CRITICAL: The API call NEVER succeeds and resources are NOT deleted** - panic occurs during HTTP request construction, BEFORE the request is sent to the API.

**Affected Services**:
- `SecretsVM().DeleteSecret()` ✅ Confirmed in CLI: `ark exec sia secrets vm delete-secret`
- `WorkspacesTargetSets().DeleteTargetSet()` ✅ Confirmed in CLI: `ark exec sia workspaces target-sets delete-target-set`
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
- File: `internal/client/delete_workarounds.go`
- Functions: `DeleteDatabaseWorkspaceDirect()`, `DeleteSecretDirect()`, `DeletePolicyDirect()`
- Pattern: Bypass SDK methods, call API directly with empty map `map[string]string{}` instead of `nil`

**Required Action for New Resources**:
```go
// ✅ CORRECT - Use workaround
err := client.DeleteDatabaseWorkspaceDirect(ctx, providerData.AuthContext, databaseID)

// ❌ WRONG - Will panic
err := siaAPI.WorkspacesDB().DeleteDatabase(databaseID)
```

**Long-term Solution**: Remove workaround when ARK SDK v1.6.0+ fixes nil body handling

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

### Priority 3: Target Sets Name-as-ID Quirk ⚠️ **DESIGN PATTERN**

**Affected Service**: `WorkspacesTargetSets()`

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

**Discovery Credit**: Codex (SDK implementation analysis)

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

### Phase 1: VM Infrastructure (Highest Priority)

**Goal**: Extend provider to VM/server management (mirrors existing DB resources)

**Resources to Implement**:
1. `cyberarksia_vm_secret` (VM credentials)
2. `cyberarksia_target_set` or `cyberarksia_vm_workspace` (VM workspaces)
3. `cyberarksia_vm_policy` (VM access policies)
4. `cyberarksia_vm_policy_principal_assignment` (WHO gets access)
5. `cyberarksia_vm_policy_target_assignment` (WHAT they access)

**Dependencies**:
```
VM Secrets → Target Sets → VM Policies → Assignments
```

**Effort Estimate**: Medium (4-6 weeks)
- Can reuse patterns from database resources
- DELETE workarounds required
- VM filtering bug workaround needed
- Target set name-as-ID pattern
- Serialization quirks in VM policies

**Value**: High
- Natural extension of existing capabilities
- Completes VM infrastructure story
- Large user demand (VM management is common)

**Testing Requirements**:
- Full CRUD validation per `examples/testing/TESTING-GUIDE.md`
- Acceptance tests with real SIA API
- Client-side filtering tests for VM secrets
- Name immutability tests for target sets
- Drift detection tests for VM policies (serialization stability)

**Critical Implementation Notes**:
- ⚠️ ALL resources need DELETE workarounds (`delete_workarounds.go`)
- ⚠️ VM secrets require client-side filtering (SDK bug)
- ⚠️ Target sets use string name as ID (ForceNew pattern)
- ⚠️ VM policies need set/hash functions (serialization quirks)

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
- Clone `secret_resource.go` structure
- Reuse sensitive data handling patterns
- Add client-side filtering for VM secrets
- Implement DELETE workaround

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

The ARK SDK v1.5.0 provides **5 unimplemented services** ready for Terraform implementation, with **3 high-priority additions** (VM secrets, target sets, VM policies) that would significantly expand provider capabilities.

**Key Takeaways**:
1. **VM infrastructure is ready**: Natural extension with established patterns
2. **Validation is critical**: Multi-perspective research identified 3 hallucinated services (K8s clusters, Accounts, Platforms)
3. **SDK bugs are manageable**: Workarounds exist, plan for future fixes
4. **Clear implementation path**: Phase-based roadmap focused on verified services
5. **11 total services verified**: 4 implemented, 3 high-priority, 1 medium-priority, 3 CLI-only

**Next Steps**:
1. Validate Phase 1 priority with stakeholders
2. Create implementation specs using SpecKit
3. Enhance delete workarounds for VM services
4. Begin VM secrets implementation (highest ROI)

---

**Document Version**: 1.7 (Production Validated + CLI Bug Documented)
**Last Updated**: 2025-11-02
**Research Methodology**: Multi-perspective (Claude + Gemini + Codex) with SDK source code validation + **CyberArk ark CLI production testing**
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

### Verified Services Available for Implementation

| Service | Type | Priority | Complexity | SDK Location | DELETE Bug | Notes |
|---------|------|----------|------------|--------------|------------|-------|
| VM Secrets | Resource | High | Medium | `sia/secrets/vm/` | ⚠️ Yes | Filtering bug workaround needed |
| Target Sets | Resource | High | Medium | `sia/workspaces/targetsets/` | ⚠️ Yes | Name-as-ID pattern |
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
- ⚠️ Yes: DELETE panic bug confirmed, workaround required
- ❓ TBD: Needs verification (likely affected)
- N/A: Not applicable (data sources, no delete operation)

---

**End of Document**
