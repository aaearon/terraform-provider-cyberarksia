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

**Reference**: See `docs/days-of-week-drift-fix.md` for complete implementation details.

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
cd ~/terraform-provider-cyberark-sia

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
cp ~/terraform-provider-cyberark-sia/examples/testing/crud-test-*.tf .

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
cd ~/terraform-provider-cyberark-sia
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

## Complete CRUD Test with Azure PostgreSQL

### Overview

This is the **comprehensive testing workflow** for all 6 SIA provider resources with a real Azure PostgreSQL flexible database. This workflow validates the complete lifecycle including infrastructure provisioning, policy management, and principal assignments.

**Test Scope**: ALL 6 resources + Azure infrastructure
**Duration**: 20-30 minutes
**Cost**: < $0.01 USD (Azure B1ms for ~15 minutes)

### Resources Validated
1. Azure PostgreSQL Flexible Server (B1ms)
2. `cyberarksia_certificate` - Azure SSL certificate
3. `cyberarksia_database_secret` - Database admin credentials
4. `cyberarksia_database_workspace` - Azure PostgreSQL configuration
5. `cyberarksia_database_policy` - Access policy with conditions
6. `cyberarksia_database_policy_principal_assignment` - User assignments
7. `cyberarksia_database_policy_workspace_assignment` - Database to policy assignment

### Phase 1: Setup (2 minutes)

```bash
# 1. Create timestamped working directory
export TEST_DIR="/tmp/sia-crud-validation-$(date +%Y%m%d-%H%M%S)"
mkdir -p $TEST_DIR
cd $TEST_DIR

# 2. Copy Azure PostgreSQL template
cp -r ~/terraform-provider-cyberark-sia/examples/testing/azure-postgresql/* .

# 3. Export environment variables (recommended)
export CYBERARK_USERNAME="your-username@cyberark.cloud.XXXX"
export CYBERARK_PASSWORD="<your-password-here>"
export TF_ACC=1

# Verify environment
cd ~/terraform-provider-cyberark-sia
make check-env

# Alternative: Create terraform.tfvars (if not using environment variables)
cat > terraform.tfvars <<EOF
sia_username              = "your-username@cyberark.cloud.XXXX"
sia_password         = "<your-password-here>"

# Azure settings
azure_subscription_id     = "YOUR_AZURE_SUBSCRIPTION_ID"
azure_region              = "westus2"

# PostgreSQL settings
postgres_admin_username   = "pgadmin"
postgres_admin_password   = "ChangeMe123!SecureP@ss"

# Test principals (UPDATE THESE)
test_principal_email      = "tim.schindler@cyberark.cloud.40562"
azure_ad_directory_id     = "YOUR_AZURE_AD_DIRECTORY_ID"
azure_ad_directory_name   = "AzureAD-Test"
EOF

# 4. Build and install provider
cd ~/terraform-provider-cyberark-sia
make build && make install

# 5. Initialize Terraform
cd $TEST_DIR
terraform init
```

**Validation**:
- [ ] Working directory created with timestamp
- [ ] Azure template files copied successfully
- [ ] `terraform.tfvars` created with credentials from `.env`
- [ ] Provider built without errors
- [ ] Terraform initialized successfully

### Phase 2: CREATE - Azure Infrastructure (5-10 minutes)

```bash
# Create Azure resources first
terraform apply -target=random_string.suffix -auto-approve
terraform apply -target=azurerm_resource_group.sia_test -auto-approve
terraform apply -target=azurerm_postgresql_flexible_server.sia_test -auto-approve
terraform apply -target=azurerm_postgresql_flexible_server_firewall_rule.allow_azure_services -auto-approve
terraform apply -target=azurerm_postgresql_flexible_server_firewall_rule.allow_all -auto-approve
terraform apply -target=azurerm_postgresql_flexible_server_database.testdb -auto-approve
```

**Validation**:
- [ ] Resource group created
- [ ] PostgreSQL server provisioned (B1ms, v16, 32GB)
- [ ] Firewall rules configured (Azure services + all IPs)
- [ ] Test database created
- [ ] Server FQDN available: `terraform output azure_postgres_fqdn`
- [ ] Public access enabled

**Common Issues**:
- **LocationIsOfferRestricted**: Change `azure_region` to "westus2" in terraform.tfvars
- **Slow provisioning**: PostgreSQL creation takes 5-10 minutes (normal)

### Phase 3: CREATE - SIA Certificate (< 1 minute)

```bash
terraform apply -target=cyberarksia_certificate.azure_cert -auto-approve
```

**Validation**:
- [ ] Certificate ID is numeric string
- [ ] `expiration_date` is ISO 8601 timestamp
- [ ] `metadata.issuer` contains "Microsoft RSA Root Certificate Authority 2017"
- [ ] `metadata.subject` populated correctly
- [ ] Labels saved: `environment=test`, `purpose=sia-azure-integration`

### Phase 4: CREATE - SIA Secret (< 1 minute)

```bash
terraform apply -target=cyberarksia_database_secret.admin -auto-approve
```

**Validation**:
- [ ] Secret ID is UUID format
- [ ] `created_at` timestamp populated
- [ ] `authentication_type` = "local"
- [ ] Username matches `postgres_admin_username` from tfvars
- [ ] Tags saved: `environment=test`, `managed_by=terraform`

### Phase 5: CREATE - SIA Database Workspace (< 1 minute)

```bash
terraform apply -target=cyberarksia_database_workspace.azure_postgres -auto-approve
```

**Validation**:
- [ ] Database ID is numeric
- [ ] `secret_id` matches created secret (from Phase 4)
- [ ] `certificate_id` matches created certificate (from Phase 3)
- [ ] `database_type` = "postgres-azure-managed"
- [ ] `cloud_provider` = "azure"
- [ ] `address` matches Azure PostgreSQL FQDN
- [ ] `port` = 5432
- [ ] `region` = "westus2"
- [ ] Tags saved correctly

### Phase 6: CREATE - SIA Database Policy with Inline Assignments (< 1 minute)

**IMPORTANT**: The SIA API requires at least ONE target database AND ONE principal when creating a policy. Use inline `target_database` and `principal` blocks to meet this requirement.

**Example Configuration** (`crud-test-policy.tf`):
```hcl
# ==============================================================================
# DATA SOURCE - Principal Lookup (RECOMMENDED Pattern - New in v0.2.0!)
# ==============================================================================
# No more hardcoded UUIDs! Look up principals by name.
data "cyberarksia_principal" "tim_user" {
  name = "tim.schindler@cyberark.cloud.40562"
  type = "USER"  # Optional: USER, GROUP, or ROLE
}

# ==============================================================================
# DATABASE POLICY with Inline Assignments
# ==============================================================================
resource "cyberarksia_database_policy" "test" {
  name                       = "CRUD-Test-Policy-${formatdate("YYYYMMDDhhmmss", timestamp())}"
  description                = "Comprehensive CRUD test policy for Azure PostgreSQL"
  status                     = "active"
  delegation_classification  = "unrestricted"
  time_zone                  = "GMT"

  conditions {
    max_session_duration = 4
    idle_time            = 10

    access_window {
      days_of_the_week = [1, 2, 3, 4, 5]  # Monday-Friday
      from_hour        = "09:00"
      to_hour          = "17:00"
    }
  }

  # ============================================================================
  # INLINE TARGET DATABASE - Required (at least 1)
  # ============================================================================
  # Azure PostgreSQL database assigned inline
  # Note: Singular block name matches AWS/GCP patterns (ingress/egress)

  target_database {
    database_workspace_id = cyberarksia_database_workspace.azure_postgres.id
    authentication_method = "db_auth"

    db_auth_profile {
      roles = ["pg_read_all_settings"]
    }
  }

  # ============================================================================
  # INLINE PRINCIPAL - Required (at least 1)
  # ============================================================================
  # RECOMMENDED: Use data source for principal lookup (NO hardcoded UUIDs!)
  # tim.schindler@cyberark.cloud.40562 assigned inline via data source
  # Note: Singular block name matches AWS/GCP patterns

  principal {
    principal_id          = data.cyberarksia_principal.tim_user.id
    principal_type        = data.cyberarksia_principal.tim_user.principal_type
    principal_name        = data.cyberarksia_principal.tim_user.name
    source_directory_name = data.cyberarksia_principal.tim_user.directory_name
    source_directory_id   = data.cyberarksia_principal.tim_user.directory_id
  }

  # ALTERNATIVE: Hardcoded UUIDs (not recommended, but shown for reference)
  # principal {
  #   principal_id          = "c2c7bcc6-9560-44e0-8dff-5be221cd37ee"
  #   principal_type        = "USER"
  #   principal_name        = "tim.schindler@cyberark.cloud.40562"
  #   source_directory_name = "CyberArk Cloud Directory"
  #   source_directory_id   = "09B9A9B0-6CE8-465F-AB03-65766D33B05E"
  # }

  policy_tags = ["test:crud", "environment:test", "managed-by:terraform"]

  depends_on = [cyberarksia_database_workspace.azure_postgres]
}
```

**Create Policy**:
```bash
terraform apply -target=cyberarksia_database_policy.test -auto-approve
```

**Validation**:
- [ ] Policy ID is UUID format
- [ ] Name: "CRUD-Test-Policy-[timestamp]"
- [ ] Status: "active"
- [ ] `delegation_classification` = "unrestricted"
- [ ] `time_zone` = "GMT"
- [ ] `conditions.max_session_duration` = 4 hours
- [ ] `conditions.idle_time` = 10 minutes
- [ ] `conditions.access_window` configured (Monday-Friday, 9am-5pm)
- [ ] **Inline target_database block** with Azure PostgreSQL database ID
- [ ] **Inline principal block** with tim.schindler user details
- [ ] `created_by` block populated (service account)
- [ ] `updated_on` timestamp present
- [ ] Policy appears in SIA UI with 1 target and 1 principal

**Verify in SIA UI**:
1. Navigate to policy: "CRUD-Test-Policy-[timestamp]"
2. Check "Assigned To" section → Should show tim.schindler@cyberark.cloud.40562
3. Check "Targets" section → Should show Azure PostgreSQL database
4. Verify authentication method = "db_auth" with roles = ["pg_read_all_settings"]

### Phase 7: (OPTIONAL) CREATE - Additional Principal via Assignment Resource (< 1 minute)

**Pattern**: This demonstrates the HYBRID pattern where the initial principal is inline (Phase 6), and additional principals are managed via separate assignment resources.

**Note**: To use this pattern, add `lifecycle { ignore_changes = [principal] }` to the policy resource to prevent drift detection.

**Example Configuration** (`crud-test-principal-assignment.tf`):
```hcl
# Policy resource from Phase 6 should include:
resource "cyberarksia_database_policy" "test" {
  # ... (policy configuration from Phase 6)

  lifecycle {
    ignore_changes = [principal]  # Allow assignment resources to manage principals
  }
}

# Additional principal via assignment resource
resource "cyberarksia_database_policy_principal_assignment" "additional_user" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  principal_id          = "another-uuid-here"
  principal_type        = "USER"
  principal_name        = "additional.user@example.com"
  source_directory_name = "CyberArk Cloud Directory"
  source_directory_id   = "09B9A9B0-6CE8-465F-AB03-65766D33B05E"
}
```

**Create Additional Principal** (if using hybrid pattern):
```bash
terraform apply -target=cyberarksia_database_policy_principal_assignment.additional_user -auto-approve
```

**Validation** (if using hybrid pattern):
- [ ] Additional principal created (composite ID format: `policy-id:principal-id:USER`)
- [ ] Uses `principal_type = "USER"` for CyberArk Cloud Directory users
- [ ] `source_directory_id` and `source_directory_name` populated
- [ ] Additional principal appears in SIA UI "Assigned To" section (total: 2 principals)
- [ ] No drift detected on policy resource (lifecycle.ignore_changes working)

### Phase 8: Inline Assignments Only - No Separate Database Assignment

**Note**: In the inline assignment pattern (Phase 6), the database is already assigned via the `target_database` block. No separate `cyberarksia_database_policy_workspace_assignment` resource is needed.

**If using hybrid pattern** (separate assignment resources for databases):
```hcl
# Policy resource must include lifecycle block
resource "cyberarksia_database_policy" "test" {
  # ... config

  lifecycle {
    ignore_changes = [principal, target_database]  # Delegate to assignment resources
  }
}

# Separate database assignment
resource "cyberarksia_database_policy_workspace_assignment" "additional_db" {
  policy_id             = cyberarksia_database_policy.test.policy_id
  database_workspace_id = cyberarksia_database_workspace.another_db.id
  authentication_method = "db_auth"

  db_auth_profile {
    roles = ["pg_read_all_data"]
  }
}
```

**Validation**:
- [ ] Policy targets include inline Azure PostgreSQL database
- [ ] If using hybrid: Additional database assignment created with composite ID
- [ ] All databases appear in SIA UI "Targets" section
- [ ] Uses "FQDN/IP" target set (confirmed for Azure/AWS/GCP/on-premise)

### Phase 9: READ - State Refresh (< 1 minute)

```bash
# Refresh state from API
terraform refresh

# Verify no changes detected
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] `terraform plan` shows 0 to add, 0 to change, 0 to destroy
- [ ] All computed fields populated correctly
- [ ] No drift detected between state and API
- [ ] All outputs display correctly: `terraform output`

### Phase 10: READ - Verify Complete Dependency Chain

```bash
# Review comprehensive outputs
terraform output validation_summary
```

**Validation**:
- [ ] Azure infrastructure: Server, database, firewall rules
- [ ] Certificate: ID, expiration, metadata
- [ ] Secret: ID, authentication type, timestamps
- [ ] Database workspace: Links to certificate + secret
- [ ] Policy: ID, status, conditions, timestamps
- [ ] Principal assignments: 2 users with directory info
- [ ] Database assignment: Policy-database link

**Check SIA UI**:
1. Navigate to policy: "CRUD-Test-Policy-[timestamp]"
2. Verify "Assigned To" section shows 2 principals
3. Verify "Targets" section shows Azure PostgreSQL database
4. Verify conditions match configuration

### Phase 11: UPDATE - Modify All Resources (2-3 minutes)

Edit the Terraform configuration to test UPDATE operations:

**Certificate Updates**:
```hcl
labels = {
  environment = "test"
  purpose     = "sia-azure-integration"
  updated     = "true"  # NEW
  updated_at  = formatdate("YYYY-MM-DD", timestamp())  # NEW
}
```

**Secret Updates**:
```hcl
tags = {
  environment = "test"
  managed_by  = "terraform"
  updated     = "true"  # NEW
}
```

**Database Workspace Updates**:
```hcl
tags = {
  environment = "test"
  purpose     = "crud-validation"
  updated     = "true"  # NEW
}
```

**Database Policy Updates**:
```hcl
description                = "Updated CRUD test policy - Azure PostgreSQL"  # CHANGED
conditions {
  max_session_duration = 8   # CHANGED from 4
  idle_time            = 30  # CHANGED from 10

  access_window {
    days_of_the_week = [1, 2, 3, 4, 5]
    from_hour        = "08:00"  # CHANGED from 09:00
    to_hour          = "18:00"  # CHANGED from 17:00
  }
}
```

**Principal Assignment Updates**:
```hcl
principal_name = "Test User - UPDATED NAME"  # CHANGED
```

**Database Assignment Updates**:
```hcl
db_auth_profile {
  roles = ["pg_read_all_settings", "pg_read_all_data"]  # ADDED role
}
```

Apply updates:
```bash
terraform apply
```

**Expected Output**: `Plan: 0 to add, 6 to change, 0 to destroy`

**Validation**:
- [ ] All 6 resource updates applied successfully
- [ ] Certificate labels updated
- [ ] Secret tags updated
- [ ] Database workspace tags updated
- [ ] Policy description and conditions changed
- [ ] Principal name updated in SIA UI
- [ ] Database assignment roles updated
- [ ] No forced replacements (in-place updates only)
- [ ] `terraform plan` shows no further changes

### Phase 12: IMPORT - Test Import Functionality (3-5 minutes)

Test import for each resource type:

```bash
# Get resource IDs from state
CERT_ID=$(terraform output -raw certificate_id)
SECRET_ID=$(terraform output -raw secret_id)
DB_ID=$(terraform output -raw database_workspace_id)
POLICY_ID=$(terraform output -raw policy_id)
PRINCIPAL_SERVICE_ID=$(terraform state show cyberarksia_database_policy_principal_assignment.service_account | grep "^id " | awk '{print $3}' | tr -d '"')
PRINCIPAL_USER_ID=$(terraform state show cyberarksia_database_policy_principal_assignment.test_user | grep "^id " | awk '{print $3}' | tr -d '"')
ASSIGNMENT_ID=$(terraform state show cyberarksia_database_policy_workspace_assignment.azure_postgres | grep "^id " | awk '{print $3}' | tr -d '"')

# Remove resources from state
terraform state rm cyberarksia_certificate.azure_cert
terraform state rm cyberarksia_database_secret.admin
terraform state rm cyberarksia_database_workspace.azure_postgres
terraform state rm cyberarksia_database_policy.test
terraform state rm cyberarksia_database_policy_principal_assignment.service_account
terraform state rm cyberarksia_database_policy_principal_assignment.test_user
terraform state rm cyberarksia_database_policy_workspace_assignment.azure_postgres

# Import each resource
terraform import cyberarksia_certificate.azure_cert "$CERT_ID"
terraform import cyberarksia_database_secret.admin "$SECRET_ID"
terraform import cyberarksia_database_workspace.azure_postgres "$DB_ID"
terraform import cyberarksia_database_policy.test "$POLICY_ID"
terraform import cyberarksia_database_policy_principal_assignment.service_account "$PRINCIPAL_SERVICE_ID"
terraform import cyberarksia_database_policy_principal_assignment.test_user "$PRINCIPAL_USER_ID"
terraform import cyberarksia_database_policy_workspace_assignment.azure_postgres "$ASSIGNMENT_ID"

# Verify no changes after import
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] All 7 imports succeeded
- [ ] Certificate: Imported with numeric ID
- [ ] Secret: Imported with UUID
- [ ] Database workspace: Imported with numeric ID
- [ ] Policy: Imported with UUID
- [ ] Principal assignments: Imported with 3-part composite IDs
- [ ] Database assignment: Imported with 2-part composite ID
- [ ] All attributes populated correctly after import
- [ ] No changes detected in `terraform plan`

### Phase 13: DELETE - Cleanup ⚠️ USER APPROVAL REQUIRED

**STOP**: Before proceeding, confirm you want to DELETE all test resources.

Delete in reverse dependency order:

```bash
# Delete assignment resources first
terraform destroy -target=cyberarksia_database_policy_workspace_assignment.azure_postgres -auto-approve
terraform destroy -target=cyberarksia_database_policy_principal_assignment.test_user -auto-approve
terraform destroy -target=cyberarksia_database_policy_principal_assignment.service_account -auto-approve

# Delete policy
terraform destroy -target=cyberarksia_database_policy.test -auto-approve

# Delete database workspace
terraform destroy -target=cyberarksia_database_workspace.azure_postgres -auto-approve

# Delete secret and certificate
terraform destroy -target=cyberarksia_database_secret.admin -auto-approve
terraform destroy -target=cyberarksia_certificate.azure_cert -auto-approve

# Delete Azure infrastructure
terraform destroy -auto-approve
```

**Validation**:
- [ ] All SIA assignments removed from policy
- [ ] Policy deleted from SIA UI
- [ ] Database workspace deleted
- [ ] Secret deleted
- [ ] Certificate deleted
- [ ] Azure PostgreSQL server deleted
- [ ] Azure resource group deleted
- [ ] No orphaned resources in SIA UI
- [ ] No orphaned Azure resources
- [ ] State is clean: `terraform state list` returns empty

**Cost Verification**:
```bash
# Verify Azure resources are deleted
az postgres flexible-server list --query "[?name=='psql-sia-test-*'].name"
# Should return: []
```

### Success Criteria

- ✅ All 6 SIA resources created successfully
- ✅ Azure PostgreSQL B1ms server provisioned and accessible
- ✅ Complete dependency chain validated (cert → secret → database → policy → principals → assignment)
- ✅ No schema validation errors
- ✅ No warnings about unknown attributes
- ✅ All computed fields populated correctly (timestamps, IDs, metadata)
- ✅ UPDATE operations work for all 6 resources (in-place, no forced replacements)
- ✅ IMPORT works with correct ID formats (numeric, UUID, composite)
- ✅ DELETE cleans up without errors (reverse dependency order)
- ✅ SIA UI reflects all changes correctly throughout lifecycle
- ✅ Policy assignment uses "FQDN/IP" target set (Azure cloud_provider confirmed)
- ✅ Principal assignments support multiple users with directory metadata
- ✅ Read-modify-write pattern preserves other principals/targets during updates
- ✅ Total cost < $0.01 USD

### Test Results Documentation

Save test results for future reference:

```bash
# Create test results file
cat > TEST-RESULTS-$(date +%Y%m%d-%H%M%S).md <<'EOF'
# Azure PostgreSQL CRUD Test Results

**Test Date**: $(date)
**Test Directory**: $TEST_DIR
**Duration**: [FILL IN] minutes
**Cost**: $[FILL IN] USD

## Resources Created
- Azure PostgreSQL Flexible Server: [SERVER_NAME]
- Certificate ID: [CERT_ID]
- Secret ID: [SECRET_ID]
- Database Workspace ID: [DB_ID]
- Policy ID: [POLICY_ID]
- Principal Assignments: 2 (service account + test user)
- Database Assignment ID: [ASSIGNMENT_ID]

## Test Phases
- [x] Phase 1: Setup
- [x] Phase 2: Azure Infrastructure
- [x] Phase 3: Certificate
- [x] Phase 4: Secret
- [x] Phase 5: Database Workspace
- [x] Phase 6: Policy
- [x] Phase 7: Principal Assignments
- [x] Phase 8: Database Assignment
- [x] Phase 9: READ - State Refresh
- [x] Phase 10: READ - Verify Dependencies
- [x] Phase 11: UPDATE - All Resources
- [x] Phase 12: IMPORT - All Resources
- [x] Phase 13: DELETE - Cleanup

## Validation Results
- All resources created successfully: YES/NO
- No drift detected: YES/NO
- UPDATE operations successful: YES/NO
- IMPORT operations successful: YES/NO
- DELETE operations successful: YES/NO
- SIA UI matches state: YES/NO

## Issues Encountered
[FILL IN]

## Notes
[FILL IN]
EOF
```

### Cleanup

```bash
# Remove test directory (optional - keep for review)
cd ~ && rm -rf $TEST_DIR
```

---

## Resource Dependencies

```
┌─────────────────────────┐
│ Data Source:            │
│ access_policy (lookup)  │◄────┐
└─────────────────────────┘     │
                                │
┌─────────────────────────┐     │
│ Resource:               │     │
│ certificate (TLS cert)  │     │
└──────────┬──────────────┘     │
           │                     │
           │  ┌─────────────────────────┐
           │  │ Resource:               │
           │  │ secret (DB credentials) │
           │  └──────────┬──────────────┘
           │             │
           │             │
           ▼             ▼
┌─────────────────────────────────────┐
│ Resource:                           │
│ database_workspace                  │
│ (references cert + secret)          │
└──────────┬──────────────────────────┘
           │
           │
           ▼
┌─────────────────────────────────────┐
│ Resource:                           │
│ policy_workspace_assignment          │
│ (assigns database to policy)        │
└─────────────────────────────────────┘
```

---

## Expected Output

After `terraform apply`, you should see:

```
Outputs:

validation_summary = {
  "assignment_created" = true
  "assignment_has_database" = true
  "assignment_has_policy" = true
  "certificate_created" = true
  "database_created" = true
  "database_has_certificate" = true
  "database_has_secret" = true
  "policy_found" = true
  "secret_created" = true
  "total_resources_created" = 4
}

test_completion_message = "✅ All 4 resources created successfully! Review validation_summary for dependency verification."
```

---

## Validation Checklists

### Certificate Resource
- [ ] Certificate ID is numeric string
- [ ] `expiration_date` is ISO 8601 timestamp
- [ ] `metadata` object populated (issuer, subject, valid_from, valid_to)
- [ ] Labels saved correctly

### Secret Resource
- [ ] Secret ID is UUID format
- [ ] `created_at` timestamp populated
- [ ] `authentication_type` matches input
- [ ] Tags saved correctly

### Database Workspace Resource
- [ ] Database ID is numeric
- [ ] `secret_id` matches created secret
- [ ] `certificate_id` matches created certificate
- [ ] `database_type` set correctly
- [ ] `cloud_provider` defaults to "on_premise"

### Policy Database Assignment Resource
- [ ] Composite ID format: `{policy_id}:{database_id}`
- [ ] `policy_id` matches policy data source
- [ ] `database_workspace_id` matches created database
- [ ] `authentication_method` set to "db_auth"
- [ ] `platform` computed from database workspace
- [ ] `db_auth_profile.roles` saved correctly

### Data Source
- [ ] Policy found by name
- [ ] Policy ID retrieved
- [ ] Policy status shown

---

## Troubleshooting

### Error: Policy Not Found
**Symptom**: `Error: Policy not found: Terraform-Test-Policy`

**Solution**: Ensure "Terraform-Test-Policy" exists in your tenant, or modify the policy name in `main.tf`:
```hcl
data "cyberarksia_database_policy" "test_policy" {
  name = "Your-Policy-Name-Here"
}
```

### Error: No UAP Service
**Symptom**: DNS lookup failures or "service not available" errors

**Solution**: Verify tenant has UAP service provisioned:
```bash
curl -s "https://platform-discovery.cyberark.cloud/api/v2/services/subdomain/{your-tenant}" | jq '.jit // .dpa'
```

If UAP/JIT/DPA is not in the response, contact CyberArk support to provision the service.

### Error: Provider Binary Not Found
**Symptom**: `Error: Failed to query available provider packages`

**Solution**: Rebuild and reinstall provider:
```bash
cd ~/terraform-provider-cyberark-sia
make clean && make build && make install
```

### Error: Schema Validation Failed
**Symptom**: `Error: Missing required argument` or `Unsupported argument`

**Solution**: Reinitialize Terraform:
```bash
rm -rf .terraform .terraform.lock.hcl
terraform init
```

### Error: Lock File Checksum Mismatch
**Symptom**: `cached package does not match any of the checksums recorded`

**Solution**:
```bash
rm .terraform.lock.hcl
terraform init
```

---

## UPDATE Testing

To test UPDATE operations, modify `main.tf`:

### Certificate Updates
```hcl
cert_description = "CRUD validation test certificate - UPDATED"
labels = {
  environment = "test"
  purpose     = "crud-validation"
  suite       = "full"
  created_at  = formatdate("YYYY-MM-DD", timestamp())
  updated     = "true"  # NEW
}
```

### Secret Updates
```hcl
tags = {
  environment = "test"
  purpose     = "crud-validation"
  suite       = "full"
  updated     = "true"  # NEW
}
```

### Database Workspace Updates
```hcl
tags = {
  environment = "test"
  purpose     = "crud-validation"
  suite       = "full"
  updated     = "true"  # NEW
}
```

### Policy Assignment Updates
```hcl
db_auth_profile {
  roles = ["crud_test_admin", "crud_test_auditor"]  # CHANGED
}
```

Then apply:
```bash
terraform apply -auto-approve
```

**Expected result**: `0 to add, 4 to change, 0 to destroy`

---

## Clean Up

```bash
# Remove all test resources
terraform destroy -auto-approve

# Remove test directory (optional)
cd ~ && rm -rf /tmp/sia-crud-validation
```

---

## Files in Test Directory

After setup, your test directory should contain:

```
/tmp/sia-crud-validation/
├── provider.tf       # Provider configuration (from template)
├── main.tf           # All 4 resources + policy data source (from template)
├── outputs.tf        # Comprehensive validation outputs (from template)
├── test-cert.pem     # Generated test certificate
└── key.pem           # Certificate private key (not used by provider)
```

---

## Best Practices

### 1. Always Use Templates
**DO NOT** create ad-hoc test configurations. Always start from:
- `examples/testing/crud-test-provider.tf`
- `examples/testing/crud-test-main.tf`
- `examples/testing/crud-test-outputs.tf`
- `examples/testing/azure-postgresql-with-policy/` - Complete policy workflow (recommended)

### 2. Use Context-Efficient Testing Workflows
**IMPORTANT**: Use automation scripts to minimize LLM context consumption.

```bash
# Run tests from project directory, not /tmp!
cd ~/terraform-provider-cyberark-sia/examples/testing/azure-postgresql-with-policy
./setup.sh   # All terraform output → /tmp/ logs (98% context savings!)
```

**See `examples/testing/CONTEXT-OPTIMIZATION.md` for detailed patterns and best practices.**

Key principles:
- ✅ Test configurations live in project (`examples/testing/`)
- ✅ Only logs go to `/tmp/` (not test configs)
- ✅ Use automation scripts for efficient logging
- ✅ Extract only relevant information (summaries, errors)

### 3. Never Modify Templates Directly
Templates in `examples/testing/` are canonical references. Copy to working directory first (or run in place).

### 4. Always Rebuild After Code Changes
```bash
cd ~/terraform-provider-cyberark-sia
make build && make install
```

### 5. Reinitialize After Provider Updates
```bash
# In your test directory
rm -rf .terraform .terraform.lock.hcl
terraform init
```

### 6. Verify Outputs After Each Operation
```bash
terraform output validation_summary
```

---

## Cloud Provider Integration Testing

### Overview

Testing with cloud-managed databases (AWS RDS, Azure PostgreSQL, GCP Cloud SQL) requires additional setup for cloud provider authentication and resource provisioning.

**RECOMMENDED WORKFLOW**: See [Complete CRUD Test with Azure PostgreSQL](#complete-crud-test-with-azure-postgresql) section above for the comprehensive 13-phase testing workflow that covers ALL 6 resources with a real Azure database.

### Key Finding: Database Target Sets

**CRITICAL**: ALL database workspaces use `"FQDN/IP"` target set in policies, **regardless of cloud_provider attribute**.

| Cloud Provider | Target Set (Actual) |
|----------------|---------------------|
| on_premise | FQDN/IP |
| aws | FQDN/IP |
| azure | FQDN/IP |
| gcp | FQDN/IP |
| atlas | FQDN/IP |

**Impact**: The `cloud_provider` field is **metadata only** for database workspaces. Policy assignment always uses `"FQDN/IP"` target set.

### Azure PostgreSQL Testing

**Canonical Test Templates**:
1. **Basic CRUD**: `examples/testing/azure-postgresql/` - Database workspace onboarding only
2. **Complete Policy Workflow**: `examples/testing/azure-postgresql-with-policy/` - Database workspace + policy + principal assignments (NEW!)

#### Option 1: Basic Database Workspace CRUD Test

**Directory**: `examples/testing/azure-postgresql/`

This configuration validates Azure PostgreSQL database workspace creation and assignment to an existing policy.

```bash
cd examples/testing/azure-postgresql
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your credentials
terraform init && terraform apply
# Verify success, then clean up
terraform destroy
```

#### Option 2: Complete Policy Workflow (Recommended for Full Testing)

**Directory**: `examples/testing/azure-postgresql-with-policy/`

This configuration creates a complete testing environment including:
- Azure PostgreSQL Flexible Server (B1ms)
- SIA secret and database workspace (**certificate validation disabled**)
- **Database policy** (with inline service account principal + database target)
- **Principal assignment** (Tim Schindler via separate resource)

**Quick Start**:
```bash
cd ~/terraform-provider-cyberark-sia/examples/testing/azure-postgresql-with-policy
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars  # Fill in credentials and principal UUIDs
./setup.sh  # Automated setup with context-efficient logging!
```

**Features**:
- ✅ Automated setup/cleanup scripts (98% context savings!)
- ✅ Creates database policy (not just assigns to existing)
- ✅ Manages principal assignments
- ✅ No `timestamp()` functions (avoids unnecessary updates)
- ✅ All logs to `/tmp/` (see `CONTEXT-OPTIMIZATION.md`)

See `examples/testing/azure-postgresql-with-policy/QUICK-START.md` for detailed instructions.

#### Prerequisites
1. Azure CLI authentication: `az login`
2. Valid subscription with unrestricted regions
3. SIA credentials in project root `.env` file
4. Existing SIA policy with `locationType: "FQDN/IP"`

#### Recommended Configuration
```hcl
# Azure PostgreSQL Flexible Server (B1ms - cheapest option)
resource "azurerm_postgresql_flexible_server" "test" {
  name                = "psql-sia-test-${random_string.suffix.result}"
  resource_group_name = azurerm_resource_group.test.name
  location            = "westus2"  # Check subscription restrictions

  sku_name   = "B_Standard_B1ms"  # ~$0.017/hour
  storage_mb = 32768              # 32 GB minimum
  version    = "16"

  administrator_login    = var.admin_username
  administrator_password = var.admin_password

  public_network_access_enabled = true  # Required for SIA connectivity
}

# SIA Database Workspace
resource "cyberarksia_database_workspace" "azure_postgres" {
  name                          = "azure-postgres-test-${random_string.suffix.result}"
  database_type                 = "postgres-azure-managed"
  cloud_provider                = "azure"  # Metadata only
  region                        = var.azure_region
  address                       = azurerm_postgresql_flexible_server.test.fqdn
  port                          = 5432
  secret_id                     = cyberarksia_database_secret.admin.id
  enable_certificate_validation = false  # Simplify testing
  certificate_id                = cyberarksia_certificate.azure_cert.id
}

# Policy Assignment (uses "FQDN/IP" target set for ALL databases)
resource "cyberarksia_database_policy_workspace_assignment" "azure_postgres" {
  policy_id              = data.cyberarksia_database_policy.test_policy.id
  database_workspace_id  = cyberarksia_database_workspace.azure_postgres.id
  authentication_method  = "db_auth"

  db_auth_profile {
    roles = ["pg_read_all_settings"]
  }
}
```

#### Cost Estimate
- **Hourly**: $0.017 USD (B1ms compute)
- **Test Duration**: ~15-30 minutes
- **Test Cost**: < $0.01 USD

#### Common Issues

**Issue**: LocationIsOfferRestricted
```
Error: Subscriptions are restricted from provisioning in location 'eastus'
```
**Solution**: Change region to `westus2` or check `az account list-locations` for allowed regions.

**Issue**: Firewall Rule Already Exists
**Solution**: Import existing rule:
```bash
terraform import azurerm_postgresql_flexible_server_firewall_rule.allow_azure \
  "/subscriptions/.../firewallRules/AllowAzureServices"
```

#### Azure Certificate

Azure PostgreSQL uses **Microsoft RSA Root Certificate Authority 2017**:

```hcl
resource "cyberarksia_certificate" "azure_cert" {
  cert_name = "azure-postgres-ssl-cert"
  cert_type = "PEM"
  cert_body = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIIFqDCCA5CgAwIBAgIQHtOXCV/YtLNHcB6qvn9FszANBgkqhkiG9w0BAQwFADBl
    [... full certificate content ...]
    -----END CERTIFICATE-----
  EOT
}
```

**Download**: https://www.microsoft.com/pkiops/certs/Microsoft%20RSA%20Root%20Certificate%20Authority%202017.crt

### AWS RDS Testing (Template)

**Prerequisites**:
1. AWS CLI authentication: `aws configure`
2. Valid AWS credentials with RDS permissions

**Recommended Configuration**:
- RDS Instance Class: `db.t3.micro` (cheapest)
- Storage: 20 GB minimum
- Public accessibility: Yes (for testing)
- Certificate: AWS RDS CA bundle

### GCP Cloud SQL Testing (Template)

**Prerequisites**:
1. gcloud CLI authentication: `gcloud auth login`
2. Valid GCP project with Cloud SQL API enabled

**Recommended Configuration**:
- Tier: `db-f1-micro` (cheapest)
- Storage: 10 GB minimum
- Public IP: Yes (for testing)
- Certificate: GCP server CA

### Testing Workflow

1. **Setup Cloud Resources** (5-10 minutes)
   ```bash
   terraform apply -target=azurerm_postgresql_flexible_server.test
   terraform apply -target=azurerm_postgresql_flexible_server_database.testdb
   ```

2. **Create SIA Resources** (< 1 minute)
   ```bash
   terraform apply -target=cyberarksia_certificate.azure_cert
   terraform apply -target=cyberarksia_database_secret.admin
   terraform apply -target=cyberarksia_database_workspace.azure_postgres
   ```

3. **Test Policy Assignment** (< 1 minute)
   ```bash
   terraform apply -target=cyberarksia_database_policy_workspace_assignment.azure_postgres
   ```

4. **Verify in SIA UI**
   - Navigate to policy: Terraform-Test-Policy
   - Confirm database appears in targets
   - Verify authentication method and profile

5. **Cleanup** (2-5 minutes)
   ```bash
   terraform destroy -auto-approve
   ```

### Cost Management

**Best Practices**:
- Use smallest instance sizes (B1ms, db.t3.micro, db-f1-micro)
- Add `auto_delete = "true"` tag to all resources
- Set up automatic cleanup GitHub Actions
- Stop/delete resources immediately after testing

**Estimated Costs per Test**:
- Azure PostgreSQL B1ms: < $0.01 USD (15 min test)
- AWS RDS db.t3.micro: < $0.01 USD (15 min test)
- GCP Cloud SQL db-f1-micro: < $0.01 USD (15 min test)

---

## Database Policy Management Testing

### cyberarksia_database_policy Resource

**Template**: [`crud-test-policy.tf`](./crud-test-policy.tf)

**Testing Workflow** (15-20 minutes):

1. **CREATE** - Create policy with conditions
   ```bash
   terraform apply
   # Verify: Policy appears in SIA UI with correct metadata and conditions
   ```

2. **READ** - Validate state matches API
   ```bash
   terraform refresh
   terraform plan
   # Expected: No changes detected
   ```

3. **UPDATE** - Modify policy attributes
   ```bash
   # Edit: Change description, status (Active ↔ Suspended), conditions
   terraform apply
   # Verify: Changes reflected in SIA UI
   ```

4. **DELETE** - Remove policy
   ```bash
   terraform destroy
   # Verify: Policy removed from SIA UI (cascade deletes principals/targets)
   ```

**Validation Summary Outputs**:
- `policy_metadata` - Policy ID, name, status
- `policy_conditions` - Session duration, idle time, access window
- `computed_fields` - Created by, updated on timestamps

### cyberarksia_database_policy_principal_assignment Resource

**Template**: [`crud-test-principal-assignment.tf`](./crud-test-principal-assignment.tf)

**Prerequisites**:
1. Existing policy (created via `cyberarksia_database_policy` or SIA UI)
2. Valid identity directory (Azure AD, LDAP) with test principals
3. Principal IDs for USER/GROUP types
4. Directory name and ID for USER/GROUP assignments

**Testing Workflow** (10-15 minutes):

1. **CREATE** - Assign principals to policy
   ```bash
   terraform apply
   # Verify: Principals appear in policy's "Assigned To" section in SIA UI
   # Test: USER (with directory), GROUP (with directory), ROLE (no directory)
   ```

2. **READ** - Validate state matches API
   ```bash
   terraform refresh
   terraform plan
   # Expected: No changes detected
   # Verify: Composite ID format: policy-id:principal-id:principal-type
   ```

3. **UPDATE** - Modify principal name (in-place)
   ```bash
   # Edit: Change principal_name
   terraform apply
   # Verify: Updated name appears in SIA UI
   # Note: policy_id, principal_id, principal_type are ForceNew
   ```

4. **DELETE** - Remove principal assignment
   ```bash
   terraform destroy
   # Verify: Principal removed from policy in SIA UI
   # Verify: Other principals remain (read-modify-write pattern)
   ```

**Composite ID Testing**:
```bash
# Test 3-part format for all principal types
terraform import cyberarksia_database_policy_principal_assignment.user_test \
  "policy-id:user@example.com:USER"

terraform import cyberarksia_database_policy_principal_assignment.group_test \
  "policy-id:group@example.com:GROUP"

terraform import cyberarksia_database_policy_principal_assignment.role_test \
  "policy-id:role-name:ROLE"
```

**Validation Tests**:
- ✅ USER/GROUP require `source_directory_name` + `source_directory_id`
- ✅ ROLE works without directory fields
- ✅ Duplicate principal detection (same ID + type on policy fails)
- ✅ Read-modify-write preserves other principals
- ✅ Principal type validation (USER, GROUP, ROLE only)

**Validation Summary Outputs**:
- `assignment_ids` - Composite IDs for all principals
- `principal_types` - Count by type (USER, GROUP, ROLE)
- `directory_sources` - Azure AD, LDAP counts

### Integration Testing: Full Policy Lifecycle

**RECOMMENDED**: Use the [Complete CRUD Test with Azure PostgreSQL](#complete-crud-test-with-azure-postgresql) workflow above, which includes:
- Database policy creation with conditions
- Principal assignments (service account + test user)
- Database assignments (Azure PostgreSQL)
- Full UPDATE/IMPORT/DELETE testing
- Real-world Azure integration

**Alternative: Template-Based Testing** (without cloud resources):

**Template**: [`crud-test-full-lifecycle.tf`](./crud-test-full-lifecycle.tf) *(to be created)*

**Comprehensive workflow** (20-30 minutes):

```bash
# 1. Create policy with metadata and conditions
terraform apply -target=cyberarksia_database_policy.test

# 2. Assign principals (users, groups, roles)
terraform apply -target=cyberarksia_database_policy_principal_assignment.users
terraform apply -target=cyberarksia_database_policy_principal_assignment.groups
terraform apply -target=cyberarksia_database_policy_principal_assignment.roles

# 3. Assign database workspaces to policy
terraform apply -target=cyberarksia_database_policy_assignment.databases

# 4. Verify complete access chain
# - Policy exists with correct conditions
# - Principals assigned (check SIA UI "Assigned To")
# - Databases assigned (check SIA UI "Targets")
# - Test database access via SIA portal

# 5. Update policy conditions (preserve principals/targets)
# Edit: Change max_session_duration, idle_time, access_window
terraform apply

# 6. Add/remove principals independently
terraform apply -target=cyberarksia_database_policy_principal_assignment.new_user

# 7. Suspend policy (should block access)
# Edit: status = "suspended"
terraform apply
# Verify: Access blocked in SIA portal

# 8. Reactivate policy
# Edit: status = "active"
terraform apply
# Verify: Access restored in SIA portal

# 9. Full cleanup
terraform destroy -auto-approve
```

**Success Criteria**:
- ✅ Policy CRUD operations work independently
- ✅ Principal assignments work independently
- ✅ Database assignments work independently
- ✅ Update policy preserves principals and targets (read-modify-write)
- ✅ Policy status changes (Active ↔ Suspended) affect access
- ✅ Policy deletion cascades to principals and targets
- ✅ Import works for all resource types

---

## Production-Ready Testing Checklist

Use this checklist before committing resource changes or releasing new provider versions.

### Pre-Test Preparation

**Environment**:
- [ ] `.env` file exists with valid `CYBERARK_USERNAME` and `CYBERARK_PASSWORD`
- [ ] Azure CLI authenticated: `az login && az account show`
- [ ] Provider built and installed: `cd ~/terraform-provider-cyberark-sia && make build && make install`
- [ ] Clean working directory: `/tmp/sia-crud-validation-$(date +%Y%m%d-%H%M%S)`

**Credentials**:
- [ ] SIA service account username (from `.env`)
- [ ] SIA client secret (from `.env`)
- [ ] Test principal email addresses (for USER assignments)
- [ ] Azure AD directory ID (for directory-based principals)
- [ ] Azure subscription ID (for cloud testing)

**Prerequisites Verified**:
- [ ] UAP service provisioned on tenant: `curl -s "https://platform-discovery.cyberark.cloud/api/v2/services/subdomain/{tenant}" | jq '.jit'`
- [ ] Azure region allowed: `westus2` recommended (check subscription restrictions)
- [ ] PostgreSQL admin credentials prepared (strong password required)

### During Test Validation

**CREATE Phase**:
- [ ] All resources created without errors
- [ ] No schema validation warnings
- [ ] All computed fields populated (IDs, timestamps, metadata)
- [ ] SIA UI shows all resources correctly
- [ ] Azure infrastructure provisioned (if testing with cloud resources)

**READ Phase**:
- [ ] `terraform refresh` succeeds without changes
- [ ] `terraform plan` shows "No changes"
- [ ] All outputs display expected values
- [ ] State file matches API responses
- [ ] No drift detected

**UPDATE Phase**:
- [ ] All UPDATE operations successful (in-place, no forced replacements)
- [ ] Changes reflected in SIA UI
- [ ] `terraform plan` after updates shows no further changes
- [ ] Read-modify-write preserves other resources (principals, targets)
- [ ] Timestamps updated (`last_modified`, `updated_on`)

**IMPORT Phase**:
- [ ] All imports succeed with correct ID formats:
  - Certificate: numeric ID
  - Secret: UUID
  - Database workspace: numeric ID
  - Policy: UUID
  - Principal assignment: 3-part composite (`policy:principal:type`)
  - Database assignment: 2-part composite (`policy:database`)
- [ ] `terraform plan` after import shows no changes
- [ ] All attributes populated correctly

**DELETE Phase** (requires user approval):
- [ ] Delete in reverse dependency order
- [ ] All resources removed from SIA UI
- [ ] No orphaned resources
- [ ] Azure resources deleted (if applicable)
- [ ] State is clean: `terraform state list` returns empty
- [ ] Cost verification: Azure resources confirmed deleted

### Post-Test Verification

**SIA UI Checks**:
- [ ] No orphaned certificates
- [ ] No orphaned secrets
- [ ] No orphaned database workspaces
- [ ] No orphaned policies
- [ ] No orphaned principal assignments
- [ ] No orphaned database assignments

**Azure Cost Verification** (if applicable):
- [ ] Resource group deleted: `az group list --query "[?name contains 'sia-test']"`
- [ ] PostgreSQL server deleted: `az postgres flexible-server list --query "[?name contains 'sia-test']"`
- [ ] Total test cost < $0.01 USD
- [ ] No ongoing charges

**Documentation**:
- [ ] Test results saved to `TEST-RESULTS-$(date).md`
- [ ] Any issues documented with reproduction steps
- [ ] Success criteria met (see checklist in Azure CRUD workflow)
- [ ] Working directory preserved for review (or cleaned up)

### Resource-Specific Validation

**Certificate Resource**:
- [ ] `expiration_date` is valid ISO 8601 timestamp
- [ ] `metadata` object complete (issuer, subject, valid_from, valid_to, serial_number)
- [ ] Labels/tags saved correctly
- [ ] No warnings about unknown attributes

**Secret Resource**:
- [ ] `created_at` timestamp populated
- [ ] `authentication_type` matches input (local, domain, aws_iam)
- [ ] Password marked as sensitive in state
- [ ] Username stored correctly

**Database Workspace Resource**:
- [ ] `secret_id` required and links correctly
- [ ] `certificate_id` optional but links correctly if provided
- [ ] `database_type` validated (60+ engine types supported)
- [ ] `cloud_provider` metadata saved correctly
- [ ] `address` and `port` correct

**Database Policy Resource**:
- [ ] `policy_id` is UUID
- [ ] `created_by` block populated
- [ ] `updated_on` timestamp present
- [ ] Conditions saved correctly (`max_session_duration`, `idle_time`, `access_window`)
- [ ] Policy appears in SIA UI with correct metadata

**Database Policy Principal Assignment Resource**:
- [ ] Composite ID format: `policy-id:principal-id:type` (3 parts)
- [ ] USER/GROUP require `source_directory_name` + `source_directory_id`
- [ ] ROLE works without directory fields
- [ ] Duplicate principal detection works
- [ ] Read-modify-write preserves other principals

**Policy Database Assignment Resource**:
- [ ] Composite ID format: `policy-id:database-id` (2 parts)
- [ ] `authentication_method` validated (6 methods supported)
- [ ] Profile type matches authentication method
- [ ] Uses "FQDN/IP" target set regardless of `cloud_provider`
- [ ] Read-modify-write preserves other database assignments

### Troubleshooting Reference

**Quick Diagnostics**:
0. **Start with automated testing**: `make check-env && make test-crud DESC=test` - validates environment and runs basic CRUD cycle

**Common Issues**:
1. **Provider binary not found**: `make build && make install` or `make build && make install`
2. **Schema validation failed**: `rm -rf .terraform .terraform.lock.hcl && terraform init`
3. **UAP service not available**: Verify tenant provisioning, may need to contact CyberArk support
4. **Azure location restricted**: Change `azure_region` to "westus2"
5. **Terraform state drift**: Run `terraform refresh` then `terraform plan`
6. **Environment variables missing**: Run `make check-env` to verify `CYBERARK_USERNAME` and `CYBERARK_PASSWORD`

**Error Patterns**:
- **"Policy not found"**: Verify policy name or create new test policy
- **"Certificate in use"**: Remove database workspace reference before deleting certificate
- **"Invalid composite ID"**: Check ID format (2-part vs 3-part, correct delimiters)
- **"Directory required for USER/GROUP"**: Add `source_directory_name` and `source_directory_id`

### Success Criteria Summary

For a test to be considered successful, ALL of the following must be true:
- ✅ All resources created without errors
- ✅ No schema validation warnings
- ✅ All computed fields populated correctly
- ✅ UPDATE operations work (in-place, no forced replacements)
- ✅ IMPORT works with correct ID formats
- ✅ DELETE cleans up without errors
- ✅ SIA UI reflects all changes correctly
- ✅ State matches API throughout lifecycle
- ✅ No orphaned resources after cleanup
- ✅ Azure costs < $0.01 USD (if applicable)

---

## Target Set Resource Testing

### Overview

Target sets define logical groupings of virtual machines and servers that share common access credentials for Just-In-Time (JIT) privileged access. This section covers CRUD testing for the `cyberarksia_target_set` resource.

**Template**: [`crud-test-target-set.tf`](./crud-test-target-set.tf)

**Test Scope**: Target set lifecycle with all matching patterns (Domain, Suffix, Target)
**Duration**: 10-15 minutes
**Prerequisites**: Existing VM secret resource

### Resources Validated
1. `cyberarksia_virtual_machine_secret` - VM credentials (ProvisionerUser or PCloudAccount)
2. `cyberarksia_target_set` - VM/server target set with matching patterns

### Key Features Tested

**Matching Patterns**:
- **Domain**: Matches all servers in a domain (e.g., `*.example.com`)
- **Suffix**: Matches servers with hostname suffix (e.g., `*.dc1.example.com`)
- **Target**: Matches specific server hostname

**Critical Behaviors**:
- ✅ Type changes (Domain ↔ Suffix ↔ Target) without resource recreation
- ✅ provision_format clearing prevention (plan-time error)
- ✅ Forward slash warning in names (deletion will fail with 403)
- ✅ In-place rename with ID following name
- ✅ Certificate validation toggle

### Testing Workflow (15-20 minutes)

#### Phase 1: Setup (2 minutes)

```bash
# 1. Create timestamped working directory
export TEST_DIR="/tmp/sia-crud-validation-target-set-$(date +%Y%m%d-%H%M%S)"
mkdir -p $TEST_DIR
cd $TEST_DIR

# 2. Copy target set template
cp ~/terraform-provider-cyberark-sia/examples/testing/crud-test-target-set.tf .

# 3. Export environment variables (recommended)
export CYBERARK_USERNAME="your-username@cyberark.cloud.XXXX"
export CYBERARK_PASSWORD="<your-password-here>"
export TF_ACC=1

# Verify environment
cd ~/terraform-provider-cyberark-sia
make check-env

# 4. Build and install provider
make build && make install

# 5. Initialize Terraform
cd $TEST_DIR
terraform init
```

**Validation**:
- [ ] Working directory created with timestamp
- [ ] Template copied successfully
- [ ] Environment variables set
- [ ] Provider built without errors
- [ ] Terraform initialized successfully

#### Phase 2: CREATE - VM Secret (< 1 minute)

**IMPORTANT**: Target sets require an existing VM secret. Create one first or reference an existing secret.

```bash
# Option A: Create test secret (recommended for isolated testing)
terraform apply -target=cyberarksia_virtual_machine_secret.test -auto-approve

# Option B: Use existing secret by updating secret_id in template
# Edit crud-test-target-set.tf and set secret_id to existing secret ID
```

**Validation**:
- [ ] VM secret created successfully
- [ ] Secret ID is UUID format
- [ ] secret_type is "ProvisionerUser" or "PCloudAccount"

#### Phase 3: CREATE - Target Set (< 1 minute)

**Edit Template**: Update `local.test_suffix` for unique naming (e.g., "01", "02", "03")

```hcl
locals {
  test_suffix = "01"  # Change this for each test run
}
```

**Create Target Set**:
```bash
terraform apply -auto-approve
```

**Validation**:
- [ ] Target set created successfully
- [ ] Name format: `crud-test-{suffix}.example.com`
- [ ] Type: "Domain"
- [ ] ID equals name (name-as-ID pattern)
- [ ] secret_id matches VM secret
- [ ] provision_format: `<user>-test-<session-guid>` (custom value)
- [ ] enable_certificate_validation: true (default)
- [ ] description populated correctly

#### Phase 4: READ - State Refresh (< 1 minute)

```bash
# Refresh state from API
terraform refresh

# Verify no changes detected
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] `terraform plan` shows 0 to add, 0 to change, 0 to destroy
- [ ] All computed fields populated (id, provision_format defaults)
- [ ] No drift detected between state and API

#### Phase 5: UPDATE - Type Change (< 1 minute)

**Test Type Mutability**: Change from Domain → Suffix (in-place update, no recreation)

Edit `crud-test-target-set.tf`:
```hcl
resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}.example.com"
  type        = "Suffix"  # CHANGED from "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"

  description                   = "UPDATED: CRUD validation test - modified"  # CHANGED
  provision_format              = "jit-<user>-<session-guid>"                  # CHANGED
  enable_certificate_validation = false                                        # CHANGED
}
```

Apply changes:
```bash
terraform apply -auto-approve
```

**Expected Output**: `Plan: 0 to add, 1 to change, 0 to destroy`

**Validation**:
- [ ] Type changed to "Suffix" (no resource recreation)
- [ ] Description updated
- [ ] provision_format changed successfully
- [ ] enable_certificate_validation toggled to false
- [ ] ID still equals name (unchanged)
- [ ] `terraform plan` shows no further changes

#### Phase 6: UPDATE - Rename Test (< 1 minute)

**Test In-Place Rename**: Verify ID follows name

Edit `crud-test-target-set.tf`:
```hcl
resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}-renamed.example.com"  # CHANGED
  type        = "Suffix"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"

  description                   = "UPDATED: CRUD validation test - modified"
  provision_format              = "jit-<user>-<session-guid>"
  enable_certificate_validation = false
}
```

Apply rename:
```bash
terraform apply -auto-approve
```

**Expected Output**: `Plan: 0 to add, 1 to change, 0 to destroy`

**Validation**:
- [ ] `terraform plan` shows update in-place (NOT replacement)
- [ ] After apply, ID equals new name
- [ ] Old name would return 404 if queried manually
- [ ] `terraform plan` shows no changes after rename

#### Phase 7: NEGATIVE TEST - provision_format Clearing (< 1 minute)

**Test Clearing Prevention**: Attempt to remove provision_format (should fail at plan time)

Edit `crud-test-target-set.tf` and **remove** the `provision_format` line:
```hcl
resource "cyberarksia_target_set" "test" {
  name        = "crud-test-${local.test_suffix}-renamed.example.com"
  type        = "Suffix"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"

  description                   = "UPDATED: CRUD validation test - modified"
  # provision_format = ""  # REMOVED - should ERROR
  enable_certificate_validation = false
}
```

Try to plan:
```bash
terraform plan
```

**Expected Output**: Error at plan time

**Validation**:
- [ ] `terraform plan` shows error: "Cannot Clear Attribute"
- [ ] Error message explains: "cannot be removed once set due to API limitations"
- [ ] Plan does NOT proceed (blocked at plan phase)

**Restore provision_format** before continuing:
```hcl
  provision_format = "jit-<user>-<session-guid>"
```

#### Phase 8: NEGATIVE TEST - Forward Slash Warning (< 1 minute)

**Test Forward Slash Detection**: Create target set with forward slashes (warning, not error)

**WARNING**: This test creates a resource that CANNOT be deleted via Terraform. Manual SIA UI deletion required.

Create separate test configuration:
```hcl
resource "cyberarksia_target_set" "forward_slash_test" {
  name        = "env/test/servers-${local.test_suffix}"  # Contains forward slashes
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.test.id
  secret_type = "ProvisionerUser"
}
```

Apply:
```bash
terraform apply -target=cyberarksia_target_set.forward_slash_test -auto-approve
```

**Validation**:
- [ ] `terraform plan` shows WARNING (not error)
- [ ] Warning mentions "forward slashes which will cause deletion failures"
- [ ] Resource CAN be created (warning doesn't block)

**Cleanup Attempt** (will fail):
```bash
terraform destroy -target=cyberarksia_target_set.forward_slash_test -auto-approve
```

**Expected Output**: Error 403 Forbidden

**Validation**:
- [ ] `terraform destroy` FAILS with 403 Forbidden
- [ ] Manual deletion via SIA UI required
- [ ] Forward slash validator working correctly

**Manual Cleanup**:
1. Log into SIA UI
2. Navigate to Target Sets
3. Manually delete `env/test/servers-{suffix}`

#### Phase 9: IMPORT - Test Import Functionality (< 1 minute)

**Test Import by Name**: Target sets use name as ID

```bash
# Get target set name (ID equals name)
TARGET_SET_ID=$(terraform output -raw target_set_id)

# Remove from state
terraform state rm cyberarksia_target_set.test

# Import by name
terraform import cyberarksia_target_set.test "$TARGET_SET_ID"

# Verify no changes after import
terraform plan
```

**Expected Output**: `No changes. Your infrastructure matches the configuration.`

**Validation**:
- [ ] Import succeeded with name as ID
- [ ] All attributes populated correctly after import
- [ ] name matches ID
- [ ] type, secret_id, provision_format restored
- [ ] No changes detected in `terraform plan`

#### Phase 10: DELETE - Cleanup (< 1 minute)

**Delete in Dependency Order**:

```bash
# Delete target set first
terraform destroy -target=cyberarksia_target_set.test -auto-approve

# Delete VM secret
terraform destroy -target=cyberarksia_virtual_machine_secret.test -auto-approve
```

**Validation**:
- [ ] Target set deleted successfully
- [ ] VM secret deleted successfully
- [ ] No orphaned resources in SIA UI
- [ ] State is clean: `terraform state list` returns empty

### Success Criteria

- ✅ VM secret created successfully (ProvisionerUser or PCloudAccount)
- ✅ Target set created with all attributes
- ✅ ID equals name (name-as-ID pattern)
- ✅ Type changes (Domain → Suffix → Target) without recreation
- ✅ provision_format can be updated but not cleared (plan-time error)
- ✅ Forward slash validator warns but allows creation
- ✅ In-place rename with ID following name
- ✅ Certificate validation can be toggled
- ✅ Credential rotation (update secret_id) works in-place
- ✅ IMPORT works with name as ID
- ✅ DELETE cleans up without errors
- ✅ SIA UI reflects all changes correctly

### Advanced Testing: All 6 Type Change Combinations

**Test All Bidirectional Changes** (Domain ↔ Suffix ↔ Target):

```bash
# Start with Domain
terraform apply -auto-approve
# Verify: type = "Domain"

# Change to Suffix
# Edit: type = "Suffix"
terraform apply -auto-approve
# Verify: type = "Suffix", no recreation

# Change to Target
# Edit: type = "Target"
terraform apply -auto-approve
# Verify: type = "Target", no recreation

# Change back to Domain
# Edit: type = "Domain"
terraform apply -auto-approve
# Verify: type = "Domain", no recreation

# Test direct Domain → Target
# Edit: type = "Target"
terraform apply -auto-approve
# Verify: type = "Target", no recreation

# Test direct Target → Suffix (completes all 6 combinations)
# Edit: type = "Suffix"
terraform apply -auto-approve
# Verify: type = "Suffix", no recreation
```

**Validation**:
- [ ] All 6 type change combinations work (Target↔Domain, Target↔Suffix, Domain↔Suffix)
- [ ] No ForceNew on type attribute
- [ ] ID remains stable across type changes
- [ ] `terraform plan` shows no changes after each apply

### Common Issues

**Issue**: Forward Slash Deletion Failure
```
Error: 403 Forbidden - Cannot delete target set with forward slashes in name
```
**Solution**: Manually delete via SIA UI. Forward slash validator warns about this during creation.

**Issue**: provision_format Clearing Error
```
Error: Cannot Clear Attribute: provision_format cannot be removed once set
```
**Solution**: Expected behavior. You can UPDATE provision_format but not remove it. This is due to API PATCH semantics.

**Issue**: Secret Not Found
```
Error: Failed to read VM secret: 404 Not Found
```
**Solution**: Ensure VM secret exists before creating target set. Create via template or reference existing secret ID.

### Troubleshooting Reference

**Quick Diagnostics**:
1. **Validate environment**: `make check-env` - verifies `CYBERARK_USERNAME` and `CYBERARK_PASSWORD`
2. **Rebuild provider**: `cd ~/terraform-provider-cyberark-sia && make build && make install`
3. **Reinitialize Terraform**: `rm -rf .terraform .terraform.lock.hcl && terraform init`
4. **Check template**: Ensure `local.test_suffix` is updated to avoid name conflicts

**Common Patterns**:
- **Name conflicts**: Increment `local.test_suffix` for each test run
- **Drift detection**: Run `terraform refresh` then `terraform plan`
- **State issues**: Use `terraform state list` and `terraform state show` to inspect

### Test Results Documentation

Save test results:

```bash
cat > TARGET-SET-TEST-RESULTS-$(date +%Y%m%d-%H%M%S).md <<'EOF'
# Target Set CRUD Test Results

**Test Date**: $(date)
**Test Directory**: $TEST_DIR
**Duration**: [FILL IN] minutes

## Resources Created
- VM Secret ID: [SECRET_ID]
- Target Set Name: [TARGET_SET_NAME]
- Type Tested: Domain → Suffix → Target (all combinations)

## Test Phases
- [x] Phase 1: Setup
- [x] Phase 2: VM Secret Creation
- [x] Phase 3: Target Set Creation
- [x] Phase 4: READ - State Refresh
- [x] Phase 5: UPDATE - Type Change
- [x] Phase 6: UPDATE - Rename
- [x] Phase 7: NEGATIVE - provision_format Clearing
- [x] Phase 8: NEGATIVE - Forward Slash Warning
- [x] Phase 9: IMPORT
- [x] Phase 10: DELETE

## Validation Results
- All resources created successfully: YES/NO
- No drift detected: YES/NO
- Type changes without recreation: YES/NO
- provision_format clearing prevented: YES/NO
- Forward slash warning displayed: YES/NO
- Rename successful with ID update: YES/NO
- IMPORT successful: YES/NO
- DELETE successful: YES/NO
- SIA UI matches state: YES/NO

## Issues Encountered
[FILL IN]

## Notes
[FILL IN]
EOF
```

---

## See Also

### Documentation
- [`CLAUDE.md`](../../CLAUDE.md) - Development guidelines (references this guide)
- [`docs/testing-framework.md`](../../docs/testing-framework.md) - Conceptual testing framework
- [`docs/resources/policy_workspace_assignment.md`](../../docs/resources/policy_workspace_assignment.md) - Policy database assignment documentation
- [`docs/resources/database_policy.md`](../../docs/resources/database_policy.md) - Database policy documentation
- [`docs/resources/database_policy_principal_assignment.md`](../../docs/resources/database_policy_principal_assignment.md) - Principal assignment documentation
- [`examples/resources/`](../resources/) - Per-resource usage examples

### Automation
- [`scripts/test-crud-resource.sh`](../../scripts/test-crud-resource.sh) - Automated CRUD testing script
- [`Makefile`](../../Makefile) - Build and test targets (`make help` for all commands)
  - `make test-crud DESC=<description>` - Run automated CRUD validation
  - `make check-env` - Verify environment variables
  - `make build && make install` - Build and install provider
  - `make testacc` - Run acceptance tests

### Test Results
- `/tmp/sia-azure-test-20251027-185657/TEST-RESULTS.md` - Azure PostgreSQL test results (2025-10-27)
- `/tmp/sia-crud-validation-*/` - Automated test directories (timestamped)
