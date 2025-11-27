#!/bin/bash
set -e

# ============================================================================
# Setup terraform.tfvars from project .env file
# ============================================================================

echo "📋 Creating terraform.tfvars from .env file..."
echo ""

# Check .env exists
if [ ! -f ~/terraform-provider-cyberarksia/.env ]; then
    echo "❌ ERROR: .env file not found at ~/terraform-provider-cyberarksia/.env"
    exit 1
fi

# Source .env file
source ~/terraform-provider-cyberarksia/.env

# Check required variables
if [ -z "$CYBERARK_USERNAME" ]; then
    echo "❌ ERROR: CYBERARK_USERNAME not set in .env"
    exit 1
fi

if [ -z "$CYBERARK_PASSWORD" ]; then
    echo "❌ ERROR: CYBERARK_PASSWORD not set in .env"
    exit 1
fi

echo "✅ Found SIA credentials in .env"
echo ""

# Prompt for remaining required variables
echo "📝 Please provide the following information:"
echo ""

read -p "Azure Subscription ID: " AZURE_SUBSCRIPTION_ID
read -p "Azure Region [westus2]: " AZURE_REGION
AZURE_REGION=${AZURE_REGION:-westus2}

read -p "PostgreSQL Admin Username [pgadmin]: " PG_USERNAME
PG_USERNAME=${PG_USERNAME:-pgadmin}

read -sp "PostgreSQL Admin Password (strong password required): " PG_PASSWORD
echo ""

read -p "Service Account Principal UUID (from SIA UI): " SERVICE_UUID
read -p "Tim Schindler Principal UUID [c2c7bcc6-9560-44e0-8dff-5be221cd37ee]: " TIM_UUID
TIM_UUID=${TIM_UUID:-c2c7bcc6-9560-44e0-8dff-5be221cd37ee}

echo ""
echo "📄 Creating terraform.tfvars..."

# Create terraform.tfvars
cat > terraform.tfvars <<EOF
# ============================================================================
# Auto-generated from .env file on $(date)
# ============================================================================

# SIA Provider Configuration (from .env)
sia_username = "$CYBERARK_USERNAME"
sia_password = "$CYBERARK_PASSWORD"

# Azure Configuration
azure_subscription_id = "$AZURE_SUBSCRIPTION_ID"
azure_region          = "$AZURE_REGION"

# PostgreSQL Configuration
postgres_admin_username = "$PG_USERNAME"
postgres_admin_password = "$PG_PASSWORD"

# Test Principal Configuration (Tim Schindler)
test_principal_id    = "$TIM_UUID"
test_principal_email = "tim.schindler@cyberark.cloud.40562"

# CyberArk Cloud Directory ID (standard)
cyberark_cloud_directory_id = "09B9A9B0-6CE8-465F-AB03-65766D33B05E"

# Service Account Principal
service_account_principal_id = "$SERVICE_UUID"
EOF

echo "✅ terraform.tfvars created successfully!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Configuration Summary:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "SIA Username:      $CYBERARK_USERNAME"
echo "Azure Subscription: $AZURE_SUBSCRIPTION_ID"
echo "Azure Region:      $AZURE_REGION"
echo "PG Admin User:     $PG_USERNAME"
echo "Service Account:   $SERVICE_UUID"
echo "Tim Schindler:     $TIM_UUID"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✅ Ready to proceed!"
echo ""
echo "Next steps:"
echo "  1. Review terraform.tfvars if needed: vim terraform.tfvars"
echo "  2. Run setup: ./setup.sh"
echo ""
