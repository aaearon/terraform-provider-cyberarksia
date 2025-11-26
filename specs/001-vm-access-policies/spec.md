# Feature Specification: VM Access Policy Management

**Feature Branch**: `001-vm-access-policies`
**Created**: 2025-11-16
**Status**: Draft
**Input**: User description: "I need Terraform resources for managing CyberArk SIA virtual machine access policies that enable just-in-time privileged access to servers and VMs."

## Clarifications

### Session 2025-11-16

- Q: When an administrator attempts to create a policy with conflicting target criteria (e.g., specifying both FQDN/IP targets AND AWS targets in the same policy configuration), what should happen? → A: Reject at validation with clear error message explaining the one-location-type constraint
- Q: When a policy is updated while users have active privileged sessions using that policy, what should happen to those existing sessions? → A: Out of scope for Terraform provider - provider updates policy configuration; SIA backend system determines session handling behavior
- Q: When an administrator attempts to assign a principal (user, group, or role) to a policy, but that principal ID does not exist in CyberArk Identity, what should happen? → A: Accept assignment; defer validation to SIA backend (let API reject)
- Q: When a user attempts to access a server outside the configured time window (e.g., accessing at 8PM when policy allows only 9AM-5PM), what should happen? → A: Out of scope for Terraform provider - provider manages policy configuration; SIA backend enforces time window restrictions
- Q: What logging or audit events should the Terraform provider emit for VM policy operations (create, update, delete, assignment changes)? → A: Standard Terraform logging via tflog (structured logging) for all CRUD operations and API interactions
- Q: Can a VM policy be created without any principals assigned? → A: No - at least one principal MUST be provided at policy creation time; additional principals can be assigned later via separate assignment resource

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create Basic On-Premises Server Access Policy (Priority: P1)

Security administrators need to create VM access policy configurations for on-premises servers using FQDN and IP address matching criteria with time-based access conditions and at least one principal assignment.

**Why this priority**: This is the foundation of VM policy configuration management - without the ability to create and store policy configurations defining target criteria, access conditions, and initial principal assignments, no other policy management scenarios are possible. This scenario delivers immediate value by enabling infrastructure-as-code policy configuration.

**Independent Test**: Can be fully tested by creating a policy configuration via Terraform with a principal assignment, reading it back from SIA API, and verifying all configuration values including the assigned principal match expected settings. Delivers core policy configuration management.

**Acceptance Scenarios**:

1. **Given** administrator wants to configure access for production web servers, **When** they create a policy with FQDN suffix rule "-prod.example.com", access window (Monday-Friday, 9AM-5PM), 2-hour session duration, and assign principal "admin@example.com", **Then** the policy is created in SIA with status "Active", the principal is assigned, and all configuration values can be read back correctly
2. **Given** administrator needs to configure access for DMZ network servers, **When** they create a policy with IP rule "192.168.100.0/24", logical network name "DMZ Network", SSH username "admin", and assign user principal "ops-team@example.com", **Then** the policy configuration is stored in SIA with the principal assignment and Terraform state reflects all specified values
3. **Given** administrator creates a policy with 1-hour maximum session duration, 10-minute idle timeout, and multiple initial principals, **When** the policy is read back from SIA API, **Then** the session duration, idle timeout, and all assigned principals match the specified configuration

---

### User Story 2 - Assign Additional Users and Groups to Access Policies (Priority: P1)

DevOps teams and security administrators need to add additional principal assignments to existing VM access policies beyond the initial principals configured at policy creation.

**Why this priority**: While policies require at least one principal at creation, the ability to add additional principals dynamically is essential for managing evolving access requirements. This scenario enables flexible "WHO gets access" management without recreating policies.

**Independent Test**: Can be fully tested by creating a policy with one initial principal, then adding a second principal via separate assignment resource, reading the policy back from SIA, and verifying both principals appear in the policy's principals list. Delivers additional principal assignment configuration management.

**Acceptance Scenarios**:

1. **Given** a VM access policy exists with initial principal "admin@company.com", **When** administrator adds user "jane.doe@company.com" via assignment resource, **Then** both principals are stored in SIA and both appear in the policy's principals list when read back
2. **Given** administrator manages access for an entire team, **When** they add CyberArk group "DevOps-Team" to an existing policy, **Then** the group assignment is persisted in policy configuration and visible in Terraform state alongside existing principals
3. **Given** administrator configures broad access, **When** they add role "Database Administrator" to a policy, **Then** the role assignment is stored in policy configuration with correct principal type and ID
4. **Given** administrator attempts to assign a principal that already exists in the policy (either inline or via assignment), **When** the duplicate assignment is submitted, **Then** the operation fails with a clear error message indicating the principal is already assigned

---

### User Story 3 - Manage AWS Cloud VM Access (Priority: P2)

Cloud infrastructure teams need to create VM access policy configurations for AWS EC2 instances using cloud-native target criteria like regions, VPC IDs, account IDs, and resource tags.

**Why this priority**: Critical for cloud-first organizations but builds on the foundation of P1 stories. Can be implemented and tested independently once core policy configuration management works.

**Independent Test**: Can be fully tested by creating a policy with AWS location type, specifying region "us-east-1" and tag "Environment=production", then reading the policy back and verifying all AWS target criteria match expected configuration. Delivers cloud-specific policy configuration.

**Acceptance Scenarios**:

1. **Given** organization uses AWS resource tags for classification, **When** administrator creates a policy with AWS targets specifying tags "Environment=production" and "Application=WebServer", **Then** the policy configuration is stored in SIA with location type "AWS" and both tags can be read back correctly
2. **Given** organization operates in multiple AWS regions, **When** administrator creates a policy specifying regions ["us-east-1", "eu-west-1"] and VPC ID "vpc-abc123", **Then** the AWS target configuration is persisted with all specified criteria and visible in Terraform state
3. **Given** organization has multiple AWS accounts, **When** administrator creates a policy specifying account IDs ["123456789012", "987654321098"], **Then** the account IDs are stored in the policy's AWS target configuration and can be read back correctly

---

### User Story 4 - Configure SSH and RDP Connection Behavior (Priority: P2)

Administrators need to configure connection behavior settings for VM access policies, specifying SSH usernames or RDP ephemeral user configurations with group assignments.

**Why this priority**: Essential for complete policy configuration but can be added after core policy and assignment functionality works. Each connection type can be tested independently.

**Independent Test**: Can be fully tested by creating a policy with SSH username "ec2-user" in Terraform, reading the policy back from SIA, and verifying the SSH behavior configuration matches the specified username. Delivers connection behavior configuration management.

**Acceptance Scenarios**:

1. **Given** administrator needs to configure SSH access, **When** they create a policy with SSH username "centos", **Then** the SSH behavior configuration is stored in SIA with the specified username and can be read back correctly
2. **Given** administrator configures RDP access with local ephemeral users, **When** they create a policy with RDP local ephemeral user settings specifying groups ["Administrators", "Remote Desktop Users"] and reconnection enabled, **Then** the RDP behavior configuration is persisted with all specified settings and visible in Terraform state
3. **Given** administrator configures domain-joined RDP access, **When** they create a policy with RDP domain ephemeral user settings specifying domain groups ["CORP\\ServerAdmins"], **Then** the domain ephemeral user configuration is stored correctly in policy behavior settings
4. **Given** mixed infrastructure requires both SSH and RDP, **When** administrator creates a policy with both SSH username "admin" and RDP ephemeral user configuration, **Then** both connection behavior configurations are stored in the policy and can be read back independently

---

### User Story 5 - Manage Azure and GCP Cloud VM Access (Priority: P3)

Cloud infrastructure teams need to create VM access policy configurations for Azure VMs and GCP instances using their respective cloud-native target criteria.

**Why this priority**: Important for multi-cloud organizations but lower priority than AWS (most common). Can be implemented after AWS configuration pattern is proven.

**Independent Test**: Can be fully tested by creating a policy with Azure location type, resource group "production-rg", and tag "Team=Platform", then reading the policy back and verifying all Azure target criteria match expected configuration. Delivers multi-cloud policy configuration support.

**Acceptance Scenarios**:

1. **Given** organization uses Azure resource groups, **When** administrator creates a policy with Azure targets specifying resource group "production-rg" and regions ["eastus2", "westus"], **Then** the policy configuration is stored in SIA with location type "Azure" and all specified criteria can be read back correctly
2. **Given** organization uses Azure tags for classification, **When** administrator creates a policy with Azure tags "Environment=production" and "CostCenter=IT", **Then** both tags are persisted in the policy's Azure target configuration and visible in Terraform state
3. **Given** organization uses GCP labels for classification, **When** administrator creates a policy with GCP location type, label "env=production", project "my-project-123", and region "us-central1", **Then** the GCP target configuration is stored with all specified criteria and can be read back correctly
4. **Given** organization uses GCP VPC segmentation, **When** administrator creates a policy specifying GCP VPC ID "projects/my-project/global/networks/production-vpc", **Then** the VPC ID is stored in the policy's GCP target configuration and matches when read back

---

### User Story 6 - Update Existing Access Policies (Priority: P2)

Administrators need to modify existing policy configurations to adjust time windows, session limits, target criteria, or connection settings as requirements evolve.

**Why this priority**: Critical for long-term policy configuration management but can be implemented after create functionality is proven. Updates must preserve unmodified configuration elements.

**Independent Test**: Can be fully tested by creating a policy with 1-hour session duration, updating it to 4 hours via Terraform, reading it back from SIA, and verifying the new value is persisted while all other settings remain unchanged. Delivers policy update configuration management.

**Acceptance Scenarios**:

1. **Given** policy configuration needs session duration adjustment, **When** administrator updates maximum session duration from 1 hour to 4 hours, **Then** the updated value is persisted in SIA and all other policy settings remain unchanged when read back
2. **Given** business hours configuration changes seasonally, **When** administrator updates access window from "Monday-Friday 9AM-5PM" to "Monday-Friday 8AM-6PM", **Then** the new time window configuration is stored correctly and visible in Terraform state
3. **Given** infrastructure changes require different target criteria, **When** administrator adds a new FQDN rule matching "-staging" servers to an existing policy, **Then** the new rule is added to the policy's target configuration while preserving existing rules
4. **Given** connection configuration requirements change, **When** administrator updates RDP ephemeral user group assignments from ["Remote Desktop Users"] to ["Administrators", "Remote Desktop Users"], **Then** the updated group list is persisted in RDP behavior configuration and can be read back correctly

---

### User Story 7 - Remove Access and Decommission Policies (Priority: P3)

Administrators need to remove principal assignments from policies and delete policy configurations that are no longer needed, supporting complete policy lifecycle management.

**Why this priority**: Important for configuration hygiene but lower priority than creation and update scenarios. Deletion operations are simpler to implement than creation.

**Independent Test**: Can be fully tested by creating a policy, assigning a principal, removing the assignment via Terraform, deleting the policy, and verifying the policy no longer exists in SIA. Delivers complete configuration lifecycle management.

**Acceptance Scenarios**:

1. **Given** principal assignment is no longer needed, **When** administrator removes a user's assignment from a policy, **Then** the principal is removed from the policy's principals list in SIA while other assigned principals remain unchanged
2. **Given** a policy configuration is no longer needed, **When** administrator deletes the policy via Terraform, **Then** the policy is removed from SIA and no longer appears in list operations
3. **Given** policy has been deleted externally (drift scenario), **When** Terraform refresh runs, **Then** the provider detects the policy no longer exists in SIA and removes the resource from state without error

---

### Edge Cases

- **Access outside time window**: Out of scope for Terraform provider - provider manages policy configuration; SIA backend enforces time window restrictions and handles access denials
- **Zero session duration or idle timeout**: Validation rejects configuration values of zero; minimum session duration is 1 hour, minimum idle timeout is 1 minute (per FR-004, FR-005)
- **Policy creation without principals**: Validation rejects policy creation attempts with zero principals; at least one principal required at creation time with error message indicating minimum principal requirement (per FR-041)
- **Duplicate principal assignments**: Validation prevents duplicate assignments across both inline principals and assignment resources; operation fails with error message indicating principal already assigned (per FR-022, FR-044)
- **Conflicting target criteria**: When administrators specify both FQDN/IP targets AND cloud targets (AWS/Azure/GCP) in the same policy configuration, validation rejects the configuration with a clear error message explaining the one-location-type constraint (exactly one location type required per policy)
- **Exceeding metadata character limits**: Validation rejects configurations exceeding limits (name 200 chars, description 200 chars, tags 50 chars each) with clear error messages (per FR-001, FR-028, FR-029)
- **IP rules without logical names**: Validation rejects IP rules missing required logical network name with error message (per FR-014, FR-035)
- **Invalid FQDN operators**: Validation rejects operators not in allowed set (EXACTLY, WILDCARD, PREFIX, SUFFIX, CONTAINS) with error message listing valid operators (per FR-013)
- **Empty connection behavior**: Validation rejects policies without at least one connection profile (SSH or RDP) with error message requiring at least one profile (per FR-011)
- **Empty SSH username or RDP groups**: Validation rejects empty required fields with specific error messages identifying missing configuration (per FR-008, FR-009, FR-010)
- **Time frame conflicts**: Validation rejects access window configurations where fromTime is after toTime with error message explaining valid time range requirements (per FR-006)
- **Policy updates with active sessions**: Out of scope for Terraform provider - provider updates policy configuration; SIA backend system determines how existing sessions are handled
- **Non-existent cloud resources**: Configuration is accepted by provider; SIA backend validates resource existence at enforcement time (validation deferred to API)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow administrators to create VM access policies with unique names (1-200 characters)
- **FR-002**: System MUST allow administrators to specify policy status as either "Active" or "Suspended"
- **FR-003**: System MUST support exactly ONE location type per policy (FQDN/IP, AWS, Azure, or GCP)
- **FR-003a**: System MUST reject policy configurations that specify multiple conflicting location types with a clear validation error message
- **FR-004**: System MUST allow administrators to define maximum session duration between 1 and 24 hours (default 1 hour)
- **FR-005**: System MUST allow administrators to define idle timeout between 1 and 120 minutes (default 10 minutes)
- **FR-006**: System MUST support optional daily access windows specifying days of week (0-6) and time ranges (fromHour/toHour)
- **FR-007**: System MUST support optional policy activation timeframes with start and end timestamps
- **FR-008**: System MUST allow administrators to configure SSH connection behavior with a required username
- **FR-009**: System MUST allow administrators to configure RDP connection behavior with local ephemeral user settings (assigned groups and reconnection policy)
- **FR-010**: System MUST allow administrators to configure RDP connection behavior with domain ephemeral user settings (assigned local and domain groups, reconnection policy)
- **FR-011**: System MUST require at least one connection profile (SSH or RDP) per policy
- **FR-012**: System MUST support both SSH and RDP profiles simultaneously on the same policy
- **FR-013**: System MUST allow administrators to define FQDN matching rules using operators (EXACTLY, WILDCARD, PREFIX, SUFFIX, CONTAINS)
- **FR-014**: System MUST allow administrators to define IP address matching rules using operators (EXACTLY, WILDCARD) with required logical network names
- **FR-015**: System MUST support multiple FQDN rules and multiple IP rules within a single FQDN/IP location type policy
- **FR-016**: System MUST allow administrators to define AWS target criteria using regions, resource tags (key-value pairs), VPC IDs, and account IDs
- **FR-017**: System MUST allow administrators to define Azure target criteria using regions, resource tags (key-value pairs), resource groups, VNet IDs, and subscriptions
- **FR-018**: System MUST allow administrators to define GCP target criteria using regions, labels (key-value pairs, not tags), VPC IDs, and projects
- **FR-019**: System MUST allow administrators to assign users to policies by specifying user ID and name with required source directory information
- **FR-020**: System MUST allow administrators to assign groups to policies by specifying group ID and name with required source directory information
- **FR-021**: System MUST allow administrators to assign roles to policies by specifying role ID and name with optional source directory information
- **FR-022**: System MUST prevent duplicate principal assignments to the same policy
- **FR-023**: System MUST allow administrators to remove principal assignments from policies without affecting other assigned principals
- **FR-024**: System MUST allow administrators to update existing policies while preserving unmodified configuration elements
- **FR-025**: System MUST allow administrators to delete policies
- **FR-026**: System MUST detect when policies have been deleted externally and handle drift gracefully
- **FR-027**: System MUST track policy creation and update metadata (creator, timestamps)
- **FR-028**: System MUST support optional policy descriptions (up to 200 characters)
- **FR-029**: System MUST support optional policy tags (up to 20 tags, each up to 50 characters)
- **FR-030**: System MUST support optional policy timezone configuration (default "GMT")
- **FR-031**: System MUST read delegation classification computed by SIA backend based on policy configuration
- **FR-032**: System MUST read unique policy IDs generated by SIA backend upon policy creation
- **FR-033**: System MUST validate FQDN computername patterns do not exceed 300 characters
- **FR-034**: System MUST validate FQDN domain names do not exceed 1000 characters
- **FR-035**: System MUST validate IP rule logical names are between 1 and 256 characters
- **FR-036**: System MUST validate IP addresses arrays do not exceed 1000 entries
- **FR-037**: System MUST validate principal IDs do not exceed 40 characters
- **FR-038**: System MUST validate principal names are between 1 and 512 characters
- **FR-039**: System MUST validate source directory names do not exceed 50 characters
- **FR-040**: System MUST allow administrators to import existing policies into Terraform management using policy ID
- **FR-041**: System MUST require at least one principal assignment at VM policy creation time
- **FR-042**: System MUST support multiple initial principal assignments at policy creation (inline principals)
- **FR-043**: System MUST allow additional principals to be assigned after policy creation via separate assignment resource
- **FR-044**: System MUST preserve both inline principals (from policy resource) and assigned principals (from assignment resources) during policy updates

### Non-Functional Requirements

- **NFR-001**: System MUST emit structured logs via Terraform's tflog for all policy CRUD operations (create, read, update, delete) and principal assignment changes
- **NFR-002**: System MUST log API interactions with sufficient detail for troubleshooting (request/response metadata, errors, retry attempts)

### Key Entities *(include if feature involves data)*

- **VM Access Policy**: Represents a VM access policy configuration defining WHO is assigned (principals - at least one required at creation), WHAT servers are targeted (target criteria), WHEN access is configured (time windows), and HOW connections are configured (SSH/RDP settings). Key attributes include policy name, status (Active/Suspended), location type, initial principal assignments (minimum one), target criteria, connection behavior (SSH/RDP), time-based conditions (session duration, idle timeout, access windows), and policy metadata (description, tags, timezone)

- **Policy Target**: Represents the target selection criteria configuration defining which servers/VMs a policy applies to. Varies by location type:
  - FQDN/IP: FQDN matching rules (operator, pattern, domain) and IP matching rules (operator, addresses, logical name)
  - AWS: Regions, resource tags, VPC IDs, account IDs
  - Azure: Regions, resource tags, resource groups, VNet IDs, subscriptions
  - GCP: Regions, labels, VPC IDs, projects

- **Connection Behavior**: Represents the connection configuration settings for a policy. Includes SSH profile (username) and/or RDP profile (local or domain ephemeral user configuration with group assignments and reconnection settings)

- **Access Conditions**: Represents time-based access configuration settings. Includes maximum session duration (1-24 hours), idle timeout (1-120 minutes), access windows (days of week, time ranges), and policy activation timeframes (start/end timestamps)

- **Principal Assignment**: Represents the configuration association between a user, group, or role and a VM access policy. Principals can be assigned in two ways: (1) inline principals defined at policy creation (required - minimum one), or (2) additional principals added via separate assignment resource after policy creation (optional). Key attributes include principal ID, principal name, principal type (USER/GROUP/ROLE), and source directory information. Both inline and assigned principals are stored together in the policy's principals array

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Policy updates preserve all unmodified configuration elements with zero data loss
- **SC-002**: System detects policy drift (external deletions) and updates state correctly on next Terraform run
- **SC-003**: Policy creation fails with actionable error messages when validation rules are violated (e.g., missing SSH username, duplicate principals)
- **SC-004**: Compliance auditors can review all active VM access policy configurations through version-controlled Terraform code
- **SC-005**: Security teams can deploy consistent policy configurations across multiple environments using Terraform modules
- **SC-006**: Policy configuration changes are tracked in version control with complete audit trail of who made changes and when
