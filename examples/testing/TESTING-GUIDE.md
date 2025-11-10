# Comprehensive CRUD Testing Guide

> **STATUS**: ⚠️ **CANONICAL REFERENCE** ⚠️ (Mandatory)

This guide is the **single source of truth** for CRUD testing of the CyberArk SIA Terraform provider.

**Last Updated**: 2025-10-30 (Added automated CRUD testing via make test-crud + scripts/test-crud-resource.sh)

---

## ⚠️ CRITICAL WARNINGS - Read Before Testing

### 1. NEVER Use `timestamp()` on Azure Resources

**Problem**: Using `timestamp()` in Azure resource tags causes Terraform to recompute the value on every apply, triggering unnecessary resource updates that can take 1+ minute per resource.

```hcl
# ❌ BAD - Causes unnecessary updates on every apply
resource "azurerm_postgresql_flexible_server" "test" {
  tags = {
    created_at = timestamp()  # DON'T DO THIS!
  }
}

# ✅ GOOD - Static values only
resource "azurerm_postgresql_flexible_server" "test" {
  tags = {
    created_at  = "2025-10-28"  # Static date
    environment = "test"
  }
}
```

**Impact**: Each apply with `timestamp()` in tags:
- Triggers ~1 minute PostgreSQL tag update
- Adds 1-2 minutes total to every `terraform apply`
- Multiplies testing time by 2-3x

**Solution**: Use static strings for Azure resource tags. Only use `timestamp()` on **SIA resources** where it doesn't trigger cloud provider API calls.

### 2. CyberArk Cloud Directory Users Are `USER` Type, Not `ROLE`

**Problem**: CyberArk Cloud Directory users (format: `user@cyberark.cloud.XXXXX`) must use `principal_type = "USER"`, NOT `"ROLE"`.

```hcl
# ❌ BAD - ROLE is incorrect for Cloud Directory users
resource "cyberarksia_database_policy_principal_assignment" "example" {
  principal_id   = "tim@cyberark.cloud.40562"
  principal_type = "ROLE"  # WRONG!
}

# ✅ GOOD - USER is correct for Cloud Directory users
resource "cyberarksia_database_policy_principal_assignment" "example" {
  principal_id   = "tim@cyberark.cloud.40562"
  principal_type = "USER"  # CORRECT!
}
```

**When to use each type**:
- `"USER"` - CyberArk Cloud Directory users (`user@cyberark.cloud.XXXXX`)
- `"GROUP"` - Active Directory groups, Azure AD groups
- `"ROLE"` - CyberArk roles, service roles (NOT Cloud Directory users)

### 3. Database Policy `time_frame` is Optional

**Behavior**: If `time_frame` block is omitted, the policy **never expires** (valid indefinitely).

```hcl
# ✅ Policy never expires
resource "cyberarksia_database_policy" "permanent" {
  name   = "Never-Expires-Policy"
  status = "active"
  # No time_frame block = policy valid forever
}

# ✅ Policy expires on Dec 31, 2026
resource "cyberarksia_database_policy" "temporary" {
  name   = "Temporary-Policy"
  status = "active"

  time_frame {
    from_time = "2025-10-28T00:00:00Z"
    to_time   = "2026-12-31T23:59:59Z"
  }
}
```

### 4. ~~Known Limitation~~: `days_of_the_week` Order - FIXED ✅

**Status**: ✅ **RESOLVED** as of commit e7d8fa7 (2025-10-29)

**Previous Issue**: CyberArk API returned `days_of_the_week` in different order than configured, causing false positive drift detection.

**Fix Implemented**: Changed from `ListAttribute` to `SetAttribute` - days can now be specified in any order!

```hcl
# ✅ All of these are equivalent and will NOT trigger drift:
resource "cyberarksia_database_policy" "example" {
  conditions {
    access_window {
      days_of_the_week = [1, 2, 3, 4, 5]  # Ascending
      days_of_the_week = [5, 4, 3, 2, 1]  # Descending
      days_of_the_week = [5, 1, 3, 2, 4]  # Random order
      # All work identically - order doesn't matter!
      from_hour        = "09:00"
      to_hour          = "17:00"
    }
  }
}
```

**What Changed**:
- `days_of_the_week` is now a **Set** (unordered collection) instead of a List
- Terraform framework automatically normalizes order comparison
- No more "Provider produced inconsistent result" errors
- No `lifecycle { ignore_changes }` blocks needed
- HCL syntax remains identical: `[1,2,3,4,5]` works the same

**Benefits**:
- Users can specify days in any natural order
- No drift detection when API returns different order
- Cleaner code (-192 lines of workaround removed)
- Matches semantic meaning (days are a set, not a sequence)

---

## About This Guide

### Document Authority

- **Location**: `examples/testing/TESTING-GUIDE.md`
- **Referenced by**: `CLAUDE.md`, `docs/testing-framework.md`, all template files
- **Automation**: `scripts/test-crud-resource.sh`, `Makefile` (targets: `test-crud`, `check-env`)
- **Maintainers**: Must update when resource schemas or testing requirements change
- **Version**: See git history for changes

### When to Update This Guide

Update this guide when:
1. ✅ Adding a new resource type
2. ✅ Changing resource schemas or dependencies
3. ✅ Discovering new validation requirements
4. ✅ Adding new troubleshooting scenarios
5. ✅ Updating provider configuration requirements
6. ✅ Modifying automation scripts (`scripts/test-crud-resource.sh`, Makefile targets)

### When NOT to Follow This Guide

This guide is for **integration testing** (real API). For:
- **Unit testing**: See `internal/*/` test files
- **Acceptance testing**: See `tests/` (future)
- **CI/CD testing**: See `.github/workflows/` (future)

### Reporting Issues

If templates or procedures are outdated:
1. Create issue in GitHub repo
2. Update this guide first (it's the source of truth)
3. Then update templates to match

---

## Resources Tested

This test validates all CyberArk SIA Terraform provider resources:
1. **Certificate** - TLS/mTLS certificates
2. **Secret** - Database credentials
3. **Database Workspace** - Database connection configurations
4. **Database Policy** - Access policy metadata and conditions
5. **Database Policy Principal Assignment** - Assign users/groups/roles to policies
6. **Policy Database Assignment** - Assign databases to access policies

---

## Prerequisites

### Basic Testing (On-Premise/Mock Resources)
- ✅ CyberArk SIA tenant with UAP service provisioned
- ✅ Valid credentials (username + password) - exported as environment variables (see CLAUDE.md → Environment Setup)
- ✅ Provider built and installed (`make build && make install`)

### Cloud Provider Testing (Azure/AWS/GCP)
- ✅ Azure CLI authenticated (`az login`) - for Azure PostgreSQL testing
- ✅ Valid Azure subscription with PostgreSQL Flexible Server permissions
- ✅ AWS CLI configured (`aws configure`) - for AWS RDS testing (optional)
- ✅ GCP CLI authenticated (`gcloud auth login`) - for GCP Cloud SQL testing (optional)

### Policy Management Testing
- ✅ Test principal email addresses (for USER assignments)
- ✅ Azure AD directory ID (for USER/GROUP assignments)
- ✅ Service account credentials (for automated testing)

---

## Testing Approaches

This guide supports two testing workflows:

### ⚡ Automated Testing (Recommended)

**Use this for**: Fast CRUD validation during development

**Tool**: `scripts/test-crud-resource.sh` (automated via `make test-crud`)

**Time**: 5-10 minutes for full CREATE → READ → DELETE cycle

**What it does**:
- ✅ Automatically copies all templates to `/tmp/sia-crud-validation-{timestamp}/`
- ✅ Runs `terraform init`
- ✅ Executes CREATE test (`terraform apply`)
- ✅ Executes READ test (`terraform plan` with drift detection)
- ✅ Skips UPDATE test (requires manual modification)
- ✅ Executes DELETE test (`terraform destroy`)
- ✅ Provides detailed success/failure report

**Prerequisites**:
```bash
# Verify environment variables are set
make check-env

# Should confirm:
# ✅ CYBERARK_USERNAME
# ✅ CYBERARK_PASSWORD
# ⚠️  TF_ACC=1 (recommended)
```

**Usage**:
```bash
# From project root
cd ~/terraform-provider-cyberarksia

# Run automated CRUD test
make test-crud DESC=policy-principal-assignment

# Or run script directly
./scripts/test-crud-resource.sh "my-test-description"
```

**Output**:
```
╔════════════════════════════════════════════════════════════════════╗
║  CRUD Test Summary                                                  ║
╚════════════════════════════════════════════════════════════════════╝

  CREATE:  ✅ PASSED
  READ:    ✅ PASSED (no drift detected)
  UPDATE:  ⏭️  SKIPPED (manual modification required)
  DELETE:  ✅ PASSED

Test directory preserved: /tmp/sia-crud-validation-my-test-{timestamp}/
```

**When to use automated testing**:
- ✅ Quick validation during development
- ✅ Verifying bug fixes (CREATE/DELETE cycle)
- ✅ Pre-commit smoke testing
- ✅ CI/CD pipeline integration (future)

**When to use manual testing** (see below):
- 🔧 Testing UPDATE operations (requires template modification)
- 🔧 Testing specific cloud providers (Azure/AWS/GCP)
- 🔧 Complex policy assignment scenarios
- 🔧 Debugging drift detection issues

---

## 🔧 Manual Testing (Full Control)

**Use this for**: Detailed CRUD validation, UPDATE testing, cloud provider testing

**Time**: 15-30 minutes for full CREATE → READ → UPDATE → DELETE cycle

### 1. Setup Test Environment

```bash
# Create test directory
mkdir -p /tmp/sia-crud-validation
cd /tmp/sia-crud-validation

# Copy templates from project
cp ~/terraform-provider-cyberarksia/examples/testing/crud-test-*.tf .

# Generate test certificate
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out test-cert.pem \
  -days 365 -nodes -subj "/CN=crud-test-full/O=CRUDValidation/C=US"
```

### 2. Configure Provider

**Option A: Use Environment Variables (Recommended)**

Ensure environment variables are exported (see CLAUDE.md → Environment Setup):
```bash
export CYBERARK_USERNAME="your-username@cyberark.cloud.XXXX"
export CYBERARK_PASSWORD="<your-password-here>"
export TF_ACC=1  # For acceptance testing
```

Provider configuration (no hardcoded credentials):
```hcl
provider "cyberarksia" {
  # Credentials automatically read from environment variables
}
```

**Option B: Explicit Configuration (Alternative)**

Edit `crud-test-provider.tf`:
```hcl
provider "cyberarksia" {
  username      = "your-username@cyberark.cloud.XXXX"
  password = var.password  # Use variables, not hardcoded
}
```

### 3. Build and Install Provider

```bash
cd ~/terraform-provider-cyberarksia
make build
make install
```

### 4. Initialize Terraform

```bash
cd /tmp/sia-crud-validation
terraform init
```

### 5. Run Complete CRUD Test

```bash
# CREATE - Create all 4 resources
terraform apply -auto-approve

# READ - Verify state matches API
terraform plan  # Should show "No changes"

# UPDATE - Modify tags/labels in main.tf, then:
terraform apply -auto-approve

# DELETE - Clean up all resources
terraform destroy -auto-approve
```

---


---

## Resource-Specific Test Guides

For detailed testing procedures for specific resources, see the following guides:

### Database Resources

- **[Database Full CRUD Test with Azure PostgreSQL](resources/database-full-crud.md)**
  - Comprehensive workflow for all 6 SIA provider resources
  - Validates complete lifecycle with real Azure PostgreSQL
  - Duration: 20-30 minutes
  - Covers: certificate, secret, workspace, policy, and assignment resources

- **[Database Policy Management Testing](resources/database-policy.md)**
  - Focused testing for database policy and assignment resources
  - Principal assignment patterns
  - Workspace assignment patterns
  - Policy condition testing

### VM/Server Resources

- **[Target Set Resource Testing](resources/target-set.md)**
  - VM/server target set testing
  - Domain, suffix, and target matching patterns
  - Virtual machine secret integration

---

## Quick Navigation

**For automation**: Use `make test-crud DESC=<description>` (see above)

**For manual testing**: See resource-specific guides above

**For troubleshooting**: See [docs/troubleshooting.md](../../docs/troubleshooting.md)

**For development workflow**: See [CLAUDE.md](../../CLAUDE.md)
