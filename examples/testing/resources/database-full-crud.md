# Database Full CRUD Test with Azure PostgreSQL

**Part of**: [Comprehensive CRUD Testing Guide](../TESTING-GUIDE.md)  
**Last Updated**: See git history  
**Duration**: 20-30 minutes  
**Cost**: < $0.01 USD

---

# Complete CRUD Test with Azure PostgreSQL

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
cp -r ~/terraform-provider-cyberarksia/examples/testing/azure-postgresql/* .

# 3. Export environment variables (recommended)
export CYBERARK_USERNAME="your-username@cyberark.cloud.XXXX"
export CYBERARK_PASSWORD="<your-password-here>"
export TF_ACC=1

# Verify environment
cd ~/terraform-provider-cyberarksia
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
cd ~/terraform-provider-cyberarksia
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
cd ~/terraform-provider-cyberarksia
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
cd ~/terraform-provider-cyberarksia/examples/testing/azure-postgresql-with-policy
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
cd ~/terraform-provider-cyberarksia
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
cd ~/terraform-provider-cyberarksia/examples/testing/azure-postgresql-with-policy
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
