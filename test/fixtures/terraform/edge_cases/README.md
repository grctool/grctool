# Edge Cases Test Fixtures

## Purpose
Test edge cases and error handling for the Terraform indexer and analysis tools.

## Test Scenarios

### 1. Large Scale Infrastructure (`large_scale.tf`)
- **Purpose:** Performance testing
- **Content:** 100+ resources to test indexing performance
- **Expected:** Index build completes in <5 seconds

### 2. Complex Module References (`complex_modules.tf`)
- **Purpose:** Test module resolution and dependency tracking
- **Content:** Module calls with complex variable passing
- **Expected:** Correct dependency mapping

### 3. Invalid HCL Syntax (`invalid_hcl.tf`)
- **Purpose:** Test error handling
- **Content:** Deliberately invalid Terraform syntax
- **Expected:** Graceful error handling, fallback to regex scanner

### 4. Circular Dependencies (`circular_deps.tf`)
- **Purpose:** Test dependency cycle detection
- **Content:** Resources with circular `depends_on`
- **Expected:** Detect and report circular dependencies

## Usage

These fixtures should be used in error handling and edge case tests:

```go
func TestIndexer_InvalidHCL(t *testing.T) {
    // Should handle gracefully without crashing
    _, err := indexer.BuildIndex("test/fixtures/terraform/edge_cases/invalid_hcl.tf")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "HCL parsing failed")
}

func TestIndexer_LargeScale(t *testing.T) {
    // Should complete in reasonable time
    start := time.Now()
    index, err := indexer.BuildIndex("test/fixtures/terraform/edge_cases/large_scale.tf")
    duration := time.Since(start)

    assert.NoError(t, err)
    assert.Greater(t, len(index.Resources), 100)
    assert.Less(t, duration, 5*time.Second)
}
```
