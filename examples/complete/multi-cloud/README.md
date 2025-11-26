# Multi-Cloud VM Policy Examples

VM access policies for AWS, Azure, GCP, and on-premises infrastructure.

## Prerequisites

1. CyberArk Identity tenant with SIA enabled
2. Service account with VM policy management permissions
3. At least one user/group in your identity directory

## Quick Start

```bash
# Set credentials
export CYBERARK_USERNAME="your-service-account@cyberark.cloud.12345"
export CYBERARK_PASSWORD="your-password"

# Initialize
terraform init

# Deploy AWS policy
terraform apply -var="principal_name=admin@example.com" -var="aws_account_id=123456789012"

# Or deploy on-premises policy
terraform apply -var="principal_name=admin@example.com" -var='fqdn_targets=["*.prod.example.com"]'
```

## Files

| File | Cloud | Description |
|------|-------|-------------|
| `provider.tf` | All | Provider config and shared principal lookup |
| `aws-vm-policy.tf` | AWS | Target by account ID and VPC |
| `azure-vm-policy.tf` | Azure | Target by subscription and resource group |
| `gcp-vm-policy.tf` | GCP | Target by project and VPC network |
| `onprem-vm-policy.tf` | On-prem | Target by FQDN, hostname, IP, or CIDR |

## Target Format Reference

### AWS

```hcl
aws_targets {
  account_ids = ["123456789012"]          # 12-digit account ID
  vpc_ids     = ["vpc-0a1b2c3d4e5f6789a"] # Optional: vpc- prefix
}
```

### Azure

**Important**: Use full ARM resource paths, not just names.

```hcl
azure_targets {
  subscription_ids = ["759a039e-dc44-4762-9f40-2696323c2fa5"]
  resource_groups  = ["/subscriptions/759a039e-.../resourceGroups/my-rg"]
}
```

### GCP

```hcl
gcp_targets {
  project_ids     = ["my-project-123"]
  vpc_network_ids = ["projects/my-project-123/global/networks/my-vpc"]
}
```

### On-Premises (FQDN/IP)

```hcl
fqdn_targets = [
  "server1.example.com",     # Specific host
  "*.prod.example.com",      # Wildcard
  "192.168.1.100",           # IP address
  "10.0.0.0/24"              # CIDR range
]
```

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid azure resource group` | Using name instead of ARM path | Use full `/subscriptions/.../resourceGroups/name` path |
| `principal not found` | User/group doesn't exist | Verify principal exists in CyberArk Identity |
| `invalid AWS account ID` | Wrong format | Must be exactly 12 digits |
