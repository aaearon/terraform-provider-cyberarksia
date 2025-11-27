#!/bin/bash
set -e

# ============================================================================
# Azure PostgreSQL + SIA Policy Test Setup Script
# ============================================================================

echo "🚀 Starting Azure PostgreSQL + SIA Policy Test Setup"
echo ""

# ============================================================================
# Validation
# ============================================================================

echo "📋 Validating prerequisites..."

# Check terraform.tfvars exists
if [ ! -f "terraform.tfvars" ]; then
    echo "❌ ERROR: terraform.tfvars not found!"
    echo ""
    echo "Please create terraform.tfvars from the example:"
    echo "  cp terraform.tfvars.example terraform.tfvars"
    echo "  vim terraform.tfvars"
    echo ""
    exit 1
fi

# Check Azure CLI
if ! command -v az &> /dev/null; then
    echo "❌ ERROR: Azure CLI not found!"
    echo "Install: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check Azure login
if ! az account show &> /dev/null; then
    echo "❌ ERROR: Not logged into Azure!"
    echo "Run: az login"
    exit 1
fi

# Check provider installed
if ! terraform providers 2>/dev/null | grep -q "terraform.local/local/cyberarksia"; then
    echo "⚠️  Provider may not be installed. Run:"
    echo "  cd ~/terraform-provider-cyberarksia"
    echo "  go build -v && go install"
fi

echo "✅ Prerequisites validated"
echo ""

# ============================================================================
# Terraform Init
# ============================================================================

echo "📦 Initializing Terraform..."
terraform init -upgrade > /tmp/tf-init.log 2>&1

if [ $? -eq 0 ]; then
    echo "✅ Terraform initialized"
else
    echo "❌ Terraform init failed! Check /tmp/tf-init.log"
    exit 1
fi
echo ""

# ============================================================================
# Terraform Plan
# ============================================================================

echo "🔍 Running Terraform plan..."
terraform plan -out=tfplan > /tmp/tf-plan.log 2>&1

if [ $? -eq 0 ]; then
    echo "✅ Plan created successfully"
    echo ""
    echo "📊 Plan Summary:"
    grep -E "Plan:|No changes" /tmp/tf-plan.log || echo "See /tmp/tf-plan.log for details"
else
    echo "❌ Plan failed! Check /tmp/tf-plan.log"
    tail -20 /tmp/tf-plan.log
    exit 1
fi
echo ""

# ============================================================================
# Confirm Apply
# ============================================================================

echo "⚠️  Ready to create resources. This will:"
echo "   - Create Azure PostgreSQL Flexible Server (B1ms)"
echo "   - Create SIA secret, database workspace, and policy"
echo "   - Add 2 principal assignments (service account + Tim Schindler)"
echo "   - Cost: < $0.01 USD for test duration"
echo ""
read -p "Continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "❌ Aborted by user"
    exit 0
fi
echo ""

# ============================================================================
# Terraform Apply (with progress tracking)
# ============================================================================

echo "🏗️  Creating resources (this takes 5-10 minutes)..."
echo "   Terraform output: /tmp/tf-apply.log"
echo ""

# Run apply in background and track progress
terraform apply -auto-approve tfplan > /tmp/tf-apply.log 2>&1 &
TF_PID=$!

# Show spinner while waiting
spin='-\|/'
i=0
while kill -0 $TF_PID 2> /dev/null; do
    i=$(( (i+1) %4 ))
    printf "\r   ${spin:$i:1} Creating resources... "
    sleep 0.5
done
echo ""

# Check if apply succeeded
wait $TF_PID
if [ $? -eq 0 ]; then
    echo "✅ All resources created successfully!"
else
    echo "❌ Apply failed! Check /tmp/tf-apply.log"
    tail -30 /tmp/tf-apply.log
    exit 1
fi
echo ""

# ============================================================================
# Display Outputs
# ============================================================================

echo "📊 Resource Summary:"
echo ""
terraform output -json | jq -r '.validation_summary.value | to_entries[] | "\(.key): \(.value)"' 2>/dev/null || terraform output validation_summary
echo ""

echo "✅ Setup Complete!"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 Next Steps:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
terraform output -raw next_steps
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "⚠️  IMPORTANT: Resources are running and incurring costs!"
echo "   When ready to clean up, run: ./cleanup.sh"
echo ""
