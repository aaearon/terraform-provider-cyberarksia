<!--
Sync Impact Report:
- Version: Template → 1.0.0
- Change Type: INITIAL RATIFICATION (first concrete constitution from template)
- Modified Principles:
  * Added 8 Core Principles (template had 5 placeholders):
    1. Test-Driven Development (NON-NEGOTIABLE)
    2. Peer Review & Validation (MANDATORY)
    3. Pattern Reuse & Consistency
    4. SDK Constraint Awareness
    5. Documentation Synchronization
    6. Git Workflow Discipline
    7. Incremental Delivery
    8. Security & Sensitive Data
- Added Sections:
  * Technology Constraints (required stack, auth, testing)
  * Development Workflow (Spec-Kit process, review gates, quality gates)
- Templates Status:
  ✅ plan-template.md - "Constitution Check" gate already present
  ✅ spec-template.md - User stories with P1/P2/P3 priorities align with Principle VII
  ✅ tasks-template.md - Task organization by user story aligns with Principle VII
- Follow-up TODOs: None - all placeholders resolved
- Date: 2025-11-02
-->

# Terraform Provider for CyberArk SIA - Constitution

## Core Principles

### I. Test-Driven Development (NON-NEGOTIABLE)

- TDD mandatory: Acceptance tests define behavior → User approves → Tests fail →
  Implement → Tests pass
- Red-Green-Refactor cycle strictly enforced
- CRUD validation required for ALL resources using `examples/testing/TESTING-GUIDE.md`
  workflow before any feature is considered complete
- All SDK workarounds MUST have tests validating the workaround works correctly
- No code ships without passing acceptance tests (TF_ACC=1)

**Rationale**: Real API testing catches integration issues that mocks miss. CyberArk SIA
API behavior cannot be reliably simulated, making acceptance tests essential.

### II. Peer Review & Validation (MANDATORY)

- ALL implementation plans, specifications, and significant code changes MUST be
  peer-reviewed by Codex via `mcp__zen__clink` tool
- Codex is instructed to NEVER update files directly - REVIEW ONLY mode
- Reviews focus on: ARK SDK correctness, Terraform Plugin Framework patterns, bug
  workarounds, naming conventions, architecture decisions
- Gemini is NOT used for peer review due to previous hallucination of non-existent
  SDK services
- Final approval remains with human maintainer after Codex review

**Rationale**: Codex provides code-level deep analysis that complements Claude's
architectural perspective. Previous Gemini hallucinations (K8s clusters, Accounts,
Platforms services) demonstrated the critical need for SDK source code validation.

### III. Pattern Reuse & Consistency

- Reuse established patterns before creating new ones: Profile Factory
  (`internal/provider/profile_factory.go`), Delete Workarounds
  (`internal/client/delete_workarounds.go`), Read-Modify-Write for policy updates,
  Composite IDs (`internal/provider/helpers/composite_ids.go`)
- Naming conventions: Use full words (`virtual_machine` not `vm`, `database` not `db`)
  for consistency, preserve domain-specific SIA terminology (`target_set`, `workspace`)
- Follow existing resource structures (`database_workspace`, `database_secret`,
  `database_policy`) as templates for new VM resources
- Architecture patterns documented in `CLAUDE.md` supersede ad-hoc decisions

**Rationale**: Pattern reuse prevents code duplication (Profile Factory eliminated
410 LOC duplication) and reduces bug surface area. Terraform community expects
consistency across resources within a provider.

### IV. SDK Constraint Awareness

- ARK SDK v1.5.0 has CRITICAL bugs requiring workarounds: DELETE panic bug (all
  Delete operations), VM secrets filtering bug (ListSecretsBy broken)
- ALWAYS use `internal/client/delete_workarounds.go` functions, NEVER call SDK Delete
  methods directly
- No `context.Context` support in `Authenticate()` - handle timeouts at HTTP client
  level
- 15-minute token expiration - rely on SDK automatic refresh, no manual token
  management
- All SDK limitations documented in `docs/sdk-integration.md` and
  `docs/troubleshooting.md`

**Rationale**: SDK bugs are proven in production (reproduced in CyberArk's own `ark`
CLI). Workarounds are temporary but essential until SDK v1.6.0+ fixes arrive.

### V. Documentation Synchronization

- `CLAUDE.md`, `docs/sdk-integration.md`, `docs/troubleshooting.md`,
  `docs/development-history.md` MUST stay synchronized with code changes
- Every new resource requires: `examples/resources/` directory with basic and
  complete examples, `tfplugindocs generate` execution, `TESTING-GUIDE.md` template
  updates
- Implementation summaries go in `docs/development/` with detailed analysis
- Breaking changes require `CHANGELOG.md` updates

**Rationale**: Documentation drift creates onboarding friction and support burden.
Generated documentation (tfplugindocs) ensures consistency between schema and docs.

### VI. Git Workflow Discipline

- Feature branches required (main branch is protected - no direct commits)
- Branch naming conventions: `feature/*` (new features), `fix/*` (bug fixes),
  `docs/*` (documentation only), `refactor/*` (code refactoring), `test/*` (test
  changes), `chore/*` (maintenance)
- Pull requests required even for sole contributors
- Squash merge to main with descriptive commit messages
- Spec-Kit feature branches use numeric prefix: `003-feature-name`,
  `004-feature-name`

**Rationale**: Protected main branch prevents accidental breaking changes. PR
workflow creates audit trail and enables pre-merge validation (CI checks, peer
review).

### VII. Incremental Delivery

- User stories MUST be prioritized (P1, P2, P3) and independently testable
- Each user story is an MVP slice that can be developed, tested, deployed
  independently
- Dependencies between resources explicitly documented in specs (e.g., VM Secrets →
  Target Sets → VM Policies)
- Terraform import functionality required for all resources to support brownfield
  adoption

**Rationale**: Independent user stories enable parallel development and early value
delivery. Import support is critical for users adopting Terraform in existing
environments.

### VIII. Security & Sensitive Data

- NEVER log sensitive data: passwords, tokens, `client_secret`,
  `aws_secret_access_key`, `provisioner_password`
- ALL sensitive attributes marked with `Sensitive: true` in Terraform schema
- Redact secrets in test output, logs, and state files
- Security considerations documented for each resource

**Rationale**: Credential leakage is a critical security risk. Terraform's
`Sensitive` flag prevents accidental exposure in plans/applies but does not
eliminate all risks (state files still contain plaintext).

## Technology Constraints

### Required Stack

- Go 1.25.0
- ARK SDK v1.5.0 (`github.com/cyberark/ark-sdk-golang`)
- Terraform Plugin Framework v1.16.1 (Plugin Framework v6)
- Terraform Plugin Log v0.9.0

### Authentication

- CyberArk Identity OAuth2 service accounts only
- ISP authentication via ARK SDK
- No API key or username/password authentication supported

### Testing Requirements

- CyberArk Identity tenant with SIA enabled required for acceptance tests
- Environment variables: `CYBERARK_USERNAME`, `CYBERARK_PASSWORD`, `TF_ACC=1`
- Real API testing preferred over mocking

## Development Workflow

### Specification Process (Spec-Kit)

1. `/speckit.constitution` - Establish project principles (this document)
2. `/speckit.specify` - Define WHAT to build (requirements, user stories) - NO tech
   stack yet
3. `/speckit.clarify` - Ask clarifying questions for underspecified areas
4. `/speckit.plan` - Define HOW to build (tech stack, architecture, implementation
   details)
5. `/speckit.tasks` - Generate actionable task breakdown with dependencies
6. `/speckit.analyze` - Cross-check consistency across spec, plan, tasks (optional
   quality gate)
7. `/speckit.implement` - Execute all tasks according to plan

### Code Review Gates

- **Pre-implementation**: Codex reviews spec, plan, and tasks
- **Post-implementation**: Codex reviews code changes, test coverage, documentation
  updates
- All reviews captured in feature spec directory for audit trail

### Quality Gates

- All tests pass (unit + acceptance)
- CRUD validation complete per `TESTING-GUIDE.md`
- Documentation generated (`tfplugindocs`)
- Examples created and tested
- Peer review by Codex complete with approval

## Governance

### Constitution Authority

- This constitution supersedes all other development practices and guidelines
- Amendments require: documentation of rationale, approval by project maintainer,
  migration plan for existing code
- All PRs and code reviews must verify compliance with constitution principles
- Complexity and deviations from established patterns must be justified in writing

### Conflict Resolution

- `CLAUDE.md` provides runtime development guidance and implementation details
- Constitution provides governing principles and constraints
- In case of conflict: Constitution principles override `CLAUDE.md` specifics
- Maintainer has final authority on interpretations and exceptions

### Review Compliance

- Use `CLAUDE.md` for pattern reference during development
- Use constitution for decision-making and architectural choices
- Document all exceptions and rationale in feature specs

**Version**: 1.0.0 | **Ratified**: 2025-11-02 | **Last Amended**: 2025-11-02
