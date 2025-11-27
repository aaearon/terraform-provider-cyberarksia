# Quickstart: Principal Lookup Data Source

**Feature**: `cyberarksia_principal` data source
**Purpose**: Look up principals (users, groups, roles) by name for policy assignments

---

## Basic Usage

### 1. Look Up a Cloud Directory User

```hcl
data "cyberarksia_principal" "admin_user" {
  name = "admin@cyberark.cloud.12345"
}

output "user_info" {
  value = {
    id         = data.cyberarksia_principal.admin_user.id
    type       = data.cyberarksia_principal.admin_user.principal_type
    directory  = data.cyberarksia_principal.admin_user.directory_name
  }
}
```

### 2. Use in Policy Assignment

```hcl
data "cyberarksia_principal" "db_admin" {
  name = "tim.schindler@cyberark.cloud.40562"
}

resource "cyberarksia_database_policy_principal_assignment" "admin_access" {
  policy_id               = cyberarksia_database_policy.prod_db.policy_id
  principal_id            = data.cyberarksia_principal.db_admin.id
  principal_type          = data.cyberarksia_principal.db_admin.principal_type
  principal_name          = data.cyberarksia_principal.db_admin.name
  source_directory_name   = data.cyberarksia_principal.db_admin.directory_name
  source_directory_id     = data.cyberarksia_principal.db_admin.directory_id
}
```

### 3. Filter by Type (Group)

```hcl
data "cyberarksia_principal" "db_admins_group" {
  name = "Database Administrators"
  type = "GROUP"  # Only search for groups
}
```

---

## Supported Directory Types

### Cloud Directory (CDS)
Native CyberArk cloud users:
```hcl
data "cyberarksia_principal" "cloud_user" {
  name = "user@cyberark.cloud.12345"
}
```

### Federated Directory (Entra ID / Okta)
Federated identity users:
```hcl
data "cyberarksia_principal" "entra_user" {
  name = "john.doe@company.com"
}
```

### Active Directory (AdProxy)
On-premises AD users:
```hcl
data "cyberarksia_principal" "ad_user" {
  name = "SchindlerT@cyberiam.tech"
}
```

---

## Supported Principal Types

### USER
```hcl
data "cyberarksia_principal" "user" {
  name = "tim@example.com"
  type = "USER"  # Optional: explicitly filter
}

# Available outputs: id, principal_type, directory_name, directory_id,
#                   display_name, email (users only), description
```

### GROUP
```hcl
data "cyberarksia_principal" "group" {
  name = "Administrators"
  type = "GROUP"
}

# Available outputs: id, principal_type, directory_name, directory_id,
#                   display_name, description
```

### ROLE
```hcl
data "cyberarksia_principal" "role" {
  name = "DB Admin Role"
  type = "ROLE"
}

# Available outputs: id, principal_type, directory_name, directory_id,
#                   display_name, description
```

---

## Complete Example

```hcl
terraform {
  required_providers {
    cyberarksia = {
      source = "terraform.local/local/cyberarksia"
      version = "~> 0.1"
    }
  }
}

provider "cyberarksia" {
  username      = "service-account@cyberark.cloud.12345"
  client_secret = var.cyberark_secret
}

# Look up user
data "cyberarksia_principal" "db_user" {
  name = "tim.schindler@cyberark.cloud.40562"
}

# Look up group
data "cyberarksia_principal" "db_group" {
  name = "Database Administrators"
  type = "GROUP"
}

# Use in policy assignments
resource "cyberarksia_database_policy_principal_assignment" "user_access" {
  policy_id               = cyberarksia_database_policy.prod.policy_id
  principal_id            = data.cyberarksia_principal.db_user.id
  principal_type          = data.cyberarksia_principal.db_user.principal_type
  principal_name          = data.cyberarksia_principal.db_user.name
  source_directory_name   = data.cyberarksia_principal.db_user.directory_name
  source_directory_id     = data.cyberarksia_principal.db_user.directory_id
}

resource "cyberarksia_database_policy_principal_assignment" "group_access" {
  policy_id               = cyberarksia_database_policy.prod.policy_id
  principal_id            = data.cyberarksia_principal.db_group.id
  principal_type          = data.cyberarksia_principal.db_group.principal_type
  principal_name          = data.cyberarksia_principal.db_group.name
  source_directory_name   = data.cyberarksia_principal.db_group.directory_name
  source_directory_id     = data.cyberarksia_principal.db_group.directory_id
}

# Outputs
output "user_details" {
  value = {
    id         = data.cyberarksia_principal.db_user.id
    type       = data.cyberarksia_principal.db_user.principal_type
    directory  = data.cyberarksia_principal.db_user.directory_name
    email      = data.cyberarksia_principal.db_user.email
  }
}

output "group_details" {
  value = {
    id         = data.cyberarksia_principal.db_group.id
    type       = data.cyberarksia_principal.db_group.principal_type
    directory  = data.cyberarksia_principal.db_group.directory_name
  }
}
```

---

## Error Handling

### Principal Not Found
```
Error: Principal Not Found
Principal 'nonexistent@example.com' not found in any directory
```

**Resolution**: Verify the principal name spelling and ensure the principal exists in one of the configured directories.

### Authentication Failed
```
Error: Authentication Failed
Failed to authenticate with CyberArk Identity: invalid credentials
```

**Resolution**: Check your provider configuration (username/client_secret).

---

## Next Steps

- **Full Documentation**: See `docs/data-sources/principal.md`
- **Examples**: Browse `examples/data-sources/cyberarksia_principal/`
- **Testing Guide**: See `examples/testing/TESTING-GUIDE.md` for CRUD validation

---

## Performance Notes

- **Users**: < 1 second lookup (fast path via UserByName API)
- **Groups/Roles**: < 2 seconds (fallback path via full entity scan)
- **Caching**: No caching (each Terraform run re-queries the API)
- **Scale**: Tested with 200 entities, supports up to 10,000 principals per tenant
