# Runner Refactoring Questions

## Architecture Questions

### 1. Runner Interface Design ✅ RESOLVED
**Answer**: Yes, both runners should have common methods since they both process environments and build manifests. The only difference is GitHub runner pulls comments.

**Final Interface**:
```go
type RunnerInterface interface {
    Initialize() error
    ProcessEnvironment(environment string) error
    BuildManifests(environment string) ([]byte, []byte, error)
    GenerateReport() error
    OutputResults() error
}
```

### 2. Common vs Specific Logic ✅ RESOLVED
**Answer**: GitHub runner has a specific function to fetch files from head/base branches, but after fetching, it becomes local files. Then both runners use the same kustomize build process.

**Final Organization**:
- **Common Logic**: `processEnvironment()`, `evaluatePolicies()`, `generateReport()`, `kustomizeBuild()`
- **GitHub-specific**: `fetchManifestsFromPR()` - fetches head/base files to local
- **Local-specific**: `getLocalManifestPaths()` - gets local file paths
- **After fetching**: Both runners use the same `kustomizeBuild()` logic

### 3. Data Flow and State Management ✅ RESOLVED
**Answer**: No, runners shouldn't store data. Runners should be stateless and return model structs. The main function stores the results.

**Final Approach**:
- **Runners**: Stateless, return model structs
- **Main function**: Stores all results and data
- **Model structs**: Abstract layer for reports, policies, etc.
- **Context data**: Keep in runners (like GitHub client, options)

### 4. Output Handling ✅ RESOLVED
**Answer**: Similar to question 2. GitHub runner has specific function to post to GitHub PR, but when calling the interface function `OutputResults()`, it will call the GitHub posting logic. This question is not important.

### 5. Main Function Simplification ✅ RESOLVED
**Answer**: Yes, agree with the proposed flow.

## Implementation Questions

### 6. Method Signatures ✅ RESOLVED
**Answer**: Agree with the proposed signatures.

### 7. Dependencies Injection ✅ RESOLVED
**Answer**: These variables (builder, differ, evaluator, reporter, renderer) are all common and used by both run modes. They should be in the base runner, maybe even in the runner interface if needed.

### 8. Error Handling ✅ RESOLVED
**Answer**: Reporting errors back to main is fine.

## Migration Strategy

### 9. Step-by-Step Migration ✅ RESOLVED
**Answer**: Order doesn't matter. Just refactor everything and make sure testing can run by running `make run-local`.

### 10. Testing Strategy ✅ RESOLVED
**Answer**: Don't worry about it for now, just check "Run Local" if it still works.

## Current Issues to Fix

### 11. Compilation Errors ✅ RESOLVED
**Answer**: Sure, can add getter/setter methods if needed, or just make fields public. Doesn't matter much.

### 12. Missing Methods ✅ RESOLVED
**Answer**: There's no general method to solve these, but GitHub Runner has specific GitHub logic and these should be internal functions. Internal functions will be called by public functions, and public functions should match the interface.

## Next Steps

Please address these questions one by one, and I'll help implement the refactoring based on your answers.
