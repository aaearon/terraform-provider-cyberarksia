# terraform-provider-cyberarksia Development Guidelines

**Purpose**: Quick reference for LLM-assisted development of the CyberArk SIA Terraform Provider

**Last Updated**: 2025-11-08 (Added Target Set resource implementation)

## Branch Protection & Workflow

**Main branch is protected** - direct commits are blocked. All changes must go through pull requests.

**Standard workflow:**
```bash
# 1. Create feature branch
git checkout -b feature/description-of-change

# 2. Make changes, commit locally
git add .
git commit -m "feat: description"

# 3. Push and create PR
git push -u origin feature/description-of-change
gh pr create --title "feat: description" --body "Details..."

# 4. Merge via PR (even if you're the only reviewer)
gh pr merge --squash

# 5. Clean up
git checkout main
git pull
git branch -d feature/description-of-change
```

**Branch naming conventions:**
- `feature/*` - New features
- `fix/*` - Bug fixes
- `docs/*` - Documentation only
- `refactor/*` - Code refactoring
- `test/*` - Test additions/changes
- `chore/*` - Maintenance tasks

## Quick Start

**Technology Stack**:
- **Go**: 1.25.0
- **ARK SDK**: github.com/cyberark/ark-sdk-golang v1.5.0 (has DELETE bug - see Critical Constraints)
- **Terraform Plugin Framework**: v1.16.1 (Plugin Framework v6)
- **Terraform Plugin Log**: v0.9.0

**Build & Run**:
```bash
go build -v                    # Build provider
go install                     # Install locally for testing
TF_ACC=1 go test ./... -v     # Run acceptance tests
```

**Key Files**:
- Authentication: `internal/client/auth.go`
- Profile Factory: `internal/provider/profile_factory.go` (centralized auth profile building)
- Error Handling: `internal/client/errors.go`, `internal/client/retry.go`
- SDK Workarounds: `internal/client/sdk_workarounds.go` (DELETE + Azure VM policy bug fixes)
- Legacy DELETE Workaround: `internal/client/delete_workarounds.go` (ARK SDK bug fix)

## Environment Setup

### Prerequisites
- CyberArk Identity tenant with SIA enabled
- OAuth2 service account credentials (username format: `service-account@cyberark.cloud.XXXXX`)
- Go 1.25.0 installed
- Terraform CLI v1.5+ installed

### Required Environment Variables

For acceptance tests and local development, export these variables:

```bash
# CyberArk Identity Authentication
export CYBERARK_USERNAME="service-account@cyberark.cloud.12345"
export CYBERARK_PASSWORD="<your-password-here>"

# Optional: CyberArk Identity URL (only needed for GovCloud or custom deployments)
# If not provided, URL is automatically resolved from username by ARK SDK
export CYBERARK_IDENTITY_URL="https://abc123.cyberark.cloud"

# Enable Terraform acceptance tests
export TF_ACC=1

# Optional: Terraform logging
export TF_LOG=DEBUG           # For verbose provider logs
export TF_LOG_PATH=./tf.log   # Save logs to file
```

### Terraform CLI Configuration (Local Development)

To use the locally-built provider, configure Terraform CLI dev overrides:

**Configuration File Location**:
- Linux/macOS: `~/.terraformrc`
- Windows: `%APPDATA%/terraform.rc`

**Configuration Content**:
```hcl
provider_installation {
  dev_overrides {
    "aaearon/cyberarksia" = "~/.terraform.d/plugins/local/aaearon/cyberark-sia/dev/linux_amd64"
  }
  direct {}
}
```

**Note**: Adjust the path to match your `make install` target. When dev overrides are active, Terraform will skip version constraints and use your local binary.

### Obtaining SIA Credentials

1. Log into CyberArk Identity admin console
2. Navigate to **Applications** → **Add Web Apps** → **Custom**
3. Create OAuth2 confidential client:
   - **Application ID**: `terraform-provider-sia` (or your preferred name)
   - **Grant Types**: `Client Credentials`
   - **Scopes**: `sia`, `identity`
4. Save the **username** (format: `app-name@cyberark.cloud.XXXXX`) and **client secret**

### Verifying Your Setup

Test that credentials work:

```bash
# Build and install provider
make build
make install

# Run provider configuration test
TF_ACC=1 go test ./internal/provider -v -run TestAccProvider_Configure
```

If successful, you're ready for development.

## Project Structure

```
terraform-provider-cyberarksia/
├── internal/
│   ├── provider/         # Terraform provider implementation
│   │   ├── profile_factory.go    # Authentication profile factory
│   │   └── helpers/               # Shared utilities
│   │       ├── id_conversion.go   # Database ID conversion
│   │       └── composite_ids.go   # Composite ID parsing/building
│   ├── client/          # ARK SDK wrappers, retry, error handling
│   ├── models/          # Data models
│   └── validators/      # Custom Terraform validators (DatabaseEngine, etc.)
├── examples/            # Terraform HCL examples
│   ├── complete/        # Complete working examples
│   ├── provider/        # Provider configuration examples
│   ├── resources/       # Per-resource examples
│   └── testing/         # CRUD testing framework templates
├── docs/                # Documentation
│   ├── guides/          # User guides
│   ├── resources/       # Resource documentation
│   ├── development/     # Design decisions, implementation summaries
│   ├── sdk-integration.md      # ARK SDK reference
│   ├── development-history.md  # Development timeline
│   └── troubleshooting.md      # Common issues & solutions
├── specs/               # Feature specifications (active)
└── specs-archive/       # Archived specifications
```

## Provider Overview

### Available Resources & Data Sources

| Type | Name | Implementation | Status | Purpose |
|------|------|----------------|--------|---------|
| Resource | `cyberarksia_database_workspace` | `internal/provider/database_workspace_resource.go` | ✅ Stable | Database target configuration (60+ engines supported) |
| Resource | `cyberarksia_database_secret` | `internal/provider/secret_resource.go` | ✅ Stable | Strong account credentials (username/password, AWS IAM) |
| Resource | `cyberarksia_certificate` | `internal/provider/certificate_resource.go` | ✅ Stable | TLS/mTLS certificates for database connections |
| Resource | `cyberarksia_database_policy` | `internal/provider/database_policy_resource.go` | ✅ Stable | Access policies with time-based conditions |
| Resource | `cyberarksia_database_policy_principal_assignment` | `internal/provider/database_policy_principal_assignment_resource.go` | ✅ Stable | Assign users/groups/roles TO policies (WHO gets access) |
| Resource | `cyberarksia_database_policy_workspace_assignment` | `internal/provider/database_policy_workspace_assignment_resource.go` | ✅ Stable | Assign database workspaces TO policies (WHAT they access) |
| Resource | `cyberarksia_virtual_machine_secret` | `internal/provider/virtual_machine_secret_resource.go` | ✅ Stable | VM credentials (ProvisionerUser/PCloudAccount) |
| Resource | `cyberarksia_target_set` | `internal/provider/target_set_resource.go` | ✅ Stable | VM/server target sets (Domain/Suffix/Target matching) |
| Resource | `cyberarksia_vm_policy` | `internal/provider/vm_policy_resource.go` | ✅ Stable | VM access policies (FQDN/IP, AWS, Azure, GCP targets with SSH/RDP) |
| Resource | `cyberarksia_vm_policy_principal_assignment` | `internal/provider/vm_policy_principal_assignment_resource.go` | ✅ Stable | Assign users/groups/roles TO VM policies (WHO gets access) |
| Data Source | `cyberarksia_principal` | `internal/provider/principal_data_source.go` | ✅ Stable | Lookup users/groups/roles by name (no manual UUID needed) |

### Quick Reference: Resource Selection Guide

**Use this decision tree to quickly identify which resources you need:**

| If you need to... | Use this resource | Notes |
|-------------------|-------------------|-------|
| Store database credentials | `cyberarksia_database_secret` | Supports username/password, AWS IAM |
| Configure a database target | `cyberarksia_database_workspace` | Requires secret_id reference |
| Add TLS/mTLS certificate | `cyberarksia_certificate` | Optional for database connections |
| Define access policy conditions | `cyberarksia_database_policy` | Time-based access windows |
| Grant users access to policies | `cyberarksia_database_policy_principal_assignment` | WHO gets access |
| Assign databases to policies | `cyberarksia_database_policy_workspace_assignment` | WHAT they access |
| Store VM/server credentials | `cyberarksia_virtual_machine_secret` | ProvisionerUser or PCloudAccount |
| Group VMs/servers by criteria | `cyberarksia_target_set` | Domain/Suffix/Target matching |
| Define VM access policies | `cyberarksia_vm_policy` | Time-based access with cloud/on-prem targets |
| Grant users access to VM policies | `cyberarksia_vm_policy_principal_assignment` | WHO gets VM access |
| Look up user/group/role UUIDs | `cyberarksia_principal` (data source) | Avoids manual UUID lookup |

**Common Tasks Quick Links:**

| Task | Resources Needed | See Section |
|------|------------------|-------------|
| Set up database access for first time | database_secret → database_workspace → database_policy → principal_assignment + workspace_assignment | [Resource Dependencies](#resource-dependencies) |
| Grant user access to database | principal_assignment | [Policy Management](#read-modify-write-for-policy-assignments) |
| Add database to existing policy | workspace_assignment | [Policy Management](#read-modify-write-for-policy-assignments) |
| Configure VM target management | virtual_machine_secret → target_set | [Resource Dependencies](#resource-dependencies) |
| Rotate database credentials | Update database_secret resource | [Error Handling Pattern](#error-handling-pattern) |
| Debug authentication issues | Check logs, verify credentials | [Debugging Test Failures](#debugging-test-failures) |

### Resource Dependencies

Typical configuration flow:

```
1. cyberarksia_database_secret (credentials)
     ↓
2. cyberarksia_database_workspace (database target, references secret)
     ↓
3. cyberarksia_database_policy (access conditions)
     ↓
     ├→ 4a. cyberarksia_database_policy_principal_assignment (WHO: assign users/groups/roles)
     └→ 4b. cyberarksia_database_policy_workspace_assignment (WHAT: assign database workspaces)
```

**Note**: Principal and database assignments can be managed independently by different teams (security team manages WHO, app team manages WHAT).

**VM/Server configuration flow:**
```
1. cyberarksia_virtual_machine_secret (VM credentials)
     ↓
2. cyberarksia_target_set (VM/server target grouping)
     ↓
3. cyberarksia_vm_policy (access policies with inline principals + cloud/on-prem targets)
     ↓
     └→ 4. cyberarksia_vm_policy_principal_assignment (optional: add more principals)
```

**Important**: At least one principal MUST be defined inline in `cyberarksia_vm_policy` at creation time. The `cyberarksia_vm_policy_principal_assignment` resource is only for adding additional principals beyond those inline ones. VM policies can target cloud resources (AWS, Azure, GCP) or on-premises servers (FQDN/IP) independently of target sets.

## Architecture Patterns

### Profile Factory Pattern

**When to Use**: Creating/updating ANY resource with authentication profiles (db_auth, ldap_auth, oracle_auth, mongo_auth, sqlserver_auth, rds_iam_user_auth)

**Location**: `internal/provider/profile_factory.go`

**Usage**:
```go
// In Create() or Update() methods
profile := BuildAuthenticationProfile(ctx, authMethod, data, &diags)
if diags.HasError() {
    return
}
instanceTarget.Profile = profile
```

**Why**: Eliminates 410 LOC duplication, centralizes validation for all 6 authentication methods, prevents auth drift bugs

**Anti-Pattern**: ❌ Don't manually construct authentication profiles in resource CRUD methods

### Helper Utilities

**Composite ID Parsing** (`internal/provider/helpers/composite_ids.go`):
```go
// Policy-database assignments (2-part ID)
policyID, databaseID, err := helpers.ParsePolicyDatabaseID(id)

// Policy-principal assignments (3-part ID)
policyID, principalID, principalType, err := helpers.ParsePolicyPrincipalID(id)
```

**Database ID Conversion** (`internal/provider/helpers/id_conversion.go`):
```go
// API returns string IDs in JSON but accepts int64 in URLs
dbID, err := helpers.ConvertDatabaseID(stringID)
```

### Error Handling Pattern

**Always Use**:
```go
import "github.com/aaearon/terraform-provider-cyberarksia/internal/client"

// Wrap SDK calls with retry logic
err := client.RetryWithBackoff(ctx, func() error {
    _, err := siaAPI.WorkspacesDB().AddDatabase(...)
    return err
})

// Convert to Terraform diagnostics
if err != nil {
    resp.Diagnostics.Append(client.MapError(err, "Failed to create database workspace")...)
    return
}
```

**Why**: Automatic exponential backoff (3 retries, 30s max delay), error classification, actionable user messages

### Read-Modify-Write for Policy Assignments

**Critical Pattern**: When updating policy assignments, ALWAYS fetch full policy first

```go
// CORRECT: Preserves other assignments
existingPolicy, err := siaAPI.AccessPolicies().GetAccessPolicy(policyID)
// Modify ONLY managed element
existingPolicy.Principals = append(existingPolicy.Principals, newPrincipal)
// Write back
updated, err := siaAPI.AccessPolicies().UpdatePolicy(policyID, existingPolicy)

// WRONG: Overwrites all other assignments
newPolicy := &models.Policy{Principals: []Principal{newPrincipal}}
updated, err := siaAPI.AccessPolicies().UpdatePolicy(policyID, newPolicy)
```

**Why**: API constraint - UpdatePolicy() accepts only ONE workspace type in Targets map per call. Must preserve unmanaged elements.

## Common Workflows

### Adding a New Resource

1. **Create Schema**: `internal/provider/<name>_resource.go`
   - Define schema with `schema.Schema{}`
   - Mark sensitive attributes: `Sensitive: true`
   - Use profile factory for authentication profiles

2. **Implement CRUD Methods**:
   - Create(): Use profile factory, wrap with RetryWithBackoff, convert errors with MapError
   - Read(): Handle 404 as deleted (drift detection)
   - Update(): Use Read-Modify-Write pattern if modifying shared resources
   - Delete(): Use `delete_workarounds.go` functions (see Critical Constraints)

3. **Add Tests**: `internal/provider/<name>_resource_test.go`
   - Acceptance tests with `TF_ACC=1`
   - Test CRUD lifecycle, ImportState, ForceNew behavior

4. **Create Examples**: `examples/resources/<name>/`
   - Basic usage example
   - Complete configuration example

5. **Generate Documentation**:
   ```bash
   tfplugindocs generate
   ```

6. **CRUD Validation**:
   - **Automated**: `make test-crud DESC=<resource-description>`
   - **Manual**: Follow `examples/testing/TESTING-GUIDE.md` for detailed workflow
   - Verify all validation checks pass (CREATE → READ → UPDATE → DELETE cycle)

### Fixing a Resource Bug

1. **Identify Affected Method**: Create/Read/Update/Delete
2. **Check Patterns**:
   - Using profile factory? (if auth-related)
   - Using delete workarounds? (if Delete method)
   - Using RetryWithBackoff? (if API calls)
   - Using Read-Modify-Write? (if policy updates)
3. **Verify SDK Mappings**: `docs/sdk-integration.md`
4. **Add/Update Test**: Reproduce bug in acceptance test
5. **CRUD Validation**: Run TESTING-GUIDE.md workflow

### Debugging Test Failures

1. **Enable Verbose Logging**:
   ```bash
   TF_LOG=DEBUG TF_ACC=1 go test ./... -v -run TestAccResourceName
   ```

2. **Common Issues**:
   - 401 Unauthorized → Check token refresh (see `docs/troubleshooting.md`)
   - 404 Not Found → Resource deleted externally (drift)
   - Nil pointer panic on Delete → Using SDK methods directly (use delete_workarounds.go)
   - Perpetual drift → Check profile pointer clearing in Read() method

3. **Consult References**:
   - API errors: `docs/troubleshooting.md`
   - SDK limitations: `docs/development/design-decisions.md`
   - Field mappings: `docs/sdk-integration.md`

## Critical Constraints

### ARK SDK v1.5.0 Limitations

1. **DELETE Panic Bug** ⚠️ **CRITICAL**
   - **Problem**: `DeleteDatabase()`, `DeleteSecret()`, `DeletePolicy()` pass nil body → panic in doRequest()
   - **Root Cause**: `pkg/common/ark_client.go:556-576` doesn't handle nil body pointer
   - **Solution**: Use `internal/client/delete_workarounds.go` functions:
     ```go
     // ✅ CORRECT
     err := client.DeleteDatabaseWorkspaceDirect(ctx, providerData.AuthContext, databaseID)
     err := client.DeleteSecretDirect(ctx, providerData.AuthContext, secretID)
     err := client.DeletePolicyDirect(ctx, providerData.AuthContext, policyID)

     // ❌ WRONG - Will panic
     err := siaAPI.WorkspacesDB().DeleteDatabase(databaseID)
     ```
   - **TODO**: Remove workaround when ARK SDK v1.6.0+ fixes nil body handling

2. **Azure VM Policy Serialization Bug** ⚠️ **CRITICAL**
   - **Problem**: Azure VM policies fail with HTTP 500/400 errors on create/update/read
   - **Root Cause**: SDK uses `"AZURE"` (uppercase) but API expects `"Azure"` (mixed case) for:
     - `targets.Azure` key (API returns/expects mixed case)
     - `metadata.policyEntitlement.locationType` value
     - `behavior.connectAs.ssh` wrapper (SDK produces `behavior.sshProfile`)
   - **Solution**: Use `internal/client/sdk_workarounds.go` functions:
     ```go
     // ✅ CORRECT - Use Azure workaround functions
     created, err = client.CreateAzureVMPolicyDirect(ctx, providerData.AuthContext, policy)
     policy, err = client.ReadAzureVMPolicyDirect(ctx, providerData.AuthContext, policyID)
     updated, err = client.UpdateAzureVMPolicyDirect(ctx, providerData.AuthContext, policyID, policy)

     // ❌ WRONG - Will fail with HTTP 500
     created, err = vmService.AddPolicy(policy)  // For Azure policies
     ```
   - **Detection**: Check `plan.LocationType.ValueString() == "Azure"` before API calls
   - **GitHub Issue**: https://github.com/cyberark/ark-sdk-golang/issues/32
   - **TODO**: Remove workaround when ARK SDK v1.6.0+ fixes Azure serialization

3. **No Context Support in Authenticate()**
   - Cannot cancel authentication mid-flight via context
   - First parameter is `*ArkProfile` (optional), NOT `context.Context`

4. **No Structured Errors**
   - SDK returns generic `error` interface with string messages
   - Use `internal/client.MapError()` for error classification

5. **15-Minute Token Expiration**
   - SDK handles automatic token refresh
   - In-memory profile pattern (stateless, container-friendly)

### Database Workspace Constraints

1. **All Cloud Providers Use "FQDN/IP" Target Set**
   - Don't create cloud-specific policy target logic
   - `cloud_provider` attribute is metadata only
   - Validated in ARK SDK: `choices:"FQDN/IP"` annotation

2. **secret_id is Functionally Required**
   - Schema: Optional (SDK allows it)
   - Reality: Required for ZSP/JIT access provisioning
   - Document this requirement in examples

### Cloud VM Policy Target Format Requirements

#### AWS (`aws_targets`)

1. **Account IDs** - 12-digit number:
   ```
   123456789012
   ```

2. **VPC IDs** - `vpc-` prefix followed by 8 or 17-character alphanumeric string:
   ```
   vpc-12345678
   vpc-1234567890abcdef0
   ```

#### Azure (`azure_targets`)

1. **Subscription IDs** - UUID format (32 alphanumeric chars in 5 groups with hyphens):
   ```
   759a039e-dc44-4762-9f40-2696323c2fa5
   ```

2. **Resource Groups** - Full ARM path format:
   ```
   /subscriptions/<subscription-id>/resourceGroups/<resource-group-name>
   ```
   Example: `/subscriptions/759a039e-dc44-4762-9f40-2696323c2fa5/resourceGroups/my-rg`

3. **VNet IDs** - Full ARM path format:
   ```
   /subscriptions/<subscription-id>/resourceGroups/<rg-name>/providers/Microsoft.Network/virtualNetworks/<vnet-name>
   ```
   Example: `/subscriptions/759a039e-dc44-4762-9f40-2696323c2fa5/resourceGroups/my-rg/providers/Microsoft.Network/virtualNetworks/my-vnet`

**Common Error**: Using just the resource group name (e.g., `my-rg`) instead of the full ARM path will result in API validation error: `invalid azure resource group`.

#### GCP (`gcp_targets`)

1. **Project IDs** - 3-60 characters, lowercase letters/numbers/hyphens, must start with letter, end with letter or number:
   ```
   my-project-123
   production-workloads
   ```

2. **VPC Network IDs** - Full VPC path format:
   ```
   projects/{project_id}/global/networks/{network_name}
   ```
   Example: `projects/my-project-123/global/networks/my-vpc`

### Policy Management Constraints

1. **UpdatePolicy() Accepts ONE Workspace Type Only**
   - Can't update database targets and VM targets in same call
   - Use Read-Modify-Write pattern to preserve unmanaged assignments

2. **Composite ID Formats**
   - Principal assignments: `policy-id:principal-id:principal-type` (3-part)
   - Database assignments: `policy-id:database-id` (2-part)
   - Parsing: Use `helpers/composite_ids.go` utilities

## Anti-Patterns (What NOT to Do)

❌ **Don't bypass profile factory** - Creates validation inconsistencies and 410 LOC duplication
❌ **Don't log sensitive data** - Passwords, tokens, password, aws_secret_access_key
❌ **Don't use SDK Delete methods directly** - Use `delete_workarounds.go` (prevents panics)
❌ **Don't assume cloud providers need different target sets** - All use "FQDN/IP"
❌ **Don't create ad-hoc test configs** - Use `examples/testing/TESTING-GUIDE.md` templates
❌ **Don't modify template files directly** - Copy to `/tmp` first, then customize
❌ **Don't skip Read-Modify-Write for policies** - Causes assignment overwrites
❌ **Don't assume SDK behavior** - Always verify in SDK source code

## Code Style

### Go Standards
- Follow standard Go conventions and idioms
- Use `gofmt` for formatting
- Run `golangci-lint` before commits
- Write godoc comments for exported functions

### Terraform Provider Patterns
- Use Terraform Plugin Framework v6
- Mark sensitive attributes with `Sensitive: true`
- Use `terraform-plugin-log/tflog` for structured logging
- **NEVER log sensitive data** (passwords, tokens, secrets)

### Error Handling
- Use `internal/client.MapError()` for Terraform diagnostics
- Wrap operations with `internal/client.RetryWithBackoff()`
- Classify errors by type (auth, permission, network, etc.)
- Provide actionable error messages with guidance

## Testing Strategy

**CANONICAL REFERENCE**: [`docs/development/TESTING-STRATEGY.md`](docs/development/TESTING-STRATEGY.md)

> **Philosophy**: "Tests should catch unique bugs, not test framework behavior."
> Prefer acceptance testing over unit testing. Each test must answer: "What bug does this catch that others don't?"

### Quick Summary

**Primary: Acceptance Tests** (Required for all resources)
- Test against real SIA API when `TF_ACC=1`
- Verify CRUD operations end-to-end
- Test ImportState functionality
- Test ForceNew behavior and drift detection
- Mock only when necessary (prefer real integration tests)

**Selective: Unit Tests** (Only for complex utilities)
- Complex helper functions (ID parsing, formatters)
- Error classification and retry logic
- Critical infrastructure code
- **NOT for**: Simple validators, SDK enums, framework behavior

**For complete testing philosophy, guidelines, and anti-patterns**, see [`docs/development/TESTING-STRATEGY.md`](docs/development/TESTING-STRATEGY.md)

### Acceptance Test Prerequisites

**Quick Prerequisites Check** before running `TF_ACC=1 go test ./...`:
- [ ] Environment variables: `CYBERARK_USERNAME`, `CYBERARK_PASSWORD`, `TF_ACC=1`
- [ ] Service account scopes: `sia`, `identity`
- [ ] CyberArk tenant with SIA enabled

**For complete prerequisites** (test data, cloud providers, troubleshooting), see `examples/testing/TESTING-GUIDE.md`

### CRUD Testing Standards

**CANONICAL REFERENCE**: `examples/testing/TESTING-GUIDE.md`

**ALL CRUD testing MUST follow** `examples/testing/TESTING-GUIDE.md`. This is the **single source of truth** for:
- Test configuration templates
- Testing workflow (CREATE → READ → UPDATE → DELETE)
- Validation checklists
- Resource dependency patterns
- Troubleshooting procedures

**Template Usage**:
1. Start from templates in `examples/testing/crud-test-*.tf`
2. Copy to `/tmp/sia-crud-validation-<timestamp>/`
3. Never modify templates directly
4. Update TESTING-GUIDE.md if resource behavior changes

**Testing Checklist (Before Committing)**:
- [ ] Run full CRUD cycle using TESTING-GUIDE.md workflow
- [ ] All validation checks pass (validation_summary outputs)
- [ ] Update TESTING-GUIDE.md if resource behavior changed
- [ ] Update template files if new dependencies added
- [ ] Document any new troubleshooting scenarios

## Commands

**Quick Reference**: Run `make help` to see all available commands

### Build & Install
```bash
make build                              # Build provider binary
make install                            # Install locally for Terraform development
make clean                              # Clean build artifacts
```

### Testing
```bash
make test                               # Run unit tests
make testacc                            # Run acceptance tests (requires TF_ACC=1)
make test-crud DESC=policy-assignment   # Run automated CRUD validation
```

### Code Quality
```bash
make fmt                                # Format Go code
make lint                               # Run golangci-lint
make generate                           # Generate provider documentation
```

### Validation (Local CI)
**Use these before committing to catch issues locally:**
```bash
make validate                           # Run ALL validations (recommended before commit)
make validate-go                        # Go: format + vet + golangci-lint
make validate-terraform                 # Terraform: format check for examples/
make validate-docs                      # Docs: verify tfplugindocs was run
make validate-security                  # Security: secrets detection + govulncheck
make check-all                          # Alias for 'validate'
```

### Pre-commit Hooks
```bash
make pre-commit-install                 # Install pre-commit hooks (one-time setup)
make pre-commit-run                     # Run pre-commit checks manually
```

### Development Setup
```bash
make tools-install                      # Install golangci-lint, tfplugindocs, govulncheck
make check-env                          # Verify environment variables are set
make deps                               # Download and tidy Go dependencies
```

### Recommended Workflow

**First-time setup:**
```bash
make tools-install                      # Install dev tools
make pre-commit-install                 # Enable automatic validation on commit
make validate                           # Verify everything works
```

**Before each commit:**
```bash
make validate                           # Run all checks locally (mirrors CI)
git commit -m "message"                 # Pre-commit hooks run automatically
```

### Manual Commands (Advanced)

If you need more control, use Go commands directly:

```bash
# Build
go build -v

# Run specific tests
go test ./internal/client/... -v
go test ./internal/provider -v -run TestAccResourceName

# Acceptance tests with verbose logging
TF_LOG=DEBUG TF_ACC=1 go test ./internal/provider -v -run TestAccResourceName

# Install to custom location
go install

# Dependencies
go mod tidy
go mod download
```

## Release & Distribution

### Version Management

**Versioning**: Follow [Semantic Versioning 2.0.0](https://semver.org/)
- **Major** (v1.0.0): Breaking changes (incompatible API changes)
- **Minor** (v0.X.0): New features, backward compatible
- **Patch** (v0.0.X): Bug fixes, backward compatible

**Pre-1.0 Status**: Currently v0.1.0 - breaking changes are acceptable before 1.0 release

### Release Checklist

Before creating a new release:

- [ ] All tests passing (`make test && make testacc`)
- [ ] CRUD validation complete for affected resources (`make test-crud DESC=...`)
- [ ] Code formatted and linted (`make fmt && make lint`)
- [ ] Documentation generated (`make generate` or `tfplugindocs generate`)
- [ ] CHANGELOG.md updated with release notes
- [ ] Version numbers bumped (if applicable)
- [ ] Git tag created: `git tag v0.X.X && git push origin v0.X.X`
- [ ] GitHub release created with release notes

### CHANGELOG.md Format

Follow [Keep a Changelog](https://keepachangelog.com/) format:

```
[0.X.X] - YYYY-MM-DD

Added:
- New features

Changed:
- Changes in existing functionality

Fixed:
- Bug fixes

Breaking Changes:
- Incompatible changes (pre-1.0 only)
```

### Future: Terraform Registry Publication

When ready for public distribution:

1. **Prerequisites**:
   - GitHub repository public
   - GPG key for signing releases
   - Terraform Registry account

2. **CI/CD Setup**:
   - GitHub Actions for automated builds
   - Automated testing on PRs
   - Release automation

3. **Registry Publishing**:
   - Follow [Terraform Registry publishing guide](https://www.terraform.io/docs/registry/providers/publishing.html)
   - Sign releases with GPG
   - Follow provider naming conventions

## Technical References

For detailed technical information, see:
- **Design Decisions**: `docs/development/design-decisions.md` - Active technologies, SDK limitations, breaking changes
- **SDK Integration**: `docs/sdk-integration.md` - ARK SDK patterns and field mappings
- **Troubleshooting**: `docs/troubleshooting.md` - Common issues and solutions
- **Development History**: `docs/development-history.md` - Complete development timeline and architectural decisions

## Known TODOs in Codebase

**Quick Scan**: `rg "TODO|FIXME" --glob "*.go"`

**Current Count**: 6 TODOs across 4 files (as of 2025-10-30)

### High-Priority TODOs

Track critical items as GitHub Issues for better visibility and prioritization:

| Priority | TODO | File | Blocked By | Notes |
|----------|------|------|------------|-------|
| **P1** | Remove delete_workarounds.go | `internal/client/delete_workarounds.go` | ARK SDK v1.6.0+ release | Critical workaround for nil body panic bug |
| **P1** | Remove Azure VM policy workarounds | `internal/client/sdk_workarounds.go` | ARK SDK v1.6.0+ release | Azure serialization bug (targets key + locationType casing) |
| **P2** | Add conditional validators for secret auth types | `internal/provider/secret_resource.go` | SDK field verification | Enforce required fields per auth type |
| **P3** | Add ParseAuthenticationProfile tests | `internal/provider/profile_factory_test.go` | - | Add tests for profile parsing and round-trip validation |

**Recommendation**: Create GitHub Issues for P1 and P2 TODOs to track SDK dependency and coordinate with upstream ARK SDK team.

**Detailed TODO Report** (with context):
```bash
rg "TODO|FIXME" --glob "*.go" -A 2 -B 1
```

## Active Technologies
- Not applicable (stateless provider, Terraform state managed externally) (001-vm-access-policies)

## Recent Changes
- 001-vm-access-policies: Added Go 1.25.0
