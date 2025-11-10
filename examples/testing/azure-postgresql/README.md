# Azure PostgreSQL + CyberArk SIA Integration Test

This directory contains a comprehensive end-to-end test of the `terraform-provider-cyberarksia` using an Azure PostgreSQL Flexible Server.

## Test Overview

**Objective**: Validate full CRUD operations of the SIA Terraform provider with a real Azure database resource.

**Infrastructure Created**:
1. **Azure Resources**:
   - Resource Group
   - PostgreSQL Flexible Server (B1ms - cheapest tier: ~$0.017/hour)
   - Firewall rules (Azure services + public access)
   - Test database

2. **SIA Resources**:
   - Secret (PostgreSQL admin credentials)
   - Database Workspace (Azure PostgreSQL, cloud_provider=azure)

## Prerequisites

- Azure CLI authenticated (`az account show`)
- CyberArk SIA credentials (in `.env` or terraform.tfvars)
- Terraform provider built (`go install` in provider directory)

## Quick Start

### 1. Initialize Terraform
```bash
terraform init
```

### 2. Review Plan
```bash
terraform plan
```

### 3. Deploy Infrastructure
```bash
terraform apply -auto-approve
```

Expected deployment time: **5-10 minutes** (Azure PostgreSQL provisioning)

### 4. Validate Resources

Check outputs:
```bash
terraform output
```

Test connectivity:
```bash
# DNS lookup
nslookup $(terraform output -raw postgres_server_fqdn)

# Port check (requires nc/netcat)
nc -zv $(terraform output -raw postgres_server_fqdn) 5432

# Direct PostgreSQL connection (requires psql)
terraform output -raw test_commands | jq -r '.psql_direct' | bash
```

### 5. CRUD Testing

**READ** - Refresh state:
```bash
terraform plan  # Should show no changes
terraform refresh
```

**UPDATE** - Modify database workspace:
```bash
# Edit main.tf - change tags in cyberarksia_database_workspace resource
terraform apply -auto-approve
```

**IMPORT** - Test import functionality:
```bash
# Remove from state
terraform state rm cyberarksia_database_workspace.azure_postgres

# Re-import using ID
terraform import cyberarksia_database_workspace.azure_postgres $(terraform output -raw sia_database_workspace_id)

# Verify
terraform plan  # Should show no changes
```

### 6. Cleanup

**Destroy all resources**:
```bash
terraform destroy -auto-approve
```

Estimated cleanup time: **2-3 minutes**

## Cost Management

**Estimated Costs**:
- Hourly: ~$0.017 USD (B1ms compute) + $0.006 storage = **$0.023/hour**
- 2-hour test: **~$0.05 USD**
- 24 hours: **~$0.55 USD**

**Cost-Saving Tips**:
1. **Stop server when not in use** (eliminates compute cost):
   ```bash
   az postgres flexible-server stop --name $(terraform output -raw postgres_server_name) --resource-group $(terraform output -raw azure_resource_group)
   ```

2. **Use Azure Free Tier** (if eligible):
   - 750 hours/month of B1ms instance
   - 32 GB storage included
   - **Total cost: $0.00**

3. **Delete immediately after testing**:
   ```bash
   terraform destroy -auto-approve
   ```

## Validation Checklist

- [ ] Azure PostgreSQL server provisions successfully
- [ ] Server is publicly accessible (firewall rules configured)
- [ ] SIA secret creates with admin credentials
- [ ] SIA database workspace creates with `cloud_provider=azure`
- [ ] Database workspace shows correct FQDN and region
- [ ] Terraform state matches remote API (no drift)
- [ ] UPDATE operations work (modify tags/description)
- [ ] IMPORT functionality works
- [ ] DESTROY cleans up all resources

## Troubleshooting

### PostgreSQL provisioning fails
```bash
# Check Azure quota
az postgres flexible-server list --query "[?sku.tier=='Burstable']" -o table

# Verify region availability
az postgres flexible-server list-skus --location eastus --query "[?name=='B_Standard_B1ms']"
```

### SIA cannot connect to PostgreSQL
```bash
# Verify firewall rules
az postgres flexible-server firewall-rule list --resource-group $(terraform output -raw azure_resource_group) --name $(terraform output -raw postgres_server_name) -o table

# Check DNS resolution
nslookup $(terraform output -raw postgres_server_fqdn)

# Test connectivity from local machine
psql "host=$(terraform output -raw postgres_server_fqdn) port=5432 dbname=postgres user=siaadmin password=<password> sslmode=require"
```

### Terraform provider errors
```bash
# Rebuild provider
cd /home/tim/terraform-provider-cyberarksia
go install
terraform init -upgrade
```

## Test Results

Record your test results in `/home/tim/terraform-provider-cyberarksia/docs/testing/azure-integration-test-results.md`.

## Files

- `main.tf` - Main Terraform configuration (Azure + SIA resources)
- `variables.tf` - Input variable definitions
- `terraform.tfvars` - Variable values (sensitive, gitignored)
- `outputs.tf` - Resource outputs and validation summary
- `.gitignore` - Exclude sensitive files from version control

## Cleanup Script

Quick cleanup script (run after testing):
```bash
#!/bin/bash
cd /tmp/sia-azure-test-20251027-185657
terraform destroy -auto-approve
cd ~ && rm -rf /tmp/sia-azure-test-20251027-185657
```

## Next Steps

After successful testing, document results in the main provider repository:
```bash
cp /tmp/sia-azure-test-20251027-185657/README.md \
   /home/tim/terraform-provider-cyberarksia/docs/testing/azure-integration-test-results.md
```
