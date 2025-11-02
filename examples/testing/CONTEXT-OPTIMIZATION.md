# Context Optimization for Terraform Operations

## The Problem: Context Burning on Long Operations

When running Terraform operations that produce large outputs (especially `terraform apply` for infrastructure provisioning), reading the entire output consumes significant token context. This is especially problematic for:

- Long-running operations (PostgreSQL provisioning: 5-10 minutes)
- Operations with verbose output (resource creation logs)
- Repeated operations during testing

## Solution: Background Execution with Log Files

### Pattern 1: Using Bash with run_in_background

```bash
# Run terraform in background, redirect output to log file
terraform apply -auto-approve > /tmp/tf-apply.log 2>&1
```

Then periodically check completion:
```bash
# Check if apply succeeded (exit code in log)
if grep -q "Apply complete" /tmp/tf-apply.log; then
    echo "✅ Apply succeeded"
    # Show only summary
    grep "Resources:" /tmp/tf-apply.log
fi
```

### Pattern 2: Using Setup Scripts (Recommended)

The provided `setup.sh` and `cleanup.sh` scripts implement this pattern:

```bash
# Run terraform in background with progress spinner
terraform apply -auto-approve tfplan > /tmp/tf-apply.log 2>&1 &
TF_PID=$!

# Show spinner without burning context
while kill -0 $TF_PID 2> /dev/null; do
    # Show progress indicator
    printf "\r   Creating resources... "
    sleep 0.5
done

# Check result and show only summary
wait $TF_PID
if [ $? -eq 0 ]; then
    echo "✅ Success"
    # Show only relevant summary lines
    terraform output validation_summary
fi
```

## Best Practices

### 1. Always Use Log Files for Long Operations

**Bad** (burns tokens):
```bash
terraform apply -auto-approve
# LLM reads entire 10,000+ line output
```

**Good** (saves context):
```bash
terraform apply -auto-approve > /tmp/tf-apply.log 2>&1
echo "Apply completed. Summary:"
grep -A 5 "Apply complete" /tmp/tf-apply.log
```

### 2. Extract Only Relevant Information

**Bad** (burns tokens):
```bash
cat /tmp/tf-apply.log
# LLM reads entire log file
```

**Good** (saves context):
```bash
# Show only errors if failed
if ! grep -q "Apply complete" /tmp/tf-apply.log; then
    echo "❌ Apply failed:"
    tail -30 /tmp/tf-apply.log
fi

# Show only summary if succeeded
terraform output -json | jq '.validation_summary.value'
```

### 3. Use Scripts for Repeated Workflows

Instead of running Terraform commands directly:
```bash
# One-time setup with efficient logging
./setup.sh

# Later cleanup
./cleanup.sh
```

Scripts handle:
- Log file management
- Progress tracking without verbose output
- Error extraction
- Summary generation

### 4. Terraform Output Filtering

**Bad** (burns tokens):
```bash
terraform output
# Shows all outputs, many redundant
```

**Good** (saves context):
```bash
# Get specific output only
terraform output -raw policy_id

# Or filtered JSON
terraform output -json | jq -r '.validation_summary.value.sia_resources'
```

### 5. Use -target for Incremental Testing

**Bad** (burns tokens):
```bash
terraform apply -auto-approve
# Creates all resources, produces massive output
```

**Good** (saves context):
```bash
# Create resources incrementally with focused output
terraform apply -target=azurerm_postgresql_flexible_server.sia_test -auto-approve | grep -A 3 "Apply complete"
terraform apply -target=cyberarksia_database_secret.admin -auto-approve | grep -A 3 "Apply complete"
```

## Implementation in This Project

### setup.sh
- Runs `init` → log file
- Runs `plan` → log file
- Runs `apply` → log file with progress spinner
- Shows only summary on completion
- **Token savings**: ~95% vs reading full output

### cleanup.sh
- Runs `destroy` → log file with progress spinner
- Verifies cleanup with minimal output
- **Token savings**: ~90% vs reading full output

### Log File Locations

All Terraform operations write to `/tmp/`:
- `/tmp/tf-init.log` - Initialization output
- `/tmp/tf-plan.log` - Plan output
- `/tmp/tf-apply.log` - Apply output
- `/tmp/tf-destroy.log` - Destroy output

These can be reviewed if errors occur, but aren't loaded into context unless needed.

## Example: Context-Efficient Testing Workflow

```bash
# Step 1: Setup (efficient logging)
cd ~/terraform-provider-cyberark-sia/examples/testing/azure-postgresql-with-policy
./setup.sh
# Output: ~200 lines (summary only)
# Actual Terraform output: ~10,000 lines (saved to /tmp/ log files)

# Step 2: Verify (targeted query)
terraform output -raw policy_id
# Output: 1 line (UUID only)

# Step 3: Manual verification
# User checks SIA UI, Azure Portal
# No LLM context consumed

# Step 4: Cleanup (efficient logging)
./cleanup.sh
# Output: ~100 lines (summary only)
# Actual Terraform output: ~5,000 lines (saved to /tmp/ log files)
```

**Total context consumed**: ~300 lines
**Without optimization**: ~15,000 lines
**Context savings**: 98%

## Error Handling

When operations fail, scripts show only the relevant error context:

```bash
if [ $? -ne 0 ]; then
    echo "❌ Operation failed!"
    echo "Last 30 lines of log:"
    tail -30 /tmp/tf-apply.log
    exit 1
fi
```

This provides error context (~600 tokens) instead of full output (~50,000 tokens).

## Monitoring Long Operations

For very long operations (Azure PostgreSQL: 5-10 minutes):

```bash
# Start in background
terraform apply -auto-approve > /tmp/tf-apply.log 2>&1 &
TF_PID=$!

# Periodic checks without reading full log
while kill -0 $TF_PID 2> /dev/null; do
    # Count completed resources
    COMPLETED=$(grep -c "Creation complete" /tmp/tf-apply.log 2>/dev/null || echo 0)
    printf "\r   Resources created: $COMPLETED"
    sleep 2
done
```

Shows progress (single line, updates in place) without consuming context.

## Summary

**Use scripts for all Terraform operations**:
- `setup.sh` - Initialize, plan, apply with efficient logging
- `cleanup.sh` - Destroy with verification
- Log files capture full output for troubleshooting
- LLM context shows only summaries and errors

**Context savings**:
- Traditional approach: 50,000-100,000 tokens per test cycle
- Optimized approach: 500-1,000 tokens per test cycle
- **Savings: 98-99%**
