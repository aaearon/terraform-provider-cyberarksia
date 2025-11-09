# Quick Start: Target Set Resource

**Resource**: `cyberarksia_target_set`
**Purpose**: Manage VM/server target sets for Just-In-Time (JIT) privileged access
**Date**: 2025-11-08

## Prerequisites

1. **CyberArk Identity Tenant** with SIA enabled
2. **OAuth2 Service Account** credentials
3. **Terraform** v1.5+ installed
4. **Provider Configuration** completed

### Provider Configuration

```hcl
terraform {
  required_providers {
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "~> 0.1.0"
    }
  }
}

provider "cyberarksia" {
  # Authentication via environment variables:
  # export CYBERARK_USERNAME="service-account@cyberark.cloud.12345"
  # export CYBERARK_PASSWORD="your-password"
}
```

---

## Basic Usage

### Example 1: Domain-Based Target Set

Matches all servers in a domain (most common use case).

```hcl
# First, create VM credentials
resource "cyberarksia_virtual_machine_secret" "admin" {
  name        = "windows-admin-credentials"
  secret_type = "ProvisionerUser"
  username    = "Administrator"
  password    = var.admin_password
}

# Then, create target set
resource "cyberarksia_target_set" "production" {
  name        = "prod.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = cyberarksia_virtual_machine_secret.admin.secret_type

  description = "Production environment servers"
}
```

**What This Does**:
- Creates a target set matching ALL servers in `prod.example.com` domain
- Uses administrator credentials for JIT access
- Terraform automatically creates the secret first (dependency graph)

**Commands**:
```bash
terraform init
terraform plan
terraform apply
```

---

### Example 2: Suffix-Based Target Set

Matches servers with a specific hostname suffix (datacenter grouping).

```hcl
resource "cyberarksia_target_set" "datacenter_east" {
  name        = "dc-east.example.com"
  type        = "Suffix"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = "ProvisionerUser"

  description                  = "East Coast Datacenter Servers"
  enable_certificate_validation = true
}
```

**What This Does**:
- Matches all servers ending with `dc-east.example.com`
- Examples: `web01.dc-east.example.com`, `db01.dc-east.example.com`
- Enables TLS certificate validation (recommended for production)

---

### Example 3: Target-Based Target Set

Matches a specific server (individual system).

```hcl
resource "cyberarksia_target_set" "critical_database" {
  name        = "db01.prod.example.com"
  type        = "Target"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = "ProvisionerUser"

  description = "Critical Production Database Server"
  provision_format = "jit-<user>-<session-guid>"
}
```

**What This Does**:
- Matches ONLY `db01.prod.example.com` (exact hostname)
- Uses custom ephemeral account naming format with "jit-" prefix
- When user "john.doe" requests access, creates account like `jit-john.doe-abc123`

---

## Complete Example (All Attributes)

```hcl
resource "cyberarksia_virtual_machine_secret" "admin" {
  name        = "windows-admin-credentials"
  secret_type = "ProvisionerUser"
  username    = "Administrator"
  password    = var.admin_password
}

resource "cyberarksia_target_set" "complete_example" {
  # Required attributes
  name        = "staging.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = cyberarksia_virtual_machine_secret.admin.secret_type

  # Optional attributes
  description                  = "Staging environment servers - US West region"
  provision_format             = "<user>-stg-<session-guid>"
  enable_certificate_validation = false  # Disable for dev/test with self-signed certs

  # Note: provision_format cannot be removed once set (audit trail consistency)
}
```

---

## Common Operations

### Rename Target Set

```hcl
resource "cyberarksia_target_set" "production" {
  name        = "new-prod.example.com"  # Changed from "old-prod.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = "ProvisionerUser"
}
```

**Behavior**: In-place update (no resource recreation), ID automatically updates to match new name.

---

### Change Matching Pattern

```hcl
resource "cyberarksia_target_set" "flexible" {
  name        = "servers.example.com"
  type        = "Suffix"  # Changed from "Domain" or "Target"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = "ProvisionerUser"
}
```

**Behavior**: In-place update (no resource recreation). All 6 type changes supported bidirectionally.

---

### Rotate Credentials

```hcl
resource "cyberarksia_virtual_machine_secret" "new_admin" {
  name        = "new-admin-credentials"
  secret_type = "ProvisionerUser"
  username    = "NewAdministrator"
  password    = var.new_admin_password
}

resource "cyberarksia_target_set" "production" {
  name        = "prod.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.new_admin.id  # Updated reference
  secret_type = cyberarksia_virtual_machine_secret.new_admin.secret_type
}
```

**Behavior**: In-place update, new credentials apply to future JIT sessions (active sessions continue with old credentials).

---

### Add Ephemeral Account Naming

```hcl
resource "cyberarksia_target_set" "production" {
  name             = "prod.example.com"
  type             = "Domain"
  secret_id        = cyberarksia_virtual_machine_secret.admin.id
  secret_type      = "ProvisionerUser"
  provision_format = "jit-<user>-<session-guid>"  # Added custom format
}
```

**Behavior**: Can be added or updated anytime, but **CANNOT be removed** once set (maintains audit trail consistency).

**Attempting to Clear** (will fail):
```hcl
resource "cyberarksia_target_set" "production" {
  name        = "prod.example.com"
  type        = "Domain"
  secret_id   = cyberarksia_virtual_machine_secret.admin.id
  secret_type = "ProvisionerUser"
  # provision_format = ""  # ❌ ERROR at plan time
}
```

**Error Message**:
```
Error: Cannot clear provision_format

The provision_format field cannot be removed once set due to API limitations.
You can update it to a different value, but cannot clear it entirely.
```

---

## Import Existing Target Sets

### Import Command

```bash
terraform import cyberarksia_target_set.example "target-set-name"
```

### Import Example

```bash
# 1. Create Terraform configuration
cat > target_set.tf <<EOF
resource "cyberarksia_target_set" "existing" {
  name        = "existing-prod.example.com"
  type        = "Domain"
  secret_id   = "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0"
  secret_type = "ProvisionerUser"
}
EOF

# 2. Import existing target set
terraform import cyberarksia_target_set.existing "existing-prod.example.com"

# 3. Verify import
terraform plan  # Should show "No changes"
```

**Note**: After import, run `terraform plan` to verify all attributes match. Update configuration if drift detected.

---

## Troubleshooting

### Issue: Duplicate Name Error

**Symptom**:
```
Error: Failed to create target set

Target set prod.example.com already exists
```

**Solution**: Target set names must be globally unique across the SIA tenant. Choose a different name or import the existing target set.

---

### Issue: Forward Slash in Name

**Symptom**:
```
Warning: Name contains forward slashes

Name contains forward slashes which will cause deletion failures (403 errors).
While the API accepts this during creation, you will not be able to destroy
this resource. Consider using hyphens or underscores instead.
```

**Solution**: Use hyphens (`-`) or underscores (`_`) instead of forward slashes (`/`):
- ❌ Bad: `"servers/production/web"`
- ✅ Good: `"servers-production-web"` or `"servers_production_web"`

---

### Issue: Cannot Clear provision_format

**Symptom**:
```
Error: Cannot clear provision_format

The provision_format field cannot be removed once set due to API limitations.
```

**Solution**: You can update `provision_format` to a different value, but cannot remove it entirely:
```hcl
# ✅ Allowed: Update to new format
provision_format = "<user>-new-format-<session-guid>"

# ❌ Not Allowed: Remove entirely
# provision_format = ""
```

---

### Issue: Secret ID Not Found

**Symptom**:
```
Error: Failed to create target set

Secret aec8cf4b-8012-4efb-9aa2-ca14db5f79c0 not found
```

**Solution**: Ensure VM secret exists and use Terraform reference for automatic dependency:
```hcl
secret_id = cyberarksia_virtual_machine_secret.admin.id
```

This ensures Terraform creates the secret before the target set.

---

### Issue: DELETE Fails with 403

**Symptom**:
```
Error: Failed to delete target set

Status: 403 Forbidden
```

**Cause**: Name contains forward slashes (URL path interpretation issue).

**Solution**: Manual cleanup required via CyberArk SIA UI or API (contact CyberArk support if needed).

---

## Best Practices

### 1. Use Terraform References for Dependencies

✅ **Good** - Automatic dependency ordering:
```hcl
secret_id   = cyberarksia_virtual_machine_secret.admin.id
secret_type = cyberarksia_virtual_machine_secret.admin.secret_type
```

❌ **Bad** - Manual UUID (no dependency):
```hcl
secret_id   = "aec8cf4b-8012-4efb-9aa2-ca14db5f79c0"
secret_type = "ProvisionerUser"
```

### 2. Use Descriptive Names

✅ **Good** - Clear, descriptive:
```hcl
name = "production-web-servers.example.com"
name = "staging-databases.example.com"
```

❌ **Bad** - Vague or unclear:
```hcl
name = "servers1"
name = "test"
```

### 3. Add Descriptions

✅ **Good** - Documented:
```hcl
description = "Production web servers - US East region - Team Alpha"
```

❌ **Bad** - No context:
```hcl
# No description field
```

### 4. Avoid Forward Slashes in Names

✅ **Good**:
```hcl
name = "prod-web-tier-1.example.com"
```

❌ **Bad**:
```hcl
name = "prod/web/tier-1.example.com"  # Will cause DELETE issues
```

### 5. Use Standard provision_format

✅ **Recommended** - Standard format (default):
```hcl
provision_format = "<user>-<session-guid>"
```

✅ **Acceptable** - Custom format with valid placeholders:
```hcl
provision_format = "jit-<user>-<session-guid>"
```

❌ **Not Recommended** - Invalid placeholders (no API validation):
```hcl
provision_format = "<invalid>-<placeholder>"  # API accepts but may not work
```

---

## Next Steps

1. **Review Examples**: See `examples/resources/cyberarksia_target_set/` for complete working examples
2. **Configure VM Secrets**: Create VM credentials before target sets
3. **Test CRUD Operations**: Follow `examples/testing/TESTING-GUIDE.md` for validation workflow
4. **Configure Access Policies**: Assign users/groups to target sets for JIT access (future resource)

---

## Additional Resources

- **Feature Specification**: [spec.md](./spec.md)
- **Implementation Plan**: [plan.md](./plan.md)
- **Data Model**: [data-model.md](./data-model.md)
- **API Contract**: [contracts/target-set-api-contract.md](./contracts/target-set-api-contract.md)
- **Provider Documentation**: `docs/resources/cyberarksia_target_set.md` (generated by tfplugindocs)
- **Testing Guide**: `examples/testing/TESTING-GUIDE.md`

---

**Quick Start Status**: ✅ COMPLETE
**Last Updated**: 2025-11-08
