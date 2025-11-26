# Quick Start - Session 7

## TL;DR
Fix the schema so RDP-only and SSH-only policies work, then complete remaining User Stories.

## 5-Minute Start

```bash
# 1. Setup
cd /home/tim/terraform-provider-cyberarksia
git checkout 001-vm-access-policies
export CYBERARK_USERNAME="timtest@cyberark.cloud.40562"
export CYBERARK_PASSWORD='nvk*phv*hfd3ATR2rfc'
export TF_ACC=1

# 2. Read context
cat specs/001-vm-access-policies/HANDOFF.md | grep -A 50 "Session 6"

# 3. The fix (in vm_policy_resource.go line 277+)
# Change: SingleNestedBlock → SingleNestedAttribute
# For: behavior.ssh, behavior.rdp, rdp.local_ephemeral_user, rdp.domain_ephemeral_user

# 4. Test
go test ./internal/provider -v -run "TestAccVMPolicy_rdp" -timeout 20m
# Expected: 8/8 tests pass (currently 1/8)
```

## The Problem
```
User tries: behavior { rdp { ... } }
Terraform says: "SSH username required"
Why: SingleNestedBlock with Required attrs makes parent required
Fix: Change to SingleNestedAttribute
```

## Key Files
- `internal/provider/vm_policy_resource.go` (lines 274-330) - Schema definition
- `specs/001-vm-access-policies/rdp-only-api-structure.json` - API reference
- `specs/001-vm-access-policies/SESSION-7-PROMPT.md` - Full instructions

## Success = All Green
```bash
go test ./internal/provider -v -run "TestAccVMPolicy" -timeout 30m
# Target: 21/21 tests pass (currently 13/21)
```

## Reference
- HashiCorp Docs: https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes/single-nested
- Issue #740: https://github.com/hashicorp/terraform-plugin-framework/issues/740
