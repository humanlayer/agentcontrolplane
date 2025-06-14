# Parallel LLM Calls Bug Fix Plan

## Objective
Fix the bug where multiple LLM calls happen in parallel, causing invalid payloads to be sent to LLMs and race conditions in the reconciliation loop.

## Background
When a task involves multiple tool calls that can run in parallel (like fetching data from multiple endpoints), the system incorrectly handles concurrent LLM interactions, leading to:
- Invalid payloads sent to LLMs
- Multiple "SendingContextWindowToLLM" events
- Multiple "LLMFinalAnswer" events
- Race conditions in controller reconciliation

## Reproduction Example
Task: "fetch the data at https://lotrapi.co./api/v1/characters and then fetch data about two of the related locations"
This causes a many-turn conversation with multiple parallel tool calls.

## Implementation Tasks

1. **Analyze Current Parallel Processing**
   - Read Task controller reconciliation logic thoroughly
   - Understand how tool calls are processed
   - Identify where parallel execution happens
   - Find the race condition sources

2. **Implement Proper Synchronization**
   - Add mutex or other synchronization mechanism
   - Ensure only one LLM call happens at a time per task
   - Properly queue or serialize LLM interactions
   - Maintain correct context window state

3. **Fix Event Generation**
   - Prevent duplicate "SendingContextWindowToLLM" events
   - Ensure single "LLMFinalAnswer" per LLM interaction
   - Fix "ValidationSucceeded" duplicate events
   - Add proper event deduplication

4. **Handle Parallel Tool Calls Correctly**
   - Allow tools to execute in parallel (this is good)
   - But serialize LLM interactions (one at a time)
   - Maintain proper state between LLM calls
   - Ensure context window consistency

5. **Testing and Validation**
   - Create test case with parallel tool calls
   - Verify no duplicate events
   - Ensure valid LLM payloads
   - Test with the LOTR API example
   - Add regression tests

## Expected Symptoms to Fix
- Multiple "SendingContextWindowToLLM" events in quick succession
- Multiple "LLMFinalAnswer" events for same interaction
- Controller reconciling same resource multiple times
- Invalid or corrupted LLM payloads

## Key Areas to Check
- Task controller reconciliation loop
- Tool call completion handling
- LLM client interaction code
- Event recording logic
- Controller requeue behavior

## Commit Strategy
- Commit after analyzing and understanding the issue
- Commit after implementing synchronization fix
- Commit after fixing event generation
- Commit after adding tests
- Final commit with any cleanup

Remember to adopt the hack/agent-developer.md persona and follow the Dan Abramov methodology throughout this implementation.