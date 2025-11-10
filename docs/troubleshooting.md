# Troubleshooting Guide: CyberArk SIA Terraform Provider

## Table of Contents

- [Partial State Failures](#partial-state-failures)
- [Specific Error Scenarios](#specific-error-scenarios)
- [State Drift Detection](#state-drift-detection)
- [Best Practices for Preventing Issues](#best-practices-for-preventing-issues)
- [Getting Help](#getting-help)
- [Additional Resources](#additional-resources)
- [Known Limitations](#known-limitations)
- [ARK SDK Authentication and Cache Management](#ark-sdk-authentication-and-cache-management)
- [Target Set Resource Issues](#target-set-resource-issues)
- [FAQ: Do All Resources Use In-Memory Profile Authentication?](#faq-do-all-resources-use-in-memory-profile-authentication)
- [Known ARK SDK Issues (v1.5.0)](#known-ark-sdk-issues-v150)

---

## Partial State Failures

### Scenario: Database Created but SIA Onboarding Failed

**Problem**: Your AWS RDS or Azure SQL database was successfully created, but the SIA database workspace registration failed. This leaves you with a database in AWS/Azure but no corresponding entry in SIA.

**Symptoms**:
```
Error: Failed to create database workspace
│ SIA API error: [specific error message]
│
│ Recommended action: [recovery guidance]
```

**Root Causes**:
1. **Authentication Issues**: ISPSS credentials expired or invalid
2. **Permission Issues**: Service account lacks required SIA roles
3. **Network Issues**: SIA API unreachable from Terraform execution environment
4. **API Validation**: Database type/version not supported by SIA
5. **Resource Limits**: SIA tenant quota exceeded

### Recovery Options

#### Option 1: Fix the Issue and Re-apply (Recommended)

This is the safest approach as it maintains Terraform state consistency.

**Steps**:
1. Diagnose and fix the underlying issue (see specific error guidance below)
2. Run `terraform apply` again
3. Terraform will detect that the AWS/Azure database exists and skip its creation
4. Terraform will retry the SIA database workspace creation
5. Verify both resources are now in state: `terraform state list`

**Example**:
```bash
# Fix authentication issue (e.g., renew ISPSS credentials)
export TF_VAR_cyberark_password="new-secret-value"

# Re-apply configuration
terraform apply

# Expected output:
# aws_db_instance.main: Refreshing state... [id=mydb]
# cyberark_sia_database_workspace.main: Creating...
# cyberark_sia_database_workspace.main: Creation complete [id=target-123]
```

#### Option 2: Import Existing Database (If SIA Entry Was Created Manually)

If you manually registered the database in SIA console during troubleshooting:

**Steps**:
1. Find the SIA database workspace ID from the SIA console
2. Import it into Terraform state:
   ```bash
   terraform import cyberark_sia_database_workspace.main <sia-target-id>
   ```
3. Run `terraform plan` to verify state matches configuration
4. Run `terraform apply` if needed to reconcile differences

#### Option 3: Start Over (Nuclear Option)

Only use this if Options 1 and 2 don't work.

**Steps**:
1. Remove the cloud database (if not needed):
   ```bash
   # For AWS
   aws rds delete-db-instance --db-instance-identifier mydb --skip-final-snapshot

   # For Azure
   az sql db delete --name mydb --resource-group myrg --server myserver
   ```
2. Remove the resource from Terraform state:
   ```bash
   terraform state rm aws_db_instance.main
   terraform state rm cyberark_sia_database_workspace.main
   ```
3. Run `terraform apply` to recreate everything

## Specific Error Scenarios

### Error: Authentication Failed

**Error Message**:
```
Authentication failed: Invalid client_id or password
```

**Resolution**:
1. Verify ISPSS credentials are correct
2. Check if service account is enabled in Identity
3. Ensure credentials haven't expired
4. Re-run with updated credentials

**Prevention**:
- Use secret managers (AWS Secrets Manager, Azure Key Vault) for credential storage
- Set up credential rotation workflows
- Monitor service account expiration dates

### Error: Insufficient Permissions

**Error Message**:
```
Insufficient permissions: Service account lacks required role memberships
```

**Resolution**:
1. Log in to CyberArk Identity Admin Portal
2. Navigate to Users & Roles
3. Find your ISPSS service account
4. Add required roles:
   - `SIA Database Administrator` (or equivalent)
   - `SIA Secrets Manager` (for secrets)
5. Wait 2-5 minutes for permissions to propagate
6. Re-run `terraform apply`

**Prevention**:
- Document required roles in your infrastructure documentation
- Use Infrastructure as Code to manage Identity roles (if supported)
- Implement least-privilege access control

### Error: Database Type Not Supported

**Error Message**:
```
Database version 9.5.0 is below minimum for postgresql (10.0.0 required)
```

**Resolution**:
1. Check SIA documentation for supported database versions
2. Upgrade your database to a supported version:
   ```hcl
   resource "aws_db_instance" "main" {
     engine_version = "14.7" # Updated to supported version
   }
   ```
3. Run `terraform apply`

**Prevention**:
- Always check [SIA supported databases documentation](https://docs.cyberark.com/sia) before provisioning
- Use Terraform variables for database versions with validation:
  ```hcl
  variable "postgres_version" {
    type = string
    validation {
      condition     = tonumber(split(".", var.postgres_version)[0]) >= 10
      error_message = "PostgreSQL version must be 10.0 or higher for SIA compatibility."
    }
  }
  ```

### Error: Network Connectivity Issues

**Error Message**:
```
SIA service unavailable: Connection timeout
```

**Resolution**:
1. Verify network connectivity from Terraform execution environment to SIA API:
   ```bash
   curl -I https://your-tenant.cyberark.cloud/api/health
   ```
2. Check firewall rules and security groups
3. Verify DNS resolution
4. Check proxy settings if applicable

**Prevention**:
- Run Terraform from a network location with reliable SIA API access
- Configure appropriate network timeouts in provider configuration:
  ```hcl
  provider "cyberark_sia" {
    request_timeout = 60 # seconds
  }
  ```

### Error: Conflict (Resource Already Exists)

**Error Message**:
```
Resource already exists: A database workspace with name 'prod-postgres' already exists
```

**Resolution**:
1. Check SIA console for existing database workspace
2. Either:
   - **Option A**: Import the existing resource:
     ```bash
     terraform import cyberark_sia_database_workspace.main <existing-target-id>
     ```
   - **Option B**: Rename your new resource:
     ```hcl
     resource "cyberark_sia_database_workspace" "main" {
       name = "prod-postgres-v2" # Changed name
     }
     ```

## State Drift Detection

### Scenario: Resource Modified Outside Terraform

**Problem**: Someone modified the SIA database workspace or secret directly in the SIA console, causing state drift.

**Detection**:
Run `terraform plan` regularly to detect drift:
```bash
terraform plan

# Output shows drift:
# Note: Objects have changed outside of Terraform
# cyberark_sia_database_workspace.main: Refreshing state... [id=target-123]
# Resource actions are indicated with the following symbols:
# ~ update in-place
```

**Resolution**:
1. Review the detected changes
2. Decide whether to:
   - **Accept the manual change**: Update your Terraform configuration to match
   - **Revert the manual change**: Run `terraform apply` to restore Terraform-managed values

### Scenario: Resource Deleted Outside Terraform

**Problem**: Someone deleted the SIA database workspace from the console.

**Detection**:
```bash
terraform plan

# Output:
# cyberark_sia_database_workspace.main: Refreshing state... [id=target-123]
# Warning: Database target not found, removing from state
```

**Resolution**:
1. Verify the resource was intentionally deleted
2. Run `terraform apply` to recreate it:
   ```bash
   terraform apply
   # Output: cyberark_sia_database_workspace.main will be created
   ```

## Best Practices for Preventing Issues

### 1. Use Terraform Workspaces for Environments

Separate state for dev, staging, and production:
```bash
terraform workspace new production
terraform workspace select production
terraform apply
```

### 2. Enable State Locking

Use remote state with locking to prevent concurrent modifications:
```hcl
terraform {
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "sia/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-lock"
    encrypt        = true
  }
}
```

### 3. Implement CI/CD Validation

Run `terraform plan` in CI/CD before merging:
```yaml
# Example GitHub Actions workflow
- name: Terraform Plan
  run: terraform plan -out=tfplan
- name: Verify No Errors
  run: terraform show tfplan | grep -q "Error:" && exit 1 || exit 0
```

### 4. Regular State Backups

Backup Terraform state regularly:
```bash
# Automated backup
terraform state pull > backups/terraform-$(date +%Y%m%d-%H%M%S).tfstate
```

### 5. Use Terraform Modules

Encapsulate common patterns in reusable modules:
```hcl
module "sia_database" {
  source = "./modules/sia-rds-integration"

  environment    = "production"
  database_type  = "postgresql"
  database_version = "14.7"
}
```

## Getting Help

### Log Collection

When opening support tickets, include:

1. **Terraform Version**:
   ```bash
   terraform version
   ```

2. **Provider Version**:
   ```bash
   terraform providers
   ```

3. **Debug Logs** (redact sensitive data):
   ```bash
   TF_LOG=DEBUG terraform apply 2>&1 | tee terraform-debug.log
   ```

4. **State Information**:
   ```bash
   terraform state list
   terraform state show cyberark_sia_database_workspace.main
   ```

### Support Channels

- **Provider Issues**: https://github.com/aaearon/terraform-provider-cyberarksia/issues
- **CyberArk SIA Support**: https://www.cyberark.com/customer-support/
- **Community Forums**: CyberArk Commons

## Additional Resources

- [CyberArk SIA Documentation](https://docs.cyberark.com/sia)
- [Terraform Provider Best Practices](https://www.terraform.io/docs/extend/best-practices/index.html)
- [ARK SDK Documentation](https://github.com/cyberark/ark-sdk-golang)

## Known Limitations

### Certificate `last_updated_by` Field Warning

**Symptom**:
```
Warning: Provider returned invalid result object after apply
After the apply operation, the provider still indicated an unknown value
for cyberarksia_certificate.*.last_updated_by
```

**Impact**: This is a cosmetic warning that does not prevent resource creation or any other CRUD operations. The field is correctly set to `null` for newly created certificates and will be populated after the first update operation.

**Root Cause**: Terraform Plugin Framework limitation in handling nullable computed fields during the plan/apply cycle. The framework automatically marks nullable computed attributes as "unknown" during planning, which conflicts with the API's behavior of returning `null` for this field on newly created certificates.

**Workaround**: None required - all operations function correctly despite the warning. You can safely ignore this message.

**Verification**: After apply completes, run `terraform show` to verify the certificate was created successfully:
```bash
terraform show | grep -A5 "cyberarksia_certificate"
```

All fields except `last_updated_by` will be populated. The field will remain `null` until the certificate is updated.

**Future Fix**: This will be resolved in a future version when either:
1. The Terraform Plugin Framework improves nullable computed field handling
2. The SIA API is updated to always return a value for this field

---

## ARK SDK Authentication and Cache Management

### Understanding ARK SDK Caching Behavior

The ARK SDK (github.com/cyberark/ark-sdk-golang v1.5.0) has built-in caching mechanisms that can cause authentication issues if not properly managed in Terraform provider context.

**Default SDK Behavior**:
- Loads profiles from `~/.ark/profiles/` when `Authenticate(nil, ...)` is called
- Caches tokens in `~/.ark_cache/` directory
- Uses keyring storage for credential caching
- Caching persists even when `NewArkISPAuth(false)` is used (false only disables one layer)

**Problem in Terraform Context**:
This caching behavior is problematic for Terraform providers because:
1. **Stale tokens**: Cached tokens may be expired or from different credentials
2. **401 Unauthorized errors**: Subsequent CRUD operations fail with authentication errors
3. **Manual workarounds required**: Users had to manually delete `~/.ark_cache` between runs
4. **State inconsistency**: Resource creation succeeds but read operations fail

### Symptoms of Cache-Related Auth Issues

**Certificate Resource Example**:
```bash
# First operation (CREATE)
terraform apply
# ✅ Success: cyberarksia_certificate.test created (ID: 1761472452063442)

# Second operation (READ) - FAILS with stale cache
terraform plan
# ❌ Error: 401 Unauthorized
# The provider could not read certificate 1761472452063442

# Manual workaround (before fix)
rm -rf ~/.ark_cache
terraform plan
# ✅ Success: No changes detected
```

**Other Symptoms**:
- Intermittent authentication failures
- Works on fresh systems but fails after first run
- Different behavior between `terraform apply` and `terraform plan`
- Errors mentioning "invalid token" or "authentication required"

### Solution: In-Memory Profile Authentication

**Implementation** (2025-10-26 - Commit: 9899527):

The provider now creates in-memory `ArkProfile` objects to completely bypass filesystem-based profile loading and caching.

**Key Components**:

#### 1. ISPAuthContext Structure (internal/client/auth.go)

```go
// ISPAuthContext holds authentication state for re-use across operations
// This prevents filesystem profile loading and keyring caching
type ISPAuthContext struct {
    ISPAuth     *auth.ArkISPAuth           // SDK auth instance
    Profile     *models.ArkProfile         // In-memory profile (NOT persisted)
    AuthProfile *authmodels.ArkAuthProfile // Auth configuration
    Secret      *authmodels.ArkSecret      // Credentials
}
```

**Why this works**:
- Holds all authentication state in memory
- Prevents SDK from falling back to `~/.ark/profiles/` loading
- Allows re-authentication with same profile during token refresh

#### 2. In-Memory Profile Creation

```go
// Create in-memory ArkProfile to bypass filesystem profile loading
inMemoryProfile := &models.ArkProfile{
    ProfileName:  "terraform-ephemeral", // Non-persisted name
    AuthProfiles: map[string]*authmodels.ArkAuthProfile{
        "isp": authProfile,
    },
}

// Authenticate with explicit in-memory profile (NOT nil)
// force=true: Always get a fresh token (no cache lookups)
_, err := ispAuth.Authenticate(
    inMemoryProfile,  // Explicit profile (prevents default profile loading)
    authProfile,      // Auth method configuration
    secret,           // Credentials
    true,             // force=true (bypass ALL cache lookups)
    false,            // refreshAuth=false (not a refresh operation)
)
```

**Critical Parameters**:
- **First parameter (`inMemoryProfile`)**: Must NOT be `nil`. Passing explicit profile prevents SDK from loading `~/.ark/profiles/`
- **Fourth parameter (`force=true`)**: Bypasses all cache lookups, ensures fresh token
- **Profile name (`terraform-ephemeral`)**: Non-standard name that won't conflict with user profiles

#### 3. Token Refresh Callback (internal/client/certificates.go)

```go
// refreshSIAAuth refreshes the authentication token when it expires.
// Called automatically by SDK when token approaches 15-min expiration.
// CRITICAL: Re-authenticates with in-memory profile to bypass cache
func (c *CertificatesClient) refreshSIAAuth(client *common.ArkClient) error {
    // Re-authenticate with in-memory profile (force=true to bypass cache)
    _, err := c.authCtx.ISPAuth.Authenticate(
        c.authCtx.Profile,     // In-memory profile (NOT nil)
        c.authCtx.AuthProfile, // Auth profile
        c.authCtx.Secret,      // Secret
        true,                  // force=true (bypass cache)
        false,                 // refreshAuth=false
    )
    if err != nil {
        return fmt.Errorf("failed to refresh authentication: %w", err)
    }

    // Refresh the client with new token
    return isp.RefreshClient(client, c.authCtx.ISPAuth)
}
```

**Why refresh callback matters**:
- ARK SDK bearer tokens expire after 15 minutes
- Refresh callback ensures long-running operations (create, update) continue working
- Using in-memory profile in callback prevents falling back to cached tokens

### Verification and Testing

**Test Scenario**: Certificate CRUD without cache deletion

```bash
# Clean test environment
cd /tmp/test-cache-bypass
rm -rf .terraform terraform.tfstate*

# Create test certificate
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=test/O=Testing/C=US"

# Test configuration
cat > main.tf << 'EOF'
terraform {
  required_providers {
    cyberarksia = {
      source  = "terraform.local/local/cyberark-sia"
      version = "0.1.0"
    }
  }
}

provider "cyberarksia" {
  username      = "your-service-account@cyberark.cloud.XXXX"
  password = "your-secret"
}

resource "cyberarksia_certificate" "test" {
  cert_name = "cache-bypass-test"
  cert_body = file("${path.module}/cert.pem")
  cert_type = "PEM"
}
EOF

# Initialize and test
terraform init
terraform apply -auto-approve
# ✅ Expected: Certificate created successfully

# Verify READ works without cache deletion
terraform plan
# ✅ Expected: No changes detected (proves READ works with fresh auth)

# Test UPDATE
terraform apply -auto-approve
# ✅ Expected: No changes applied (resource unchanged)

# Clean up
terraform destroy -auto-approve
# ✅ Expected: Certificate deleted successfully
```

**Success Criteria**:
- ✅ No manual `rm -rf ~/.ark_cache` required
- ✅ All CRUD operations complete successfully
- ✅ No 401 Unauthorized errors
- ✅ State refresh detects no drift

**Test Results** (2025-10-26):
```
Certificate CREATE: ✅ Success (ID: 1761492255149647)
Certificate READ:   ✅ Success (no drift detected)
Certificate DELETE: ✅ Success (clean removal)
Cache Bypass:       ✅ Verified (no manual cleanup needed)
```

### Architecture Benefits

**Before** (Cache-Dependent):
```
Provider Init → Authenticate → Cache Token → ~/.ark_cache
                                                ↓
Resource CRUD → Load Token from Cache → 401 Error (stale token)
                                                ↓
Manual Fix:  rm -rf ~/.ark_cache → Re-run operation
```

**After** (In-Memory):
```
Provider Init → Authenticate (force=true) → In-Memory Profile
                                                ↓
Resource CRUD → Fresh Auth (bypass cache) → Success
                                                ↓
Token Refresh → Re-auth with In-Memory Profile → Success
```

**Key Advantages**:
1. **Stateless authentication**: No filesystem dependencies
2. **Terraform-friendly**: Each run gets fresh credentials
3. **No manual cleanup**: Works reliably without user intervention
4. **Concurrent runs**: No cache contention between parallel executions
5. **Container-friendly**: No persistent cache directory needed

### When to Apply This Pattern

**Use in-memory profiles when**:
- Building Terraform providers
- Creating automation tools that run repeatedly
- Running in containerized/ephemeral environments
- Managing multiple tenants/accounts concurrently
- Implementing CI/CD pipelines

**Default SDK behavior is OK when**:
- Building interactive CLI tools
- Single-user desktop applications
- Long-running services with stable credentials
- Tools where persistent auth state is desired

### References

**ARK SDK Authentication**:
- Package: `github.com/cyberark/ark-sdk-golang/pkg/auth`
- Profile Model: `github.com/cyberark/ark-sdk-golang/pkg/models.ArkProfile`
- Auth Method: `IdentityServiceUser` (OAuth 2.0 client credentials flow)

**Authenticate() Signature**:
```go
func (a *ArkISPAuth) Authenticate(
    profile *models.ArkProfile,           // Profile (nil loads default, explicit bypasses)
    authProfile *authmodels.ArkAuthProfile, // Auth configuration
    secret *authmodels.ArkSecret,          // Credentials
    force bool,                             // true = bypass cache, false = use cache
    refreshAuth bool,                       // true = refresh operation
) (*authmodels.ArkAuthentication, error)
```

**Key Discovery**:
- Passing `nil` for first parameter triggers default profile loading from `~/.ark/profiles/`
- Passing explicit `ArkProfile` object bypasses filesystem entirely
- `force=true` ensures fresh token on every call (critical for Terraform use case)

### Related Issues and Commits

- **Issue**: Certificate READ operations failing with 401 after CREATE success
- **Root Cause**: ARK SDK loading stale tokens from `~/.ark_cache`
- **Fix Commit**: `9899527` - "fix: Bypass ARK SDK filesystem cache with in-memory profiles"
- **Date**: 2025-10-26
- **Files Changed**: 5 (auth.go, sia_client.go, certificates.go, provider.go, resource_certificate.go)

### Future Considerations

**If ARK SDK evolves**:
- Monitor SDK releases for improved caching controls
- Consider contributing in-memory profile pattern to SDK examples
- Watch for `context.Context` support in `Authenticate()` method

**Provider Enhancements**:
- Consider making profile name configurable (advanced use case)
- Add debug logging for auth lifecycle (troubleshooting)
- Document pattern in SDK integration guide

---

## Target Set Resource Issues

### Forward Slashes in Target Set Names (⚠️ WARNING)

**Status**: ⚠️ **KNOWN LIMITATION** - API design constraint

**Issue**: Target sets with forward slashes (`/`) in the name can be created successfully but **cannot be deleted** via Terraform or the API.

**Symptoms**:
```
Error: DELETE operation failed with status 403 Forbidden
│
│   with cyberarksia_target_set.example,
│   on main.tf line 10, in resource "cyberarksia_target_set" "example":
│   10: resource "cyberarksia_target_set" "example" {
│
│ Failed to delete target set 'env/test/servers'. Manual deletion via SIA UI required.
```

**Root Cause**: CyberArk SIA API limitation:
- **POST (Create)**: Accepts names with forward slashes (e.g., `env/test/servers`)
- **DELETE**: Treats forward slashes as URL path separators, resulting in malformed DELETE requests
- **API Endpoint**: `/WorkspacesVMs/TargetSets/{name}` - forward slashes in `{name}` break the URL parsing

**Example Problematic Names**:
```hcl
# ❌ AVOID - These will create successfully but cannot be deleted
resource "cyberarksia_target_set" "problematic" {
  name        = "env/test/servers"        # Contains forward slashes
  name        = "domain/datacenter/host"  # Contains forward slashes
  name        = "tier1/production/db"     # Contains forward slashes
}

# ✅ RECOMMENDED - Use dots, dashes, or underscores instead
resource "cyberarksia_target_set" "recommended" {
  name        = "env.test.servers"        # Dots are safe
  name        = "domain-datacenter-host"  # Dashes are safe
  name        = "tier1_production_db"     # Underscores are safe
}
```

#### Detection

The provider includes a **custom validator** that warns about forward slashes during `terraform plan`:

```bash
terraform plan

# Output:
│ Warning: Forward Slashes in Target Set Name
│
│   with cyberarksia_target_set.example,
│   on main.tf line 10, in resource "cyberarksia_target_set" "example":
│   10:   name = "env/test/servers"
│
│ The target set name contains forward slashes which will cause deletion failures.
│ DELETE operations will fail with 403 Forbidden due to API URL parsing issues.
│
│ Recommended alternatives:
│   - Use dots:        env.test.servers
│   - Use dashes:      env-test-servers
│   - Use underscores: env_test_servers
```

**Validator Implementation**: `internal/validators/target_set_name_validator.go`

**Why it's a Warning (not Error)**:
- Creation DOES work correctly
- Some users may need forward slashes for organizational naming conventions
- Manual SIA UI deletion is available as a workaround
- Blocking creation would be overly restrictive

#### Recovery Options

If you've already created a target set with forward slashes:

**Option 1: Manual Deletion via SIA UI + State Removal**

```bash
# 1. Log into CyberArk SIA UI
# 2. Navigate to VM Access → Target Sets
# 3. Find the problematic target set (e.g., "env/test/servers")
# 4. Click Delete button in UI
# 5. Confirm deletion

# 6. Remove from Terraform state
terraform state rm cyberarksia_target_set.example

# 7. Continue with destroy for remaining resources
terraform destroy -auto-approve
```

**Option 2: Leave Resource in Place**

If the target set is actively used:
```bash
# Remove only the Terraform management (keep resource in SIA)
terraform state rm cyberarksia_target_set.example

# Target set remains in SIA and continues to function
# Manually manage it via SIA UI going forward
```

**Option 3: Rename via Update (if supported)**

Target sets support in-place rename. Try updating the name to remove forward slashes:

```hcl
# Change configuration
resource "cyberarksia_target_set" "example" {
  name        = "env.test.servers"  # CHANGED: dots instead of slashes
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = "ProvisionerUser"
}
```

```bash
terraform apply -auto-approve
# If successful: Target set renamed, DELETE will now work
# If fails: Fall back to Option 1 or 2
```

#### Prevention

**Best Practices**:
1. ✅ **Use the validator warnings** - Don't ignore warnings during `terraform plan`
2. ✅ **Naming conventions**:
   - Dots: `production.datacenter1.web`
   - Dashes: `production-datacenter1-web`
   - Underscores: `production_datacenter1_web`
3. ✅ **Test in dev/staging** - Verify full CRUD lifecycle before production
4. ✅ **Code review** - Check for forward slashes in target set names before merging

**CI/CD Integration**:
```bash
# Add validation step to CI/CD pipeline
rg --glob '*.tf' 'name\s*=\s*"[^"]*/' || echo "✅ No forward slashes in target set names"
```

#### Impact on Testing

**CRUD Testing**:
- **CREATE**: ✅ Works with forward slashes
- **READ**: ✅ Works correctly
- **UPDATE**: ✅ Works correctly (can rename to fix)
- **DELETE**: ❌ Fails with 403 Forbidden

**Acceptance Testing**:
```go
// Test forward slash warning (acceptance test pattern)
func TestAccTargetSet_forwardSlashWarning(t *testing.T) {
    targetSetName := "env/test/servers"  // Contains forward slashes

    resource.Test(t, resource.TestCase{
        // ... test setup ...
        Steps: []resource.TestStep{
            {
                Config: testAccTargetSetConfigBasic(targetSetName),
                Check: resource.ComposeAggregateTestCheckFunc(
                    // Creation succeeds despite forward slashes
                    testAccCheckTargetSetExists("cyberarksia_target_set.test"),
                    resource.TestCheckResourceAttr("cyberarksia_target_set.test", "name", targetSetName),
                ),
                // NOTE: Validator emits warning during plan but doesn't block creation
            },
        },
        // CheckDestroy will fail - manual cleanup required for this test
    })
}
```

#### Future Considerations

**If API evolves**:
- Monitor CyberArk SIA API updates for improved DELETE endpoint (e.g., ID-based deletion)
- Consider URL encoding target set names in DELETE requests (may fix issue)
- Track feedback with CyberArk product team

**Provider Enhancements**:
- Could upgrade validator to ERROR severity if strict mode is requested
- Consider automatic name sanitization option (replace `/` with `.`)
- Add import/export functionality for problematic resources

#### Related Files

- **Validator**: `internal/validators/target_set_name_validator.go`
- **Validator Tests**: `internal/validators/target_set_name_validator_test.go`
- **Resource Implementation**: `internal/provider/target_set_resource.go`
- **Testing Guide**: `examples/testing/TESTING-GUIDE.md` (Section: Target Set Resource Testing → Phase 8)
- **CRUD Template**: `examples/testing/crud-test-target-set.tf` (Advanced Test: Forward Slash Warning)

#### References

**ARK SDK**:
- Target Set Service: `github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/targetsets`
- DeleteTargetSet: Uses name-based URL construction
- AddTargetSet: Accepts any string for name field

**API Constraint**:
```
DELETE /WorkspacesVMs/TargetSets/{name}
        └── {name} = "env/test/servers"
        └── Interpreted as: /WorkspacesVMs/TargetSets/env/test/servers (path traversal)
        └── Results in: 403 Forbidden or 404 Not Found
```

**Workaround Pattern**:
```go
// Custom DELETE workaround (if needed in future)
// URL-encode the name to preserve forward slashes
encodedName := url.PathEscape(targetSetName)  // "env%2Ftest%2Fservers"
deleteURL := fmt.Sprintf("/WorkspacesVMs/TargetSets/%s", encodedName)
```

---

## FAQ: Do All Resources Use In-Memory Profile Authentication?

**Q: Does the in-memory profile fix apply to database workspace and secret resources?**

**A: Yes, all resources automatically benefit** from the in-memory profile implementation. Here's why:

### Architecture Overview

All resources share the same authenticated `ISPAuth` instance created during provider initialization:

| Resource | Authentication Source | Refresh Mechanism |
|----------|----------------------|-------------------|
| Certificate | CertificatesClient (custom) | Custom refresh callback with in-memory profile |
| Database Workspace | SIAAPI.WorkspacesDB() (SDK) | SDK's built-in refresh using ActiveProfile |
| Secret | SIAAPI.SecretsDB() (SDK) | SDK's built-in refresh using ActiveProfile |

### Why SDK Services Don't Need Custom Refresh

The ARK SDK services (WorkspacesDB, SecretsDB) automatically use the in-memory profile because:

1. **Shared ISPAuth Reference**: All SDK services receive the same `ISPAuth` instance that was authenticated with the in-memory profile

2. **ActiveProfile Preservation**: The SDK stores the in-memory profile:
   ```go
   // Set during Authenticate() in provider initialization
   ispAuth.ActiveProfile = inMemoryProfile  // "terraform-ephemeral"
   ```

3. **SDK Refresh Pattern**: SDK services call `LoadAuthentication(ispAuth.ActiveProfile, true)` which:
   - Uses the stored `ActiveProfile` (our in-memory profile)
   - Skips cache because `CacheKeyring = nil` (set by `NewArkISPAuth(false)`)
   - Re-authenticates using the same in-memory credentials

4. **No Filesystem Access**: The SDK never loads profiles from `~/.ark/profiles/` because:
   - `ActiveProfile` is already set (not nil)
   - Cache is disabled (`CacheKeyring = nil`)

### Certificate Resource Exception

The certificate resource required a **custom refresh callback** because:

1. **Manual ISP Client Creation**: Uses `isp.FromISPAuth()` directly instead of SDK service
2. **Custom Refresh Callback**: Must explicitly re-authenticate with in-memory profile
3. **Not a Built-in SDK Service**: CertificatesAPI isn't part of the standard SDK services

### Verification

No cache-related issues have been reported for database workspace or secret resources, confirming that SDK services properly maintain the in-memory profile authentication state.

**Summary**: Database workspace and secret resources work correctly without additional changes because they use SDK services that automatically preserve and use the in-memory profile.

---

## Known ARK SDK Issues (v1.5.0)

### Database Workspace DELETE Panic (✅ FIXED - 2025-10-27)

**Status**: ✅ **WORKAROUND IMPLEMENTED** - Provider v0.1.0+ includes fix

**Issue**: `terraform destroy` panics when deleting database_workspace resources.

**Error**:
```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x8 pc=0x70f024]

goroutine 57 [running]:
bytes.(*Buffer).Len(...)
net/http.NewRequestWithContext(...)
github.com/cyberark/ark-sdk-golang/pkg/common.(*ArkClient).doRequest(...)
github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/db.(*ArkSIAWorkspacesDBService).DeleteDatabase(...)
```

**Root Cause**: ARK SDK v1.5.0 bug in `ArkClient.doRequest()` method:

1. `DeleteDatabase()` passes `nil` body to HTTP DELETE request (line 188):
   ```go
   response, err := s.client.Delete(context.Background(),
       fmt.Sprintf(resourceURL, deleteDatabase.ID), nil)  // nil body
   ```

2. `doRequest()` doesn't handle nil body properly:
   ```go
   var bodyBytes *bytes.Buffer
   if body != nil {
       // ... marshal body
       bodyBytes = bytes.NewBuffer(bodyB)
   }
   // bodyBytes remains nil if body was nil!
   req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyBytes)
   ```

3. `http.NewRequestWithContext()` calls `bodyBytes.Len()` which panics on nil pointer

**Affected Resources**: Both `database_workspace` and `secret` DELETE operations

**Status**: Reported in golang/go issue #26666 (nil *bytes.Buffer in http.NewRequest)

**SDK Version**: ARK SDK v1.5.0 (latest as of 2025-10-27)

#### ✅ Fixed in Provider

**Provider v0.1.0+** includes automatic workaround - no manual steps required! The provider bypasses the SDK's buggy DELETE methods and makes direct HTTP calls with proper body handling.

**For Older Provider Versions** - Use Manual Workarounds:

**Option 1: Manual Deletion via SIA UI + State Removal**

```bash
# 1. Manually delete database workspace in SIA UI
# 2. Remove from Terraform state
terraform state rm cyberarksia_database_workspace.production

# 3. Continue with destroy for remaining resources
terraform destroy -auto-approve
```

**Option 2: Delete Other Resources First**

```bash
# Delete secret and certificate (these work correctly)
terraform destroy -target=cyberarksia_database_secret.db_admin -auto-approve
terraform destroy -target=cyberarksia_certificate.db_cert -auto-approve

# Manually delete database workspace in SIA UI
# Remove from state
terraform state rm cyberarksia_database_workspace.production
```

**Option 3: Wait for SDK Fix**

Monitor the [ark-sdk-golang repository](https://github.com/cyberark/ark-sdk-golang) for updates that fix the nil body handling in `doRequest()`.

**Expected Fix**: SDK should use `http.NoBody` or `bytes.NewBuffer(nil)` instead of nil pointer:

```go
// Fixed version
var bodyBytes io.Reader = http.NoBody  // or bytes.NewBuffer(nil)
if body != nil {
    bodyB, err := json.Marshal(body)
    if err != nil {
        return nil, err
    }
    bodyBytes = bytes.NewBuffer(bodyB)
}
req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyBytes)
```

#### Impact on CRUD Testing

DELETE tests for `database_workspace` and `secret` resources will fail with panic. Until SDK is fixed:

**Database Workspace & Secret Resources**:
- **CREATE**: ✅ Works correctly
- **READ**: ✅ Works correctly
- **UPDATE**: ✅ Works correctly
- **DELETE**: ❌ Panics (use manual workaround)

**Certificate Resource**:
- **CREATE**: ✅ Works correctly
- **READ**: ✅ Works correctly
- **UPDATE**: ✅ Works correctly (after v0.1.0 fix)
- **DELETE**: ✅ Works correctly
