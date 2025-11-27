#!/bin/bash
set -e

# CRUD Testing Automation for terraform-provider-cyberarksia
#
# Purpose: Automates the CRUD validation workflow from examples/testing/TESTING-GUIDE.md
# Usage: ./scripts/test-crud-resource.sh [resource_description]
# Example: ./scripts/test-crud-resource.sh "policy-principal-assignment"
#
# This script:
# 1. Creates timestamped test directory in /tmp
# 2. Copies all CRUD test templates from examples/testing/
# 3. Runs Terraform init
# 4. Executes CREATE test (terraform apply)
# 5. Executes READ test (terraform plan - expect no changes)
# 6. Skips UPDATE test (requires manual template modification)
# 7. Executes DELETE test (terraform destroy)
# 8. Reports results

RESOURCE_DESC="${1:-crud-validation}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
TESTDIR="/tmp/sia-crud-validation-${RESOURCE_DESC}-${TIMESTAMP}"
TEMPLATE_DIR="./examples/testing"

echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  CRUD Test Automation for terraform-provider-cyberarksia          ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
echo "Resource: ${RESOURCE_DESC}"
echo "Test directory: ${TESTDIR}"
echo ""

# Check prerequisites
echo "→ Checking prerequisites..."

if [ ! -d "$TEMPLATE_DIR" ]; then
    echo "❌ ERROR: Template directory not found: ${TEMPLATE_DIR}"
    echo "   Run this script from the project root directory"
    exit 1
fi

if [ -z "$CYBERARK_USERNAME" ]; then
    echo "❌ ERROR: CYBERARK_USERNAME not set"
    echo "   See CLAUDE.md → Environment Setup"
    exit 1
fi

if [ -z "$CYBERARK_PASSWORD" ]; then
    echo "❌ ERROR: CYBERARK_PASSWORD not set"
    echo "   See CLAUDE.md → Environment Setup"
    exit 1
fi

if [ -z "$TF_ACC" ]; then
    echo "⚠️  WARNING: TF_ACC not set, acceptance tests may fail"
    echo "   Recommended: export TF_ACC=1"
fi

if ! command -v terraform &> /dev/null; then
    echo "❌ ERROR: terraform command not found"
    echo "   Install Terraform CLI: https://www.terraform.io/downloads"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Create test directory
echo "→ Creating test directory..."
mkdir -p "${TESTDIR}"
cd "${TESTDIR}"
echo "✅ Test directory created: ${TESTDIR}"
echo ""

# Copy all template files
echo "→ Copying CRUD test templates..."
TEMPLATE_COUNT=0
for template in "${TEMPLATE_DIR}"/crud-test-*.tf; do
    if [ -f "$template" ]; then
        cp "$template" .
        TEMPLATE_COUNT=$((TEMPLATE_COUNT + 1))
        echo "   Copied: $(basename "$template")"
    fi
done

if [ $TEMPLATE_COUNT -eq 0 ]; then
    echo "❌ ERROR: No template files found in ${TEMPLATE_DIR}"
    echo "   Expected files: crud-test-*.tf"
    exit 1
fi

echo "✅ Copied ${TEMPLATE_COUNT} template files"
echo ""

# Initialize Terraform
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  Step 1: Terraform Init                                            ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
terraform init
if [ $? -ne 0 ]; then
    echo ""
    echo "❌ ERROR: terraform init failed"
    exit 1
fi
echo ""
echo "✅ Terraform initialized successfully"
echo ""

# CREATE test
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  Step 2: CREATE (terraform apply)                                  ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
terraform apply -auto-approve
CREATE_EXIT=$?

if [ $CREATE_EXIT -ne 0 ]; then
    echo ""
    echo "❌ CREATE failed (exit code: ${CREATE_EXIT})"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check provider configuration in crud-test-provider.tf"
    echo "  2. Verify credentials: CYBERARK_USERNAME, CYBERARK_PASSWORD"
    echo "  3. Review error messages above"
    echo "  4. See docs/troubleshooting.md for common issues"
    echo ""
    echo "Test directory preserved for debugging: ${TESTDIR}"
    exit 1
fi

echo ""
echo "✅ CREATE successful"
echo ""

# READ test (terraform plan should show no changes)
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  Step 3: READ (terraform plan - expect no changes)                 ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
terraform plan -detailed-exitcode
PLAN_EXIT=$?

if [ $PLAN_EXIT -eq 0 ]; then
    echo ""
    echo "✅ READ successful (no drift detected)"
    READ_STATUS="✅ PASSED"
elif [ $PLAN_EXIT -eq 2 ]; then
    echo ""
    echo "⚠️  WARNING: Drift detected (plan shows changes)"
    echo ""
    echo "This indicates the provider is not correctly reading state from the API."
    echo "Common causes:"
    echo "  - Read() method not handling all attributes"
    echo "  - Type conversions causing false drift"
    echo "  - API returning different values than configured"
    echo ""
    echo "Running plan again to show drift details:"
    terraform plan
    READ_STATUS="⚠️  DRIFT"
else
    echo ""
    echo "❌ READ failed (plan error, exit code: ${PLAN_EXIT})"
    READ_STATUS="❌ FAILED"
fi
echo ""

# UPDATE test (manual - skip in automation)
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  Step 4: UPDATE (skipped - requires manual modification)           ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
echo "⚠️  UPDATE testing requires manual modification of template files"
echo "   See examples/testing/TESTING-GUIDE.md for manual UPDATE workflow"
echo ""
echo "To test UPDATE manually:"
echo "  1. cd ${TESTDIR}"
echo "  2. Modify resource attributes in crud-test-*.tf files"
echo "  3. Run: terraform apply"
echo "  4. Run: terraform plan (verify no drift)"
echo ""
UPDATE_STATUS="⏭️  SKIPPED"

# DELETE test
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  Step 5: DELETE (terraform destroy)                                ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
terraform destroy -auto-approve
DELETE_EXIT=$?

if [ $DELETE_EXIT -ne 0 ]; then
    echo ""
    echo "❌ DELETE failed (exit code: ${DELETE_EXIT})"
    echo ""
    echo "Resources may still exist in CyberArk SIA tenant!"
    echo "Manual cleanup may be required."
    DELETE_STATUS="❌ FAILED"
else
    echo ""
    echo "✅ DELETE successful"
    DELETE_STATUS="✅ PASSED"
fi
echo ""

# Summary
echo "╔════════════════════════════════════════════════════════════════════╗"
echo "║  CRUD Test Summary                                                  ║"
echo "╚════════════════════════════════════════════════════════════════════╝"
echo ""
echo "  CREATE:  ✅ PASSED"
echo "  READ:    ${READ_STATUS}"
echo "  UPDATE:  ${UPDATE_STATUS}"
echo "  DELETE:  ${DELETE_STATUS}"
echo ""
echo "Test directory preserved: ${TESTDIR}"
echo ""
echo "Next steps:"
echo "  - Review terraform state: cd ${TESTDIR} && terraform show"
echo "  - Manual UPDATE test: See examples/testing/TESTING-GUIDE.md"
echo "  - Cleanup: rm -rf ${TESTDIR}"
echo ""

# Exit with non-zero if any test failed
if [ "$CREATE_EXIT" -ne 0 ] || [ "$PLAN_EXIT" -gt 2 ] || [ "$DELETE_EXIT" -ne 0 ]; then
    echo "⚠️  Some tests failed or showed warnings"
    exit 1
fi

echo "✅ All automated tests passed!"
