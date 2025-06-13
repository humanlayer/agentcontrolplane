# Merge Agent Plan

## Objective
Monitor the progress of 6 worker agents and incrementally merge their changes into the integration branch, handling conflicts and ensuring clean builds.

## Context
Six worker agents are implementing different features in parallel:
1. SRS Implementation (acp-srs-*)
2. ContactChannel Project ID (acp-projectid-*)
3. ContactChannel Task Spec (acp-taskspec-*)
4. Channel API Key/ID (acp-channelapikey-*)
5. V1Beta3 Events Support (acp-v1beta3-*)
6. Parallel LLM Calls Fix (acp-parallel-*)

## Merge Strategy and Dependencies

### Phase 1: Foundation Changes (No Dependencies)
1. **SRS Implementation** - Can be merged first as it's a utility change
2. **Parallel LLM Calls Fix** - Bug fix that doesn't depend on other changes

### Phase 2: ContactChannel Enhancements (Dependent on Each Other)
1. **Channel API Key/ID** - Adds new fields to ContactChannel
2. **ContactChannel Project ID** - Depends on API key functionality
3. **ContactChannel Task Spec** - Depends on ContactChannel being ready

### Phase 3: Integration Features
1. **V1Beta3 Events Support** - Depends on all ContactChannel features

## Monitoring and Merge Process

1. **Initial Setup**
   - Check out integration branch
   - Verify clean build state
   - List all worker branches

2. **Continuous Monitoring Loop**
   - Every 2 minutes (sleep 120):
     - Check each worker branch for new commits
     - Identify branches ready for merging
     - Merge in dependency order

3. **Merge Procedure for Each Branch**
   - Check for new commits: `git log --oneline -3 [branch-name]`
   - If new commits found:
     - Attempt merge: `git merge [branch-name]`
     - Run tests: `make -C acp fmt vet lint test`
     - If conflicts, resolve based on feature priority
     - Commit merge if successful

4. **Conflict Resolution Strategy**
   - For CRD changes: Take union of fields
   - For controller changes: Ensure all features work together
   - For test changes: Include all tests
   - Always maintain backward compatibility

5. **Build Validation**
   - After each merge:
     - Run full test suite
     - Deploy to local kind cluster
     - Verify controller starts successfully
     - Check for any runtime errors

## Branches to Monitor
- acp-srs-[suffix]
- acp-projectid-[suffix]
- acp-taskspec-[suffix]  
- acp-channelapikey-[suffix]
- acp-v1beta3-[suffix]
- acp-parallel-[suffix]

## Success Criteria
- All changes merged without conflicts
- All tests passing
- Controller deploys and runs successfully
- No duplicate or conflicting implementations
- Clean commit history maintained

## Adoption Note
Adopt the hack/agent-merger.md persona for this task. Focus on systematic merging, thorough testing, and maintaining code quality throughout the integration process.