# Quick Start Guide

## Prerequisites Check

```bash
# 1. Azure login
az login
az account show

# 2. Provider built
cd ~/terraform-provider-cyberarksia
go build -v && go install

# 3. Get principal UUIDs from SIA UI
# - Service account UUID (from Identity & Access > Users)
# - Tim Schindler UUID (tim.schindler@cyberark.cloud.40562)
```

## Setup (5 minutes)

```bash
# 1. Navigate to test directory (in project, not /tmp!)
cd ~/terraform-provider-cyberarksia/examples/testing/azure-postgresql-with-policy

# 2. Create configuration
cp terraform.tfvars.example terraform.tfvars
vim terraform.tfvars
# Fill in:
#   - sia_username (from .env)
#   - sia_client_secret (from .env)
#   - azure_subscription_id
#   - postgres_admin_password (strong password!)
#   - test_principal_id (Tim's UUID from SIA UI)
#   - service_account_principal_id (your UUID from SIA UI)

# 3. Run setup
./setup.sh
```

## What Gets Created

**Azure** (~10 minutes, ~$0.017/hour):
- Resource group: `rg-sia-policy-test`
- PostgreSQL B1ms: `psql-sia-policy-XXXXXX`
- Database: `testdb`
- Firewall rules: Azure services + all IPs

**SIA** (~1 minute):
- Secret: PostgreSQL admin credentials
- Database workspace: Azure PostgreSQL (cert validation DISABLED)
- Policy: With inline service account principal
- Principal assignment: Tim Schindler (separate resource)

## Verification Steps

### 1. Check Terraform State
```bash
terraform state list
# Should show 11 resources
```

### 2. Check Outputs
```bash
terraform output validation_summary
```

### 3. SIA UI Verification
Navigate to policy (name in outputs):
- **Assigned To**: Should show 2 principals
  * Service account (inline)
  * tim.schindler@cyberark.cloud.40562 (separate resource)
- **Targets**: Should show Azure PostgreSQL
  * Target set: "FQDN/IP" (all cloud providers use this)
  * Auth method: `db_auth`
  * Roles: `["pg_read_all_settings"]`

### 4. Azure Portal Verification
- Resource group exists
- PostgreSQL server running
- FQDN resolves

### 5. Test Access (Optional)
- Log into SIA portal as Tim Schindler
- Request access to policy
- Verify conditions:
  * Access window: Monday-Friday, 09:00-17:00 GMT
  * Max session: 4 hours
  * Idle timeout: 10 minutes

## Verify No Drift

```bash
terraform plan
# Expected: "No changes. Your infrastructure matches the configuration."
```

## Manual Verification Complete? Clean Up!

```bash
./cleanup.sh
```

**Verification**:
```bash
# State should be empty
terraform state list

# Azure resources deleted
az postgres flexible-server list --query "[?name contains 'sia-policy'].name"
# Should return: []
```

## If Something Goes Wrong

### Terraform Logs
All operations write to `/tmp/`:
```bash
cat /tmp/tf-init.log    # Initialization
cat /tmp/tf-plan.log    # Planning
cat /tmp/tf-apply.log   # Creation
cat /tmp/tf-destroy.log # Deletion
```

### Common Issues

**LocationIsOfferRestricted**:
```bash
# Change region in terraform.tfvars
azure_region = "westus2"
```

**Principal UUID Not Found**:
```bash
# Verify UUIDs in SIA UI:
# Identity & Access > Users > Search for user > Copy UUID
```

**Policy Not Found**:
```bash
# Wait a few seconds and retry
terraform refresh
terraform plan
```

## File Structure

```
~/terraform-provider-cyberarksia/examples/testing/azure-postgresql-with-policy/
├── main.tf                   # All resources
├── variables.tf              # Variable declarations
├── outputs.tf                # Comprehensive outputs
├── terraform.tfvars          # Your credentials (git-ignored)
├── setup.sh                  # Automated setup (efficient logging)
├── cleanup.sh                # Automated cleanup (with verification)
├── README.md                 # Detailed documentation
├── QUICK-START.md            # This file
└── CONTEXT-OPTIMIZATION.md   # Context-efficient patterns
```

## Key Design Decisions

1. **No timestamp()**: Azure resources use static tags to avoid unnecessary updates
2. **Certificate validation DISABLED**: Simplifies testing (not for production!)
3. **Hybrid principal pattern**: Service account inline + Tim via separate resource
4. **All logs to /tmp/**: Saves LLM context (98% reduction vs reading full output)

## Cost Management

- **Hourly**: $0.017 USD (B1ms SKU)
- **Test duration**: ~15-30 minutes
- **Total cost**: < $0.01 USD

**Monitor costs**:
```bash
az postgres flexible-server show \
  --resource-group rg-sia-policy-test \
  --name psql-sia-policy-XXXXXX \
  --query sku.name
```

**Always clean up when done**:
```bash
./cleanup.sh
```

## Success Criteria

- [x] All Azure resources created
- [x] All SIA resources created
- [x] Policy has 2 principals (service account + Tim)
- [x] Policy has 1 target (Azure PostgreSQL)
- [x] No drift detected (`terraform plan` shows no changes)
- [x] SIA UI matches Terraform state
- [x] Cost < $0.01 USD
- [x] Clean cleanup (no orphaned resources)

## Next Steps After Verification

Once you've verified everything works:

1. **Document Results**:
   - Screenshot SIA UI (policy with principals and targets)
   - Note any issues or observations
   - Save test outputs

2. **Clean Up**:
   ```bash
   ./cleanup.sh
   ```

3. **Review**:
   - Check TESTING-GUIDE.md for comprehensive testing patterns
   - Review CONTEXT-OPTIMIZATION.md for efficient LLM workflows
   - Consider adding to CI/CD pipeline

## Getting Help

- **TESTING-GUIDE.md**: Comprehensive testing framework
- **README.md**: Detailed setup and troubleshooting
- **CONTEXT-OPTIMIZATION.md**: Efficient terraform operations
- **Logs**: `/tmp/tf-*.log` files for debugging
