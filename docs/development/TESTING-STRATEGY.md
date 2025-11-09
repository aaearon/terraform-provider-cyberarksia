# Testing Strategy

**Last Updated**: 2025-11-09
**Purpose**: Define testing philosophy and guidelines for the CyberArk SIA Terraform Provider

## Philosophy

> **"Tests should catch unique bugs, not test framework behavior."**

Our testing strategy follows HashiCorp's guidance: **prefer acceptance testing over unit testing**. Each test should answer: *"What bug does this test catch that others don't?"* If the answer is "none", the test should be deleted.

---

## Testing Approach

### Acceptance Tests (Primary) ✅

**When to Use**: All resources and data sources (REQUIRED)

**Purpose**:
- Verify real Terraform lifecycle behavior (plan, apply, refresh, destroy)
- Test against actual API using real credentials
- Guarantee provider works as users expect

**Coverage Target**:
- All CRUD operations (Create, Read, Update, Delete)
- ImportState functionality
- ForceNew behavior
- Error cases and validation
- Drift detection
- Edge cases (multiple assignments, updates, etc.)

**Example** (Good Acceptance Test):
```go
func TestAccDatabaseWorkspace_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read
            {
                Config: testAccDatabaseWorkspaceConfigBasic,
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttrSet("cyberarksia_database_workspace.test", "id"),
                    resource.TestCheckResourceAttr("cyberarksia_database_workspace.test", "database_type", "postgres"),
                ),
            },
            // ImportState
            {
                ResourceName:      "cyberarksia_database_workspace.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

**What Makes a Good Acceptance Test**:
- ✅ Tests real Terraform configurations
- ✅ Exercises full lifecycle
- ✅ Tests business logic, not implementation details
- ✅ Each test adds unique value
- ✅ Clear test name describing scenario

---

### Unit Tests (Selective) ⚠️

**When to Use**:
- Complex utility functions (ID parsing, formatters)
- Error handling logic (retry, error classification)
- Critical infrastructure code (shared helpers)

**When NOT to Use**:
- Simple validators (regex, list membership)
- Testing framework behavior
- Testing standard library functions
- Testing SDK constants or enums

**Coverage Target**:
- Representative test cases, not exhaustive
- 5-8 cases per function typically sufficient
- Focus on edge cases and error paths

**Example** (Good Unit Test - Utility):
```go
func TestBuildCompositeID(t *testing.T) {
    tests := []struct {
        name     string
        parts    []string
        expected string
    }{
        {
            name:     "two parts",
            parts:    []string{"policy-123", "db-456"},
            expected: "policy-123:db-456",
        },
        {
            name:     "empty parts edge case",
            parts:    []string{"", ""},
            expected: ":",
        },
    }
    // ... test execution
}
```

**Example** (Bad Unit Test - Over-Testing):
```go
// ❌ DON'T: Testing every value from SDK list
func TestDatabaseEngineValidator(t *testing.T) {
    tests := []struct {
        name  string
        value string
    }{
        {"valid postgres", "postgres"},
        {"valid mysql", "mysql"},
        {"valid mariadb", "mariadb"},
        // ... 60 more identical cases
    }
}

// ✅ DO: Test representative cases
func TestDatabaseEngineValidator(t *testing.T) {
    tests := []struct {
        name      string
        value     string
        expectErr bool
    }{
        {"valid generic engine", "postgres", false},
        {"valid platform-specific", "postgres-aws-rds", false},
        {"invalid unknown engine", "invalid", true},
        {"invalid case sensitivity", "POSTGRES", true},
        {"null value skips validation", nil, false},
    }
}
```

---

## Test Selection Criteria

### Representative vs Exhaustive Testing

**Exhaustive Testing** (❌ Avoid for simple validators):
```
Test every possible valid input +
Test every possible invalid input +
Test every edge case
= Diminishing returns after ~10 cases
```

**Representative Testing** (✅ Prefer):
```
Test 2-3 valid happy paths (different categories) +
Test 2-3 invalid error paths (different failures) +
Test 1-2 edge cases (null, unknown)
= 5-8 cases per validator (sufficient for 100% coverage)
```

### The "Unique Value" Question

Before adding a test, ask:

> **"What bug does this test catch that my other tests don't?"**

If the answer is:
- ❌ "It tests another valid value" → DELETE (redundant)
- ❌ "It tests framework behavior" → DELETE (not our responsibility)
- ❌ "It increases coverage percentage" → DELETE (false metric)
- ✅ "It tests a different error path" → KEEP (unique value)
- ✅ "It catches a regression from production bug" → KEEP (proven value)
- ✅ "It tests complex business logic" → KEEP (high value)

---

## Red Flags (Over-Testing)

### Indicators of Test Bloat

| Indicator | Threshold | Action |
|-----------|-----------|--------|
| **Test-to-code ratio** | > 3:1 for simple validators | Review for redundancy |
| **Test cases** | > 15 cases for single function | Consolidate to representative |
| **Repeated patterns** | Same assertion, different values | Reduce to 2-3 examples |
| **Testing SDK/stdlib** | Validating list membership | Remove, trust upstream |

### Common Anti-Patterns

❌ **Testing every enum value**:
```go
// Bad: 65 test cases for list membership check
{"valid postgres", "postgres", false},
{"valid mysql", "mysql", false},
{"valid mariadb", "mariadb", false},
// ... 62 more
```

❌ **Testing regex variations exhaustively**:
```go
// Bad: 43 test cases for simple email regex
{"valid with dots", "user.name@example.com", false},
{"valid with hyphens", "user-name@example.com", false},
{"valid with underscores", "user_name@example.com", false},
// ... 40 more variations
```

❌ **Testing framework behavior**:
```go
// Bad: Testing Terraform's null/unknown handling
{"null returns no error", types.StringNull(), false},
// Framework guarantees this, don't retest
```

✅ **Testing business logic**:
```go
// Good: Testing unique error conditions
{"invalid composite ID missing separator", "policy123", true},
{"invalid composite ID empty parts", ":", true},
{"invalid composite ID wrong principal type", "p:u:ADMIN", true},
```

---

## Guidelines by Component Type

### Validators (Simple)

**Target**: 5-8 test cases
- 2-3 valid cases (different categories)
- 2-3 invalid cases (different failures)
- 1-2 edge cases (null, unknown)

**Anti-pattern**: Testing all valid inputs from a list

### Validators (Complex with Business Logic)

**Target**: 10-15 test cases
- Valid cases covering all code branches
- Invalid cases for each validation rule
- Edge cases for boundary conditions

**Example**: `profile_validator` (validates mutual exclusivity)

### Helper Functions

**Target**: 8-12 test cases
- Happy path with representative inputs
- Error paths for each error type
- Edge cases (empty strings, special characters, nil)

**Example**: `ParsePolicyDatabaseID()`, `BuildCompositeID()`

### Client/Retry Logic

**Target**: 10-20 test cases
- Success scenarios
- Transient failures (retries succeed)
- Permanent failures (retries exhausted)
- Edge cases (timeouts, connection errors)

**Justification**: Infrastructure code is high-value and complex

### Resources (Acceptance Only)

**Target**: 10-15 acceptance tests per resource
- Basic CRUD lifecycle
- Each authentication method (if applicable)
- Update scenarios
- ForceNew triggers
- Error validation
- Import functionality
- Drift detection

**No unit tests needed**: Acceptance tests cover behavior

---

## Test Organization

### File Structure

```
internal/
├── validators/
│   ├── email_like_validator.go          (51 lines)
│   ├── email_like_validator_test.go     (80 lines) ← 1.6:1 ratio ✅
│   ├── database_engine_validator.go     (84 lines)
│   └── database_engine_validator_test.go (128 lines) ← 1.5:1 ratio ✅
├── provider/
│   ├── database_workspace_resource.go
│   └── database_workspace_resource_test.go  ← Acceptance tests only
└── helpers/
    ├── composite_ids.go
    └── composite_ids_test.go  ← Unit tests for utilities
```

### Naming Conventions

**Acceptance Tests**:
```go
func TestAccResourceName_scenario(t *testing.T)
func TestAccDatabaseWorkspace_basic(t *testing.T)
func TestAccDatabaseWorkspace_withConditions(t *testing.T)
```

**Unit Tests**:
```go
func TestFunctionName(t *testing.T)
func TestFunctionName_edgeCase(t *testing.T)
func TestEmailLikeValidator(t *testing.T)
func TestEmailLikeValidator_Description(t *testing.T)
```

---

## Metrics & Quality Gates

### Acceptable Ratios

| Component | Test-to-Code Ratio | Acceptance% | Assessment |
|-----------|-------------------|-------------|------------|
| **Resources** | 1.0-1.5:1 | 100% | ✅ Good |
| **Simple Validators** | 1.5-2.5:1 | 0% | ✅ Good |
| **Complex Validators** | 2.0-3.0:1 | 0% | ✅ Good |
| **Helpers/Utils** | 1.5-2.5:1 | 0% | ✅ Good |
| **Client/Retry** | 1.5-2.5:1 | 0% | ✅ Good |

### Code Review Checklist

When reviewing test PRs, verify:

- [ ] **Unique Value**: Each test catches different bugs
- [ ] **No Framework Testing**: Not testing Terraform/Go stdlib behavior
- [ ] **Representative Coverage**: Not exhaustive, just sufficient
- [ ] **Clear Names**: Test name describes scenario clearly
- [ ] **Appropriate Type**: Acceptance for resources, unit for utils only
- [ ] **Ratio Check**: Test-to-code ratio < 3:1 for validators
- [ ] **Business Logic**: Tests behavior, not implementation

### Quality Over Quantity

**Bad Metrics**:
- ❌ Number of test cases
- ❌ Lines of test code
- ❌ Test-to-code ratio alone

**Good Metrics**:
- ✅ Bug detection rate (tests that caught real bugs)
- ✅ Confidence in deployments
- ✅ Ease of refactoring
- ✅ Clarity of test failures

---

## Migration Guide

### Consolidating Existing Tests

**Before** (email_like_validator_test.go - 289 lines, 43 cases):
```go
{"valid standard email", "user@example.com", false},
{"valid email with dots", "john.doe@company.com", false},
{"valid with numbers in local part", "user123@test.com", false},
{"valid with multiple dots in domain", "user.123@test.domain.co.uk", false},
{"valid with hyphens in local part", "first-last@company.com", false},
// ... 38 more similar cases
```

**After** (80 lines, 13 cases):
```go
{"valid standard email", "user@example.com", false},
{"valid CyberArk cloud format", "tim@cyberark.cloud.12345", false},
{"valid complex with special chars", "user.name+tag-test_123@sub.domain.co.uk", false},
{"invalid no @ symbol", "username.example.com", true},
{"invalid missing domain", "user@", true},
{"invalid missing TLD", "user@domain", true},
{"invalid spaces", "user name@example.com", true},
{"null value skips validation", nil, false},
```

**Result**: 72% reduction, same coverage, clearer intent

---

## References

### HashiCorp Official Guidance

> "Similar to other provider concepts, many provider developers **prefer acceptance testing over unit testing**."
> — [Terraform Plugin Framework Testing](https://developer.hashicorp.com/terraform/plugin/framework/acctests)

> Unit tests are "better suited for functions requiring **extensive input value testing**"
> — [Terraform Plugin Framework Functions Testing](https://developer.hashicorp.com/terraform/plugin/framework/functions/testing)

### Industry Best Practices

1. **Test Pyramid**: Few E2E tests (acceptance), more integration (acceptance for Terraform), many unit (selective)
2. **Test Behavior, Not Implementation**: Focus on outcomes, not internal mechanics
3. **Avoid Testing Third-Party Code**: Trust framework and stdlib, test your logic
4. **Each Test Must Have Value**: If it doesn't catch unique bugs, delete it

---

## Appendix: Historical Changes

### 2025-11-09: Test Consolidation

**Motivation**: Validator tests had excessive redundancy (4.5:1 ratio vs 1.5:1 target)

**Changes**:
- Reduced `database_engine_validator_test.go`: 424 → 128 lines (70% reduction)
- Reduced `email_like_validator_test.go`: 289 → 132 lines (54% reduction)
- Reduced other validator tests by 60-70%

**Impact**:
- Total test code: 10,250 → ~8,800 lines (14% reduction)
- Validator test ratio: 4.5:1 → 1.5:1
- Same code coverage, clearer test intent
- Faster CI/CD, easier maintenance

**Principle Applied**: Representative over exhaustive testing

---

## Summary

**DO**:
- ✅ Write acceptance tests for all resources/data sources
- ✅ Use unit tests for complex utilities and infrastructure code
- ✅ Test representative cases covering different categories
- ✅ Focus on business logic and unique error paths
- ✅ Ask "what unique bug does this catch?" before adding tests

**DON'T**:
- ❌ Test every value in SDK lists or enums
- ❌ Test framework or standard library behavior
- ❌ Write 50+ cases for simple validators
- ❌ Test implementation details
- ❌ Add tests just to increase coverage percentage

**Remember**: Quality over quantity. 10 valuable tests beat 100 redundant ones.
