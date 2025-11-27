# Feature Specification: Principal Lookup Data Source

**Feature Branch**: `003-principal-lookup`
**Created**: 2025-10-29
**Status**: Draft
**Input**: User description: "Build a Terraform data source for the CyberArk SIA provider that enables users to look up principals (users, groups, or roles) by their name and automatically retrieve all the information needed for policy assignments, eliminating the need for manual UUID lookups and portal navigation."

## Clarifications

### Session 2025-10-29

- Q: What happens when a principal name matches multiple principals across different directories? → A: Throw an error when multiple principals are found
- Q: For operational monitoring and debugging, what observability signals should the data source emit? → A: Log errors and successful lookups with principal names (detailed logging)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cloud Directory User Lookup (Priority: P1)

Sarah is a DevOps engineer managing infrastructure-as-code for her team. She needs to grant her colleague Tim access to a PostgreSQL database through a Terraform-managed policy. Tim's cloud username is "tim.schindler@cyberark.cloud.40562" - Sarah knows this because it's how Tim logs into the system. However, Sarah doesn't know Tim's UUID, which directory he belongs to, or the directory's UUID.

Currently, Sarah must leave her Terraform workflow, log into the CyberArk Identity portal, navigate through menus to find Tim's user profile, copy his UUID, determine he's in the Cloud Directory, find the Cloud Directory UUID through API calls or browser inspection, and paste all these values into her Terraform configuration. This manual process is error-prone and frustrating.

With the principal lookup data source, Sarah simply adds a data block with Tim's name. When she runs `terraform plan`, it automatically looks up Tim's UUID, identifies the Cloud Directory, retrieves the directory UUID, and returns all five fields needed for the policy assignment.

**Why this priority**: This is the most common use case. Most CyberArk deployments use Cloud Directory for native cloud users, and user-to-policy assignments are the primary workflow.

**Independent Test**: Can be fully tested by creating a data source referencing a Cloud Directory user by name, running `terraform plan`, and verifying all required attributes (id, type, name, directory_name, directory_id) are populated correctly.

**Acceptance Scenarios**:

1. **Given** a Cloud Directory user exists with username "tim.schindler@cyberark.cloud.40562", **When** Terraform reads the data source with `name = "tim.schindler@cyberark.cloud.40562"`, **Then** it returns the user's UUID, type (USER), directory name ("CyberArk Cloud Directory"), and directory UUID
2. **Given** multiple Cloud Directory users exist, **When** Terraform reads the data source for one specific user by name, **Then** it returns only that user's information (no ambiguity)
3. **Given** a Cloud Directory user's name, **When** the name uses different casing than stored (e.g., "Tim.Schindler@..." vs "tim.schindler@..."), **Then** the lookup succeeds (case-insensitive)
4. **Given** a Cloud Directory user lookup succeeds, **When** the returned attributes are used in a policy assignment resource, **Then** the policy assignment creation succeeds without errors

---

### User Story 2 - Federated Directory User Lookup (Priority: P1)

Mike works at a company that uses Microsoft Entra ID (formerly Azure AD) federated with CyberArk Identity. He needs to grant access to a user named "john.doe@company.com" who exists in the federated directory. Mike doesn't know which directory this user is in - is it the cloud directory? Is it one of the federated directories? Which federated directory?

Mike shouldn't need to understand the directory architecture. He uses the data source with the name "john.doe@company.com", and the lookup automatically identifies that this user is in the federated directory, finds the UUID, gets the directory name (something like "Federation with company.com"), retrieves the directory UUID, and returns everything needed.

**Why this priority**: Federated directories are increasingly common in enterprise deployments. Supporting this use case ensures the data source works in modern identity architectures where users authenticate through external identity providers (Entra ID, Okta, etc.).

**Independent Test**: Can be fully tested by creating a data source referencing a federated user by name, running `terraform plan`, and verifying all attributes are populated with FDS directory information.

**Acceptance Scenarios**:

1. **Given** a federated user exists with username "john.doe@company.com", **When** Terraform reads the data source with that name, **Then** it returns the user's UUID, type (USER), FDS directory name, and directory UUID
2. **Given** a user exists in both Cloud Directory and Federated Directory with similar names, **When** looking up by exact system name, **Then** the correct user from the correct directory is returned
3. **Given** a federated user's system name, **When** the lookup executes, **Then** the directory_name field contains the localized, human-readable directory name (e.g., "Federation with company.com")

---

### User Story 3 - Group Lookup (Priority: P1)

Maria manages database access for her organization and wants to assign an entire group of database administrators to a policy. The group is called "Database Administrators" and exists in one of the directories. Maria doesn't know if it's in the cloud directory or a federated directory, and she doesn't know the group's UUID.

Maria uses the data source with the group name and optionally specifies `type = "GROUP"` to narrow the search. The lookup finds the group, identifies which directory it's in, and returns all required information. Maria can now assign the entire group to the policy without manually looking up any UUIDs.

**Why this priority**: Group-based access management is a best practice for enterprise security. Supporting groups alongside users ensures the data source works for both individual and group-based policy assignments.

**Independent Test**: Can be fully tested by creating a data source referencing a group by name, optionally filtering by type, and verifying all attributes are populated correctly.

**Acceptance Scenarios**:

1. **Given** a group named "Database Administrators" exists, **When** Terraform reads the data source with `name = "Database Administrators"`, **Then** it returns the group's UUID, type (GROUP), directory name, and directory UUID
2. **Given** a principal name that could match users or groups, **When** the data source specifies `type = "GROUP"`, **Then** only group matches are considered
3. **Given** a group exists across multiple directories, **When** looking up by name only, **Then** the system raises an error indicating multiple matches were found

---

### User Story 4 - Active Directory User Lookup (Priority: P2)

Tom's company has an on-premises Active Directory connected to CyberArk through an AD Connector (AdProxy). He wants to grant access to a user whose AD username is "SchindlerT@cyberiam.tech". Tom doesn't know the user's UUID or the AdProxy directory UUID.

Tom uses the data source with the AD username. The lookup automatically identifies this user is in the Active Directory, finds the user UUID, retrieves the AdProxy directory information, and returns everything. Tom doesn't need to understand how AdProxy works.

**Why this priority**: While less common than Cloud Directory and Federated Directory in new deployments, many enterprises still use on-premises Active Directory. Supporting AdProxy ensures the data source works for hybrid identity architectures.

**Independent Test**: Can be fully tested by creating a data source referencing an AD user by username, running `terraform plan`, and verifying all attributes include AdProxy directory information.

**Acceptance Scenarios**:

1. **Given** an Active Directory user exists with username "SchindlerT@cyberiam.tech", **When** Terraform reads the data source with that name, **Then** it returns the user's UUID, type (USER), AdProxy directory name, and directory UUID
2. **Given** an AD user lookup succeeds, **When** the directory_name attribute is examined, **Then** it contains a recognizable AD directory name (e.g., "Active Directory (cyberiam.tech)")

---

### User Story 5 - Error Handling and Not Found (Priority: P1)

Alex is writing Terraform configuration and tries to look up a user named "nonexistent.user@example.com" who doesn't actually exist in any directory. Instead of getting a cryptic API error or Terraform crash, Alex receives a clear, actionable error message.

The error message states: "Principal 'nonexistent.user@example.com' not found in any directory". This helps Alex realize they might have mistyped the name or the user doesn't exist yet, prompting them to verify the principal name or create the user first.

**Why this priority**: Clear error handling is critical for user experience. Cryptic errors lead to support tickets, frustration, and abandoned adoption. Good error messages help users self-service and understand what went wrong.

**Independent Test**: Can be fully tested by creating a data source referencing a non-existent principal name, running `terraform plan`, and verifying a clear error message is displayed (not a stack trace or API error).

**Acceptance Scenarios**:

1. **Given** a principal name that doesn't exist in any directory, **When** Terraform reads the data source, **Then** it fails with a clear error message indicating the principal was not found
2. **Given** the CyberArk API is unavailable, **When** Terraform attempts to read the data source, **Then** it fails with a clear error message about connectivity (not a generic timeout)
3. **Given** authentication credentials are invalid or expired, **When** Terraform attempts to read the data source, **Then** it fails with a clear error message about authentication failure

---

### Edge Cases

- **Multiple matches across directories**: When a principal name matches multiple principals across different directories, the system raises a clear error indicating multiple matches were found. The error message lists the matched principals with their directory names to help users understand the ambiguity.
- How does the system handle special characters in principal names (e.g., spaces, unicode, special symbols)?
- What happens when a principal exists but has incomplete metadata (e.g., no email address for a user)?
- How does the system handle principal names that are extremely long (edge case for string length limits)?
- What happens when looking up a role (ROLE type) - are roles structured differently than users and groups in the directory system?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept a principal name (string) as the primary lookup parameter
- **FR-002**: System MUST support lookup of all three principal types: USER, GROUP, and ROLE
- **FR-003**: System MUST support lookup across all three directory types: CDS (Cloud Directory), FDS (Federated Directory Service), and AdProxy (Active Directory Proxy)
- **FR-004**: System MUST return five required attributes for policy assignments:
  - Principal UUID (id)
  - Principal type (USER, GROUP, or ROLE)
  - Principal name (system name as provided)
  - Directory name (localized, human-readable directory name)
  - Directory UUID
- **FR-005**: System MUST perform case-insensitive matching on principal names (e.g., "tim.schindler@..." matches "Tim.Schindler@...")
- **FR-006**: System MUST support optional type filtering (USER, GROUP, or ROLE) to narrow search scope
- **FR-007**: System MUST return additional informational attributes when available:
  - Display name (human-readable name)
  - Email address (for users)
  - Description (if available)
- **FR-008**: System MUST provide clear, actionable error messages when:
  - Principal not found in any directory
  - Authentication fails
  - API connectivity issues occur
  - Multiple principals match the same name across different directories (error must list matched principals with directory names)
- **FR-009**: System MUST work with existing provider authentication mechanisms (service account, OAuth, etc.)
- **FR-010**: System MUST NOT cache lookup results (each Terraform data source read must query current state)
- **FR-011**: System MUST log errors and successful lookup attempts including principal name, lookup duration, and result status for operational debugging

### Key Entities

- **Principal**: Represents a user, group, or role in the CyberArk Identity system
  - Attributes: UUID (unique identifier), name (system name), type (USER/GROUP/ROLE), display name, email (optional), description (optional)
  - Belongs to exactly one directory
- **Directory**: Represents an identity source (CDS, FDS, or AdProxy)
  - Attributes: UUID (unique identifier), name (localized human-readable name), type (CDS/FDS/AdProxy)
  - Contains multiple principals

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% lookup accuracy - no false positives where wrong principal is returned
- **SC-002**: Users can complete policy assignment configuration without leaving Terraform workflow - zero manual UUID lookups required
- **SC-003**: Data source follows Terraform conventions - all computed attributes properly marked, schema matches Terraform provider patterns
- **SC-004**: Documentation includes working examples for all three directory types and all three principal types
- **SC-005**: Lookups work for all principal types (USER, GROUP, ROLE) with equal reliability

## Assumptions

Since the user description provides extensive context, minimal assumptions are needed:

- **Assumption 1**: The ARK SDK (mentioned in investigation document context) provides necessary APIs for principal and directory lookups
- **Assumption 2**: Principal system names MAY NOT be unique across all directories within a tenant (same name could exist in multiple directories - see Clarifications)
- **Assumption 3**: Directory information is always available when a principal is found (principals are never orphaned without directory context)
- **Assumption 4**: The data source will use read-only operations and will not modify any principals or directories
- **Assumption 5**: Terraform data source refresh behavior follows standard patterns (reads are idempotent and stateless)

## Out of Scope

These capabilities are explicitly NOT included in this feature specification:

- **Caching**: No provider-level caching of directory mappings or principal lookups across multiple data source instances
- **Batch Lookup**: No support for looking up multiple principals in a single data source operation
- **Filtering by Additional Attributes**: No support for filtering by email, display name, or other secondary attributes (only by system name and optionally type)
- **Wildcard or Partial Matching**: No support for partial name matching or wildcard searches (only exact system name matching)
- **Principal Creation**: The data source only looks up existing principals; it does not create new principals
- **Directory Listing**: The data source does not provide a way to list all available directories or all principals in a directory
