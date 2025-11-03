# Feature Specification: Virtual Machine Secret Management

**Feature Branch**: `003-virtual-machine-secret`
**Created**: 2025-11-02
**Completed**: 2025-11-03
**Status**: Implementation Complete - Testing PASSED (18/18 tests) ✅
**Input**: User description: "Terraform users need to manage virtual machine and server credentials for privileged access through CyberArk Secure Infrastructure Access (SIA). This feature provides credential lifecycle management for VM/server access, similar to how database credentials are currently managed in the provider."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create VM Secrets (Priority: P1)

Infrastructure teams need to create virtual machine and server credentials in SIA to enable privileged access for administrators and automation systems. Users must be able to create either self-contained credentials (username/password managed in SIA) or references to existing PAM vault accounts.

**Why this priority**: Secret creation is the foundational capability. Without the ability to create secrets, no other operations are possible. This is the minimum viable functionality that delivers immediate value - users can establish credentials for VM access.

**Independent Test**: Can be fully tested by creating a VM secret with ProvisionerUser type (username and password), verifying it appears in SIA UI with correct secret ID and name, and confirming the secret can be referenced by its ID in subsequent operations.

**Acceptance Scenarios**:

1. **Given** an authenticated Terraform provider, **When** I define a VM secret resource with secret_type "ProvisionerUser", secret_name "app-server-admin", provisioner_username "admin", and provisioner_password "SecureP@ss123", **Then** the secret is created successfully in SIA with a unique secret_id returned
2. **Given** an authenticated Terraform provider, **When** I define a VM secret resource with secret_type "PCloudAccount", secret_name "vault-db-admin", pcloud_safe_name "Production-Safe", and pcloud_account_name "db-admin-account", **Then** the secret is created successfully referencing the PAM vault account
3. **Given** I attempt to create a VM secret with an invalid secret_type, **When** I run terraform plan, **Then** I receive a validation error before API call indicating valid secret types are "ProvisionerUser" and "PCloudAccount"
4. **Given** I create a VM secret with provisioner_password, **When** I run terraform plan on subsequent runs, **Then** the password does not appear in plan output (marked sensitive)

---

### User Story 2 - Read and Verify VM Secrets (Priority: P1)

Operations teams need to read existing VM secret metadata to verify configuration, detect drift from desired state, and validate that secrets exist before referencing them in target sets or policies.

**Why this priority**: Read operations are essential for Terraform's core functionality - drift detection and state management. Without read capability, Terraform cannot verify that the actual infrastructure matches the desired state defined in configuration. This is foundational for maintaining infrastructure integrity.

**Independent Test**: Can be fully tested by creating a VM secret, running terraform plan with no changes, and verifying Terraform detects no drift. Additionally, manually modifying the secret name in SIA UI and running terraform plan should detect the drift.

**Acceptance Scenarios**:

1. **Given** a VM secret exists in SIA, **When** Terraform reads the secret during refresh, **Then** secret metadata (secret_id, secret_name, secret_type) is retrieved successfully but sensitive data (passwords) is never returned
2. **Given** a VM secret was created by Terraform, **When** I run terraform plan with no configuration changes, **Then** no modifications are detected (idempotent read)
3. **Given** a VM secret was manually modified in SIA UI (name changed), **When** I run terraform plan, **Then** Terraform detects the drift and proposes updating the secret to match configuration
4. **Given** a VM secret was deleted outside Terraform, **When** Terraform reads the secret during refresh, **Then** Terraform detects the secret is missing and proposes recreating it

---

### User Story 3 - Update VM Secret Metadata (Priority: P2)

Security teams need to update VM secret metadata (name, credentials) as access requirements change, password rotation policies require new credentials, or organizational naming conventions evolve.

**Why this priority**: After creation and verification (P1), teams need to maintain secrets over time. This is essential lifecycle management but less critical than initial creation since users can work around updates by destroying and recreating resources if necessary.

**Independent Test**: Can be fully tested by creating a VM secret, modifying the secret_name or credentials in Terraform configuration, running terraform apply, and verifying the changes are reflected in SIA without changing the secret_id.

**Acceptance Scenarios**:

1. **Given** an existing VM secret named "old-server-admin", **When** I change the secret_name to "new-server-admin" in configuration and run terraform apply, **Then** the secret name is updated in-place (same secret_id) without recreating the secret
2. **Given** an existing ProvisionerUser secret, **When** I change the provisioner_password in configuration and run terraform apply, **Then** the password is updated and marked as sensitive in plan output
3. **Given** an existing PCloudAccount secret, **When** I change the pcloud_account_name reference, **Then** the secret is updated to reference the new PAM account
4. **Given** I attempt to change secret_type from "ProvisionerUser" to "PCloudAccount", **When** I run terraform plan, **Then** Terraform proposes ForceNew (destroy and recreate) since secret type is immutable

---

### User Story 4 - Import Existing VM Secrets (Priority: P2)

Platform engineers need to import VM secrets that were created manually in SIA UI or via other tools into Terraform management to enable infrastructure-as-code adoption for existing environments.

**Why this priority**: Import enables brownfield adoption - critical for teams with existing SIA deployments who want to adopt Terraform without disrupting running systems. Not as critical as core CRUD since it's a one-time migration activity.

**Independent Test**: Can be fully tested by manually creating a VM secret in SIA UI, running terraform import with the secret_id, and verifying the secret is imported into Terraform state with all metadata correctly populated.

**Acceptance Scenarios**:

1. **Given** a VM secret exists in SIA created outside Terraform (secret_id "abc-123-def"), **When** I run terraform import cyberarksia_virtual_machine_secret.example abc-123-def, **Then** the secret is imported into Terraform state with metadata populated
2. **Given** an imported secret, **When** I run terraform plan, **Then** no changes are detected (imported state matches actual resource)
3. **Given** I attempt to import a non-existent secret_id, **When** I run terraform import, **Then** I receive a clear error message indicating the secret was not found
4. **Given** an imported secret contains sensitive data (passwords), **When** viewing Terraform state, **Then** passwords are not included in state file (API never returns them)

---

### User Story 5 - Delete VM Secrets (Priority: P3)

Operations teams need to delete VM secrets when servers are decommissioned, access is revoked, or secrets are no longer needed to maintain clean infrastructure state.

**Why this priority**: Deletion is important for cleanup but lowest priority because it's less frequent than creation/updates and typically happens during decommissioning rather than active operations.

**Independent Test**: Can be fully tested by creating a VM secret, removing it from Terraform configuration, running terraform destroy, and verifying the secret no longer exists in SIA.

**Acceptance Scenarios**:

1. **Given** a VM secret exists in Terraform management, **When** I remove the resource from configuration and run terraform destroy, **Then** the secret is deleted from SIA successfully
2. **Given** I attempt to delete a secret that is referenced by target sets or policies, **When** I run terraform destroy, **Then** the operation succeeds (deletion not blocked by references per SIA API behavior)
3. **Given** I delete a secret, **When** I attempt to read it again via API, **Then** I receive a not found error (404)
4. **Given** a secret was already deleted outside Terraform, **When** I run terraform destroy, **Then** the operation succeeds gracefully (idempotent deletion)

---

### Edge Cases

**Secret Creation**:
- What happens when creating a ProvisionerUser secret without provisioner_password? (Validation error - password required for this secret type)
- What happens when creating a PCloudAccount secret with invalid pcloud_safe_name that doesn't exist in PAM? (API error returned with clear message indicating safe not found)
- What happens when secret_name exceeds maximum length constraints? (Validation error before API call with length limit specified)
- What happens when creating a secret with duplicate secret_name? (API allows duplicate names - secret_id is unique identifier, not name)

**Secret Reading**:
- What happens when reading a secret that was deleted outside Terraform? (404 not found error, Terraform marks resource for recreation)
- What happens when SIA API is temporarily unavailable during read? (Retry with exponential backoff per provider retry logic, eventual timeout with clear error)
- What happens when reading a secret with credentials that have expired or been rotated in PAM? (Read succeeds - SIA stores reference, not actual credentials; PAM handles expiration)

**Secret Updates**:
- What happens when updating provisioner_password to an empty value? (Validation error - passwords cannot be empty)
- What happens when updating PCloudAccount to reference a safe the user doesn't have access to? (API error returned with permission denied message)
- What happens when attempting to update a deleted secret? (404 not found error, Terraform proposes recreation)
- What happens when changing from ProvisionerUser to PCloudAccount? (ForceNew required - secret type is immutable, must destroy and recreate)

**Secret Import**:
- What happens when importing a secret with an incorrectly formatted secret_id? (Clear validation error indicating expected format)
- What happens when importing the same secret into multiple Terraform resources? (Last import wins - state management issue, user responsibility to avoid)
- What happens when secret exists but user lacks read permissions? (API error returned with permission denied message)

**Secret Deletion**:
- What happens when deleting a secret that is currently in use by active target sets? (Deletion succeeds per API behavior - target sets will fail validation when accessed)
- What happens when deletion fails due to API error? (Error surfaced to user, secret remains in Terraform state)
- What happens when attempting to delete an already-deleted secret? (Idempotent - no error, treat as success)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support creating VM secrets with two distinct secret types: "ProvisionerUser" (username/password stored in SIA) and "PCloudAccount" (reference to PAM vault account)
- **FR-002**: System MUST validate secret_type values and reject invalid types before API calls with clear error messages
- **FR-003**: System MUST mark all password fields (provisioner_password) as sensitive to prevent exposure in Terraform plans and state output
- **FR-004**: System MUST treat passwords as write-only - passwords can be set during create/update but are never returned from API reads
- **FR-005**: System MUST use secret_id (UUID) as the unique identifier for all read/update/delete operations
- **FR-006**: System MUST support full CRUD lifecycle: Create, Read, Update, Delete operations for VM secrets
- **FR-007**: System MUST support Terraform import functionality using secret_id to import existing secrets into Terraform management
- **FR-008**: System MUST implement idempotent operations - repeated applies with same configuration produce no changes
- **FR-009**: System MUST detect drift when secrets are modified or deleted outside Terraform and propose corrective actions
- **FR-010**: System MUST validate required fields per secret type: provisioner_username and provisioner_password for ProvisionerUser; pcloud_safe_name and pcloud_account_name for PCloudAccount
- **FR-011**: System MUST follow the same resource patterns as cyberarksia_database_secret for consistency within the provider
- **FR-012**: System MUST use internal/client/delete_workarounds.go for Delete operations to avoid ARK SDK DELETE panic bug
- **FR-013**: System MUST handle API errors gracefully with clear error messages and appropriate HTTP status code classification
- **FR-014**: System MUST implement exponential backoff retry logic for transient API failures per provider standards
- **FR-015**: System MUST support secret_name updates in-place using PUT method (ARK SDK ChangeSecret uses POST causing updates to fail, ChangeVMSecretDirect workaround required)
- **FR-016**: System MUST enforce ForceNew (destroy/recreate) when secret_type is changed since this is immutable
- **FR-017**: System MUST allow duplicate secret_name values since secret_id is the unique identifier

### Key Entities

- **Virtual Machine Secret**: A credential entity for VM/server access with two variants:
  - **ProvisionerUser**: Self-contained username/password credential stored in SIA
    - Attributes: secret_id (UUID), secret_name (string), secret_type (enum), provisioner_username (string), provisioner_password (sensitive string, write-only)
  - **PCloudAccount**: Reference to an existing PAM vault account
    - Attributes: secret_id (UUID), secret_name (string), secret_type (enum), pcloud_safe_name (string), pcloud_account_name (string)
  - Common attributes: secret_id serves as immutable identifier, secret_name is mutable user-facing label
  - Lifecycle: Created via SIA API, referenced by target sets for VM access provisioning, deleted when access is revoked

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-002**: Drift detection accurately identifies 100% of manual changes (name modifications, deletions) within next terraform plan execution
- **SC-003**: Password fields never appear in plaintext in Terraform plan output or standard logs (marked sensitive)
- **SC-004**: Import operations successfully bring existing secrets under Terraform management with zero data loss in metadata
- **SC-005**: Users familiar with cyberarksia_database_secret resource can create VM secrets without consulting documentation (consistent resource patterns)
- **SC-007**: Teams can adopt VM secret management in brownfield environments within 1 day using import functionality
- **SC-008**: All CRUD operations complete the full testing workflow defined in examples/testing/TESTING-GUIDE.md with 100% validation checks passing
