# Testing Guide

Comprehensive testing guide for terraform-provider-cyberarksia.

## Quick Start

### Prerequisites

1. Valid SIA credentials (username and password)
2. Provider built and installed: `go install`
3. Test certificates available (can generate with `openssl`)

### Running Tests

#### Unit Tests

```bash
# Run all unit tests
go test ./...

# Run specific package
go test ./internal/provider/helpers/... -v
```

#### Acceptance Tests

```bash
# Set environment variable
export TF_ACC=1

# Run all acceptance tests
go test ./... -v

# Run specific resource tests
go test ./internal/provider -run TestPolicyDatabaseAssignment -v
```

## CRUD Testing Framework

**CANONICAL REFERENCE**: [`examples/testing/TESTING-GUIDE.md`](examples/testing/TESTING-GUIDE.md)

For comprehensive CRUD testing of all resources, follow the guide at `examples/testing/TESTING-GUIDE.md`. This includes:

- Step-by-step testing workflow (CREATE → READ → UPDATE → DELETE)
- Test configuration templates
- Validation checklists
- Resource dependency patterns
- Troubleshooting procedures

### Quick CRUD Test

```bash
# 1. Create test directory
mkdir -p /tmp/sia-crud-validation
cd /tmp/sia-crud-validation

# 2. Generate test certificate
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out test-cert.pem \
  -days 365 -nodes -subj "/CN=crud-test-cert/O=Testing/C=US"

# 3. Copy templates from examples/testing/
cp ~/terraform-provider-cyberarksia/examples/testing/crud-test-*.tf .

# 4. Edit provider.tf with your credentials

# 5. Run CRUD cycle
terraform init
terraform plan
terraform apply -auto-approve
terraform show  # Verify state
terraform destroy -auto-approve
```

## Writing Tests

### Test Philosophy

- **Primary**: Acceptance tests (test against real SIA API)
- **Selective**: Unit tests for complex validators and helpers
- **No Mocks**: Prefer real integration tests over mocks

### Test Structure

```go
func TestResourceName_CRUD(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: testAccResourceConfig_basic(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("cyberarksia_resource.test", "id"),
                    resource.TestCheckResourceAttr("cyberarksia_resource.test", "name", "test"),
                ),
            },
            // Update
            {
                Config: testAccResourceConfig_updated(),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("cyberarksia_resource.test", "name", "updated"),
                ),
            },
            // Import
            {
                ResourceName:      "cyberarksia_resource.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

## Troubleshooting

For common testing issues and solutions, see [`docs/troubleshooting.md`](docs/troubleshooting.md).

### Common Issues

**Authentication Errors**:
```bash
# Verify credentials are set correctly
echo $CYBERARK_USERNAME
echo $CYBERARK_PASSWORD  # Should show value
```

**Provider Not Found**:
```bash
# Rebuild and install provider locally
go install
```

**API Rate Limiting**:
- Tests include retry logic with exponential backoff (3 retries, 500ms-30s delays)
- If persistent, add delays between test runs

## Documentation

- **CRUD Testing**: [`examples/testing/TESTING-GUIDE.md`](examples/testing/TESTING-GUIDE.md) - Comprehensive CRUD testing framework
- **Testing Framework**: [`docs/testing-framework.md`](docs/testing-framework.md) - Conceptual overview
- **Troubleshooting**: [`docs/troubleshooting.md`](docs/troubleshooting.md) - Common issues and solutions

## Contributing

When adding new resources:

1. ✅ Implement full CRUD operations
2. ✅ Add acceptance tests following existing patterns
3. ✅ Test against real SIA API (not mocks)
4. ✅ Update `examples/testing/TESTING-GUIDE.md` with new resource patterns
5. ✅ Document resource in `docs/resources/`
6. ✅ Add examples in `examples/resources/`

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for full contributor guidelines.
