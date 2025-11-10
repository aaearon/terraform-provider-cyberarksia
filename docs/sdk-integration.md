# ARK SDK Integration Reference

**ARK SDK Version**: v1.5.0
**Package**: `github.com/cyberark/ark-sdk-golang`
**Last Updated**: 2025-11-10

**Purpose**: Reference guide for using ARK SDK with **implemented** resources

**Note**: This document covers database workspace and secret resources. For newer resources (target_set, virtual_machine_secret, policies), see resource-specific implementation files and specs.

**Related**: For analysis of available but **not yet implemented** SDK services, see [ark-sdk-sia-services-analysis.md](development/ark-sdk-sia-services-analysis.md)

---

## Table of Contents

- [Overview](#overview)
- [Confirmed SDK Packages](#confirmed-sdk-packages)
- [SIA Database Workspace CRUD Operations](#sia-database-workspace-crud-operations)
- [SIA Database Secrets CRUD Operations](#sia-database-secrets-crud-operations)
- [Authentication Pattern](#authentication-pattern)
- [Data Models](#data-models)
- [Error Handling](#error-handling)

---

## Overview

This document serves as a reference for Phase 3+ development, documenting the confirmed ARK SDK packages, methods, and integration patterns for SIA database and secrets management.

---

## Confirmed SDK Packages

### Authentication
```go
import (
	"github.com/cyberark/ark-sdk-golang/pkg/auth"
	authmodels "github.com/cyberark/ark-sdk-golang/pkg/models/auth"
)
```

### SIA Services
```go
import (
	"github.com/cyberark/ark-sdk-golang/pkg/services/sia"
	dbmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/workspaces/db/models"
	dbsecretsmodels "github.com/cyberark/ark-sdk-golang/pkg/services/sia/secrets/db/models"
)
```

---

## SIA Database Workspace CRUD Operations

### Add Database
```go
database, err := siaAPI.WorkspacesDB().AddDatabase(
	&dbmodels.ArkSIADBAddDatabase{
		Name:              "MyDatabase",
		ProviderEngine:    dbmodels.EngineTypeAuroraMysql, // or other engine types
		ReadWriteEndpoint: "myrds.com",
		SecretID:          secret.SecretID, // From secrets operation
	},
)
```

**Engine Types Available**:
- `EngineTypeAuroraMysql`
- `EngineTypePostgres` (likely `EngineTypePostgresql`)
- SQL Server, Oracle, MongoDB, MariaDB, Db2 (exact constants TBD)

### Get Database
```go
database, err := siaAPI.WorkspacesDB().GetDatabase(databaseID)
```

### Update Database
```go
updated, err := siaAPI.WorkspacesDB().UpdateDatabase(
	databaseID,
	&dbmodels.ArkSIADBUpdateDatabase{
		// Only changed fields
		Name: "UpdatedName",
	},
)
```

### Delete Database
```go
err := siaAPI.WorkspacesDB().DeleteDatabase(databaseID)
```

---

## SIA Database Secrets CRUD Operations

### Add Secret (Strong Account)
```go
secret, err := siaAPI.SecretsDB().AddSecret(
	&dbsecretsmodels.ArkSIADBAddSecret{
		SecretType: "username_password", // or other types
		Username:   "db_admin",
		Password:   "SecurePassword123!",
	},
)
// Returns: secret.SecretID for use in database registration
```

**Secret Types**:
- `"username_password"` - Local authentication
- `"domain"` - Active Directory (SQL Server, Db2)
- `"aws_iam"` - AWS IAM credentials (RDS)

### Get Secret
```go
secret, err := siaAPI.SecretsDB().GetSecret(secretID)
```

**Note**: Response contains **metadata only**, no sensitive credentials per SIA security model.

### Update Secret (Credential Rotation)
```go
updated, err := siaAPI.SecretsDB().UpdateSecret(
	secretID,
	&dbsecretsmodels.ArkSIADBUpdateSecret{
		Password: "NewPassword456!",
	},
)
```

**Behavior**: SIA updates credentials immediately. New sessions use updated credentials. Existing session handling is SIA's responsibility.

### Delete Secret
```go
err := siaAPI.SecretsDB().DeleteSecret(secretID)
```

---

## Authentication Pattern

### Provider Initialization (Already Implemented in Phase 2)
```go
ispAuth := auth.NewArkISPAuth(true) // Enable caching

profile := &authmodels.ArkAuthProfile{
	Username:   fmt.Sprintf("%s@cyberark.cloud.%s", clientID, tenantSubdomain),
	AuthMethod: authmodels.Identity,
	AuthMethodSettings: &authmodels.IdentityArkAuthMethodSettings{
		IdentityURL:             identityURL,
		IdentityTenantSubdomain: tenantSubdomain,
	},
}

secret := &authmodels.ArkSecret{
	Secret: clientSecret,
}

// Note: First parameter is *ArkProfile (optional, nil for default), NOT context.Context
_, err := ispAuth.Authenticate(nil, profile, secret, false, false)
```

### SIA API Client Initialization (Already Implemented in Phase 2)
```go
siaAPI, err := sia.NewArkSIAAPI(ispAuth)
```

---

## Data Models

### Database Target (Phase 3 - VALIDATED)

**ARK SDK Model**: `ArkSIADBAddDatabase` (Create), `ArkSIADBUpdateDatabase` (Update), `ArkSIADBDatabase` (Response)

**Terraform → SDK Field Mappings**:

| Terraform Attribute | SDK Field | Type | Required? | Notes |
|---------------------|-----------|------|-----------|-------|
| `name` | `Name` | string | ✅ Required | Only truly required field per SDK validate tag |
| `network_name` | `NetworkName` | string | Optional | Network segmentation. Defaults to "ON-PREMISE" |
| `database_type` | `ProviderEngine` | string | Optional | e.g., "postgres", "mysql", "mariadb" |
| `address` | `ReadWriteEndpoint` | string | Optional | Hostname/IP/FQDN |
| `port` | `Port` | int | Optional | SDK uses family defaults (PostgreSQL=5432, etc.) |
| `auth_database` | `AuthDatabase` | string | Optional | MongoDB authentication database (default: "admin") |
| `services` | `Services` | []string | Optional | Oracle/SQL Server service list |
| `account` | `Account` | string | Optional | Snowflake/MongoDB Atlas account name |
| `authentication_method` | `ConfiguredAuthMethodType` | string | Optional | Values: "ad_ephemeral_user", "local_ephemeral_user", "rds_iam_authentication", "atlas_ephemeral_user" |
| `secret_id` | `SecretID` | string | Optional | **Required for ZSP/JIT**. References secret for ephemeral access provisioning |
| `enable_certificate_validation` | `EnableCertificateValidation` | bool | Optional | Enforce TLS cert validation (default: true) |
| `certificate_id` | `Certificate` | string | Optional | Certificate ID for TLS/mTLS. Will reference cyberark_sia_certificate resource |
| `cloud_provider` | `Platform` | string | Optional | Values: "AWS", "AZURE", "GCP", "ON-PREMISE", "ATLAS" |
| `region` | `Region` | string | Optional | **Required for RDS IAM auth**. Used in AWS Signature Version 4 token generation |
| `read_only_endpoint` | `ReadOnlyEndpoint` | string | Optional | Read replica endpoint for scaling reads |
| `description` | (Not in SDK) | string | Optional | Provider-only field (not sent to API) |
| `tags` | `Tags` | map[string]string | Optional | Key-value metadata |

**Removed Fields** (Phase 3 Cleanup - Did Not Exist in SDK):
- ❌ `database_version` - No SDK equivalent
- ❌ `aws_account_id` - No SDK equivalent (generic fields only)
- ❌ `azure_tenant_id` - No SDK equivalent
- ❌ `azure_subscription_id` - No SDK equivalent

**Not Yet Exposed** (Available in SDK, Future Enhancement - Active Directory):
- `Domain` - Windows domain name
- `DomainControllerName` - Domain controller hostname
- `DomainControllerNetbios` - Domain controller NetBIOS name
- `DomainControllerUseLDAPS` - Use LDAPS (default: false)
- `DomainControllerEnableCertValidation` - Enforce DC cert validation (default: true)
- `DomainControllerLDAPSCertificate` - Certificate ID for DC TLS

### Strong Account (Secret)

**ARK SDK Model**: `ArkSIADBAddSecret`

**Confirmed Fields**:
- `SecretType` (string) - Authentication method
- `Username` (string) - Account username
- `Password` (string) - Account password (local/domain auth)

**For AWS IAM** (inferred):
- `AWSAccessKeyID` (string)
- `AWSSecretAccessKey` (string)

---

## Error Handling Patterns

### ARK SDK Error Characteristics
- **No Structured Error Types**: SDK v1.5.0 returns standard Go `error` interface
- **No HTTP Status Codes**: Status codes embedded in error strings only
- **No Error Code Constants**: No SDK-provided error categorization

### Our Error Handling Strategy
1. Use `errors.As()` for standard Go errors (`net.Error`, `context` errors)
2. Pattern match error strings for classification
3. Comprehensive fallback for unknown errors
4. See `internal/client/errors.go` for implementation

---

## Retry and Resilience Patterns

### Retryable Operations
- Network errors (`net.Error` with `Temporary()` or `Timeout()`)
- Server errors (5xx)
- Rate limiting (429)
- Context deadline exceeded

### Non-Retryable Operations
- Authentication failures (401)
- Permission errors (403)
- Not found (404)
- Validation errors (400, 422)
- Context canceled (user requested)

### Implementation
See `internal/client/retry.go` for `RetryWithBackoff()` with exponential backoff.

---

## SDK Limitations and Workarounds

### 1. No Context Support in Authenticate()
**Limitation**: `Authenticate()` first parameter is `*ArkProfile` (optional), not `context.Context`

**Workaround**: We accept `context.Context` in our wrapper (`NewISPAuth()`) for future-proofing, but cannot pass to SDK. Document limitation in code comments.

**Impact**: Cannot cancel authentication mid-flight via context.

### 2. No Structured Errors
**Limitation**: All errors returned as generic `error` interface with string messages.

**Workaround**:
- Detect standard Go error types (`net.Error`, context errors)
- Pattern match error strings (case-insensitive, ordered by specificity)
- Comprehensive fallback handling

**Impact**: Error classification may be brittle if SDK error messages change.

### 3. Token Caching Handled by SDK
**Good**: `NewArkISPAuth(true)` enables automatic token caching/refresh

**Provider Responsibility**: Ensure `MaxRetries` and `RequestTimeout` wrap SDK calls, not configure SDK's internal HTTP client

---

## Testing Strategy

### Acceptance Tests (Primary)
- Test against real SIA API when `TF_ACC=1`
- Use test credentials from environment variables
- Verify CRUD operations end-to-end

### Unit Tests (Selective)
- Complex validators only
- Error classification logic (already tested in `errors_test.go`)
- Retry logic (already tested in `retry_test.go`)

---

## References

- **ARK SDK GitHub**: https://github.com/cyberark/ark-sdk-golang
- **ARK SDK Docs**: https://cyberark.github.io/ark-sdk-golang/
- **Context7 Documentation**: Used for SDK research (see Phase 2 reflection)
- **Terraform Plugin Framework**: https://developer.hashicorp.com/terraform/plugin/framework

---

## Version History

- **2025-10-15 (Phase 2.5)**: Initial version based on Phase 2 research and Context7 examples
- **2025-10-15 (Phase 3 Cleanup)**: Validated database workspace field mappings against ARK SDK v1.5.0, removed non-existent fields, documented actual SDK requirements
