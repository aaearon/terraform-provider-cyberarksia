# Design Decisions & Technical Reference

Key technical decisions, SDK limitations, and breaking changes for terraform-provider-cyberark-sia.

**Last Updated**: 2025-10-29

## Active Technologies

- **Go**: 1.25.0
- **ARK SDK**: github.com/cyberark/ark-sdk-golang v1.5.0
- **Terraform Plugin Framework**: v1.16.1 (Plugin Framework v6)
- **Terraform Plugin Log**: v0.9.0

## Database Workspace Field Mappings

| Terraform Attribute | SDK Field | Required? | Notes |
|---------------------|-----------|-----------|-------|
| `name` | `Name` | ✅ Required | Database name on server (e.g., "customers", "myapp") |
| `database_type` | `ProviderEngine` | ✅ Required | 60+ engine types: postgres, mysql, postgres-aws-rds, etc. |
| `network_name` | `NetworkName` | Optional | Network segmentation (default: "ON-PREMISE") |
| `address` | `ReadWriteEndpoint` | Optional | Hostname/IP/FQDN |
| `port` | `Port` | Optional | SDK uses family defaults |
| `auth_database` | `AuthDatabase` | Optional | MongoDB auth database (default: "admin") |
| `services` | `Services` | Optional | Oracle/SQL Server services ([]string) |
| `account` | `Account` | Optional | Snowflake/Atlas account name |
| `authentication_method` | `ConfiguredAuthMethodType` | Optional | ad_ephemeral_user, local_ephemeral_user, rds_iam_authentication, atlas_ephemeral_user |
| `secret_id` | `SecretID` | ✅ Required | Links to cyberarksia_database_secret resource for ZSP/JIT access |
| `enable_certificate_validation` | `EnableCertificateValidation` | Optional | Enforce TLS cert validation (default: true) |
| `certificate_id` | `Certificate` | Optional | TLS/mTLS certificate reference |
| `cloud_provider` | `Platform` | Optional | aws, azure, gcp, on_premise, atlas |
| `region` | `Region` | Optional | **Required for RDS IAM auth** |
| `read_only_endpoint` | `ReadOnlyEndpoint` | Optional | Read replica endpoint |
| `tags` | `Tags` | Optional | Key-value metadata |

**Not Exposed**: Active Directory domain controller fields (6 fields available in SDK)

## Known ARK SDK Limitations (v1.5.0)

1. **No Context Support**: `Authenticate()` doesn't accept `context.Context`
2. **No Structured Errors**: Returns generic `error` interface
3. **No HTTP Status Codes**: Status codes embedded in error strings
4. **Token Expiration**: 15-minute bearer tokens (SDK handles refresh)
5. **DELETE Panic Bug**: `DeleteDatabase()` and `DeleteSecret()` cause nil pointer panic (WORKAROUND IMPLEMENTED)

## DELETE Panic Bug Workaround (v1.5.0)

**Bug**: ARK SDK v1.5.0's `DeleteDatabase()` and `DeleteSecret()` methods pass `nil` body to HTTP DELETE requests, causing panic in `doRequest()`.

**Root Cause** (`pkg/common/ark_client.go:556-576`):
```go
var bodyBytes *bytes.Buffer  // Defaults to nil
if body != nil {
    bodyBytes = bytes.NewBuffer(json.Marshal(body))
}
req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyBytes)  // Panic if nil!
```

**Workaround**:
- **File**: `internal/client/delete_workarounds.go`
- **Functions**: `DeleteDatabaseWorkspaceDirect()`, `DeleteSecretDirect()`
- **Pattern**: Create temporary ISP client, call `client.Delete()` with `map[string]string{}` instead of `nil`
- **Why It Works**: Empty map JSON-marshals to `"{}"`, creating valid `bytes.Buffer` → no panic

**TODO**: Remove workaround when ARK SDK v1.6.0+ fixes nil body handling in `doRequest()`.

## Certificate Resource Changes (Breaking - 2025-10-25)

**Removed Fabricated Fields**: The following attributes were removed as they don't exist in the CyberArk SIA Certificates API:
- `created_by` - User who created certificate (not returned by API)
- `last_updated_by` - User who last updated (not returned by API)
- `version` - Version number (not returned by API)
- `checksum` - SHA256 hash for drift detection (not returned by API)
- `updated_time` - Last modification timestamp (not returned by API)
- `cert_password` - Password for encrypted certificates (API only supports public keys)

**Actual Certificate Attributes**:
- Core: `id`, `certificate_id`, `tenant_id`
- Input: `cert_name`, `cert_body`, `cert_description`, `cert_type`, `domain_name`, `labels`
- Computed: `expiration_date`, `metadata` (issuer, subject, valid_from, valid_to, serial_number, subject_alternative_name)

**Validation**: Python SDK's `ArkSIACertificate` class exposes 6 fields that do NOT exist in actual API responses. Our Go SDK wrapper correctly omits these.
