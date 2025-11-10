# Lessons Learned

**Project**: terraform-provider-cyberarksia
**Purpose**: Capture key insights and lessons from development phases
**Last Updated**: 2025-11-10

---

## Phase 2.5: Foundation Improvements

### SDK Research is Critical
- Spent 2+ hours researching ARK SDK via Context7 and Gemini
- Discovered signature mismatch (context vs. profile parameter)
- Found SDK doesn't expose structured errors
- **Takeaway**: Always verify SDK signatures before implementation

### Error Handling Needs Defense in Depth
- String matching alone is brittle
- Layered approach: Go error types → specific patterns → fallback
- Comprehensive test coverage catches edge cases
- **Takeaway**: Test error classification exhaustively

### Documentation Prevents Confusion
- Created `sdk-integration.md` as Phase 3 reference
- Documented SDK limitations clearly
- Saved future debugging time
- **Takeaway**: Document SDK quirks immediately

### Test-Driven Improvement Works
- Tests revealed "temporary failure" wasn't retryable
- Tests proved exponential backoff timing correct
- Tests validated error category uniqueness
- **Takeaway**: Write tests during refactoring, not after

---

## Phase 3: Schema Validation Audit

### Never Assume SDK Behavior
- ALWAYS read actual SDK structs
- Trust `validate:` tags over assumptions
- Field names may differ from expectations

### Cloud Providers Use Generic Fields
- Modern APIs favor platform-agnostic fields
- Provider-specific fields are rare (only when truly needed)
- Don't invent cloud-specific attributes without SDK verification

### Optional ≠ Useless
- SDK provides intelligent defaults
- Over-constraining reduces flexibility
- Users benefit from optional fields with smart defaults

### Validate Early
- Schema audit should have happened in Phase 2
- Would have saved implementation time
- Prevents user confusion from non-functional fields

---

## Phase 3.5: Provider Configuration

### Question Everything
- `request_timeout` sat unused for 3 phases
- Check all configuration parameters actually do something
- Delete unused code aggressively

### Research First
- Industry patterns revealed modern approach
- Comparing to GCP/Azure providers informed decision
- Expert opinion (Gemini) confirmed the direction

### Seek External Opinion
- AI consultation validated architectural decisions
- Second opinions prevent tunnel vision
- Industry best practices trump legacy patterns

### Simplify Ruthlessly
- Fewer parameters = better UX
- 99% of users don't need advanced knobs
- Default to opinionated behavior

### Break Pre-1.0
- Perfect time for architectural improvements
- Breaking changes acceptable before 1.0
- Better now than after public release

---

## Phase 4: Strong Account Implementation

### SDK Method Discovery
- SDK documentation incomplete for secrets methods
- Must read SDK source code directly
- Method names may not match expectations (Secret vs GetSecret)
- **Takeaway**: Verify SDK method signatures in source code

### Sensitive Data Flow
- Understand API contract for sensitive data before implementing CRUD
- Read() may not refresh passwords/keys (by design)
- Mark attributes `Sensitive: true` in schema
- **Takeaway**: API security model informs resource design

### Authentication Type Patterns
- Three different auth patterns with different required fields
- Runtime validation simpler than schema-level for complex conditional logic
- SDK validates missing credentials (returns 400 error)
- **Takeaway**: Defer complex validators when SDK provides validation

---

## Cross-Cutting Insights

### Documentation Strategy
1. Document SDK quirks immediately when discovered
2. Create reference docs alongside implementation
3. Update troubleshooting guide proactively
4. Examples prevent user confusion

### Testing Philosophy
1. Write tests during refactoring, not after
2. Test complex logic, not framework behavior
3. Acceptance tests over unit tests for provider code
4. High coverage for critical paths (error handling, retry logic)

### SDK Integration
1. Always read SDK source code, not just docs
2. Verify all struct fields and method signatures
3. Test assumptions about SDK behavior
4. Document SDK limitations and workarounds

### Configuration Design
1. Fewer parameters = better UX
2. Opinionated defaults preferred
3. Remove unused parameters aggressively
4. Follow modern provider patterns (GCP, Azure, not legacy AWS)

### Pre-1.0 Development
1. Breaking changes are acceptable and encouraged
2. Architectural improvements now vs. later
3. Simplify before locking in API
4. Get external opinions on design decisions
