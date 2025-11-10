# CRUD Testing Templates

> ⚠️ **FOR CONTRIBUTORS AND TESTING ONLY**
> This directory is for provider development and validation.
> **End users**: See `examples/resources/` for typical usage patterns.

> **📖 PRIMARY DOCUMENTATION**: See [`TESTING-GUIDE.md`](./TESTING-GUIDE.md)

This directory contains **canonical testing templates** for the CyberArk SIA Terraform provider.

## Quick Links

- 📖 **[TESTING-GUIDE.md](./TESTING-GUIDE.md)** - Complete testing guide (START HERE)
- 📄 **crud-test-main.tf** - Resource definitions template
- 📄 **crud-test-outputs.tf** - Validation outputs template
- 📄 **crud-test-provider.tf** - Provider configuration template

## Usage

**Always follow TESTING-GUIDE.md** - do not create ad-hoc test configurations.

```bash
# Copy templates to working directory
mkdir -p /tmp/sia-crud-validation
cd /tmp/sia-crud-validation
cp ~/terraform-provider-cyberarksia/examples/testing/crud-test-*.tf .

# Follow TESTING-GUIDE.md for complete workflow
```

## What This Tests

✅ **Create** - All 4 resources created with correct attributes
✅ **Read** - State refresh and drift detection
✅ **Update** - In-place updates without replacement
✅ **Delete** - Clean deletion with dependency handling

## Resources Covered

1. **Certificate** - TLS/mTLS certificates
2. **Secret** - Database credentials
3. **Database Workspace** - Database connection configurations
4. **Policy Database Assignment** - Assign databases to access policies

## Notes

- Uses real SIA API (not mocks)
- Tests all four resources and their dependencies
- Validates computed fields are populated
- Verifies schema correctness
- Tests dependency relationships

For complete instructions, see **[TESTING-GUIDE.md](./TESTING-GUIDE.md)**.
