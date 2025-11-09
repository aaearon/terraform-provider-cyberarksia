# Feature Specification: Target Set Resource

**Feature Branch**: `004-target-set`
**Created**: 2025-11-08
**Status**: Draft
**Input**: User description: "Implement a Terraform resource for managing VM/server target sets in CyberArk Secure Infrastructure Access (SIA). Target sets enable platform engineers to define logical groupings of virtual machines and servers that share common access policies and credentials for Just-In-Time (JIT) privileged access."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Organize Production Servers by Domain (Priority: P1)

As a platform engineer, I want to create a target set that matches all servers in my production domain (e.g., "prod.example.com") so that I can apply consistent access policies across my entire production environment.

**Why this priority**: Most common use case - organizing infrastructure by environment/domain. This is the foundational capability that enables all other target set functionality.

**Independent Test**: Can be fully tested by creating a domain-based target set with valid VM credentials and verifying that the configuration is accepted and persisted. Delivers immediate value by establishing the infrastructure grouping needed for access control.

**Acceptance Scenarios**:

1. **Given** a domain "prod.example.com" and existing VM credentials, **When** I create a domain-based target set, **Then** the system accepts the configuration and all servers in that domain become eligible for JIT access
2. **Given** an existing target set matching "prod.example.com", **When** I change it to match "staging.example.com", **Then** the system updates the matching pattern without recreating the target set
3. **Given** an existing target set, **When** I update which credentials it uses, **Then** the system applies the new credentials for future JIT access sessions without disrupting the target set

---

### User Story 2 - Match Servers by Hostname Pattern (Priority: P1)

As a platform engineer, I want to create target sets that match servers by hostname suffix (e.g., "dc1.example.com") or specific hostname (e.g., "db01.example.com") so that I can organize infrastructure by datacenter, application tier, or individual critical systems.

**Why this priority**: Essential flexibility for real-world infrastructure organization. Different matching patterns serve different organizational needs (broad domain grouping vs. narrow datacenter/system targeting).

**Independent Test**: Can be fully tested by creating suffix-based and target-based target sets and verifying they are accepted. Delivers value by enabling granular infrastructure segmentation beyond simple domain matching.

**Acceptance Scenarios**:

1. **Given** valid VM credentials, **When** I create a suffix-based target set matching "dc1.example.com", **Then** the system accepts the configuration for all servers with that hostname suffix
2. **Given** valid VM credentials, **When** I create a target-based target set matching "db01.example.com", **Then** the system accepts the configuration for that specific server
3. **Given** an existing domain-based target set, **When** I change it to suffix-based or target-based, **Then** the system updates the matching pattern type without recreation

---

### User Story 3 - Define Ephemeral Account Naming (Priority: P2)

As a compliance officer, I want to control how temporary privileged accounts are named on our servers so that I can easily identify and audit which user had access at what time.

**Why this priority**: Critical for audit trails and compliance, but less frequent than basic setup. Typically configured once during initial setup and rarely changed thereafter.

**Independent Test**: Can be fully tested by creating a target set with a custom account naming format and verifying it is stored correctly. Delivers value by enabling consistent audit trail formatting across JIT access sessions.

**Acceptance Scenarios**:

1. **Given** a target set configuration, **When** I specify a custom account naming format like "jit-<user>-<session-guid>", **Then** the system stores and applies this format for JIT access provisioning
2. **Given** a target set without an account naming format, **When** I add a custom format, **Then** the system accepts and stores the format
3. **Given** a target set with an existing account naming format, **When** I try to remove the format entirely, **Then** the system prevents this change and shows a clear error explaining why (to maintain audit consistency)

---

### User Story 4 - Update Target Set Details Without Disruption (Priority: P2)

As a platform engineer, I want to update target set properties (description, matching pattern, credentials) without causing access outages or requiring me to recreate the entire configuration.

**Why this priority**: Operational flexibility - infrastructure changes frequently. Platform engineers need to adapt target sets as infrastructure evolves without disrupting active access policies.

**Independent Test**: Can be fully tested by creating a target set, then updating various properties and verifying changes are applied without resource recreation. Delivers value by enabling seamless operational changes.

**Acceptance Scenarios**:

1. **Given** an existing target set named "production-servers", **When** I rename it to "prod-environment", **Then** the system updates the name and all references without disrupting access
2. **Given** a target set matching domain "old.example.com", **When** I change it to match domain "new.example.com", **Then** the system updates the matching pattern without recreation
3. **Given** a target set with description "Dev environment", **When** I update the description to "Development servers - US region", **Then** the system saves the new description
4. **Given** a target set with TLS certificate validation enabled, **When** I disable certificate validation, **Then** the system applies the change without recreation

---

### User Story 5 - Import Existing Target Sets (Priority: P3)

As a platform engineer, I want to import existing target sets that were created outside of Terraform so that I can bring them under infrastructure-as-code management without recreating them.

**Why this priority**: Important for teams adopting Terraform after manual setup, but less critical than core CRUD operations.

**Independent Test**: Can be fully tested by manually creating a target set in the SIA UI, then importing it into Terraform state and verifying all properties are correctly recognized. Delivers value by enabling incremental Terraform adoption.

**Acceptance Scenarios**:

1. **Given** a target set created manually in the SIA UI, **When** I run terraform import with the target set name, **Then** the system imports the target set into Terraform state with all properties intact
2. **Given** an imported target set, **When** I run terraform plan with no configuration changes, **Then** the system shows no differences (no drift)

---

### Edge Cases

- What happens when I try to create two target sets with the same name? **Expected**: System rejects with clear error message about name uniqueness constraint
- How does the system handle target sets created with credentials that don't exist yet (forward references)? **Expected**: API accepts the configuration (no pre-flight validation), but target set will be non-functional for JIT access until valid credentials exist
- What happens if I include special characters (forward slashes, unicode) in target set names? **Expected**: Forward slashes cause DELETE issues (403 errors), unicode and other special characters are accepted
- How does the system handle very long target set names (1000+ characters)? **Expected**: API accepts extremely long names (no enforced length limit)
- What happens when I delete a target set that's referenced by active access policies? **Expected**: Deletion succeeds, but access policies referencing it may become invalid (API does not enforce referential integrity)
- What happens if I specify an invalid provision_format template? **Expected**: API accepts any string (no validation of placeholders)
- What happens when I attempt to clear the provision_format after it's been set? **Expected**: System prevents this change with a clear plan-time error explaining the audit consistency requirement

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Platform engineers MUST be able to create target sets with three matching patterns: domain-based (matches all servers in a domain), suffix-based (matches servers with hostname suffix), or target-based (matches specific hostnames)
- **FR-002**: Platform engineers MUST be able to specify which VM credentials are used for JIT access to machines in the target set
- **FR-003**: Platform engineers MUST be able to define custom naming formats for ephemeral user accounts created during JIT sessions
- **FR-004**: The system MUST prevent duplicate target set names to avoid configuration conflicts
- **FR-005**: Platform engineers MUST be able to rename target sets without disrupting existing access policies
- **FR-006**: Platform engineers MUST be able to change which credentials a target set uses (credential rotation)
- **FR-007**: Platform engineers MUST be able to switch the matching pattern type (e.g., from domain-based to target-based) for operational flexibility
- **FR-008**: The system MUST preserve account naming formats once set to maintain audit trail consistency (prevent clearing after initial configuration)
- **FR-009**: Platform engineers MUST be able to add or update account naming formats at any time (but not remove once set)
- **FR-010**: The system MUST provide clear feedback when configuration changes cannot be applied and explain why
- **FR-011**: Platform engineers MUST be able to import existing target sets from other infrastructure-as-code tools using the target set name as the identifier
- **FR-012**: The system MUST detect when target sets have been modified or deleted outside of this management interface (drift detection)
- **FR-013**: The system MUST support optional description fields for documenting target set purpose
- **FR-014**: The system MUST support toggling TLS/SSL certificate validation for target set connections
- **FR-015**: The system MUST handle target set renames by updating the internal identifier to match the new name

### Key Entities

- **Target Set**: A logical grouping of VM/server targets that share common access credentials and policies. Has a unique name (which also serves as the identifier), matching pattern type (domain/suffix/target), associated credentials reference, optional account naming format, optional description, and certificate validation setting.

- **Matching Pattern**: Defines which machines belong to the target set based on one of three patterns: domain name (e.g., "example.com" matches all servers in that domain), hostname suffix (e.g., "dc1.example.com" matches servers ending with that suffix), or specific hostname (e.g., "server01.example.com" matches only that exact server).

- **Credentials Reference**: Points to the VM credentials (managed separately) that will be used for JIT access to machines in this target set. Consists of two parts: secret_id (UUID of the VM secret) and secret_type (either "ProvisionerUser" for username/password credentials or "PCloudAccount" for PAM vault account references).

- **Account Naming Format**: Template that defines how temporary privileged accounts are named, using placeholders like "<user>" (requesting user) and "<session-guid>" (unique session identifier). For example, "<user>-<session-guid>" might generate accounts like "john.doe-abc123def456". Cannot be removed once set to maintain audit trail consistency.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Target set configurations accurately match desired infrastructure groupings (100% of created target sets correctly define their matching pattern and scope)
- **SC-002**: Configuration changes apply without requiring resource recreation or disruption (all property updates except name use in-place updates, rename updates identifier without recreation)
- **SC-003**: Drift detection identifies when target sets are modified externally (terraform plan detects all out-of-band changes including modifications and deletions)
- **SC-004**: Audit trails clearly identify which target set was used for each JIT session (ephemeral account names follow configured naming format 100% of the time)
- **SC-005**: Error messages enable platform engineers to independently resolve configuration issues (all validation errors include specific guidance on what needs to be corrected)
- **SC-006**: Import operations successfully bring existing target sets under Terraform management (imported target sets show zero diff on first plan after import)
- **SC-007**: Credential rotation updates apply without access disruption (updating secret_id or secret_type completes successfully without requiring resource recreation)

## Assumptions *(mandatory)*

- VM credentials (cyberarksia_virtual_machine_secret resource) are managed separately and must exist before target sets can reference them
- The underlying CyberArk SIA API handles the actual JIT access provisioning - this resource only manages target set configuration metadata
- Access policies that assign users to target sets are managed through separate Terraform resources (not part of this specification)
- Platform engineers have appropriate permissions in CyberArk SIA to manage target sets (authentication and authorization are handled at the provider level)
- Target set names serve as unique identifiers across the SIA tenant (name-as-ID pattern)
- The CyberArk SIA API uses PATCH-like semantics for updates (server preserves fields not included in update requests)
- The default account naming format when not specified is "<user>-<session-guid>"
- TLS/SSL certificate validation is enabled by default when not explicitly configured
- The API accepts target set creation even if secret_id is completely omitted or references a non-existent UUID (no referential integrity enforcement or field requirement)
- Forward slashes in target set names should be avoided as they cause DELETE operation failures due to URL path interpretation issues

## Dependencies *(mandatory)*

- **Upstream Dependency**: Requires cyberarksia_virtual_machine_secret resource to exist before target sets can reference credentials (Terraform dependency via secret_id reference)
- **Downstream Dependency**: Target sets must be created before they can be assigned to access policies (other resources will reference target sets)
- **SDK Dependency**: Requires ARK SDK v1.5.0+ for target set management APIs
- **Workaround Dependency**: DELETE operation requires workaround due to ARK SDK bug (nil body panic) - direct API call bypassing SDK is necessary until v1.6.0+

## Technical Notes *(optional)*

### Implementation Reference

The technical investigation document (`docs/development/target-sets-poc-investigation.md`) contains:
- 50 validated tests against live CyberArk tenant (98% pass rate)
- Complete API behavior documentation (PATCH semantics, field mutability, validation gaps)
- SDK bug workarounds (DELETE panic, UPDATE without name field)
- Field-by-field validation results (provision_format constraints, name character limitations)
- All 6 type change combinations validated (bidirectional mutability confirmed)

### Critical API Behaviors

- **Name-as-ID Pattern**: The `id` field always equals the `name` field; renaming requires updating both URL path and request body
- **PATCH Semantics**: Server preserves fields omitted from update requests (enables partial updates but prevents clearing provision_format)
- **No Pre-flight Validation**: API accepts non-existent secret_id references (validation occurs at JIT access time, not configuration time)
- **Destructive UPDATE Bug**: Sending UPDATE without `name` field returns 500 error AND deletes the target set (must always include name field)
- **DELETE Workaround Required**: SDK DELETE method panics; must use direct HTTP DELETE call until SDK v1.6.0+
- **Minimal Name Validation**: API only enforces uniqueness; accepts unicode, spaces, backslashes, and extremely long names (1000+ characters)
- **Forward Slash Limitation**: API accepts names with forward slashes but DELETE returns 403 due to URL encoding issues
- **No provision_format Validation**: API accepts any string including invalid placeholders, plain text, or extremely long values
