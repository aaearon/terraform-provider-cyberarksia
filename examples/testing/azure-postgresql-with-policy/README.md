# Azure PostgreSQL + SIA Database Policy Test Configuration

This Terraform configuration creates a comprehensive test environment for validating CyberArk SIA database policy management with a real Azure PostgreSQL Flexible Server.

## What This Test Creates

### Azure Infrastructure
- ✅ Resource Group
- ✅ PostgreSQL Flexible Server (B1ms SKU - ~$0.017/hour)
- ✅ Test Database (`testdb`)
- ✅ Firewall Rules (Azure services + all IPs for testing)

### CyberArk SIA Resources
- ✅ Secret (PostgreSQL admin credentials)
- ✅ Database Workspace (Azure PostgreSQL connection with **certificate validation DISABLED**)
- ✅ Database Policy (with inline service account principal + inline database target)
- ✅ Principal Assignment (Tim Schindler added as separate resource)

## Prerequisites

1. **Azure CLI authenticated**: `az login && az account show`
2. **Valid Azure subscription** with PostgreSQL permissions
3. **SIA credentials** from project root `.env` file
4. **Provider built**: `cd ~/terraform-provider-cyberarksia && go build -v && go install`
5. **Principal UUIDs** from SIA UI (see below)

## Getting Principal UUIDs

You need two UUIDs from the SIA UI:

### 1. Service Account UUID
```bash
# This is the UUID of the account in sia_username variable
# 1. Log into SIA UI
# 2. Navigate to: Identity & Access > Users
# 3. Search for your service account email
# 4. Copy the UUID from user details
```

### 2. Tim Schindler's UUID
```bash
# 1. Log into SIA UI
# 2. Navigate to: Identity & Access > Users
# 3. Search for: tim.schindler@cyberark.cloud.40562
# 4. Copy the UUID from user details
```

## Setup Instructions

### Step 1: Navigate to Test Directory
```bash
cd ~/terraform-provider-cyberarksia/examples/testing/azure-postgresql-with-policy
```

### Step 2: Create Configuration
```bash
cp terraform.tfvars.example terraform.tfvars
```

### Step 3: Edit terraform.tfvars
```bash
# Edit with your actual values
vim terraform.tfvars
```

Fill in:
- `sia_username` - From project `.env` file
- `sia_client_secret` - From project `.env` file
- `azure_subscription_id` - Your Azure subscription ID
- `postgres_admin_password` - Strong password for PostgreSQL
- `test_principal_id` - Tim Schindler's UUID from SIA UI
- `service_account_principal_id` - Your service account UUID from SIA UI

### Step 4: Run Automated Setup (Recommended)
```bash
./setup.sh
```

This script will:
- Initialize Terraform
- Create a plan
- Apply the configuration (with confirmation)
- Display summary and next steps
- **All Terraform output goes to /tmp/ logs** (context-efficient!)

OR manually:

### Step 4 (Alternative): Manual Setup
```bash
# Initialize
terraform init

# Review plan
terraform plan

# Apply
terraform apply
```

Expected resources:
- 7 Azure resources (resource group, PostgreSQL server, database, firewall rules)
- 4 SIA resources (secret, database workspace, policy, principal assignment)

### Step 5: Apply Configuration
```bash
terraform apply
```

**Duration**: 5-10 minutes (PostgreSQL provisioning is the slowest part)

## Validation Steps

After `terraform apply` completes, follow the steps in the `next_steps` output:

### 1. SIA UI Verification
- Navigate to the policy (name shown in outputs)
- **Assigned To** section should show:
  * Service account (inline principal)
  * Tim Schindler (separate principal assignment)
- **Targets** section should show:
  * Azure PostgreSQL database
  * FQDN/IP target set (regardless of cloud_provider)
  * Authentication: `db_auth` with roles `["pg_read_all_settings"]`

### 2. Test Access
- Log into SIA portal as Tim Schindler
- Request access to the policy
- Verify access conditions:
  * Access window: Monday-Friday, 09:00-17:00 GMT
  * Max session: 4 hours
  * Idle timeout: 10 minutes

### 3. Azure Portal Verification
- Check resource group created
- Verify PostgreSQL server running
- Confirm FQDN resolves

### 4. Cost Check
```bash
# Verify Azure costs
az postgres flexible-server show \
  --resource-group $(terraform output -raw azure_resource_group) \
  --name $(terraform output -raw azure_postgres_server_name) \
  --query sku.name

# Should show: B_Standard_B1ms (~$0.017/hour)
```

## Key Design Decisions

### 1. Certificate Validation DISABLED
As requested, `enable_certificate_validation = false` and no `certificate_id` is provided. This simplifies testing but **should not be used in production**.

### 2. Static Tags/Attributes Only
Following TESTING-GUIDE.md warning, **NO `timestamp()` functions** are used on Azure resources. This prevents unnecessary updates on every `terraform apply`.

### 3. Hybrid Principal Assignment Pattern
- Service account assigned **inline** in policy resource
- Tim Schindler assigned via **separate resource** (`cyberarksia_database_policy_principal_assignment`)
- Policy has `lifecycle { ignore_changes = [principal] }` to allow this pattern

### 4. All Cloud Providers Use "FQDN/IP" Target Set
The guide confirms that ALL database workspaces (AWS, Azure, GCP, on-premise) use `"FQDN/IP"` target set in policies, regardless of the `cloud_provider` attribute. The `cloud_provider` field is **metadata only**.

## Cleanup

⚠️ **IMPORTANT**: You requested to be asked before destroying anything for manual verification.

When you're ready to clean up:

```bash
# Delete in reverse dependency order
terraform destroy -target=cyberarksia_database_policy_principal_assignment.tim_schindler
terraform destroy -target=cyberarksia_database_policy.test
terraform destroy -target=cyberarksia_database_workspace.azure_postgres
terraform destroy -target=cyberarksia_database_secret.admin

# Delete Azure infrastructure
terraform destroy
```

Or destroy everything at once:
```bash
terraform destroy -auto-approve
```

### Verify Cleanup
```bash
# Check SIA UI - no orphaned resources
# Check Azure Portal - resource group deleted

# Verify with Azure CLI
az postgres flexible-server list \
  --query "[?name contains 'sia-policy'].name"
# Should return: []
```

## Troubleshooting

### Issue: LocationIsOfferRestricted
```
Error: Subscriptions are restricted from provisioning in location 'eastus'
```
**Solution**: Change `azure_region` in `terraform.tfvars` to `"westus2"` or check allowed regions:
```bash
az account list-locations --query "[].name"
```

### Issue: Policy Not Found
```
Error: Policy not found: SIA-Policy-Test-XXXXXX
```
**Solution**: Policy may still be creating. Wait a few seconds and retry.

### Issue: Principal UUID Not Found
```
Error: Principal not found in directory
```
**Solution**: Verify the UUIDs in `terraform.tfvars` are correct by checking the SIA UI.

## Test Results

After testing, document results:

- [ ] All Azure resources created successfully
- [ ] Secret onboarded to SIA
- [ ] Database workspace created (certificate validation disabled)
- [ ] Policy created with inline service account principal
- [ ] Policy has inline Azure PostgreSQL target
- [ ] Tim Schindler principal assignment created
- [ ] SIA UI shows 2 principals assigned to policy
- [ ] SIA UI shows Azure PostgreSQL in targets section
- [ ] Access conditions correct (Monday-Friday, 09:00-17:00)
- [ ] No drift detected: `terraform plan` shows no changes
- [ ] Total cost < $0.01 USD

## Files

```
/tmp/sia-azure-policy-test/
├── main.tf                      # All resource definitions
├── variables.tf                 # Variable declarations
├── outputs.tf                   # Comprehensive outputs
├── terraform.tfvars.example     # Template for credentials
├── terraform.tfvars             # Your actual credentials (git-ignored)
└── README.md                    # This file
```

## See Also

- [TESTING-GUIDE.md](~/terraform-provider-cyberarksia/examples/testing/TESTING-GUIDE.md) - Canonical testing reference
- [Phase 6 Documentation](~/terraform-provider-cyberarksia/examples/testing/TESTING-GUIDE.md#phase-6-create---sia-database-policy-with-inline-assignments) - Policy creation with inline assignments
- [Phase 7 Documentation](~/terraform-provider-cyberarksia/examples/testing/TESTING-GUIDE.md#phase-7-optional-create---additional-principal-via-assignment-resource) - Separate principal assignments
