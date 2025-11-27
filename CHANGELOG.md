# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2025-11-26

### BREAKING CHANGES

1. **Provider Schema**: Renamed provider attribute `client_secret` to `password` for improved clarity
   - Update your provider configuration blocks: change `client_secret` to `password`
   - Example: `provider "cyberarksia" { password = var.password }`

2. **Environment Variable**: Renamed `CYBERARK_CLIENT_SECRET` to `CYBERARK_PASSWORD`
   - Update environment variables in CI/CD pipelines and local development
   - Update `.env` files and Terraform variable references

3. **Resource Rename**: `cyberarksia_database_policy_database_assignment` has been renamed to `cyberarksia_database_policy_workspace_assignment` for better consistency with other resource names
   - **Rationale**: The new name follows the established pattern where assignment resources are named after their base resource (e.g., `cyberarksia_database_workspace`)
   - **Action Required**: Update Terraform state and configuration files (see migration guide below)

4. **Resource Rename**: `cyberarksia_secret` has been renamed to `cyberarksia_database_secret` to accurately reflect that it manages database credentials only (not VM secrets)

5. **AWS IAM Schema**: New required fields for AWS IAM authentication:
   - `aws_account` (string, 12 digits) - AWS account number
   - `aws_username` (string) - IAM username from ARN
   - These fields are now **required** when `authentication_type = "aws_iam"`

### Added

- **VM policy resource** (`cyberarksia_vm_policy`) with multi-cloud support
  - AWS targets: Account IDs, VPC IDs
  - Azure targets: Subscription IDs, Resource Groups, VNet IDs
  - GCP targets: Project IDs, VPC Network IDs
  - On-premises: FQDN/IP targets
  - SSH and RDP protocol support
  - Time-based access windows and session limits
- **VM policy principal assignment resource** (`cyberarksia_vm_policy_principal_assignment`)
  - Assign users, groups, or roles to VM policies
  - Supports inline and separate assignment patterns
- **Target set resource** (`cyberarksia_target_set`)
  - Group VMs/servers by domain, suffix, or target matching
  - Integration with VM secrets for credential management
- **Virtual machine secret resource** (`cyberarksia_virtual_machine_secret`)
  - ProvisionerUser: Username/password credentials for VM provisioning
  - PCloudAccount: Reference to PAM vault accounts
- **PR template**: Standardized pull request format
- **CODEOWNERS**: Automated reviewer assignment

### Changed

- All examples updated to use `password` attribute
- All documentation updated to reference `password` terminology
- Error messages now use "username or password" terminology instead of "client_id or client_secret"
- Renamed `cyberarksia_secret` resource to `cyberarksia_database_secret` in all:
  - Resource implementation
  - Examples and documentation
  - Test files

### Fixed

- **database_policy_workspace_assignment**: Improved error messages when API rejects deletion due to constraint violations (≥1 target required). The Delete() method now translates cryptic API errors like "List should have at least 1 item" into clear, actionable guidance for users.
- **database_policy_principal_assignment**: Improved error messages when API rejects deletion due to constraint violations (≥1 principal required). Provides clear resolution steps when attempting to remove the last principal from a policy.
- **GoReleaser**: Fixed incorrect repository URLs in release notes

### Migration Guide

#### Provider Configuration

1. **Update Provider Configuration**: Change `client_secret` attribute to `password` in all `provider "cyberarksia"` blocks
2. **Update Environment Variables**: Rename `CYBERARK_CLIENT_SECRET` to `CYBERARK_PASSWORD` in:
   - Shell environment (`export CYBERARK_PASSWORD="..."`)
   - CI/CD pipeline secrets
   - Terraform Cloud/Enterprise workspace variables
   - `.env` files
3. **Update Terraform Variables**: Rename any variables like `cyberark_client_secret` to `cyberark_password`
4. **Update Scripts**: Search for `CYBERARK_CLIENT_SECRET` in automation scripts and update to `CYBERARK_PASSWORD`

#### Policy Database Assignment Rename

**Step 1: Update Terraform State**

To upgrade existing Terraform state for policy database assignments without recreating resources:

```bash
terraform state mv 'cyberarksia_database_policy_database_assignment.example' 'cyberarksia_database_policy_workspace_assignment.example'
```

Replace `example` with your actual resource names.

**Step 2: Update HCL Resource Blocks**

Update your Terraform configuration files:

**Before:**
```hcl
resource "cyberarksia_database_policy_database_assignment" "prod_postgres" {
  policy_id             = cyberarksia_database_policy.prod_access.id
  database_workspace_id = cyberarksia_database_workspace.postgres.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["read_only"]
  }
}
```

**After:**
```hcl
resource "cyberarksia_database_policy_workspace_assignment" "prod_postgres" {
  policy_id             = cyberarksia_database_policy.prod_access.id
  database_workspace_id = cyberarksia_database_workspace.postgres.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["read_only"]
  }
}
```

#### Database Secret Rename

**Step 1: Update Terraform State**

To upgrade existing Terraform state without recreating resources, run the following command for each secret resource:

```bash
terraform state mv 'cyberarksia_secret.example' 'cyberarksia_database_secret.example'
```

Replace `example` with your actual resource names. This preserves existing secrets in CyberArk SIA without deletion/recreation.

**Example:**
```bash
# If you have: resource "cyberarksia_secret" "postgres_admin" { ... }
terraform state mv 'cyberarksia_secret.postgres_admin' 'cyberarksia_database_secret.postgres_admin'

# If you have: resource "cyberarksia_secret" "rds_iam_user" { ... }
terraform state mv 'cyberarksia_secret.rds_iam_user' 'cyberarksia_database_secret.rds_iam_user'
```

#### Step 2: Update HCL Resource Blocks

Update your Terraform configuration files (`.tf` files) to use the new resource type name:

**Before:**
```hcl
resource "cyberarksia_secret" "example" {
  name                = "my-database-secret"
  authentication_type = "local"
  username            = "admin"
  password            = "secret"
}
```

**After:**
```hcl
resource "cyberarksia_database_secret" "example" {
  name                = "my-database-secret"
  authentication_type = "local"
  username            = "admin"
  password            = "secret"
}
```

#### Step 3: Update AWS IAM Secrets (If Applicable)

If you use AWS IAM authentication (`authentication_type = "aws_iam"`), you MUST add the new required fields or your configuration will fail validation.

**Before (will fail validation):**
```hcl
resource "cyberarksia_database_secret" "rds_iam" {
  name                  = "rds-iam-secret"
  authentication_type   = "aws_iam"
  aws_access_key_id     = "AKIAIOSFODNN7EXAMPLE"
  aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  # Missing required fields - will cause error!
}
```

**After (correct):**
```hcl
resource "cyberarksia_database_secret" "rds_iam" {
  name                  = "rds-iam-secret"
  authentication_type   = "aws_iam"
  aws_access_key_id     = "AKIAIOSFODNN7EXAMPLE"
  aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  aws_account           = "123456789012"        # NEW: Your 12-digit AWS account number
  aws_username          = "database-admin"      # NEW: IAM username from ARN
}
```

**Finding your AWS account and username:**
- `aws_account`: Your 12-digit AWS account ID (e.g., "123456789012")
- `aws_username`: The IAM username portion from your ARN (e.g., if ARN is `arn:aws:iam::123456789012:user/database-admin`, use `"database-admin"`)

## [0.1.2] - 2025-11-01

### Fixed
- Provider description now includes comprehensive feature list (ZSP/JIT access, 60+ database engines, OAuth2)
- Documentation regenerated to ensure all 6 resources appear in Terraform Registry
  - `cyberarksia_database_workspace` (was missing from Registry)
  - `cyberarksia_secret` (was missing from Registry)
  - `cyberarksia_database_policy_principal_assignment` (was missing from Registry)
  - All resource and data source documentation updated with current schema
- Makefile now detects OS/architecture automatically (works on macOS, Linux, Windows)

### Added
- Complete end-to-end workflow example in `examples/complete/end-to-end-workflow/`
  - Demonstrates secret management, database workspaces, policies, and assignments
  - Shows both inline and modular assignment patterns
  - Includes comprehensive README with troubleshooting guide
- Missing `examples/resources/cyberarksia_database_workspace/` with multiple cloud provider examples

## [0.1.1] - 2025-10-30

### Fixed
- Binary naming to match repository rename (terraform-provider-cyberarksia)
- Terraform Registry installation now works correctly

## [0.1.0] - 2025-10-30

### Added
- Initial provider implementation
- Certificate resource (`cyberarksia_certificate`)
  - Create, read, update, delete TLS/SSL certificates
  - Support for PEM and DER formats
  - Automatic X.509 metadata extraction
  - Label-based organization
- Database workspace resource (`cyberarksia_database_workspace`)
  - Configure database targets with 60+ supported engines
  - Multi-cloud support (AWS, Azure, GCP, Atlas, on-premise)
  - Certificate-based authentication
  - Network segmentation support
  - Authentication method configuration
- Database policy resource (`cyberarksia_database_policy`)
  - Access policies with session limits and time-based restrictions
  - Policy tags, time frames, and access windows
  - Support for inline principal and database assignments
- Database policy principal assignment resource (`cyberarksia_database_policy_principal_assignment`)
  - Assign users, groups, or roles to policies
  - Support for multiple directory types (Cloud Directory, Azure AD, LDAP)
- Policy database assignment resource (`cyberarksia_database_policy_workspace_assignment`)
  - Connect database workspaces to policies
  - Support for 6 authentication methods
- Secret resource (`cyberarksia_secret`)
  - Store database credentials (local auth, Active Directory, AWS IAM)
- Principal data source (`cyberarksia_principal`)
  - Look up users, groups, and roles by name
- Database policy data source (`cyberarksia_database_policy`)
  - Reference existing policies by name or ID
- Provider authentication using CyberArk Identity OAuth2
- ARK SDK integration with automatic token refresh
- Comprehensive error handling and retry logic with exponential backoff
- Acceptance test suite
- Example configurations for common use cases

### Security
- All sensitive fields (passwords, secrets, certificate bodies) properly marked as sensitive
- Certificate validation enabled by default
- Secure OAuth2 token handling with automatic refresh

### Documentation
- Complete resource documentation
- SDK integration guide
- Development guidelines
- Troubleshooting guide
- Multiple example configurations

---

## Version History Notes

This provider was developed using a test-driven approach with comprehensive planning and specification documents available in the `specs/` directory.

For detailed architectural decisions and implementation insights, see [docs/development-history.md](docs/development-history.md).
