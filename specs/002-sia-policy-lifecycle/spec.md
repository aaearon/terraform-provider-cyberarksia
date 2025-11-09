# Feature Specification: Database Policy Management - Modular Assignment Pattern

**Feature Branch**: `002-sia-policy-lifecycle`
**Created**: 2025-10-28
**Status**: Draft
**Input**: User description: "Users want to be able to manage the full lifecycle of CyberArk Secure Infrastructure Access database access policies. They only care about database access policies, and not anything else!"

**Architecture**: This specification defines a modular assignment pattern where principals and targets are managed via separate assignment resources for maximum composability and team workflow flexibility.

## ⚠️ CRITICAL IMPLEMENTATION NOTE: Inline Assignments Required

**Date Added**: 2025-10-28
**Reason**: SIA API constraint discovered during implementation

The **SIA API enforces a critical constraint**: Policies CANNOT be created without at least **one principal AND one target database**. This differs from the originally envisioned modular-only pattern.

**Impact on Implementation**:

The `cyberarksia_database_policy` resource MUST support **inline assignments** using singular, repeatable blocks matching familiar Terraform patterns (AWS `ingress`/`egress`, GCP `disk`/`network_interface`):

```hcl
resource "cyberarksia_database_policy" "example" {
  name   = "Production Access"
  status = "active"

  # Required: At least 1 target_database block
  target_database {
    database_workspace_id  = cyberarksia_database_workspace.postgres.id
    authentication_method  = "db_auth"
    db_auth_profile { roles = ["reader"] }
  }

  # Required: At least 1 principal block
  principal {
    principal_id   = "abc-123"
    principal_type = "USER"
    principal_name = "user@example.com"
  }

  conditions {
    max_session_duration = 8
  }
}
```

**Why Singular Block Names** (`target_database`, `principal`):
- Matches AWS Security Group pattern: `ingress {}`, `egress {}` (not `ingreses {}`)
- Follows GCP Compute Instance pattern: `disk {}`, `network_interface {}`
- Familiar to Terraform users from popular providers
- Repeatable blocks feel more natural with singular names

**Relationship to Separate Assignment Resources**:
- Inline blocks handle **initial policy creation** (API requirement)
- Separate `cyberarksia_database_policy_principal_assignment` and `cyberarksia_policy_workspace_assignment` resources manage **additional assignments**
- Teams can use `lifecycle { ignore_changes = [principal, target_database] }` if managing all assignments externally

This hybrid approach satisfies both the API constraint AND the composability goals of the modular pattern.

## Clarifications

### Session 2025-10-28

- Q: What happens to principal and database assignments when a policy is deleted? → A: Cascade delete - API automatically removes all principal and database assignments when the policy is deleted
- Q: What is the concurrent modification behavior when multiple workspaces modify the same policy? → A: Last-write-wins - No conflict detection; second write overwrites first (standard REST behavior)
- Q: When/how is principal directory validation performed? → A: API-only validation - Provider sends request; API returns error if directory doesn't exist
- Q: Which policy status values are user-controllable vs. system-managed? → A: Only two statuses exist - "Active" and "Suspended" (both user-controllable)
- Q: How are invalid import ID formats handled? → A: Split-and-validate - Provider splits on first colon; if not exactly 2 parts, return clear error message

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create Database Policy with Metadata and Conditions (Priority: P1)

Infrastructure teams need to create database policies that establish the foundational access control framework, including policy metadata (name, description, status) and access conditions (time windows, session limits, idle timeouts).

**Why this priority**: Policy creation is the foundational step. Without a policy resource, no principals or databases can be assigned. This creates the access control framework that all assignments depend on.

**Independent Test**: Can be fully tested by creating a policy with name, description, and access conditions (weekday 9-5, 8-hour sessions), verifying it appears in SIA UI with correct settings. Policy functions without principals or targets (they're added via separate assignment resources).

**Acceptance Scenarios**:

1. **Given** an authenticated Terraform provider, **When** I define a database policy resource with name "Database-Admins", description "Admin access policy", and access window for weekdays 9AM-5PM, **Then** the policy is created successfully in SIA with unique policy ID and configured conditions
2. **Given** I have created a policy with conditions, **When** I run terraform plan with no changes, **Then** no modifications are detected (idempotent read)
3. **Given** I attempt to create a policy with duplicate name, **When** I run terraform apply, **Then** I receive clear error message indicating policy names must be unique
4. **Given** a policy exists, **When** I update max_session_duration from 4 hours to 8 hours, **Then** the policy condition updates without affecting principals or database assignments (managed separately)

---

### User Story 2 - Assign Principals to Policy (Priority: P1)

Platform teams need to assign users, groups, and roles from identity directories to policies to control who has access to databases governed by the policy.

**Why this priority**: Policies without principals grant no access. Principal assignment is foundational - until users/groups are assigned, the policy cannot provide access regardless of database assignments.

**Independent Test**: Create policy, assign USER principal with valid directory info, assign GROUP principal, verify both principals appear in SIA UI and can access assigned databases (when databases are later assigned).

**Acceptance Scenarios**:

1. **Given** a policy exists, **When** I assign a USER principal with valid source_directory_id and source_directory_name, **Then** the principal is added to policy and appears in SIA
2. **Given** a policy with one principal, **When** I add a GROUP principal, **Then** both principals coexist on the policy without conflict
3. **Given** I assign a USER principal without required source_directory_id, **When** I apply configuration, **Then** validation error occurs before API call with clear message about missing required field
4. **Given** I remove a principal assignment resource, **When** I run terraform destroy, **Then** only that specific principal is removed from policy, other principals remain unchanged
5. **Given** a principal assignment exists, **When** I run terraform import with composite ID "policy-id:principal-id:principal-type", **Then** assignment is imported successfully into state

---

### User Story 3 - Assign Databases to Policy (Priority: P1)

Application teams need to assign database workspaces to policies to control which databases the policy governs access to.

**Why this priority**: Database assignment completes the access control chain (policy + principals + databases). Without database assignments, principals have nothing to access.

**Independent Test**: Create policy with principals, assign database workspace with authentication profile, verify database appears in policy targets and principals can access it.

**Acceptance Scenarios**:

1. **Given** a policy with principals exists, **When** I assign a database workspace with db_auth authentication method and roles, **Then** the database is added to policy targets and principals can access it
2. **Given** a policy with database assignments, **When** I add another database assignment, **Then** both databases coexist in policy targets
3. **Given** I remove a database assignment resource, **When** I run terraform destroy, **Then** only that database is removed from policy, other database assignments remain unchanged
4. **Given** a database assignment exists, **When** I run terraform import with composite ID "policy-id:database-id", **Then** assignment is imported successfully into state

---

### User Story 4 - Update Existing Policy Attributes (Priority: P2)

Security teams need to modify policy metadata attributes such as name, description, and status as organizational requirements evolve. Changes to policy metadata should not disrupt existing principal or database assignments.

**Why this priority**: After creation (P1), teams need to maintain policies over time. This is core lifecycle management but not as critical as initial policy/principal/database setup.

**Independent Test**: Can be tested by creating a policy with principals and databases, modifying metadata (description, status), and verifying changes are reflected in SIA without affecting assignments.

**Acceptance Scenarios**:

1. **Given** an existing policy named "Dev-Access", **When** I change the description to "Updated development access policy", **Then** the policy description is updated in-place and existing principal/database assignments remain intact
2. **Given** a policy with status "Active", **When** I change status to "Suspended", **Then** the policy status updates in-place and access based on this policy is immediately suspended
3. **Given** an existing policy named "Dev-Access", **When** I change the name to "Production-Access", **Then** the policy name is updated in-place (policy ID remains unchanged) and existing principal/database assignments remain intact

---

### User Story 5 - Delete Policies (Priority: P3)

Operations teams need to remove obsolete policies when they are no longer needed, ensuring clean infrastructure state management.

**Why this priority**: Policy deletion is important for cleanup but is lowest priority because it's rarely performed compared to creation and updates.

**Independent Test**: Can be tested by creating a policy (with or without assignments), running terraform destroy, and verifying complete removal from SIA.

**Acceptance Scenarios**:

1. **Given** a policy exists, **When** I remove the policy from Terraform configuration and run terraform destroy, **Then** the policy is deleted from SIA successfully
2. **Given** a policy with active principal or database assignments, **When** I delete the policy, **Then** the policy is deleted and assignments are handled per API behavior (either orphaned or cascade deleted)
3. **Given** I delete a policy, **When** I attempt to read it again, **Then** I receive a not found error

---

### User Story 6 - Import Existing Policies and Assignments (Priority: P2)

Platform engineers need to import existing policies and their assignments (principals and databases) created through the SIA UI into Terraform management, enabling infrastructure-as-code adoption for legacy environments.

**Why this priority**: Import is critical for migration scenarios but not as essential as CRUD operations. Teams adopting Terraform need this to manage existing infrastructure.

**Independent Test**: Can be tested by creating a policy with principals and databases manually in SIA UI, importing each resource using appropriate IDs, and verifying all attributes are correctly populated in state.

**Acceptance Scenarios**:

1. **Given** a policy exists in SIA created outside Terraform, **When** I run terraform import with the policy ID, **Then** the policy is imported into Terraform state with metadata and conditions populated
2. **Given** an imported policy, **When** I run terraform plan, **Then** no changes are detected (state matches remote)
3. **Given** a policy with principals exists in SIA, **When** I import each principal assignment with composite ID "policy-id:principal-id:principal-type", **Then** principal assignments are imported correctly
4. **Given** a policy with database assignments exists in SIA, **When** I import each database assignment with composite ID "policy-id:database-id", **Then** database assignments are imported correctly
5. **Given** I import policy metadata but not assignments, **When** I run terraform plan, **Then** Terraform does not detect drift for unmanaged assignments (non-authoritative pattern)

---

### Edge Cases

**Policy Lifecycle**:
- What happens when attempting to delete a policy that has principal or database assignments? (Policy deletion succeeds with cascade delete - API automatically removes all principal and database assignments)
- How does the system handle concurrent updates to the same policy metadata from multiple Terraform workspaces? (Last-write-wins with no conflict detection - second write overwrites first; users must coordinate workspaces)
- What happens when a policy name exceeds maximum length constraints (200 chars)? (Validation error before API call)
- How does the system handle policy creation when UAP service is not provisioned on the tenant? (Clear error with guidance to contact CyberArk support)
- What happens when importing a policy that no longer exists in SIA? (Not found error with clear message)
- How does the system handle special characters in policy names or descriptions? (Should support standard UTF-8 characters per API constraints)
- What happens when transitioning policy status from Active to Suspended and back? (In-place update via UpdatePolicy, change takes effect immediately; no graceful shutdown period)
- What happens to active sessions when policy status changes to Suspended? (API behavior - assumed immediate termination or session completion, provider does not control)
- What happens when policy is deleted and assignment resources still exist in Terraform state? (Next terraform apply/refresh will fail with "policy not found" error for assignments; user must remove orphaned assignment resources from state manually using terraform state rm)

**Principal Assignment**:
- What happens when assigning duplicate principal to same policy? (Provider detects during plan and prevents duplicate assignment via composite ID uniqueness)
- What happens when removing a principal assignment that doesn't exist? (Terraform no-op, resource already removed from state and API)
- How does system handle concurrent principal assignments to same policy from multiple workspaces? (Read-modify-write race condition possible, same limitation as database assignments)
- What happens when principal_id doesn't exist in the specified directory? (API error: "Principal not found in directory {directory_id}", provider surfaces clear error)
- What happens when assigning USER principal without required source_directory_id? (Client-side validation error before API call)
- How does system handle principal assignment when policy is deleted? (Assignment resource references non-existent policy, apply fails with policy not found error)
- **What happens when source directory is deleted outside Terraform?** (Provider detects during Read operation, terraform plan fails with "directory not found" error following standard Terraform pattern per aws_security_group_rule, aws_iam_role_policy_attachment)

**Database Assignment**:
- How does system handle concurrent database assignments to same policy from multiple workspaces? (Last-write-wins with read-modify-write race condition possible - second workspace's changes may overwrite first; documented known limitation)
- What happens when database_workspace_id doesn't exist? (API error: "Database workspace not found", provider surfaces clear error)
- What happens when assignment references deleted policy? (Apply fails with policy not found error)
- **What happens when database workspace is deleted outside Terraform?** (Provider detects during Read operation, terraform plan fails with "database workspace not found" error following standard Terraform pattern per aws_security_group_rule)

## Requirements *(mandatory)*

### Functional Requirements

**Policy Resource (Metadata + Conditions)**:
- **FR-001**: System MUST allow users to create database policies with all policy attributes: name, description, status, time_frame, policy_entitlement (location_type, policy_type), policy_tags, time_zone, delegation_classification, and conditions (max_session_duration, idle_time, access_window). Principals and targets are managed via separate assignment resources.
- **FR-002**: System MUST assign a unique policy ID (UUID format) to each created policy
- **FR-003**: System MUST allow users to modify policy attributes with appropriate semantics: (1) In-place update: name, description, status, delegation_classification, conditions, policy_tags, time_zone, time_frame; (2) Fixed/Non-configurable: target_category (always "DB"), policy_type (always "Recurring"), location_type (always "FQDN/IP"). Rationale: Policy ID is the unique identifier. All user-configurable attributes support in-place updates. Fixed attributes (target_category, policy_type, location_type) are set by the provider and not exposed for user modification. Updating principals or targets requires using the respective assignment resources.
- **FR-004**: System MUST support policy status values: "Active" and "Suspended" (both user-controllable)
- **FR-005**: System MUST configure access conditions including max_session_duration (1-24 hours), idle_time (1-120 minutes), and access_window (days, hours)
- **FR-006**: System MUST validate location_type is "FQDN/IP" (this is the ONLY valid location_type for database access policies). All database workspaces (AWS, Azure, GCP, on-premise, Atlas) use "FQDN/IP" regardless of their cloud_provider attribute.
- **FR-006a**: System MUST set policy_type to "Recurring" (ONLY valid value for database access policies). This value is fixed and not user-configurable. OnDemand policies are not supported for database access.
- **FR-007**: System MUST allow users to delete policies regardless of whether they have active principal or database assignments. The API uses cascade delete behavior: all principal and database assignments are automatically removed when the policy is deleted.
- **FR-008**: System MUST support importing existing policies by policy ID
- **FR-009**: System MUST validate policy names are unique within the tenant
- **FR-010**: System MUST support idempotent read operations (terraform plan with no changes shows no modifications)
- **FR-011**: System MUST detect and handle policy drift (changes made outside Terraform)
- **FR-012**: System MUST provide comprehensive documentation for all resources that enables LLMs/AI assistants to understand resource purpose, required attributes, optional attributes, valid values, relationships between resources, and complete usage examples. Documentation must include: resource description, attribute descriptions with constraints, dependency relationships, and working HCL examples. **Measurable criteria**: (1) 100% of schema attributes documented in attribute table, (2) ≥3 working HCL examples per resource covering basic/intermediate/complete scenarios, (3) All constraints explicitly stated (length limits, valid enum values, required combinations).
- **FR-013**: System MUST structure resource documentation to follow Terraform provider documentation best practices per registry.terraform.io/docs/providers standards, including: (1) Clear attribute tables with Required/Optional/Computed flags, (2) Type information (string, int, bool, block), (3) Default values explicitly stated, (4) Usage patterns demonstrating common scenarios, (5) Import section with composite ID format examples.

**Principal Assignment Resource**:
- **FR-014**: System MUST support assigning individual principals to policies via separate assignment resource
- **FR-015**: System MUST validate principal type (USER, GROUP, or ROLE)
- **FR-016**: System MUST require source_directory_id and source_directory_name for USER and GROUP principal types
- **FR-017**: System MUST allow ROLE principal type without directory requirements
- **FR-018**: System MUST use read-modify-write pattern to preserve principals assigned outside Terraform (UI, other workspaces)
- **FR-019**: System MUST support composite ID format "policy-id:principal-id:principal-type" for Terraform import operations (this is a provider-level import ID format, not a CyberArk API schema). Provider splits on colons and validates exactly 3 parts result; returns clear error message if format is invalid. Three-part format is required because principal IDs can be duplicated across different principal types (e.g., user "admin" and role "admin").
- **FR-020**: System MUST force replacement (ForceNew) when policy_id or principal_id changes in assignment (standard Terraform pattern - changing identifying fields requires recreating the resource)
- **FR-021**: System MUST prevent duplicate principal assignments (same principal_id + policy_id combination)
- **FR-022**: System MUST support importing existing principal assignments by composite ID
- **FR-023**: System MUST provide comprehensive documentation per FR-012 and FR-013 (LLM-friendly attribute descriptions, constraints, relationships, and examples)

**Database Assignment Resource**:
- **FR-024**: System MUST support assigning individual databases to policies via separate assignment resource
- **FR-025**: System MUST use read-modify-write pattern to preserve databases assigned outside Terraform
- **FR-026**: System MUST support composite ID format "policy-id:database-id" for Terraform import operations (this is a provider-level import ID format, not a CyberArk API schema). Provider splits on first colon and validates exactly 2 parts result; returns clear error message if format is invalid.
- **FR-027**: System MUST force replacement (ForceNew) when policy_id or database_workspace_id changes in assignment (standard Terraform pattern - changing identifying fields requires recreating the resource)
- **FR-028**: System MUST support all 6 authentication methods: db_auth, ldap_auth, oracle_auth, mongo_auth, sqlserver_auth, rds_iam_user_auth
- **FR-029**: System MUST support importing existing database assignments by composite ID
- **FR-030**: System MUST provide comprehensive documentation per FR-012 and FR-013 (LLM-friendly attribute descriptions, constraints, relationships, and examples)

**Error Handling**:
- **FR-031**: System MUST map ARK SDK API errors to Terraform diagnostics using client.MapError() pattern. Provider relies on API to return errors and surfaces them to users with minimal transformation.
- **FR-032**: System MUST provide clear error messages for provider-level validation failures (composite ID format, required field validation) before making API calls.
- **FR-033**: System MUST use retry logic with exponential backoff for transient failures (network errors, rate limits, temporary unavailability): Max retries: 3, Base delay: 500ms, Max delay: 30s.

**Validation Requirements**:
- **FR-034**: System MUST use API-only validation for time_frame, access_window, policy name length, and policy_tags count. Provider sends request to API and surfaces validation errors with clear messages. Rationale: Reduces code complexity, eliminates drift between provider and API validation rules, follows established pattern (Assumption 12).
- **FR-035**: System MUST handle empty string vs null values in optional attributes per Terraform Plugin Framework semantics (empty strings treated as unset for API calls)
- **FR-036**: System MUST implement client-side validation only for provider-level constructs (composite ID format validation per FR-019, FR-026) and simple enums with custom validators (policy_status, principal_type, location_type) to provide immediate feedback during terraform plan

### Key Entities

This feature manages database policies through THREE separate resources:

#### 1. Database Policy Resource (`cyberarksia_database_policy`)

Manages policy metadata and access conditions. Does NOT manage principals or database assignments.

**Metadata Attributes**:

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| policy_id | UUID string | No | Yes | Unique policy identifier assigned by API |
| name | string | Yes | No | Policy name (1-200 chars, unique per tenant) |
| description | string | No | No | Policy description (max 200 chars) |
| status | string | Yes | No | Policy status: "Active" or "Suspended" (user-controllable) |
| policy_entitlement | object | No | Yes | Contains location_type (fixed: "FQDN/IP") and policy_type (fixed: "Recurring"). Both values are provider-managed and not user-configurable for database policies. |
| time_frame | object | No | No | Policy validity period (from_time, to_time as ISO 8601 timestamps) |
| policy_tags | []string | No | No | Policy tags (max 20 tags) |
| time_zone | string | No | No | Timezone for access windows (default: GMT, max 50 chars). Accepts IANA timezone names (e.g., "America/New_York", "Europe/London") or GMT offset format (e.g., "GMT", "GMT+5", "GMT-8"). |
| delegation_classification | string | Yes | No | User rights: Restricted or Unrestricted (default: Unrestricted) |
| created_by | object | No | Yes | Creator info (user, timestamp) |
| updated_on | object | No | Yes | Last modification info (user, timestamp) |

**Conditions Attributes**:

| Attribute | Type | Required | Default | Range | Description |
|-----------|------|----------|---------|-------|-------------|
| max_session_duration | int | Yes | 1 | 1-24 hours | Maximum session length in hours. Terraform accepts hours (1-24), provider converts to minutes for API if needed. |
| idle_time | int | No | 10 | 1-120 minutes | Idle timeout before disconnect |
| access_window.days_of_the_week | []int | No | [0-6] | 0=Sunday through 6=Saturday | Days when access is allowed |
| access_window.from_hour | string | No | - | "HH:MM" format | Start time for daily access window |
| access_window.to_hour | string | No | - | "HH:MM" format | End time for daily access window |

**What This Resource Does NOT Manage**: Principals and database assignments (managed by separate resources below)

#### 2. Principal Assignment Resource (`cyberarksia_database_policy_principal_assignment`)

Manages assignment of individual principals (users, groups, roles) to policies. Uses read-modify-write pattern to preserve principals assigned outside Terraform.

**Attributes**:

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| id | string | No | Yes | Composite ID: "policy-id:principal-id:principal-type" |
| policy_id | string | Yes (ForceNew) | No | Policy to assign principal to |
| principal_id | string | Yes (ForceNew) | No | Unique principal identifier (max 40 chars) |
| principal_name | string | Yes | No | Display name (max 512 chars, pattern: `^[\w.+\-@#]+$`). Pattern supports alphanumerics, dots, plus signs, hyphens, at signs, and hash symbols. **Limitation**: Does not support spaces or Unicode characters. For principals with spaces in names, API may accept but provider validation will reject - validation can be relaxed if needed. Use for email addresses and standard ASCII principal identifiers. Edge cases (Unicode names, very long names) are API-validated - provider sends request and surfaces API errors. |
| principal_type | string | Yes | No | USER, GROUP, or ROLE |
| source_directory_name | string | Conditional | No | Required for USER/GROUP types (max 50 chars) |
| source_directory_id | string | Conditional | No | Required for USER/GROUP types |

**Pattern**: Non-authoritative, one assignment per resource. Multiple assignment resources can coexist for the same policy.

#### 3. Database Assignment Resource (`cyberarksia_database_policy_assignment`)

Manages assignment of individual database workspaces to policies. Uses read-modify-write pattern to preserve databases assigned outside Terraform.

**Attributes**:

| Attribute | Type | Required | Computed | Description |
|-----------|------|----------|----------|-------------|
| id | string | No | Yes | Composite ID: "policy-id:database-id" |
| policy_id | string | Yes (ForceNew) | No | Policy to assign database to |
| database_workspace_id | string | Yes (ForceNew) | No | Database workspace to assign |
| authentication_method | string | Yes | No | One of: db_auth, ldap_auth, oracle_auth, mongo_auth, sqlserver_auth, rds_iam_user_auth |
| *_profile | object | Yes | No | Authentication profile matching the method (see schemas below) |

**Authentication Profile Schemas** (from ARK SDK v1.5.0):

1. **db_auth_profile** (Local DB Authentication):
   - `roles` ([]string, required): Database roles to assign (min=1, max=50, max_length=50 per role)

2. **ldap_auth_profile** (LDAP Authentication):
   - `assign_groups` ([]string, required): LDAP groups to assign (min=1, max=50, max_length=50 per group)

3. **oracle_auth_profile** (Oracle DB Authentication):
   - `roles` ([]string, required): Oracle roles to assign (min=1, max=50, max_length=50 per role)
   - `dba_role` (bool, optional): Grant DBA role
   - `sysdba_role` (bool, optional): Grant SYSDBA role
   - `sysoper_role` (bool, optional): Grant SYSOPER role

4. **mongo_auth_profile** (MongoDB Authentication):
   - `global_builtin_roles` ([]string, optional): Global built-in roles (max=50, max_length=50 per role)
   - `database_builtin_roles` (map[string][]string, optional): Map of database names to built-in roles (max=1000 total, db_name max_length=256, role max_length=50)
   - `database_custom_roles` (map[string][]string, optional): Map of database names to custom roles (max=1000 total, db_name max_length=256, role max_length=50)
   - **Validation**: At least one global role required if no database roles specified

5. **sqlserver_auth_profile** (SQL Server Authentication):
   - `global_builtin_roles` ([]string, required): Global built-in roles (max=50, max_length=50 per role)
   - `global_custom_roles` ([]string, optional): Global custom roles (max=50, max_length=50 per role)
   - `database_builtin_roles` (map[string][]string, optional): Map of database names to built-in roles (max=1000 total, db_name max_length=256, role max_length=50)
   - `database_custom_roles` (map[string][]string, optional): Map of database names to custom roles (max=1000 total, db_name max_length=256, role max_length=50)

6. **rds_iam_user_auth_profile** (AWS RDS IAM Authentication):
   - `db_user` (string, required): Database username for IAM authentication (min=1, max=256)

**Pattern**: Non-authoritative, one assignment per resource. Multiple assignment resources can coexist for the same policy.

## Success Criteria *(mandatory)*

### Measurable Outcomes

**Policy Resource**:
- **SC-001**: Users can successfully create database policies with all supported attributes (name, description, status, conditions, time_frame, policy_tags, etc.)
- **SC-002**: Policy CRUD operations are idempotent (multiple terraform apply commands with no configuration changes result in no changes to infrastructure)
- **SC-003**: Policy import operations preserve all policy attributes in Terraform state
- **SC-004**: Policy deletion succeeds regardless of existing principal or database assignments
- **SC-005**: Policy drift is detectable via terraform plan (changes made outside Terraform are identified)

**Principal Assignment Resource**:
- **SC-006**: Principal assignment operations are isolated from database assignments (assigning/removing principals does not modify database assignments)
- **SC-007**: Multiple principal assignment resources can manage different principals on the same policy without conflicts (composability)
- **SC-008**: Principal assignment removal affects only the specified principal (other principals remain on policy)
- **SC-009**: Read-modify-write pattern preserves principals assigned outside Terraform (UI-managed principals persist after Terraform operations)

**Database Assignment Resource**:
- **SC-010**: Database assignment operations are isolated from principal assignments (assigning/removing databases does not modify principals)
- **SC-011**: Multiple database assignment resources can manage different databases on the same policy without conflicts (composability)
- **SC-012**: Database assignment removal affects only the specified database (other databases remain on policy)
- **SC-013**: Read-modify-write pattern preserves databases assigned outside Terraform (UI-managed databases persist after Terraform operations)

**Documentation Quality**:
- **SC-014**: LLMs/AI assistants can successfully generate valid Terraform configurations for all three resources (policy, principal assignment, database assignment) when given a natural language description of the desired infrastructure
- **SC-015**: Documentation includes attribute constraint validation rules (e.g., max lengths, valid enum values, required field combinations) that enable AI assistants to generate valid configurations on first attempt without trial-and-error

## Assumptions *(mandatory)*

1. **UAP Service Availability**: Assumes UAP service is provisioned on the target CyberArk tenant. If not, operations will fail with DNS lookup errors. Users must verify tenant capabilities before using policy resources.

2. **Authentication**: Assumes users have valid SIA service account credentials with UAP policy management permissions. Provider authentication is already implemented and working.

3. **ARK SDK Version**: Assumes ARK SDK v1.5.0 or later is available, which includes UAP policy management methods: AddPolicy, UpdatePolicy, DeletePolicy, Policy, ListPolicies.

4. **Modular Resource Design**: Database policy management is implemented through three separate Terraform resources: `cyberarksia_database_policy` (policy metadata + conditions), `cyberarksia_database_policy_principal_assignment` (principal assignments), and `cyberarksia_database_policy_assignment` (database assignments). This specification covers the complete lifecycle management for all three resources, including policy CRUD operations and assignment management for both principals and databases.

5. **Single Tenant**: Assumes all operations target a single CyberArk tenant (configured at provider level). Multi-tenant scenarios are out of scope.

6. **Naming Constraints**: Assumes policy names follow CyberArk SIA naming conventions (alphanumeric, hyphens, underscores, spaces allowed). Maximum length constraints will be validated against API error responses.

7. **Terraform Version**: Assumes Terraform 1.0+ with Plugin Framework v6 support.

8. **Location Type Constant**: SIA currently only supports "FQDN/IP" (with forward slash) as the location type for all database access policies. All database workspaces (AWS, Azure, GCP, on-premise, Atlas) are assigned to this single location type. This is a current platform constraint, not a policy-level configuration.

9. **Concurrent Access**: The API uses last-write-wins behavior with no conflict detection for concurrent updates from multiple Terraform workspaces. This is standard REST behavior without optimistic locking. The second write will overwrite the first with no error or warning. Users are responsible for workspace coordination (same limitation as AWS security groups).

10. **API Rate Limits**: Assumes CyberArk SIA API has standard rate limiting. Provider retry logic (exponential backoff, 3 attempts) is sufficient to handle transient failures.

10a. **Pagination**: Assumes ARK SDK v1.5.0 handles pagination transparently for ListPolicies operations. Provider relies on SDK to fetch all pages automatically without manual pagination handling.

11. **Modular Assignment Pattern**: Principal and target assignments use non-authoritative read-modify-write pattern. Multiple Terraform workspaces can assign different principals/targets to the same policy, but concurrent modifications may cause race conditions (same limitation as `aws_security_group_rule`). Users responsible for workspace coordination.

12. **Principal Source Directories**: Principal source directories (AzureAD, LDAP, Okta, Active Directory) must exist in the SIA tenant before principal assignment. Provider uses API-only validation: it sends the request directly and the API returns an error if the directory doesn't exist. The provider does not pre-validate directory existence to avoid extra API overhead and potential drift. Users must configure identity directories through SIA UI or other means before assigning principals.

12a. **Database Workspace Dependencies**: Database workspaces must exist in the SIA tenant before they can be assigned to policies via database assignment resources. The database_workspace_id attribute references existing cyberarksia_database_workspace resources or workspaces created through the SIA UI. Provider uses API-only validation - API returns "Database workspace not found" error if workspace doesn't exist.

13. **Consistent Separation**: Both principals and targets are embedded in the CyberArk API policy object, but are managed via separate assignment resources for consistency, composability, and team workflow flexibility. This design choice prioritizes modular infrastructure management over API structure mirroring.

14. **Development Standards**: This feature follows established Terraform Provider development best practices including: Plugin Framework v6 conventions, read-modify-write patterns for non-authoritative resources, structured error handling with MapError, retry logic with exponential backoff, comprehensive testing per TESTING-GUIDE.md, and LLM-friendly documentation standards. These practices serve as the de facto constitution in absence of a formal project constitution document.

## Out of Scope *(mandatory)*

1. **Non-Database Policies**: This specification explicitly excludes SSH access policies, Kubernetes policies, or any other non-database policy types. Focus is 100% on database access policies.

2. **Policy Template Management**: Advanced policy templating, versioning, or template libraries are out of scope. Each policy is managed independently.

3. **Inline Principal/Database Management**: Managing principals or databases inline within the policy resource (monolithic pattern) is out of scope. This specification defines separate assignment resources for maximum composability.

4. **Authoritative Assignment Resources**: Managing ALL principals or ALL targets via single authoritative resource (like `aws_iam_policy_attachment`) is out of scope. Only non-authoritative per-assignment resources are provided. Users wanting authoritative control must manually remove unmanaged assignments.

5. **Advanced RBAC Integration**: Integration with external identity providers, complex role hierarchies, or custom RBAC frameworks is out of scope. Policy status (Active/Inactive) is the only access control mechanism within policy scope.

6. **Policy Analytics and Reporting**: Policy usage statistics, access logs, audit trails, or compliance reporting are out of scope. These are SIA platform features accessed through UI/API separately.

7. **Bulk Operations**: Bulk policy creation, updates, or deletions are out of scope. Users can leverage Terraform count/for_each meta-arguments for bulk management.

8. **Policy Cloning**: Ability to duplicate/clone existing policies is out of scope. Users can use Terraform modules or copy configuration.

9. **Policy Validation Rules**: Advanced policy validation (e.g., conflicting policies, policy compliance checks) beyond basic attribute validation is out of scope.

10. **Cross-Tenant Policy Management**: Managing policies across multiple CyberArk tenants in a single Terraform workspace is out of scope. Each provider instance targets one tenant.

11. **ARK SDK Version Management**: Automatic handling of ARK SDK breaking changes or API version migrations is out of scope. Provider is pinned to ARK SDK v1.5.0. SDK updates require explicit provider version bump, code updates, and testing. Users must coordinate provider version upgrades with SDK compatibility requirements.

## Resource Architecture *(informative)*

### Modular Assignment Pattern

This specification defines a **modular assignment pattern** where principals and targets are managed via separate assignment resources rather than inline within the policy resource.

### Three Resources

**1. `cyberarksia_database_policy`** (NEW) - Policy metadata + conditions
- Purpose: Define policy framework (name, description, access conditions)
- Manages: Metadata, status, time windows, session limits
- Does NOT manage: Principals or database assignments

**2. `cyberarksia_database_policy_principal_assignment`** (NEW) - Principal assignments
- Purpose: Assign users/groups/roles to policies
- Pattern: One resource per principal assignment
- Uses: Read-modify-write to preserve UI-managed principals

**3. `cyberarksia_database_policy_assignment`** (existing, renamed) - Database assignments
- Purpose: Assign database workspaces to policies
- Pattern: One resource per database assignment
- Uses: Read-modify-write to preserve UI-managed databases

### Why Separate Resources?

**CyberArk API Reality**: Both principals and targets are embedded inside the policy API object. However, we use separate assignment resources because:

1. **Composability**: Different teams/modules can manage different principals and databases independently
2. **Consistency**: Same pattern for both principals and targets (no hybrid inconsistency)
3. **Isolation**: Changes to principal assignments don't affect target assignments (and vice versa)
4. **Flexibility**: Security team manages principals centrally, app teams manage their databases

**Pattern Match**: Follows `aws_security_group_rule` pattern (not `aws_iam_policy_attachment`)
- AWS IAM: Principals and policies are SEPARATE API objects (doesn't apply here)
- AWS Security Groups: Rules embedded in group object, managed via separate rule resources (MATCHES our case)

### Usage Example

```hcl
# Security team creates policy with conditions
resource "cyberarksia_database_policy" "db_admins" {
  name        = "Database Administrators"
  description = "Admin access to production databases"

  conditions {
    max_session_duration = 8  # 8 hour sessions
    idle_time            = 30  # 30 minute idle timeout

    access_window {
      days_of_the_week = [1, 2, 3, 4, 5]  # Monday-Friday
      from_hour        = "09:00"
      to_hour          = "17:00"
    }
  }
}

# Security team assigns principals (users and groups)
resource "cyberarksia_database_policy_principal_assignment" "alice" {
  policy_id             = cyberarksia_database_policy.db_admins.id
  principal_id          = "user-123"
  principal_name        = "alice@example.com"
  principal_type        = "USER"
  source_directory_name = "AzureAD"
  source_directory_id   = "dir-456"
}

resource "cyberarksia_database_policy_principal_assignment" "db_admins_group" {
  policy_id             = cyberarksia_database_policy.db_admins.id
  principal_id          = "group-789"
  principal_name        = "DB-Admins"
  principal_type        = "GROUP"
  source_directory_name = "AzureAD"
  source_directory_id   = "dir-456"
}

# App team A assigns their databases
resource "cyberarksia_database_policy_assignment" "team_a_prod_postgres" {
  policy_id             = cyberarksia_database_policy.db_admins.id
  database_workspace_id = cyberarksia_database_workspace.team_a_prod.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["db_admin", "db_writer"]
  }
}

# App team B assigns their databases (in different module/workspace)
resource "cyberarksia_database_policy_assignment" "team_b_prod_mysql" {
  policy_id             = cyberarksia_database_policy.db_admins.id
  database_workspace_id = cyberarksia_database_workspace.team_b_prod.id
  authentication_method = "ldap_auth"

  ldap_auth_profile {
    assign_groups = ["dbadmins"]
  }
}
```

### Multi-Team Workflow

This pattern enables distributed management:

```
┌─────────────────────────────────────────────────────────────┐
│  Security Team Workspace                                    │
│  - Creates policy (metadata + conditions)                   │
│  - Assigns principals (users/groups/roles)                  │
│  - No knowledge of which databases exist                    │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ policy_id reference
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  App Team A Workspace                                       │
│  - Assigns Team A databases to policy                       │
│  - No knowledge of Team B databases                         │
│  - No ability to modify principals or policy conditions     │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ policy_id reference
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  App Team B Workspace                                       │
│  - Assigns Team B databases to policy                       │
│  - No knowledge of Team A databases                         │
│  - No ability to modify principals or policy conditions     │
└─────────────────────────────────────────────────────────────┘
```

### Known Limitations

**Race Conditions**: Multiple workspaces modifying the same policy concurrently can cause race conditions. This is the same limitation as:
- `aws_security_group_rule` (concurrent rules to same group)
- `google_project_iam_member` (concurrent members to same role)
- `azurerm_role_assignment` (concurrent assignments to same scope)

**Mitigation**: Coordinate workspace usage - one workspace per policy for assignments, or use locking mechanisms.

### Why NOT Inline?

**Alternative Considered**: Manage principals/targets inline within policy resource

```hcl
# NOT IMPLEMENTED - Monolithic pattern
resource "cyberarksia_database_policy" "admins" {
  name = "DB Admins"

  # Principals inline
  principal {
    id = "user-123"
  }

  # Targets inline
  database_assignment {
    database_id = "db-456"
  }
}
```

**Why Rejected**:
- ❌ Not composable (all management in one place)
- ❌ Massive blast radius (one change affects everything)
- ❌ Doesn't support distributed team workflows
- ❌ Forces tight coupling of security and app teams

### Why NOT Hybrid?

**Alternative Considered**: Principals inline, targets separate (or vice versa)

**Why Rejected**: Inconsistent treatment of similar concepts. Both principals and targets are embedded in the API policy object, so they should be treated the same way in Terraform for consistency and user experience.

---

## Requirements Traceability

### User Stories → Functional Requirements

**User Story 1: Create Database Policy with Metadata and Conditions**
- Maps to: FR-001 (policy attributes), FR-002 (policy ID), FR-003 (update semantics), FR-004 (status values), FR-005 (conditions), FR-006/FR-006a (policy_entitlement fixed values), FR-007 (delete), FR-008 (import), FR-009 (unique names), FR-010 (idempotent reads), FR-011 (drift detection), FR-012/FR-013 (documentation), FR-034/FR-035/FR-036 (validation strategy)
- Acceptance Scenarios: Create with conditions → FR-001, FR-005; Idempotent read → FR-010; Duplicate name → FR-009, FR-034 (API validates); Update conditions → FR-003

**User Story 2: Assign Principals to Policy**
- Maps to: FR-014 (individual assignments), FR-015 (principal type validation), FR-016/FR-017 (conditional directory requirements), FR-018 (read-modify-write), FR-019 (composite ID), FR-020 (ForceNew), FR-021 (prevent duplicates), FR-022 (import), FR-023 (documentation)
- Acceptance Scenarios: Assign USER with directory → FR-016; Multiple principals → FR-018; Validation error → FR-016, FR-032; Remove specific principal → FR-018; Import → FR-019, FR-022

**User Story 3: Assign Databases to Policy**
- Maps to: FR-024 (individual assignments), FR-025 (read-modify-write), FR-026 (composite ID), FR-027 (ForceNew), FR-028 (6 authentication methods), FR-029 (import), FR-030 (documentation)
- Acceptance Scenarios: Assign database with auth profile → FR-024, FR-028; Multiple databases → FR-025; Remove specific database → FR-025; Import → FR-026, FR-029

**User Story 4: Update Existing Policy Attributes**
- Maps to: FR-003 (in-place updates), FR-018 (preserve principals), FR-025 (preserve targets)
- Acceptance Scenarios: Update description → FR-003; Update status → FR-003, FR-004; Update name → FR-003

**User Story 5: Delete Policies**
- Maps to: FR-007 (cascade delete)
- Acceptance Scenarios: Delete policy → FR-007; Assignments cascade → FR-007

**User Story 6: Import Existing Policies and Assignments**
- Maps to: FR-008 (policy import), FR-019 (principal composite ID), FR-022 (principal import), FR-026 (database composite ID), FR-029 (database import)
- Acceptance Scenarios: Import policy → FR-008; Import principals → FR-019, FR-022; Import databases → FR-026, FR-029; No changes after import → FR-010

### Functional Requirements → User Stories (Reverse Mapping)

**Policy Management (FR-001 to FR-013, FR-034 to FR-036)**
- Primary: User Story 1 (Create Database Policy)
- Secondary: User Story 4 (Update Policy), User Story 5 (Delete), User Story 6 (Import)

**Principal Assignment (FR-014 to FR-023)**
- Primary: User Story 2 (Assign Principals)
- Secondary: User Story 6 (Import)

**Database Assignment (FR-024 to FR-030)**
- Primary: User Story 3 (Assign Databases)
- Secondary: User Story 6 (Import)

**Cross-Cutting (FR-031 to FR-033)**
- Applied to: All User Stories (error handling, validation, retry logic)

### Success Criteria → Functional Requirements

- SC-001 (create with all attributes) → FR-001
- SC-002 (idempotent CRUD) → FR-010
- SC-003 (import preserves attributes) → FR-008, FR-022, FR-029
- SC-004 (delete regardless of assignments) → FR-007
- SC-005 (drift detection) → FR-011
- SC-006 (principal isolation from databases) → FR-018, FR-025
- SC-007 (multiple principal assignments composable) → FR-014, FR-018
- SC-008 (selective principal removal) → FR-018
- SC-009 (preserve UI-managed principals) → FR-018
- SC-010 (database isolation from principals) → FR-018, FR-025
- SC-011 (multiple database assignments composable) → FR-024, FR-025
- SC-012 (selective database removal) → FR-025
- SC-013 (preserve UI-managed databases) → FR-025
- SC-014/SC-015 (LLM-friendly documentation) → FR-012, FR-013
