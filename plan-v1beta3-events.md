# V1Beta3 Events Support Implementation Plan

## Objective
Implement support for v1Beta3 events as inbound to a specific server route with a contact channel ID and API key. Create tasks from these events and handle the special "respond_to_human" tool call pattern.

## Background
We need to support incoming v1Beta3 events that create conversations via webhooks. These events contain channel-specific API keys and IDs, and require special handling for human responses.

## Implementation Tasks

1. **Create V1Beta3 Event Types**
   - Define Go structs for V1Beta3ConversationCreated event
   - Include fields: is_test, type, channel_api_key, event (with nested fields)
   - Create appropriate validation for these types

2. **Implement Server Route Handler**
   - Create new HTTP endpoint for v1Beta3 events
   - Parse and validate incoming webhook events
   - Extract channel_api_key and contact_channel_id

3. **Create ContactChannel from Event**
   - Use channel_api_key and contact_channel_id from event
   - Create a new ContactChannel resource dynamically
   - Set appropriate status and validation

4. **Create Task with ContactChannel Reference**
   - Create new Task resource referencing the ContactChannel
   - Include user_message in the initial context
   - Set up proper agent_name from event

5. **Implement Special Response Handling**
   - When task has final answer (no tool calls), don't append to contextWindow
   - Instead, create a "respond_to_human" tool call
   - Pass the content as the argument to this tool call
   - Create ToolCall resource and let it poll until completion

6. **Update HumanLayer Client Integration**
   - Ensure client uses embedded channel_api_key and contact_channel_id
   - Support createHumanContact with proper channel routing
   - Handle both "request_more_information" and "done_for_now" intents

## Event Structure
```go
type V1Beta3ConversationCreated struct {
    IsTest         bool   `json:"is_test"`
    Type           string `json:"type"`
    ChannelAPIKey  string `json:"channel_api_key"`
    Event struct {
        UserMessage      string `json:"user_message"`
        ContactChannelID int    `json:"contact_channel_id"`
        AgentName        string `json:"agent_name"`
    } `json:"event"`
}
```

## Special Handling Logic
- Final answers trigger human contact creation instead of context append
- Support thread_id in state for conversation continuity
- Handle both intermediate questions and final responses

## Dependencies
- Requires ContactChannel with channelApiKey support
- Needs Task with contactChannel reference support
- May need new ToolCall type for "respond_to_human"

## Commit Strategy
- Commit after creating event types
- Commit after implementing route handler
- Commit after ContactChannel creation logic
- Commit after Task creation and special handling
- Commit after testing the full flow

Remember to adopt the hack/agent-developer.md persona and follow the Dan Abramov methodology throughout this implementation.