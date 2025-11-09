# Complete End-to-End Workflow Example

This example demonstrates a complete CyberArk SIA implementation using Terraform, showcasing:

- **Secret Management**: Storing database credentials securely
- **Database Workspaces**: Registering databases with SIA (AWS RDS, Azure SQL, on-premise)
- **Access Policies**: Time-based access control with inline assignments
- **Assignment Patterns**: Both inline (all-in-one) and modular (separate resources) patterns

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     CyberArk SIA                             │
│                                                              │
│  ┌──────────────┐      ┌─────────────────────────────────┐ │
│  │   Secrets    │      │      Database Workspaces        │ │
│  ├──────────────┤      ├─────────────────────────────────┤ │
│  │ postgres-    │──┬──>│ customers (PostgreSQL/RDS)      │ │
│  │ admin        │  │   │ orders (MySQL/Azure)            │ │
│  │              │  │   │ analytics (PostgreSQL/RDS+IAM)  │ │
│  │ rds-iam-user │──┘   │ staging-customers (PostgreSQL)  │ │
│  └──────────────┘      └─────────────────────────────────┘ │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Developer Access Policy (Business Hours)      │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ WHO (inline principals):                             │  │
│  │   └─ developers@example.com (GROUP)                  │  │
│  │ WHO (modular assignment):                            │  │
│  │   └─ senior-developers@example.com (GROUP)           │  │
│  │                                                        │  │
│  │ WHAT (inline databases):                             │  │
│  │   ├─ customers DB (db_auth)                          │  │
│  │   └─ orders DB (db_auth)                             │  │
│  │ WHAT (modular assignment):                           │  │
│  │   └─ staging-customers DB (db_auth)                  │  │
│  │                                                        │  │
│  │ WHEN: Mon-Fri, 8 AM - 6 PM EST                       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         On-Call Access Policy (24/7)                  │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │ WHO: oncall-engineers@example.com (GROUP)            │  │
│  │ WHAT: analytics DB (RDS IAM auth)                    │  │
│  │ WHEN: 24/7                                            │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **CyberArk Identity tenant** with SIA enabled
2. **Service account credentials**:
   ```bash
   export CYBERARK_USERNAME="service-account@cyberark.cloud.12345"
   export CYBERARK_PASSWORD="<your-password-here>"
   ```
3. **Existing users/groups** in CyberArk Identity:
   - `developers@example.com` (GROUP)
   - `senior-developers@example.com` (GROUP)
   - `oncall-engineers@example.com` (GROUP)

## Assignment Patterns: Inline vs. Modular

This example demonstrates **both** assignment patterns:

### Inline Assignments (policies.tf)

Manage the entire policy configuration in one resource:

```hcl
resource "cyberarksia_database_policy" "developer_access" {
  name   = "Developer Access"
  status = "active"

  # Inline principal assignment
  principal {
    principal_id   = data.cyberarksia_principal.developer_group.principal_id
    principal_type = "GROUP"
    # ...
  }

  # Inline database assignment
  target_database {
    database_workspace_id = cyberarksia_database_workspace.production_postgres.id
    authentication_method = "db_auth"
    db_auth_profile {
      roles = ["app_developer"]
    }
  }
}
```

**Use when:**
- You want all-in-one policy configuration
- One team manages the entire access model
- Simpler for small teams

### Modular Assignments (assignments.tf)

Manage principals and databases separately:

```hcl
# Add principal to existing policy
resource "cyberarksia_database_policy_principal_assignment" "senior_devs" {
  policy_id      = cyberarksia_database_policy.developer_access.policy_id
  principal_id   = data.cyberarksia_principal.senior_developers.principal_id
  # ...
}

# Add database to existing policy
resource "cyberarksia_database_policy_workspace_assignment" "staging_access" {
  policy_id             = cyberarksia_database_policy.developer_access.policy_id
  database_workspace_id = cyberarksia_database_workspace.staging_postgres.id
  # ...
}
```

**Use when:**
- Different teams manage WHO (security) vs. WHAT (app teams)
- Need to add principals/databases incrementally
- Want more granular Terraform state management

**IMPORTANT**: Don't mix patterns for the same policy! Choose inline OR modular, not both.

## What Gets Created

### 1. Secrets (secrets.tf)
- `postgres-admin-credentials` - Username/password for PostgreSQL
- `rds-iam-user-credentials` - AWS IAM credentials for RDS

### 2. Database Workspaces (workspaces.tf)
- **customers** - PostgreSQL on AWS RDS (production VPC)
- **orders** - MySQL on Azure (with TLS certificate)
- **analytics** - PostgreSQL on AWS RDS (with IAM authentication)
- **staging-customers** - PostgreSQL on-premise (via modular assignment)

### 3. Access Policies (policies.tf)
- **Developer Access** - Business hours (Mon-Fri, 8 AM-6 PM EST) with inline assignments
- **On-Call Access** - 24/7 access with inline assignments

### 4. Modular Assignments (assignments.tf)
- Senior developers → Developer Policy
- Staging database → Developer Policy

## Usage

### 1. Initialize Terraform
```bash
terraform init
```

### 2. Review the Plan
```bash
terraform plan
```

### 3. Apply Configuration
```bash
terraform apply
```

### 4. View Outputs
```bash
terraform output access_summary
```

Example output:
```hcl
{
  developer_group = {
    access_time = "Monday-Friday, 8 AM - 6 PM EST"
    databases   = ["customers (PostgreSQL)", "orders (MySQL)", "staging-customers (PostgreSQL)"]
    policy      = "Developer Database Access"
  }
  oncall_group = {
    access_time = "24/7"
    databases   = ["analytics (PostgreSQL with RDS IAM)"]
    policy      = "On-Call Engineer Access"
  }
  senior_developers = {
    access_time = "Monday-Friday, 8 AM - 6 PM EST"
    note        = "Added via modular assignment pattern"
    policy      = "Developer Database Access"
  }
}
```

## Configuration Notes

### Time-Based Access

Days of the week are integers (not strings):
- `0` = Sunday
- `1` = Monday
- `2` = Tuesday
- `3` = Wednesday
- `4` = Thursday
- `5` = Friday
- `6` = Saturday

Example:
```hcl
access_window {
  days_of_the_week = [1, 2, 3, 4, 5]  # Monday-Friday
  from_hour        = "08:00"           # 8 AM
  to_hour          = "18:00"           # 6 PM
}
```

### Authentication Profiles

Each authentication method has its own profile block:

- `db_auth_profile` - Standard database authentication
- `ldap_auth_profile` - LDAP/Active Directory
- `oracle_auth_profile` - Oracle database
- `mongo_auth_profile` - MongoDB
- `sqlserver_auth_profile` - SQL Server
- `rds_iam_user_auth_profile` - AWS RDS IAM

### Required Blocks

Every `cyberarksia_database_policy` resource must have:
1. At least **one** `principal` block (WHO can access)
2. At least **one** `target_database` block (WHAT they can access)
3. A `conditions` block with `max_session_duration`

## Production Best Practices

1. **Secrets Management**:
   - Use Terraform variables for sensitive data
   - Consider external secret managers (Vault, AWS Secrets Manager)
   - Never commit credentials to version control

2. **Certificate Management**:
   - Use `cert_body`, not `certificate` (correct attribute name)
   - Store certificates outside the Terraform directory
   - Rotate certificates before expiration

3. **Time Zones**:
   - Use IANA time zone names (e.g., "America/New_York")
   - Account for daylight saving time
   - Document time zone decisions

4. **Principal Lookups**:
   - Use `name` and `type` attributes (not `principal_name`/`principal_type`)
   - Verify principal names match Identity directory exactly
   - Use groups for easier access management

## Cleanup

To destroy all resources:
```bash
terraform destroy
```

**Warning**: This will revoke all database access configured through these policies!

## Troubleshooting

### Principal Not Found
```
Error: Principal not found: developers@example.com
```
**Solution**: Verify the group/user exists in CyberArk Identity and the name matches exactly.

### Missing Required Blocks
```
Error: At least 1 principal block must be defined
```
**Solution**: Add at least one `principal {}` block inside the policy resource.

### Invalid Days of Week
```
Error: days_of_the_week must be a set of integers 0-6
```
**Solution**: Use integers (0-6), not strings: `[1, 2, 3, 4, 5]` not `["Monday", "Tuesday", ...]`

### Wrong Attribute Names
```
Error: Unsupported argument: policy_name
```
**Solution**: Use `name` (not `policy_name`), `status` (not `policy_status`), `cert_body` (not `certificate`).

## Next Steps

- **Add more databases**: Extend `workspaces.tf`
- **Create custom policies**: Different teams/roles
- **Implement monitoring**: Custom validations
- **CI/CD Integration**: Automate with GitHub Actions

## Learn More

- [CyberArk SIA Documentation](https://docs.cyberark.com)
- [Provider Documentation](https://registry.terraform.io/providers/aaearon/cyberarksia/latest/docs)
- [Troubleshooting Guide](../../../docs/troubleshooting.md)
