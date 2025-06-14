# Secure Random String (SRS) Implementation Plan

## Objective
Replace all usage of random UUIDs or hex strings with a more k8s-native SRS (secure random string) approach, generating strings of 6-8 characters similar to how k8s names jobs and pods.

## Background
Currently, the codebase uses random UUIDs and hex strings in various places. We need to standardize on a k8s-style naming convention that uses shorter, more readable strings.

## Implementation Tasks

1. **Search and Identify Current UUID/Hex Usage**
   - Use grep to find all instances of UUID generation (`uuid.New()`, `uuid.NewString()`, etc.)
   - Search for hex string generation patterns
   - Create a comprehensive list of all locations that need updating

2. **Check for Existing SRS Implementation**
   - Search for any existing secure random string generation functions
   - Look for k8s-style naming utilities already in the codebase
   - Check if there's a naming utilities package

3. **Implement or Enhance SRS Function**
   - If no SRS function exists, create one in an appropriate utilities package
   - Function should generate 6-8 character strings using alphanumeric characters
   - Follow k8s naming convention: lowercase letters and numbers, starting with a letter
   - Use crypto/rand for secure randomness

4. **Replace UUID/Hex Usage**
   - Systematically replace each UUID/hex generation with the SRS function
   - Ensure the context is appropriate for shorter strings (6-8 chars vs 36 char UUIDs)
   - Update any related tests

5. **Testing and Validation**
   - Run all tests to ensure nothing breaks
   - Create specific tests for the SRS function if they don't exist
   - Verify that generated strings follow k8s naming conventions

## Expected Locations
Based on typical k8s operator patterns, check these locations:
- Controller reconciliation loops
- Resource name generation
- Unique identifier creation
- Test fixtures and mocks

## Commit Strategy
- Commit after implementing the SRS function
- Commit after each major file or package update
- Commit after updating tests
- Final commit after all replacements are complete

Remember to adopt the hack/agent-developer.md persona and follow the Dan Abramov methodology throughout this implementation.