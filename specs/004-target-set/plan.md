# Implementation Plan: Target Set Resource

**Branch**: `004-target-set` | **Date**: 2025-11-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-target-set/spec.md`

## Summary

Implement a Terraform resource (`cyberarksia_target_set`) for managing VM/server target sets in CyberArk SIA. Target sets enable platform engineers to define logical groupings of virtual machines and servers that share common access credentials for Just-In-Time (JIT) privileged access. The resource supports three matching patterns (domain, suffix, target) and handles credential rotation, optional ephemeral account naming, and full CRUD operations with drift detection.

**Technical Approach**: Follow established patterns from `cyberarksia_virtual_machine_secret` and `cyberarksia_database_workspace` resources. Use ARK SDK v1.5.0 with DELETE workaround due to known SDK bug. Implement custom plan modifier to prevent clearing `provision_format` once set (audit trail consistency). Handle name-as-ID pattern with proper rename logic.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**:
- ARK SDK v1.5.0 (`github.com/cyberark/ark-sdk-golang`)
- Terraform Plugin Framework v1.16.1 (Plugin Framework v6)
- Terraform Plugin Log v0.9.0

**Storage**: Terraform state only (stateless provider)
**Testing**:
- Acceptance tests with `TF_ACC=1` against live CyberArk SIA tenant
- CRUD validation using `examples/testing/TESTING-GUIDE.md` workflow
- Unit tests for custom validators and plan modifiers

**Target Platform**: Cross-platform (Linux, macOS, Windows) - Terraform provider binary
**Project Type**: Single project (Terraform provider Go module)
**Performance Goals**:
- Sub-second response time for CRUD operations (dependent on CyberArk API)
- Support concurrent target set management across multiple Terraform workspaces

**Constraints**:
- ARK SDK v1.5.0 DELETE bug requires workaround (direct API call)
- ARK SDK v1.5.0 UPDATE bug (missing `name` field deletes resource) requires field inclusion
- API PATCH semantics prevent clearing `provision_format` once set
- Forward slashes in names cause DELETE failures (403) - add validator warning
- No pre-flight validation of `secret_id` references (API constraint)

**Scale/Scope**:
- Single resource implementation (~500-800 LOC based on similar resources)
- 8 schema attributes (id, name, type, secret_id, secret_type, provision_format, description, enable_certificate_validation)
- 5 user stories (P1: core CRUD, P2: operational flexibility, P3: import)
- 15 functional requirements
- Integration with existing VM secrets resource

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Initial Check (Pre-Research)

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Test-Driven Development** | ✅ PASS | Spec requires CRUD validation per TESTING-GUIDE.md. Acceptance tests defined in user stories. |
| **II. Peer Review & Validation** | ⏸️ PENDING | Codex review required post-planning (before implementation). |
| **III. Pattern Reuse & Consistency** | ✅ PASS | Reuses Delete Workarounds pattern, follows `virtual_machine_secret` structure. |
| **IV. SDK Constraint Awareness** | ✅ PASS | DELETE workaround planned, UPDATE name field inclusion documented. |
| **V. Documentation Synchronization** | ✅ PASS | Examples, tfplugindocs, and TESTING-GUIDE.md updates planned. |
| **VI. Git Workflow Discipline** | ✅ PASS | Feature branch `004-target-set` created per Spec-Kit workflow. |
| **VII. Incremental Delivery** | ✅ PASS | User stories prioritized (P1/P2/P3), independently testable. |
| **VIII. Security & Sensitive Data** | ✅ PASS | No sensitive attributes in target sets (references VM secrets by ID only). |

**Overall**: ✅ PASS - No violations, no complexity justifications needed.

### Post-Design Check

*To be completed after Phase 1 design*

## Project Structure

### Documentation (this feature)

```text
specs/004-target-set/
├── spec.md              # Feature specification (/speckit.specify output)
├── plan.md              # This file (/speckit.plan output)
├── research.md          # Phase 0 output (research findings)
├── data-model.md        # Phase 1 output (schema design)
├── quickstart.md        # Phase 1 output (usage examples)
├── contracts/           # Phase 1 output (API contracts)
│   └── target-set-api-contract.md
└── tasks.md             # Phase 2 output (/speckit.tasks - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/provider/
├── target_set_resource.go           # Main resource implementation (NEW)
└── target_set_resource_test.go      # Acceptance tests (NEW)

internal/client/
└── delete_workarounds.go            # Add DeleteTargetSetDirect function (MODIFY)

internal/validators/
└── target_set_name_validator.go     # Forward slash warning validator (NEW)

internal/planmodifiers/
└── prevent_clearing_modifier.go     # Prevent clearing provision_format (NEW)

examples/resources/cyberarksia_target_set/
├── resource.tf                      # Basic example (NEW)
└── complete.tf                      # Complete example with all attributes (NEW)

examples/testing/
└── crud-test-target-set.tf          # CRUD validation template (NEW)

docs/resources/
└── cyberarksia_target_set.md        # Generated documentation (tfplugindocs)

docs/development/
└── target-set-implementation.md     # Implementation summary (NEW)
```

**Structure Decision**: Single project structure. This is a Terraform provider (Go module) with clear separation of concerns: resource implementation in `internal/provider/`, shared utilities in `internal/client/`, custom validators/modifiers in `internal/validators/` and `internal/planmodifiers/`, and examples/documentation following existing patterns.

## Complexity Tracking

> No violations detected - this section intentionally left empty per constitution.

---

## Phase 0: Research & Unknowns Resolution

*Research tasks to resolve "NEEDS CLARIFICATION" items from Technical Context.*

**Status**: No unresolved unknowns - comprehensive technical investigation already completed.

**Findings Summary**: The `docs/development/target-sets-poc-investigation.md` document provides:
- 50 validated tests against live CyberArk tenant (98% pass rate)
- Complete API behavior documentation (PATCH semantics, field mutability, validation gaps)
- SDK bug workarounds (DELETE panic, UPDATE without name field)
- Field-by-field validation results (provision_format constraints, name character limitations)
- All 6 type change combinations validated (bidirectional mutability confirmed)

**Output**: See [research.md](./research.md) for consolidated research findings.

---

## Phase 1: Design & Contracts

*Design artifacts generated after research completion.*

### Data Model

**Output**: See [data-model.md](./data-model.md) for complete schema design with:
- 8 attributes (id, name, type, secret_id, secret_type, provision_format, description, enable_certificate_validation)
- Validation rules (type enum, secret_type enum, name uniqueness, forward slash warning)
- State transitions (name changes, type changes, credential rotation)
- Computed vs. required attributes
- Plan modifiers (prevent clearing provision_format)

### API Contracts

**Output**: See [contracts/target-set-api-contract.md](./contracts/target-set-api-contract.md) for:
- ARK SDK endpoint mappings (`WorkspacesTargetSets()` service)
- Request/response formats for CRUD operations
- Error scenarios and status codes
- DELETE workaround specification
- UPDATE name field requirement

### Quick Start Guide

**Output**: See [quickstart.md](./quickstart.md) for:
- Installation and configuration
- Basic usage example (domain-based target set)
- All three matching pattern examples (domain, suffix, target)
- Credential rotation example
- Import example
- Common troubleshooting scenarios

### Agent Context Update

*Agent-specific context files updated with new technology/patterns from this plan.*

**Updated**: `.specify/memory/agent/claude/context.md` with:
- Target set resource implementation patterns
- Custom plan modifier for provision_format
- Forward slash name validation
- Name-as-ID pattern handling

---

## Phase 2: Implementation Breakdown

*This section populated by `/speckit.tasks` command - NOT by /speckit.plan*

**Next Command**: `/speckit.tasks` to generate actionable task breakdown.

---

## Constitution Re-Check (Post-Design)

| Principle | Status | Notes |
|-----------|--------|-------|
| **I. Test-Driven Development** | ✅ PASS | Acceptance tests designed in data-model.md, CRUD validation workflow defined. |
| **II. Peer Review & Validation** | ⏸️ PENDING | Codex review scheduled post-tasks generation, pre-implementation. |
| **III. Pattern Reuse & Consistency** | ✅ PASS | Confirmed reuse of existing patterns (Delete Workarounds, resource structure). |
| **IV. SDK Constraint Awareness** | ✅ PASS | DELETE workaround designed, UPDATE name field requirement documented. |
| **V. Documentation Synchronization** | ✅ PASS | Examples designed, tfplugindocs plan confirmed, TESTING-GUIDE.md update planned. |
| **VI. Git Workflow Discipline** | ✅ PASS | Feature branch workflow maintained. |
| **VII. Incremental Delivery** | ✅ PASS | User stories remain independently testable in design. |
| **VIII. Security & Sensitive Data** | ✅ PASS | No sensitive data handling in target sets (references only). |

**Overall**: ✅ PASS - Design adheres to all constitution principles.

---

**Plan Status**: Phase 0 & Phase 1 COMPLETE
**Next Step**: Run `/speckit.tasks` to generate implementation tasks
