# Terraform Provider for Idira (formerly CyberArk) Secure Infrastructure Access (SIA)

A Terraform provider for managing Idira Secure Infrastructure Access (SIA) resources, enabling infrastructure-as-code workflows for database and VM/server access control with Just-In-Time privileged access.

## Features

### Database Access Management
- **Policy Management**: Create access policies with session limits and time-based restrictions
- **User & Group Assignment**: Grant access to specific users, groups, or roles - no manual UUID lookups needed
- **Principal Lookup**: Find users and groups by name across Cloud Directory, Azure AD, Active Directory
- **Database Configuration**: Configure database workspaces with certificate-based authentication
- **Database Assignment**: Connect databases to policies with 6 authentication methods
- **Certificate Management**: Manage TLS/SSL certificates for secure database connections
- **Database Secret Management**: Store database credentials (local auth, Active Directory, AWS IAM)
- **Multiple Database Engines**: PostgreSQL, MySQL, Oracle, SQL Server, MongoDB, Snowflake, and 60+ more
- **Multi-Cloud Support**: AWS RDS, Azure SQL, GCP Cloud SQL, MongoDB Atlas, and on-premise

### VM/Server Access Management
- **VM Policy Management**: Create access policies for cloud (AWS, Azure, GCP) and on-premises servers (FQDN/IP)
- **Principal Assignment**: Grant users, groups, or roles access to VM policies - inline or via separate resources
- **Connection Behaviors**: Configure SSH usernames and RDP ephemeral users (local or domain-joined)
- **Time-Based Access**: Restrict access to specific time windows and set session limits
- **VM Secret Management**: Store VM/server credentials (username/password or PAM vault references)
- **Target Set Configuration**: Define server groupings with Domain, Suffix, or Target matching patterns
- **Ephemeral Account Provisioning**: Configure temporary account naming for JIT access
- **Multi-Cloud Targeting**: AWS (VPCs, regions, accounts, tags), Azure (VNets, subscriptions, resource groups), GCP (projects, VPCs, labels)

## Requirements

- Terraform >= 1.0
- Go >= 1.25 (for development)
- Idira Identity Security Platform Shared Services tenant with SIA and Unified Access Policies (UAP) enabled
- Valid Idira service account credentials with `DpaAdmin` role

## Installation

### From Source

```bash
git clone https://github.com/aaearon/terraform-provider-cyberarksia
cd terraform-provider-cyberarksia
go build -v
```

### Local Development Installation

```bash
make install
```

This installs the provider to `~/.terraform.d/plugins/` for local testing.

## Authentication

The provider authenticates using an Idira service account. Simply provide the service account username and password - the provider handles OAuth2 authentication automatically.

**Service Account Setup:**
1. Create a service account in Idira Identity Security Platform Shared Services
2. Assign the **`DpaAdmin`** role (required for managing SIA resources)
3. Provide the username and password to the provider

**Important:** Never commit credentials to version control. Use environment variables or secure variable storage (e.g., Idira Conjur).

## Quick Start

```hcl
terraform {
  required_providers {
    cyberarksia = {
      source  = "aaearon/cyberarksia"
      version = "0.1.0"
    }
  }
}

provider "cyberarksia" {
  username = "service-account@cyberark.cloud.1234"  # Service account username
  password = var.password                           # Service account password
}

# Create a TLS certificate
resource "cyberarksia_certificate" "postgres_cert" {
  cert_name        = "prod-postgres-tls"
  cert_description = "TLS certificate for production PostgreSQL"
  cert_body        = file("certs/postgres.pem")
  cert_type        = "PEM"

  labels = {
    environment = "production"
    database    = "postgres"
  }
}

# Configure a database workspace
resource "cyberarksia_database_workspace" "prod_postgres" {
  name                          = "prod-postgres-db"
  database_type                 = "postgres"
  address                       = "prod-postgres.example.com"
  port                          = 5432
  cloud_provider                = "aws"
  region                        = "us-west-2"
  network_name                  = "PRODUCTION"
  certificate_id                = cyberarksia_certificate.postgres_cert.id
  enable_certificate_validation = true
}
```

## Documentation

- **[Examples](examples/)**: Complete configuration examples for various scenarios
- **[SDK Integration Guide](docs/sdk-integration.md)**: ARK SDK patterns and best practices
- **[Development History](docs/development-history.md)**: Architectural decisions and implementation insights
- **[Troubleshooting](docs/troubleshooting.md)**: Common issues and solutions

## Supported Resources

### Database Access Resources

#### `cyberarksia_certificate`

Manages TLS/SSL certificates for database connections.

**Features:**
- PEM and DER format support
- Automatic X.509 metadata extraction
- Label-based organization
- Version tracking and drift detection

See [examples/resources/cyberarksia_certificate/](examples/resources/cyberarksia_certificate/) for usage examples.

#### `cyberarksia_database_workspace`

Manages database workspace configurations for secure access.

**Supported Database Engines:**
- PostgreSQL (including AWS RDS, Azure Database, GCP Cloud SQL)
- MySQL/MariaDB
- Oracle
- Microsoft SQL Server
- MongoDB (including Atlas)
- Snowflake
- And 60+ more engine types

**Features:**
- Certificate-based TLS/mTLS authentication
- Multi-cloud support (AWS, Azure, GCP, Atlas, on-premise)
- Network segmentation
- Authentication method configuration

See [examples/resources/cyberarksia_database_workspace/](examples/resources/cyberarksia_database_workspace/) for usage examples.

#### `cyberarksia_database_secret`

Manages database authentication secrets for use with database workspaces.

**Supported Authentication Types:**
- Local database authentication (username/password)
- Domain authentication (Active Directory)
- AWS IAM authentication (for RDS)

**Features:**
- Secure credential storage
- Integration with database workspaces
- Support for domain-based authentication
- AWS IAM role ARN configuration

See [examples/resources/cyberarksia_database_secret/](examples/resources/cyberarksia_database_secret/) for usage examples.

#### `cyberarksia_database_policy`

Create and manage access policies that control who can access which databases and when.

**What you can do:**
- Set session limits (max duration, idle timeout)
- Restrict access to specific time windows (e.g., business hours only, Monday-Friday 9-5)
- Set policy validity periods (e.g., temporary access for Q1 2024)
- Enable or suspend policies without deleting them
- Tag policies for organization

Think of policies as the rules. The other resources assign specific users and databases to those rules.

See [docs/resources/database_policy.md](docs/resources/database_policy.md) for usage examples.

#### `cyberarksia_database_policy_principal_assignment`

Grant specific users, groups, or roles access to databases through policies.

**What you can do:**
- Assign individual users to policies
- Assign entire groups (like "Database Admins" or "Developers")
- Assign federated users from Azure AD, Okta, etc.
- No more hunting for user UUIDs - use the `cyberarksia_principal` data source to look them up by name

**Example use case:** Your security team creates a policy for production database access. You assign your DevOps team's group to that policy without needing to know any UUIDs.

See [docs/resources/database_policy_principal_assignment.md](docs/resources/database_policy_principal_assignment.md) for usage examples.

#### `cyberarksia_database_policy_workspace_assignment`

Connect database workspaces to access policies with specific authentication settings.

**What you can do:**
- Assign databases to policies with different auth methods (standard DB auth, LDAP, AWS IAM, etc.)
- Specify which database roles users get when they connect
- Works with 6 authentication methods including passwordless AWS RDS IAM

**Example use case:** You have a production PostgreSQL database and a policy for developers. This resource connects them together and specifies that users get the `readonly` role.

See [docs/resources/database_policy_workspace_assignment.md](docs/resources/database_policy_workspace_assignment.md) and [examples/resources/cyberarksia_database_policy_workspace_assignment/](examples/resources/cyberarksia_database_policy_workspace_assignment/) for usage examples.

### VM/Server Access Resources

#### `cyberarksia_vm_secret`

Manages VM/server credentials for privileged access.

**Supported Secret Types:**
- **ProvisionerUser**: Username/password stored directly in SIA
- **PCloudAccount**: Reference to PAM vault account

**Features:**
- Secure credential storage for VM/server access
- Integration with target sets for server grouping
- Support for PAM vault account references
- Credential rotation with automatic propagation

**Example use case:** Store a privileged Linux admin account that will be used across multiple server groups (production, staging, development).

See [examples/resources/cyberarksia_vm_secret/](examples/resources/cyberarksia_vm_secret/) for usage examples.

#### `cyberarksia_target_set`

Manages server/VM target groupings for Just-In-Time privileged access.

**Matching Pattern Types:**
- **Domain**: Match all servers in a domain (e.g., `*.example.com`)
- **Suffix**: Match servers with hostname suffix (e.g., `*.dc1.example.com`)
- **Target**: Match specific server hostname (e.g., `server01.example.com`)

**Features:**
- Three flexible matching patterns for server grouping
- Custom ephemeral account naming via `provision_format`
- In-place updates (rename, credential rotation, type changes)
- Certificate validation toggle for TLS/SSL
- Drift detection with external deletion handling
- Import existing target sets by name

**Example use case:** Create a target set for all production Linux servers in a datacenter using Domain pattern, reference a VM secret for credentials, and configure ephemeral account naming for audit trails.

See [examples/resources/cyberarksia_target_set/](examples/resources/cyberarksia_target_set/) for usage examples.

#### `cyberarksia_vm_policy`

Manages VM access policies that control who can access which servers and when.

**Supported Location Types:**
- **FQDN/IP**: On-premises servers matched by hostname patterns or IP addresses
- **AWS**: EC2 instances filtered by regions, VPCs, accounts, and tags
- **Azure**: Virtual machines filtered by subscriptions, VNets, resource groups, and tags
- **GCP**: Compute instances filtered by projects, VPCs, and labels

**Connection Behaviors:**
- **SSH**: Specify connection username
- **RDP**: Local ephemeral users (with group assignment) or domain-joined ephemeral users

**Features:**
- Time-based access windows (e.g., business hours only)
- Session duration limits and idle timeouts
- Policy validity periods (start/end dates)
- At least one principal required at creation (inline)
- Additional principals via `cyberarksia_vm_policy_principal_assignment`

**Example use case:** Create a policy allowing your DevOps team SSH access to AWS EC2 instances in production VPCs during business hours, with 4-hour session limits.

See [docs/resources/vm_policy.md](docs/resources/vm_policy.md) and [examples/resources/cyberarksia_vm_policy/](examples/resources/cyberarksia_vm_policy/) for usage examples.

#### `cyberarksia_vm_policy_principal_assignment`

Add additional principals to VM policies beyond the initial inline assignments.

**What you can do:**
- Add users, groups, or roles to existing VM policies
- Manage principal assignments independently from policy definitions
- Enable different teams to manage WHO vs WHAT (security team assigns users, infra team manages policies)

**Key behaviors:**
- All attributes are ForceNew (changes require destroy + recreate)
- Duplicate detection prevents re-assigning existing principals
- Removing the last principal is blocked (policies require at least one)

**Example use case:** Your security team creates VM policies. Later, you need to grant a new contractor group access without modifying the original policy definition.

See [docs/resources/vm_policy_principal_assignment.md](docs/resources/vm_policy_principal_assignment.md) and [examples/resources/cyberarksia_vm_policy_principal_assignment/](examples/resources/cyberarksia_vm_policy_principal_assignment/) for usage examples.

## Data Sources

Data sources let you look up existing resources without creating them.

### `cyberarksia_database_policy`

Look up existing access policies by name or ID.

**Why you'd use this:**
- Reference policies created outside Terraform (in the UI or by another team)
- Share policies across multiple Terraform workspaces

See [examples/data-sources/cyberarksia_database_policy/](examples/data-sources/cyberarksia_database_policy/) for usage examples.

### `cyberarksia_principal`

Look up users, groups, or roles by name - no more hunting for UUIDs.

**What you can do:**
- Find cloud users: `tim@cyberark.cloud.12345`
- Find federated users: `john.doe@company.com` (Azure AD, Okta, etc.)
- Find Active Directory users: `SchindlerT@domain.com`
- Find groups: `Database Administrators`

Returns the UUID and directory information you need for policy assignments.

**Example:**
```hcl
data "cyberarksia_principal" "db_team" {
  name = "Database Administrators"
  type = "GROUP"
}

resource "cyberarksia_database_policy_principal_assignment" "grant_access" {
  policy_id         = cyberarksia_database_policy.prod.policy_id
  principal_id      = data.cyberarksia_principal.db_team.id
  principal_type    = data.cyberarksia_principal.db_team.principal_type
  # ... other fields populated automatically from the data source
}
```

See [docs/data-sources/principal.md](docs/data-sources/principal.md) for usage examples.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding conventions, and pull request process.

### Quick Links

- **[Contributing Guide](CONTRIBUTING.md)** - Development setup and contribution guidelines
- **[Testing Guide](TESTING.md)** - Running tests and CRUD testing framework
- **[Design Decisions](docs/development/design-decisions.md)** - Active technologies, SDK limitations, breaking changes
- **[SDK Integration](docs/sdk-integration.md)** - ARK SDK patterns and field mappings
- **[Development Guidelines](CLAUDE.md)** - Code style, commands, and project structure
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues and solutions

### Building

```bash
go build -v
```

### Testing

```bash
# Run acceptance tests (requires SIA credentials)
TF_ACC=1 go test ./... -v

# Run unit tests only
go test ./internal/client/... -v
```

For comprehensive CRUD testing, see [TESTING.md](TESTING.md) and [examples/testing/TESTING-GUIDE.md](examples/testing/TESTING-GUIDE.md).

## Project Structure

```
terraform-provider-cyberarksia/
├── internal/
│   ├── client/          # ARK SDK wrappers, retry logic, error handling
│   ├── provider/        # Terraform provider and resource implementations (tests in *_test.go)
│   ├── models/          # Data models
│   └── validators/      # Custom validators
├── examples/            # Terraform HCL examples
├── docs/                # Documentation
└── specs/               # Feature specifications and planning docs
```

## Contributing

Contributions are welcome! To get started:

**Quick Setup:**
```bash
make tools-install         # Install dev tools
make pre-commit-install    # Enable automatic validation
make validate              # Verify setup works
```

**Before submitting:**
```bash
make validate              # Run all checks locally (mirrors CI)
```

**Full contributor guide:** [CONTRIBUTING.md](CONTRIBUTING.md)

**Development reference:** [CLAUDE.md](CLAUDE.md) (for LLM-assisted development)

## Acknowledgments

This provider is built on top of:
- **[Idira ARK SDK for Go](https://github.com/cyberark/ark-sdk-golang)** - Official Go SDK for Idira platform APIs. All provider API calls use this SDK for authentication, SIA workspace management, and UAP policy operations.
- [HashiCorp Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework) - Framework for building Terraform providers with type-safe schemas and state management.

The provider implements custom OAuth2 authentication flows for Idira Identity Security Platform Shared Services integration.

## Support

For issues, questions, or contributions:
- [GitHub Issues](https://github.com/aaearon/terraform-provider-cyberarksia/issues)
- [Specifications](specs/) - Feature planning and design documentation
