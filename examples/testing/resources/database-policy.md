# Database Policy Management Testing

**Part of**: [Comprehensive CRUD Testing Guide](../TESTING-GUIDE.md)  
**Last Updated**: See git history

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
